# Feature Management Service

A production-grade feature flag management and evaluation service built with Java and Spring Boot. Designed to serve flag evaluations at high throughput (~100k evaluations/second) with sub-millisecond latency across 100+ applications and services.

---

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [API Examples](#api-examples)
- [SDK Quick Start](#sdk-quick-start)
- [Caching Strategy](#caching-strategy)
- [Explainability](#explainability)
- [Observability](#observability)
- [Development Setup](#development-setup)
- [Configuration](#configuration)

---

## Overview

The Feature Management Service (FMS) is the single source of truth for feature flag state across the entire e-commerce platform. It provides:

- **Centralized flag management** — create, update, and archive flags across web portals, backend APIs, and mobile clients from one place
- **High-throughput evaluation** — ~100k flag evaluations/second with sub-millisecond p99 latency via multi-tier caching
- **Rich targeting rules** — target by user attributes, custom segments, or percentage rollouts with full AND-logic condition matching
- **Full explainability** — every evaluation response includes a complete trace of which rules were evaluated, which conditions matched or failed, and exactly why a particular variant was returned
- **Real-time propagation** — flag changes invalidate all cache tiers within milliseconds via Redis pub/sub
- **Audit trail** — every flag mutation is recorded with actor, action, timestamp, and before/after state

### Key Capabilities

| Capability | Detail |
|---|---|
| Throughput target | ~100,000 evaluations/second |
| p99 evaluation latency | < 1ms (L1 cache hit) |
| Cache tiers | L1 in-process Caffeine (30s TTL) + L2 Redis (5min TTL) |
| Rollout granularity | 0.01% (MurmurHash3 bucketing, 0-9999 range) |
| Stickiness | Deterministic: same entity always gets same variant |
| Invalidation | Redis pub/sub, propagates to all instances in < 100ms |
| Observability | Micrometer/Prometheus metrics, OpenTelemetry traces, structured logs |

---

## Architecture

```
                    ┌─────────────────────────────────────────┐
                    │          Feature Management Service       │
                    │                                           │
  ┌───────────────┐ │ ┌──────────┐  ┌──────────────────────┐  │
  │ Management UI │─┼─►│ REST API │  │   Evaluation Engine   │  │
  └───────────────┘ │ │ (Spring  │  │                      │  │
                    │ │  MVC)    │  │  Rule priority queue  │  │
  ┌───────────────┐ │ └─────┬────┘  │  Condition matching   │  │
  │ Service SDKs  │─┼───────┘       │  Rollout bucketing    │  │
  └───────────────┘ │               │  (MurmurHash3)        │  │
                    │               └──────────┬───────────┘  │
                    │          ┌───────────────▼───────────┐  │
                    │          │      Tiered Cache          │  │
                    │          │  L1: Caffeine (30s TTL)    │  │
                    │          │  L2: Redis (5min TTL)      │  │
                    │          └───────────────┬───────────┘  │
                    │                          │               │
                    │          ┌───────────────▼───────────┐  │
                    │          │    PostgreSQL 16            │  │
                    │          │  (source of truth)         │  │
                    │          └───────────────────────────┘  │
                    └─────────────────────────────────────────┘
                                      │
                    ┌─────────────────▼────────────────────┐
                    │     Redis Pub/Sub (invalidation)       │
                    │   ff:invalidate:{appID}:{envID}        │
                    │   seg:invalidate:{appID}               │
                    └───────────────────────────────────────┘
```

The service exposes a single REST API consumed by both a management UI (for operators) and service SDKs (for application code). The evaluation engine resolves flags through a tiered cache before falling back to PostgreSQL. All write operations publish invalidation events so every running instance flushes stale data immediately.

For a deeper treatment of each component, see [docs/architecture.md](docs/architecture.md).

---

## Quick Start

### Prerequisites

- Docker and Docker Compose
- `curl` or any HTTP client for testing

### 1. Start the stack

```bash
git clone https://github.com/your-org/feature-management-service.git
cd feature-management-service
docker-compose up -d
```

This starts PostgreSQL 16, Redis 7, and the FMS API on port 8080. The database schema is applied automatically on first boot.

Verify everything is healthy:

```bash
curl -s http://localhost:8080/healthz | jq .
# {"status":"ok"}

curl -s http://localhost:8080/readyz | jq .
# {"status":"ok","checks":{"database":"ok","redis":"ok"}}
```

### 2. Create an application

```bash
curl -s -X POST http://localhost:8080/v1/applications \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"name": "checkout-service", "slug": "checkout"}' | jq .
```

Note the `id` field in the response — this is your `{appID}`.

### 3. Create an environment

```bash
APP_ID="<appID from above>"

curl -s -X POST "http://localhost:8080/v1/applications/${APP_ID}/environments" \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"name": "Production", "slug": "production", "color": "#e53e3e"}' | jq .
```

Note the `id` field — this is your `{envID}`.

### 4. Create your first flag

```bash
ENV_ID="<envID from above>"

curl -s -X POST \
  "http://localhost:8080/v1/applications/${APP_ID}/environments/${ENV_ID}/flags/checkout.new_ui" \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Checkout New UI Feature",
    "description": "Enables the redesigned checkout flow",
    "flag_type": "boolean",
    "default_value": "false",
    "variants": [
      {"key": "on",  "value": "true"},
      {"key": "off", "value": "false"}
    ]
  }' | jq .
```

### 5. Evaluate the flag

```bash
curl -s -X POST http://localhost:8080/v1/evaluate \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "flag_key":    "checkout.new_ui",
    "app_id":      "'"${APP_ID}"'",
    "environment": "production",
    "context": {
      "entity_id":   "user-123",
      "entity_type": "user",
      "attributes": {
        "email": "alice@example.com",
        "plan":  "pro"
      }
    }
  }' | jq .
```

The response includes the resolved value and a full explanation of which rules were evaluated and why.

---

## API Examples

### Create a targeting rule (100% of internal users)

```bash
curl -s -X POST \
  "http://localhost:8080/v1/applications/${APP_ID}/environments/${ENV_ID}/flags/checkout.new_ui/rules" \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name":      "Internal team override",
    "rule_type": "targeting",
    "priority":  0,
    "conditions": [
      {
        "attribute": "user.email",
        "operator":  "ends_with",
        "value":     "@internal.example.com"
      }
    ],
    "variant_key": "on"
  }' | jq .
```

### Create a percentage rollout rule

```bash
curl -s -X POST \
  "http://localhost:8080/v1/applications/${APP_ID}/environments/${ENV_ID}/flags/checkout.new_ui/rules" \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name":        "Beta users rollout",
    "rule_type":   "rollout",
    "priority":    1,
    "rollout_pct": 50,
    "variant_allocations": [
      {"variant_key": "on",  "from": 0,    "to": 5000},
      {"variant_key": "off", "from": 5000, "to": 10000}
    ]
  }' | jq .
```

### Batch evaluation (multiple flags in one round trip)

```bash
curl -s -X POST http://localhost:8080/v1/evaluate/batch \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "flag_keys":   ["checkout.new_ui", "checkout.express_pay", "homepage.banner"],
    "app_id":      "'"${APP_ID}"'",
    "environment": "production",
    "context": {
      "entity_id":   "user-123",
      "entity_type": "user",
      "attributes":  {"plan": "pro"}
    }
  }' | jq .
```

### Dry-run evaluation (test without cache side effects)

```bash
curl -s -X POST http://localhost:8080/v1/evaluate/dry-run \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "flag_key":    "checkout.new_ui",
    "app_id":      "'"${APP_ID}"'",
    "environment": "production",
    "context": {
      "entity_id":   "hypothetical-user-999",
      "entity_type": "user",
      "attributes":  {"plan": "free"}
    }
  }' | jq .
```

### Fetch the audit log

```bash
curl -s \
  "http://localhost:8080/v1/applications/${APP_ID}/audit?action=flag.updated&limit=20" \
  -H "Authorization: Bearer dev-token" | jq .
```

---

## SDK Quick Start

### Go SDK

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    fms "github.com/your-org/fms-go-sdk"
)

func main() {
    client, err := fms.NewClient(fms.Config{
        BaseURL:       "https://fms.internal.example.com",
        APIKey:        "sdk-key-abc123",
        AppID:         "app-uuid-here",
        Environment:   "production",
        LocalCacheTTL: 10 * time.Second,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    enabled, explanation, err := client.BoolVariation(ctx, "checkout.new_ui", fms.EvalContext{
        EntityID:   "user-123",
        EntityType: "user",
        Attributes: map[string]interface{}{
            "email": "alice@example.com",
            "plan":  "pro",
        },
    }, false /* default */)
    if err != nil {
        log.Printf("flag eval error: %v", err)
        // SDK returns default on error — safe to proceed
    }

    fmt.Printf("checkout.new_ui = %v (reason: %s)\n", enabled, explanation.Reason)

    variant, _, err := client.StringVariation(ctx, "homepage.banner_variant", fms.EvalContext{
        EntityID:   "user-123",
        EntityType: "user",
    }, "control")
    fmt.Printf("banner variant: %s\n", variant)
}
```

### TypeScript SDK

```typescript
import { FMSClient } from '@your-org/fms-js-sdk';

const client = new FMSClient({
  baseURL: 'https://fms.internal.example.com',
  apiKey: 'sdk-key-abc123',
  appId: 'app-uuid-here',
  environment: 'production',
  localCacheTTLMs: 10_000,
});

const { value, explanation } = await client.evaluate('checkout.new_ui', {
  entityId: 'user-123',
  entityType: 'user',
  attributes: {
    email: 'alice@example.com',
    plan: 'pro',
  },
}, /* default */ false);

console.log(`checkout.new_ui = ${value}`);
console.log(`Matched rule: ${explanation.matchedRuleName}`);
console.log(`Reason: ${explanation.reason}`);

// Batch evaluation: one network round trip for many flags
const results = await client.evaluateBatch(
  ['checkout.new_ui', 'checkout.express_pay', 'homepage.banner'],
  { entityId: 'user-123', entityType: 'user', attributes: { plan: 'pro' } },
);

for (const [flagKey, result] of Object.entries(results)) {
  console.log(`${flagKey} => ${result.value} (${result.explanation.reason})`);
}
```

Both SDKs maintain their own short-lived local cache and fall back gracefully to the configured default value if the FMS is unreachable.

---

## Caching Strategy

FMS uses a three-tier caching model to achieve sub-millisecond evaluation latency without sacrificing consistency.

### Tier Summary

| Tier | Technology | TTL | Typical Latency | Notes |
|------|-----------|-----|-----------------|-------|
| L1 | ristretto (in-process, TinyLFU eviction) | 30s | ~200ns | Per instance; 256MB max |
| L2 | Redis 7 | 5min | ~1ms | Shared across all instances |
| L3 | PostgreSQL 16 | Permanent | ~5ms | Source of truth |

### Cache Key Format

```
Flag data:    ff:{appID}:{envID}:{flagKey}
Segment data: seg:{appID}:{segmentID}
```

### Invalidation Flow

When a flag is updated:

1. The mutation is written to PostgreSQL inside a transaction that also inserts an audit event and increments the flag's version counter.
2. After the transaction commits, the API handler publishes to the Redis pub/sub channel `ff:invalidate:{appID}:{envID}`.
3. Every running FMS instance subscribes to `ff:invalidate:*` (pattern subscription). On receiving a message, each instance immediately evicts the matching key from both L1 (ristretto) and L2 (Redis).
4. The next evaluation for that flag re-warms L2 from PostgreSQL, then L1 from L2.
5. A `singleflight.Group` per flag key coalesces concurrent DB reads during the warm-up window, preventing a cache stampede.

### Negative Caching

Flags that don't exist (e.g., a typo in the flag key) are stored in L1 with a 5-second TTL. This prevents repeated database hits for missing flags from hammering the DB under load.

### Bounded Staleness

In the absence of a write, the maximum time an instance can serve stale data is 30 seconds (L1 TTL). Write-triggered invalidation reduces this to the Redis pub/sub round-trip time (roughly 10ms in practice).

For full details, see [docs/architecture.md](docs/architecture.md).

---

## Explainability

Every evaluation endpoint returns a detailed `explanation` object alongside the resolved value.

### Example Response

```json
{
  "flag_key": "checkout.new_ui",
  "value": "true",
  "variant_key": "on",
  "explanation": {
    "flag_key": "checkout.new_ui",
    "flag_name": "Checkout New UI Feature",
    "flag_version": 7,
    "environment_slug": "production",
    "evaluated_at": "2026-06-16T14:44:22Z",
    "entity_id": "user-456",
    "entity_type": "user",
    "default_served": false,
    "matched_rule_id": "550e8400-e29b-41d4-a716-446655440000",
    "matched_rule_name": "Beta users rollout",
    "matched_rule_type": "rollout",
    "bucket_value": 4521,
    "reason": "Entity bucket 4521 is within rollout threshold 5000 for rule 'Beta users rollout'",
    "steps": [
      {
        "rule_id": "111e8400-e29b-41d4-a716-446655440001",
        "rule_name": "Internal team override",
        "rule_type": "targeting",
        "priority": 0,
        "outcome": "skipped",
        "condition_results": [
          {
            "attribute": "user.email",
            "operator": "ends_with",
            "value": "@internal.example.com",
            "actual_value": "alice@acme.com",
            "matched": false
          }
        ]
      },
      {
        "rule_id": "550e8400-e29b-41d4-a716-446655440000",
        "rule_name": "Beta users rollout",
        "rule_type": "rollout",
        "priority": 1,
        "outcome": "matched",
        "rollout_bucket": 4521,
        "rollout_pct": 50
      }
    ]
  }
}
```

### Field Reference

| Field | Meaning |
|---|---|
| `flag_version` | The version of the flag definition that was evaluated. Useful for confirming which flag state was active. |
| `default_served` | `true` if no rule matched and the flag's default value was returned. |
| `matched_rule_id` | UUID of the first rule that matched. Absent when `default_served` is true. |
| `bucket_value` | The MurmurHash3 bucket (0-9999) computed for this entity. Determines rollout variant assignment. |
| `reason` | Human-readable sentence explaining the evaluation outcome. |
| `steps[].outcome` | `matched`, `skipped`, or `no_match`. Only one step will ever be `matched`. |
| `steps[].condition_results` | Per-condition breakdown for targeting rules: shows the actual attribute value and whether it matched. |
| `steps[].rollout_bucket` | Computed bucket for rollout rules. |
| `steps[].rollout_pct` | The rollout percentage threshold for the matched rule. |

---

## Observability

### Prometheus Metrics

Metrics are exposed at `GET /metrics` in standard Prometheus text format.

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `fms_flag_evaluations_total` | Counter | `flag_key`, `environment`, `result`, `cache_layer` | Total evaluations by outcome and cache tier |
| `fms_flag_evaluation_duration_seconds` | Histogram | `flag_key`, `environment` | Evaluation latency (p50/p99/p999) |
| `fms_cache_hits_total` | Counter | `tier` (`l1`/`l2`) | Cache hits per tier |
| `fms_cache_misses_total` | Counter | `tier` | Cache misses per tier |
| `fms_cache_invalidations_total` | Counter | `reason` | Invalidations triggered by writes |
| `fms_flag_updates_total` | Counter | `action` | Flag create/update/archive counts |
| `fms_active_flags_total` | Gauge | `environment`, `status` | Current count of flags by status |
| `fms_http_requests_total` | Counter | `method`, `path`, `status_code` | HTTP request counts |
| `fms_http_request_duration_seconds` | Histogram | `method`, `path` | HTTP request latency |

### OpenTelemetry Traces

The service emits OpenTelemetry traces with W3C trace context propagation. Key spans:

- `evaluate_flag` — top-level span per flag evaluation
- `cache.l1.get` / `cache.l2.get` — one span per cache tier lookup
- `db.query` — one span per database query
- `pubsub.publish` — span for each invalidation event

Configure the exporter via `OTEL_EXPORTER_OTLP_ENDPOINT`.

### Structured Logging

Logs use [zap](https://github.com/uber-go/zap). JSON format in production, console format in development (`LOG_FORMAT`).

Standard fields on every log line:

```json
{
  "level": "info",
  "ts": "2026-06-16T14:44:22.123Z",
  "caller": "handler/evaluate.go:87",
  "msg": "flag evaluated",
  "request_id": "req-abc123",
  "method": "POST",
  "path": "/v1/evaluate",
  "status": 200,
  "duration_ms": 0.42,
  "flag_key": "checkout.new_ui",
  "entity_id": "user-123",
  "cache_layer": "l1",
  "rule_matched": "Beta users rollout"
}
```

---

## Development Setup

```bash
# Start all dependencies (PostgreSQL, Redis)
make docker-up

# Run the service locally (Flyway migrations apply automatically on boot)
make run

# Run the full test suite
make test

# Compile only
make build

# Build the runnable jar
make package

# Tear down dev dependencies
make docker-down
```

---

## Configuration

All configuration is via environment variables. A `.env.example` file is provided in the repository root.

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | Yes | — | PostgreSQL DSN, e.g. `postgres://user:pass@host:5432/fms?sslmode=require` |
| `REDIS_URL` | Yes | — | Redis URL, e.g. `redis://:password@host:6379/0` |
| `HTTP_PORT` | No | `8080` | Port the HTTP server listens on |
| `HTTP_READ_TIMEOUT` | No | `10s` | HTTP server read timeout |
| `HTTP_WRITE_TIMEOUT` | No | `30s` | HTTP server write timeout |
| `DB_MAX_OPEN_CONNS` | No | `25` | Max open PostgreSQL connections per instance |
| `DB_MAX_IDLE_CONNS` | No | `10` | Max idle PostgreSQL connections per instance |
| `DB_CONN_MAX_LIFETIME` | No | `5m` | Max lifetime of a PostgreSQL connection |
| `CACHE_L1_MAX_COST` | No | `268435456` | Max memory for L1 ristretto cache in bytes (256MB) |
| `CACHE_L1_TTL` | No | `30s` | L1 cache TTL |
| `CACHE_L2_TTL` | No | `5m` | L2 Redis cache TTL |
| `CACHE_NEGATIVE_TTL` | No | `5s` | TTL for negative (missing flag) cache entries |
| `LOG_LEVEL` | No | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | No | `json` | Log format: `json` or `console` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | — | OpenTelemetry OTLP endpoint (tracing disabled if unset) |
| `API_TOKEN_SECRET` | Yes | — | Secret used to validate Bearer tokens |
| `ENVIRONMENT` | No | `development` | Runtime environment label (`development`, `production`) |
