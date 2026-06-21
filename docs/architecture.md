# Architecture

This document describes the internal design of the Feature Management Service (FMS): its goals, components, data model, caching strategy, evaluation algorithm, and scaling considerations.

---

## Table of Contents

- [System Design Goals](#system-design-goals)
- [Component Overview](#component-overview)
- [Data Model](#data-model)
- [Caching Strategy](#caching-strategy)
- [Evaluation Engine](#evaluation-engine)
- [Rollout Algorithm](#rollout-algorithm)
- [Explainability Model](#explainability-model)
- [Consistency Guarantees](#consistency-guarantees)
- [Scaling Considerations](#scaling-considerations)
- [SDK Design](#sdk-design)

---

## System Design Goals

The FMS was designed around three non-negotiable requirements:

### Latency

Flag evaluation must be fast enough to be called on every HTTP request inside product services without meaningfully affecting those services' response times. The target is:

- **p50**: < 200 microseconds
- **p99**: < 1 millisecond
- **p999**: < 5 milliseconds

This rules out any design that goes to the network on the hot path. The answer is in-process caching (L1) for the common case, with Redis (L2) as a fast fallback before touching the database.

### Throughput

The platform targets ~100,000 evaluations/second across all instances. Because individual evaluations are CPU-light (hash computation + in-memory rule matching), the bottleneck is the cache layer, not compute. Horizontal scaling of FMS instances combined with a shared Redis cluster provides the necessary headroom.

### Consistency Model

FMS is **eventually consistent** with a bounded staleness guarantee. After a flag is written:

- Instances that receive the Redis pub/sub invalidation event clear their caches immediately and serve fresh data on the next request (typically within 100ms of the write).
- Instances that miss the invalidation event (e.g., due to a network partition) continue serving the cached value for at most 30 seconds (the L1 TTL), after which the L2 TTL of 5 minutes bounds Redis staleness.

This is an intentional trade-off: strict consistency would require network round-trips on every evaluation, violating the latency goal. For feature flags — where a few seconds of staleness is always acceptable — eventual consistency is the right model.

---

## Component Overview

### REST API (Spring MVC)

The HTTP layer handles both management operations (CRUD on flags, rules, segments) and evaluation requests. It is responsible for:

- Authentication (Bearer token / API key validation)
- Request validation and deserialization
- Routing to the appropriate service layer
- Response serialization and error formatting
- Emitting HTTP metrics and trace spans

The router uses [chi](https://github.com/go-chi/chi) for its lightweight middleware chain and good performance characteristics. Every request gets a unique `request_id` header injected by middleware, which is propagated through logs and traces.

### Evaluation Engine

The core business logic component. It accepts an `EvaluationContext` (entity ID, entity type, attribute map, flag key) and returns a `EvaluationResult` (variant value + full explanation).

Internally it:

1. Resolves the flag definition from the cache hierarchy
2. Sorts the flag's rules by priority (ascending integer, lower = higher priority)
3. Iterates rules, applying the condition-matching and rollout-bucketing logic
4. Builds the step-by-step explanation as it goes
5. Returns on the first rule match, or the flag's default value if no rule matches

The engine is stateless and goroutine-safe. Multiple goroutines evaluate different flags concurrently without contention.

### Tiered Cache

A three-tier read-through cache in front of PostgreSQL.

- **L1 (Caffeine)**: An in-process cache using the [Caffeine](https://github.com/ben-manes/caffeine) library with window-TinyLFU eviction. This is the fastest possible path: a memory lookup with ~200ns latency. Each FMS instance has its own L1; there is no coordination between instances at this tier.
- **L2 (Redis 7)**: A shared Redis cache accessible to all instances. When L1 misses, the engine queries Redis. If found, the value is stored back into L1. Redis latency is ~1ms over the internal network.
- **L3 (PostgreSQL 16)**: The database is the source of truth. It is only read when both L1 and L2 miss. After a DB read, the result is written into both L2 and L1. A `singleflight.Group` prevents multiple goroutines from issuing redundant DB reads for the same flag key simultaneously.

### Redis Pub/Sub Subscriber

Each FMS instance maintains a persistent connection to Redis and runs a background goroutine that subscribes to `ff:invalidate:*` and `seg:invalidate:*` using Redis PSUBSCRIBE. When an invalidation message arrives, the goroutine evicts the corresponding key from L1 and issues a `DEL` to L2. This ensures that after a flag write, all instances converge on fresh data quickly.

### PostgreSQL 16

The relational database stores all persistent state: applications, environments, flags, rules, segments, variants, and audit events. It is the only place writes go. The schema uses UUID primary keys, optimistic concurrency via a `version` integer column, and foreign key constraints to maintain referential integrity.

---

## Data Model

The schema is organized around a hierarchy: **Applications** contain **Environments**, which contain **Flags**, which have **Rules** and **Variants**.

### Core Entities

**applications**
The top-level grouping. Each application (e.g., "checkout-service", "mobile-app") has a unique slug used in cache keys and API paths.

**environments**
An environment belongs to one application and represents a deployment stage (e.g., "production", "staging", "development"). Flag state is fully isolated per environment; the same flag key can have different rules and be enabled/disabled independently in each environment.

**flags**
A flag belongs to one application/environment pair and carries:
- A unique `flag_key` within its environment (e.g., `checkout.new_ui`)
- A `flag_type` (boolean, string, integer, json)
- A `default_value` returned when no rule matches
- A `status` (active, disabled, archived)
- A `version` counter incremented on every mutation

**variants**
Named values a flag can resolve to (e.g., `on`/`off`, `control`/`treatment_a`/`treatment_b`). A flag must have at least one variant.

**rules**
Rules belong to a flag and are evaluated in ascending `priority` order. Each rule has:
- A `rule_type`: `targeting` (conditions must match for 100% rollout to a single variant) or `rollout` (probabilistic assignment across variant buckets)
- A list of `conditions` (for targeting rules)
- A `rollout_pct` and `rollout_salt` (for rollout rules)
- An optional `schedule_start` / `schedule_end` to time-box the rule

**conditions**
A condition belongs to a targeting rule and specifies an attribute check: `{attribute, operator, value}`. All conditions in a rule are ANDed together.

**segments**
Reusable groups of conditions that can be referenced from rules. Segments belong to an application and are shared across environments within that application. Using a segment in a rule means "the entity must be a member of this segment."

**audit_events**
An append-only log of every mutation. Each event records: `actor_id`, `action` (e.g., `flag.updated`, `rule.created`), `resource_type`, `resource_id`, `before` (JSON), `after` (JSON), and `created_at`. Audit events are written in the same transaction as the mutation they describe.

### Entity Relationship Summary

```
applications
  └── environments
        └── flags
              ├── variants
              └── rules
                    └── conditions
                    └── (segment_ref → segments)

applications
  └── segments
        └── segment_conditions

applications
  └── audit_events
```

---

## Caching Strategy

### Multi-Tier Architecture

| Tier | Technology | TTL | Latency | Capacity |
|------|-----------|-----|---------|---------|
| L1 | Caffeine (in-process, TinyLFU eviction) | 30s | ~200ns | 256MB/instance |
| L2 | Redis 7 | 5min | ~1ms | Cluster-scalable |
| L3 | PostgreSQL 16 | Permanent | ~5ms | Source of truth |

### Cache Key Format

Flag cache keys encode enough information to uniquely identify a flag within an environment:

```
ff:{appID}:{envID}:{flagKey}       — full flag definition + rules
seg:{appID}:{segmentID}            — segment definition + conditions
```

The invalidation pub/sub channels mirror this structure:

```
ff:invalidate:{appID}:{envID}      — published on any flag/rule mutation
seg:invalidate:{appID}             — published on segment mutation
```

### Read Path

```
Request arrives
    │
    ├─ L1 hit (~200ns)?  ──YES──► return cached flag, evaluate rules
    │
    ├─ L2 hit (~1ms)?    ──YES──► store in L1, evaluate rules
    │
    └─ DB read (~5ms)             store in L2 + L1, evaluate rules
```

### Write Path and Invalidation

1. Handler validates and writes mutation to PostgreSQL. The UPDATE includes `version = version + 1`. An audit event INSERT runs in the same transaction.
2. On successful COMMIT, the handler publishes `{flagKey}` to `ff:invalidate:{appID}:{envID}` on Redis.
3. All FMS instances receive the pub/sub message and:
   a. Evict `ff:{appID}:{envID}:{flagKey}` from L1 (Caffeine `invalidate`)
   b. Issue `DEL ff:{appID}:{envID}:{flagKey}` to Redis (L2 eviction)
4. The first evaluation after invalidation re-warms L2 from DB, then L1 from L2.

### Stampede Prevention

When L2 is cold (e.g., immediately after invalidation) and many goroutines concurrently evaluate the same flag, they would all miss L2 and issue redundant DB reads. To prevent this, the cache layer uses a `sync/singleflight.Group` keyed by the cache key. Only one goroutine issues the DB read; the others block and receive the same result when it completes. This bounds the DB read fan-out to one query per flag per invalidation event, regardless of concurrent request volume.

### Negative Caching

When a flag key does not exist in the database (e.g., a misconfigured SDK using a wrong flag key), the cache stores a sentinel "not found" entry in L1 with a 5-second TTL. This prevents repeated DB reads for non-existent flags under load. A 5-second TTL is short enough that a flag created shortly after a miss will become visible quickly.

### Version-Gated Writes

When writing a flag definition into L2, the serialized blob includes the flag's `version` integer. Before storing into L1 from L2, the engine checks that the version is not older than any version it has recently seen for that flag. This prevents a race where a slow DB read delivers a stale version after a fast invalidation + re-warm has already placed a newer version in L1.

---

## Evaluation Engine

The evaluation engine is the heart of FMS. Given a flag definition and an `EvaluationContext`, it determines which variant the entity should receive.

### Step-by-Step Algorithm

```
evaluate(flagDef, ctx) -> EvaluationResult:

1. If flag is disabled or archived:
     return {value: flagDef.defaultValue, explanation: {defaultServed: true, reason: "flag is disabled"}}

2. Sort flagDef.rules by priority ASC (lower number = evaluated first)

3. For each rule in sorted order:
   a. If rule has schedule_start/end and current time is outside the window:
        record step outcome = "skipped (schedule)"
        continue to next rule

   b. If rule.type == "targeting":
        allMatch = true
        for each condition in rule.conditions:
            actual = ctx.attributes[condition.attribute]
            matched = applyOperator(condition.operator, actual, condition.value)
            record condition_result{attribute, operator, value, actual_value, matched}
            if !matched: allMatch = false; break (short-circuit)
        if !allMatch:
            record step outcome = "skipped"
            continue
        // All conditions matched — this rule applies at 100%
        record step outcome = "matched"
        return {value: rule.variantValue, explanation: {..., matchedRule: rule}}

   c. If rule.type == "rollout":
        if rule has segment_ref:
            segDef = cache.GetSegment(rule.segmentRef)
            if entity not in segment:
                record step outcome = "skipped (not in segment)"
                continue
        bucket = MurmurHash3_x86_32(flagKey + ":" + ctx.entityID + ":" + rule.rolloutSalt) % 10000
        if bucket >= rule.rolloutPct * 100:
            record step outcome = "skipped (outside rollout)"
            continue
        // Entity is in the rollout — pick variant by bucket range
        for each allocation in rule.variantAllocations:
            if allocation.from <= bucket < allocation.to:
                record step outcome = "matched"
                return {value: allocation.variantValue, explanation: {..., bucket, matchedRule: rule}}

4. No rule matched:
     return {value: flagDef.defaultValue, explanation: {defaultServed: true, reason: "no rule matched"}}
```

### Condition Operators

| Operator | Applies To | Semantics |
|---|---|---|
| `eq` | string, number, bool | Exact equality |
| `neq` | string, number, bool | Not equal |
| `lt` / `lte` | number | Less than / less than or equal |
| `gt` / `gte` | number | Greater than / greater than or equal |
| `contains` | string | Substring match |
| `not_contains` | string | Substring not present |
| `starts_with` | string | Prefix match |
| `ends_with` | string | Suffix match |
| `in` | string, number | Value is in a provided list |
| `not_in` | string, number | Value is not in a provided list |
| `matches_regex` | string | RE2 regex match |
| `is_set` | any | Attribute key exists in context |
| `is_not_set` | any | Attribute key absent from context |

---

## Rollout Algorithm

Percentage rollouts use a deterministic hash-based bucketing scheme to ensure that:

1. The same entity always lands in the same bucket for a given rule (sticky assignment).
2. The bucket distribution is uniform across the 0-9999 range.
3. Changing the rollout percentage changes which fraction of entities are included, without reshuffling the entire population.
4. The salt can be rotated to reassign users when needed (e.g., to reset a bad rollout without changing the percentage).

### Bucket Computation

```
bucket = MurmurHash3_x86_32( flagKey + ":" + entityID + ":" + rolloutSalt ) % 10000
```

**Why MurmurHash3?**
- Extremely fast (single-pass, no crypto overhead)
- Excellent distribution uniformity across the 0-9999 range
- Deterministic: same inputs always produce the same output
- Non-cryptographic hashes are appropriate here since the output does not need to be unpredictable

**Input construction:**
- `flagKey` scopes the bucket to the flag, so two flags with 50% rollouts split their populations independently
- `entityID` is the stable identifier of the entity (user ID, session ID, device ID)
- `rolloutSalt` is a UUID stored on the rule; rotating it effectively reassigns all users without touching the rollout percentage

**Bucket range: 0-9999**
This gives 0.01% granularity. A 50% rollout threshold is `5000`; an entity with `bucket < 5000` is in.

### Variant Allocation

For rollout rules that have more than two variants (e.g., A/B/C tests), each variant is assigned a contiguous range within `[0, 10000)`:

```json
"variant_allocations": [
  {"variant_key": "control",     "from": 0,    "to": 3334},
  {"variant_key": "treatment_a", "from": 3334, "to": 6667},
  {"variant_key": "treatment_b", "from": 6667, "to": 10000}
]
```

The engine checks `allocation.from <= bucket < allocation.to` for each allocation in order. Gaps between allocations represent entities who are in the rollout percentage but assigned to no variant — the engine falls through to the default value in that case.

### IsInRollout Check

```go
func IsInRollout(bucket uint32, rolloutPct float64) bool {
    return bucket < uint32(rolloutPct * 100)
}
```

`rolloutPct` is stored as a float in `[0, 100]`. Multiplying by 100 maps it to the `[0, 10000]` bucket range.

---

## Explainability Model

Every evaluation produces a complete explanation of how the decision was reached. This is not an afterthought — the explanation object is built incrementally as the engine works through the rule list. There is no separate "re-evaluation for audit" step; the same code path that produces the value also produces the explanation.

### Explanation Structure

```
EvaluationExplanation
├── flag_key, flag_name, flag_version
├── environment_slug
├── evaluated_at (RFC3339)
├── entity_id, entity_type
├── default_served (bool)
├── matched_rule_id, matched_rule_name, matched_rule_type (absent if default_served)
├── bucket_value (absent for targeting rules)
├── reason (human-readable sentence)
└── steps[]
      ├── rule_id, rule_name, rule_type, priority
      ├── outcome: "matched" | "skipped" | "no_match"
      ├── condition_results[] (targeting rules only)
      │     └── attribute, operator, value, actual_value, matched
      ├── rollout_bucket (rollout rules only)
      └── rollout_pct (rollout rules only)
```

### How the Explanation Is Built

- Before rule iteration begins, the engine initializes an empty `steps` slice.
- For each rule evaluated, the engine appends a `StepResult` to `steps` as soon as the outcome for that rule is known.
- For targeting rules, each condition check appends a `ConditionResult` to the current step before short-circuiting.
- When a rule matches, the engine populates the top-level fields (`matched_rule_id`, `bucket_value`, etc.) and sets `reason` to a human-readable description.
- If no rule matches, `default_served` is set to `true` and `reason` is set to `"no rule matched"`.

The `steps` array always reflects the full evaluation trace in priority order, including rules that were evaluated and skipped before the matching rule. This is essential for debugging: it shows operators exactly why higher-priority rules did not fire.

### Field Reference

| Field | Type | Description |
|---|---|---|
| `flag_version` | int | Version of the flag definition at evaluation time. Cross-reference with audit log. |
| `default_served` | bool | True when no rule matched. |
| `matched_rule_id` | UUID | The rule that produced the returned value. |
| `bucket_value` | int (0-9999) | Hash bucket for the entity. Stable for the same inputs. |
| `reason` | string | Plain-English explanation of the outcome. |
| `steps[].outcome` | enum | `matched` (rule fired), `skipped` (conditions or schedule did not match), `no_match` (rule type not applicable). |
| `steps[].condition_results[].matched` | bool | Whether this individual condition matched. |
| `steps[].condition_results[].actual_value` | string | The value the entity actually had for this attribute. |

---

## Consistency Guarantees

### After a Flag Write

The sequence of events after a flag is written:

```
t=0ms    Write committed to PostgreSQL (version incremented, audit event written)
t=1ms    Redis pub/sub message published by the writing API instance
t=5ms    All healthy FMS instances receive pub/sub, evict L1 + L2 for the flag
t=6ms    First post-invalidation evaluation re-warms L2 from DB
t=6.2ms  L1 re-warmed from L2
```

Healthy instances serve fresh data within approximately 10ms of a committed write.

### Under Network Partition

If an instance cannot reach Redis (and therefore misses the invalidation message), it continues serving the stale L1 value for at most 30 seconds (L1 TTL). After L1 expiry it falls through to L2 (Redis), which either has the fresh value (if connectivity restored) or itself expires after 5 minutes. In the absolute worst case (prolonged Redis outage), the instance falls through to PostgreSQL directly, always serving the latest data.

This means the maximum bounded staleness per instance is:
- **Normal operation**: ~10ms after a write (pub/sub invalidation latency)
- **Redis pub/sub disruption**: 30s (L1 TTL)
- **Redis fully unavailable**: 0s staleness (falls through to DB on L2 miss)

### Write Ordering

The `version` field on flags provides a monotonic counter. Clients can include `If-Match: <version>` headers on mutation requests to implement optimistic concurrency control: the server rejects the write if the current version does not match the client's expected version.

---

## Scaling Considerations

### Horizontal Scaling of FMS Instances

FMS instances are stateless with respect to writes (all writes go to PostgreSQL). Adding instances increases evaluation throughput linearly. Each instance has its own L1 cache (no cross-instance sharing), which is intentional: L1 is never a consistency bottleneck because write invalidation works at the L2/pub-sub level.

Run at least 2 instances in production for availability. Use a load balancer with health checks on `/readyz`.

### Redis Cluster

The L2 cache and pub/sub broker are both Redis. For high availability:
- Use Redis Sentinel (for simpler deployments) or Redis Cluster (for horizontal scale)
- Pub/sub messages are handled by the primary; replicas handle read traffic
- Redis Cluster requires that all keys for a given pub/sub channel land on the same shard — use hash tags (`{appID}`) if necessary

### PostgreSQL Read Replicas

Under high load, L2 cache misses (flag warm-up reads) can stress the primary. Configure a read replica and route all SELECT queries from the evaluation engine to it. The version counter on flags ensures that the engine detects when it reads from a lagging replica (the version will be older than what it expects post-invalidation) and retries on the primary.

### Evaluation Latency Under Scale

| Scenario | Latency | Bottleneck |
|---|---|---|
| L1 hit (warm cache) | ~200ns | In-process memory |
| L2 hit (Redis, instance just started) | ~1ms | Redis round-trip |
| DB read (cold start or post-invalidation) | ~5ms | PostgreSQL + singleflight |
| Singleflight contention (many goroutines, one DB reader) | ~5ms | singleflight wait |

At 100k evaluations/second with a typical L1 hit rate > 95%, the DB handles fewer than 5,000 reads/second. A single PostgreSQL 16 instance with adequate connection pooling handles this comfortably.

---

## SDK Design

### Responsibilities

Client-side SDKs are thin wrappers that:

1. Format and send evaluation requests to the FMS HTTP API
2. Maintain a short-lived local cache (configurable TTL, default 10s) to avoid a network call on every invocation
3. Return the configured default value gracefully if the FMS is unreachable (circuit breaker pattern)
4. Surface the full `explanation` object to callers who need it for debugging

### Local SDK Cache

The SDK-side cache trades a small amount of freshness for latency. Calls that hit the SDK cache never leave the process. This is acceptable for most feature flag use cases; if a flag changes, the SDK cache expires within its TTL (default 10s) and the next call fetches from FMS.

For latency-sensitive code paths (e.g., called on every database row in a loop), SDK-side caching is critical. Callers should tune `LocalCacheTTL` based on their tolerance for staleness.

### Graceful Degradation

If the FMS API is unreachable (timeout, connection refused, 5xx):

1. The SDK logs a warning with the error details.
2. It returns the default value supplied by the caller.
3. It does **not** panic or propagate the error in a way that would affect the calling service's availability.

This ensures that a FMS outage degrades gracefully — all flags return their defaults — rather than cascading into the services that depend on them.

### Batch Evaluation

SDKs expose a `EvaluateBatch` method that sends a single HTTP request for multiple flag keys. This is the recommended pattern for page-load or request-initialization code that needs to resolve many flags at once (e.g., resolving 10 flags for a page render). It avoids the latency penalty of N sequential HTTP calls.
