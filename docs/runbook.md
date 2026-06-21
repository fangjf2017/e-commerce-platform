# Operations Runbook

This runbook covers deployment, monitoring, incident response, and maintenance procedures for the Feature Management Service (FMS).

---

## Table of Contents

- [Deployment](#deployment)
- [Health Checks](#health-checks)
- [Monitoring](#monitoring)
- [Common Issues and Remediation](#common-issues-and-remediation)
- [Cache Management](#cache-management)
- [Scaling Guidance](#scaling-guidance)
- [Backup and Recovery](#backup-and-recovery)
- [On-Call Escalation Path](#on-call-escalation-path)

---

## Deployment

### Local Development (Docker Compose)

The `docker-compose.yml` in the repository root starts a full local stack including PostgreSQL, Redis, and the FMS API.

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f fms

# Apply migrations (run after first start or after adding new migrations)
docker-compose exec fms /app/fms migrate up

# Stop all services
docker-compose down

# Stop and destroy all data volumes
docker-compose down -v
```

The API is available at `http://localhost:8080`. PostgreSQL is on port `5432` and Redis on port `6379`, both accessible from the host for local debugging.

### Production Deployment (Kubernetes)

The production deployment lives in `deployments/kubernetes/`. The FMS runs as a `Deployment` with a `HorizontalPodAutoscaler` keyed on CPU utilization and the custom metric `fms_flag_evaluations_total`.

#### Pre-Deployment Checklist

- [ ] Database migrations applied (see below)
- [ ] New environment variables added to the Kubernetes `Secret` and `ConfigMap`
- [ ] Docker image built and pushed to the registry with the release tag
- [ ] Smoke test passed against staging

#### Applying Migrations

Migrations must run before deploying a new version that requires schema changes. Use the migration job:

```bash
kubectl apply -f deployments/kubernetes/migration-job.yaml
kubectl wait --for=condition=complete job/fms-migrate --timeout=120s
kubectl logs job/fms-migrate
```

The migration job runs `fms migrate up` against the production database and exits. It uses a `restartPolicy: OnFailure` so failed migrations are retried automatically.

#### Rolling Deployment

```bash
# Update image tag in the deployment
kubectl set image deployment/fms fms=registry.example.com/fms:v1.2.3

# Monitor rollout
kubectl rollout status deployment/fms

# Roll back if needed
kubectl rollout undo deployment/fms
```

The deployment uses a `RollingUpdate` strategy with `maxUnavailable: 0` and `maxSurge: 1` to ensure zero-downtime deploys. Kubernetes waits for the new pod to pass the readiness probe (`/readyz`) before terminating an old pod.

#### Kubernetes Resource Summary

| Resource | Purpose |
|---|---|
| `Deployment/fms` | FMS API pods (2 replicas minimum) |
| `HorizontalPodAutoscaler/fms` | Scales 2-20 replicas based on CPU/custom metrics |
| `Service/fms` | ClusterIP service for internal traffic |
| `Ingress/fms` | Exposes the API externally with TLS termination |
| `Secret/fms-secrets` | `DATABASE_URL`, `REDIS_URL`, `API_TOKEN_SECRET` |
| `ConfigMap/fms-config` | Non-sensitive configuration (ports, cache TTLs, log level) |
| `Job/fms-migrate` | One-shot migration job, created per deploy |
| `PodDisruptionBudget/fms` | Ensures at least 1 pod is available during node maintenance |

#### Environment Variables in Production

Sensitive values (`DATABASE_URL`, `REDIS_URL`, `API_TOKEN_SECRET`) are stored in Kubernetes Secrets and mounted as environment variables. Non-sensitive config is in a ConfigMap. Never store secrets in the deployment manifest or version control.

---

## Health Checks

FMS exposes two distinct health endpoints. Understanding the difference is important for configuring probes correctly.

### `/healthz` — Liveness Probe

**What it checks:** Only that the HTTP server process is alive and responding to requests.

**What it does NOT check:** Database connectivity, Redis connectivity, or any dependency health.

**Purpose:** Kubernetes uses this to determine if the container should be restarted. It should only return `503` if the server is fundamentally broken (e.g., deadlocked or in a graceful shutdown). It must never return `503` just because a downstream dependency is slow or unavailable — that would cause healthy pods to be restarted unnecessarily.

**Kubernetes configuration:**
```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
  failureThreshold: 3
```

### `/readyz` — Readiness Probe

**What it checks:** PostgreSQL connectivity (a lightweight `SELECT 1` ping), Redis connectivity (a `PING` command).

**What it returns:**

```json
{
  "status": "ok",
  "checks": {
    "database": "ok",
    "redis": "ok"
  }
}
```

Or on failure:

```json
{
  "status": "degraded",
  "checks": {
    "database": "ok",
    "redis": "error: dial tcp: connection refused"
  }
}
```

**Purpose:** Kubernetes uses this to determine if a pod should receive traffic. A pod that cannot reach Redis or the database should not receive requests. When this returns `503`, Kubernetes removes the pod from the Service's endpoint set until it recovers.

**Kubernetes configuration:**
```yaml
readinessProbe:
  httpGet:
    path: /readyz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  failureThreshold: 2
```

---

## Monitoring

### Key Prometheus Queries

Run these in Grafana or the Prometheus console to get a real-time view of FMS health.

**Overall evaluation rate (evaluations/second):**
```promql
rate(fms_flag_evaluations_total[1m])
```

**Evaluation latency p99 across all flags:**
```promql
histogram_quantile(0.99,
  sum(rate(fms_flag_evaluation_duration_seconds_bucket[5m])) by (le)
)
```

**Evaluation latency p99 per flag:**
```promql
histogram_quantile(0.99,
  sum(rate(fms_flag_evaluation_duration_seconds_bucket[5m])) by (le, flag_key)
)
```

**L1 cache hit rate (target: > 90%):**
```promql
rate(fms_cache_hits_total{tier="l1"}[5m])
/
(rate(fms_cache_hits_total{tier="l1"}[5m]) + rate(fms_cache_misses_total{tier="l1"}[5m]))
```

**L2 cache hit rate (target: > 98%):**
```promql
rate(fms_cache_hits_total{tier="l2"}[5m])
/
(rate(fms_cache_hits_total{tier="l2"}[5m]) + rate(fms_cache_misses_total{tier="l2"}[5m]))
```

**DB read rate (should be low when caches are healthy):**
```promql
rate(fms_cache_misses_total{tier="l2"}[5m])
```

**Invalidations per second:**
```promql
rate(fms_cache_invalidations_total[5m])
```

**HTTP 5xx error rate:**
```promql
rate(fms_http_requests_total{status_code=~"5.."}[5m])
```

**HTTP p99 request latency by endpoint:**
```promql
histogram_quantile(0.99,
  sum(rate(fms_http_request_duration_seconds_bucket[5m])) by (le, path)
)
```

**Active flag count by environment:**
```promql
fms_active_flags_total{status="active"}
```

### Alert Thresholds

| Alert | Condition | Severity | Action |
|---|---|---|---|
| High evaluation latency | p99 > 10ms for 5 minutes | Warning | Check cache hit rates, Redis latency |
| Very high evaluation latency | p99 > 50ms for 2 minutes | Critical | Likely DB or Redis issue; page on-call |
| Low L1 hit rate | L1 hit rate < 80% for 10 minutes | Warning | Check L1 cache size, TTL config |
| Low L2 hit rate | L2 hit rate < 90% for 5 minutes | Warning | Check Redis connectivity and memory |
| High error rate | HTTP 5xx rate > 1% for 5 minutes | Warning | Check logs for errors |
| Very high error rate | HTTP 5xx rate > 5% for 2 minutes | Critical | Page on-call |
| Pod count below minimum | Available pods < 2 | Critical | Check pod logs; check node health |
| Redis pub/sub lag | No invalidations received within 60s after a write | Warning | Check Redis pub/sub subscriber health |

---

## Common Issues and Remediation

### High Evaluation Latency

**Symptoms:**
- `fms_flag_evaluation_duration_seconds` p99 > 10ms
- Users reporting slow page loads
- Downstream services showing increased latency

**Diagnosis:**

1. Check the L1 hit rate. If it has dropped below 90%, the in-process cache is undersized or TTL is too short.
   ```promql
   rate(fms_cache_hits_total{tier="l1"}[5m])
   /
   (rate(fms_cache_hits_total{tier="l1"}[5m]) + rate(fms_cache_misses_total{tier="l1"}[5m]))
   ```

2. Check the L2 hit rate. If L1 is healthy but L2 is missing, Redis may be slow or evicting keys early.
   ```bash
   redis-cli --latency -h $REDIS_HOST
   redis-cli INFO memory | grep used_memory_human
   redis-cli INFO keyspace
   ```

3. Check PostgreSQL query latency. If both cache tiers are healthy, DB reads should be rare. If the DB query rate is high, a stampede may be occurring.
   ```bash
   # On the PostgreSQL server:
   SELECT query, calls, mean_exec_time FROM pg_stat_statements
   WHERE query LIKE '%flags%' ORDER BY mean_exec_time DESC LIMIT 10;
   ```

**Remediation:**

| Root Cause | Action |
|---|---|
| L1 undersized | Increase `CACHE_L1_MAX_COST` (requires rolling restart) |
| Redis high latency | Check Redis memory usage, eviction policy (`maxmemory-policy`), and network latency |
| Redis evicting keys early | Increase Redis `maxmemory` allocation or scale the Redis cluster |
| DB stampede | Verify singleflight is functioning; check for goroutine leaks in traces |
| Too many unique flags | Increase L1 cache size; consider reducing flagKey cardinality |

---

### Cache Invalidation Not Propagating

**Symptoms:**
- Flag changes made via the management UI are not reflected in evaluations
- Different FMS instances return different values for the same flag
- Staleness exceeds 30 seconds after a write

**Diagnosis:**

1. Confirm the write was committed to PostgreSQL:
   ```sql
   SELECT flag_key, version, status, updated_at FROM flags
   WHERE flag_key = 'checkout.new_ui'
   ORDER BY updated_at DESC LIMIT 5;
   ```

2. Check if the invalidation was published to Redis:
   ```bash
   # Subscribe to the invalidation channel to observe messages in real time
   redis-cli PSUBSCRIBE 'ff:invalidate:*'
   # Then trigger a flag update and watch for the message
   ```

3. Check FMS instance logs for pub/sub errors:
   ```bash
   kubectl logs -l app=fms --since=10m | grep -i "pubsub\|invalidat"
   ```

4. Verify all instances are connected to Redis:
   ```bash
   kubectl exec -it <fms-pod> -- /app/fms debug redis-ping
   ```

**Remediation:**

| Root Cause | Action |
|---|---|
| Redis pub/sub subscriber disconnected | Rolling restart of FMS pods (`kubectl rollout restart deployment/fms`) |
| Redis network partition | Restore Redis connectivity; instances will re-subscribe on reconnect |
| Bug in invalidation handler | Check logs for errors in the pubsub subscriber goroutine; escalate to engineering |
| Single instance affected | Restart the affected pod |

**Emergency manual flush:**

If propagation is broken and you need to guarantee fresh data immediately, flush the Redis cache for the affected application/environment (see [Cache Management](#cache-management)). All instances will re-warm from the database on their next evaluation.

---

### Database Connection Pool Exhaustion

**Symptoms:**
- `fms_http_request_duration_seconds` p99 suddenly increases to hundreds of milliseconds
- Log lines: `pq: sorry, too many clients already` or `driver: bad connection`
- PostgreSQL metric `pg_stat_activity` shows connection count near `max_connections`

**Diagnosis:**

1. Check how many connections FMS is holding:
   ```sql
   SELECT application_name, count(*) FROM pg_stat_activity
   WHERE application_name LIKE 'fms%'
   GROUP BY application_name;
   ```

2. Check the configured pool size:
   ```bash
   kubectl exec -it <fms-pod> -- env | grep DB_MAX
   ```

3. Check if another service is consuming connections unexpectedly:
   ```sql
   SELECT application_name, count(*) FROM pg_stat_activity
   GROUP BY application_name ORDER BY count DESC;
   ```

**Remediation:**

| Root Cause | Action |
|---|---|
| Too many FMS pods | Reduce `DB_MAX_OPEN_CONNS` per instance (total connections = pods × max_conns) |
| Connection leak | Check for goroutines that open connections and do not close them; escalate to engineering |
| PostgreSQL `max_connections` too low | Increase `max_connections` in PostgreSQL config (requires DB restart); consider PgBouncer |
| Another service consuming connections | Audit all services connecting to this PostgreSQL instance |

**Rule of thumb:** Total connections = `(number of FMS pods) × DB_MAX_OPEN_CONNS`. Keep this below 80% of PostgreSQL's `max_connections`. With 10 pods and `DB_MAX_OPEN_CONNS=25`, that's 250 connections. PostgreSQL 16 default `max_connections` is 100, so adjust accordingly or use PgBouncer connection pooling.

---

### Missing Flags Returning Default Value

**Symptoms:**
- A newly created flag always returns its default value
- An existing flag is unexpectedly returning its default value after a recent change

**Diagnosis:**

1. Verify the flag exists and is active in the correct environment:
   ```bash
   curl -s \
     "http://localhost:8080/v1/applications/${APP_ID}/environments/${ENV_ID}/flags/checkout.new_ui" \
     -H "Authorization: Bearer dev-token" | jq '{status, flag_key, default_value}'
   ```

2. Check that the request is using the correct `app_id` and `environment` slug:
   ```bash
   # Look at recent evaluation logs
   kubectl logs -l app=fms --since=5m | grep 'checkout.new_ui' | jq .
   ```

3. Use the dry-run endpoint to trace the evaluation without cache side effects:
   ```bash
   curl -s -X POST http://localhost:8080/v1/evaluate/dry-run \
     -H "Authorization: Bearer dev-token" \
     -H "Content-Type: application/json" \
     -d '{
       "flag_key": "checkout.new_ui",
       "app_id": "'"${APP_ID}"'",
       "environment": "production",
       "context": {"entity_id": "user-123", "entity_type": "user", "attributes": {}}
     }' | jq .explanation
   ```
   The `steps` array will show exactly which rules were evaluated and why they did not match.

4. Check the audit log for recent flag changes:
   ```bash
   curl -s \
     "http://localhost:8080/v1/applications/${APP_ID}/audit?resource_type=flag&limit=10" \
     -H "Authorization: Bearer dev-token" | jq '.items[] | {action, created_at, before, after}'
   ```

**Common Causes and Fixes:**

| Cause | Fix |
|---|---|
| Flag is disabled | Enable via `POST /flags/{flagKey}/enable` |
| Wrong `app_id` or `environment` in the SDK config | Correct the SDK initialization parameters |
| Flag key typo | Verify exact flag key (case-sensitive) |
| Rule schedule window not yet active | Check `schedule_start` on the rule |
| Entity attributes do not match rule conditions | Use dry-run with the entity's actual attributes; check `condition_results` |
| Rollout percentage is 0% | Increase `rollout_pct` on the rule |
| Entity bucket is outside rollout threshold | This is expected behavior; increase rollout percentage |

---

## Cache Management

### Check Current Cache State

Use the Redis CLI to inspect what is currently cached for a specific flag:

```bash
# Connect to Redis
redis-cli -h $REDIS_HOST -a $REDIS_PASSWORD

# Check if a flag is in L2 cache
EXISTS "ff:{appID}:{envID}:checkout.new_ui"

# Get the cached flag definition (will be JSON-encoded)
GET "ff:{appID}:{envID}:checkout.new_ui"

# Check TTL remaining on a cached flag
TTL "ff:{appID}:{envID}:checkout.new_ui"
```

### Force Invalidate a Specific Flag

To force all instances to evict a specific flag from both L1 and L2 immediately, publish an invalidation message manually:

```bash
redis-cli -h $REDIS_HOST -a $REDIS_PASSWORD \
  PUBLISH "ff:invalidate:{appID}:{envID}" "checkout.new_ui"
```

This triggers the pub/sub handler on all subscribed instances, causing them to evict the flag from L1 and delete it from L2.

### Flush All FMS Cache Keys for an Environment

Use this when you need to guarantee that all instances re-read all flags from the database — for example, after a bulk update or a cache inconsistency incident.

```bash
# List all flag keys for the application/environment
redis-cli -h $REDIS_HOST -a $REDIS_PASSWORD KEYS "ff:{appID}:{envID}:*"

# Delete all of them (use UNLINK for non-blocking delete in production)
redis-cli -h $REDIS_HOST -a $REDIS_PASSWORD \
  --scan --pattern "ff:{appID}:{envID}:*" | xargs redis-cli UNLINK
```

After this, each instance's L1 cache will expire naturally within 30 seconds or on the next evaluation request.

### Flush All FMS Cache Keys Globally

Only do this in an emergency. It will cause a brief spike in DB reads as all instances re-warm.

```bash
redis-cli -h $REDIS_HOST -a $REDIS_PASSWORD \
  --scan --pattern "ff:*" | xargs redis-cli UNLINK

redis-cli -h $REDIS_HOST -a $REDIS_PASSWORD \
  --scan --pattern "seg:*" | xargs redis-cli UNLINK
```

FMS instances will detect L2 misses and re-warm from PostgreSQL. The singleflight group limits concurrent DB reads to one per flag key, so the DB load spike will be bounded.

---

## Scaling Guidance

### When to Scale Out

Scale FMS horizontally when:
- CPU utilization exceeds 70% on average across pods
- p99 evaluation latency exceeds 5ms (with healthy cache hit rates)
- The HPA has been at max replicas for more than 10 minutes

Each additional FMS pod adds:
- Approximately 10,000-15,000 evaluations/second of additional capacity (L1-hit path)
- `DB_MAX_OPEN_CONNS` additional PostgreSQL connections

### Scaling Procedure

The HPA handles routine scaling automatically. For manual scaling:

```bash
# Scale to 10 replicas immediately
kubectl scale deployment/fms --replicas=10

# Check current replica count and status
kubectl get deployment fms
kubectl get pods -l app=fms
```

### Scaling Redis

If Redis becomes the bottleneck (high latency on L2 hits):

1. **Upgrade to a larger Redis instance** — most workloads are memory-bound; more RAM means fewer evictions.
2. **Enable Redis Cluster** — distributes keys across multiple Redis nodes. Update `REDIS_URL` to a cluster-aware URL. FMS's Redis client supports Redis Cluster.
3. **Add Redis read replicas** — route read commands to replicas. Pub/sub must remain on the primary.

### Scaling PostgreSQL

If DB reads become a bottleneck (high L2 miss rate during cold starts or post-invalidation):

1. **Add a read replica** and configure FMS to route SELECT queries there via a separate `DATABASE_READ_URL` environment variable.
2. **Increase `max_connections`** on the primary (or use PgBouncer connection pooling).
3. **Add an index** on frequently queried columns if slow query logs reveal missing indexes.

---

## Backup and Recovery

### PostgreSQL Backups

Production PostgreSQL is backed up using continuous WAL archiving to object storage (e.g., S3) via pgBackRest or Barman. Point-in-time recovery (PITR) is available to any moment within the retention window.

**Backup schedule:**
- Full base backup: weekly (Sunday 02:00 UTC)
- WAL archiving: continuous (every 60 seconds)
- Retention: 30 days

**Verify backups are running:**
```bash
# Check last successful backup timestamp
kubectl exec -it postgres-0 -- pgbackrest --stanza=fms info
```

**Test restore (do this in staging, not production):**
```bash
# Restore to a point in time
pgbackrest --stanza=fms --type=time \
  "--target=2026-06-16 12:00:00+00" restore
```

**Recovery Time Objective (RTO):** < 30 minutes for full restore from latest backup.
**Recovery Point Objective (RPO):** < 60 seconds (WAL archiving interval).

### Redis Persistence

Redis is configured with both RDB snapshots and AOF (Append Only File) persistence:

- **RDB**: Snapshot every 15 minutes if at least 1 key changed
- **AOF**: `appendfsync everysec` (at most 1 second of data loss)

However, Redis is a **cache**, not a source of truth. If Redis data is lost entirely:

1. FMS instances detect L2 misses and re-warm from PostgreSQL automatically.
2. There is a transient spike in DB reads during the re-warm period (bounded by singleflight).
3. All services continue to function correctly; evaluation latency temporarily increases from ~200ns (L1) to ~5ms (DB read) until L1 is warmed.

Losing Redis data is not a data-loss incident. Do not treat it as one.

### Flag Configuration Backup

Because all flag definitions are in PostgreSQL, they are covered by the PostgreSQL backup policy above. Additionally, the audit log provides a complete history of every change, which can be replayed to reconstruct the state of any flag at any point in time.

For operator peace of mind, you can export a snapshot of all flag configurations:

```bash
curl -s "http://localhost:8080/v1/applications/${APP_ID}/environments/${ENV_ID}/flags?limit=100" \
  -H "Authorization: Bearer dev-token" | jq . > flags-backup-$(date +%Y%m%d).json
```

---

## On-Call Escalation Path

### Severity Definitions

| Severity | Definition | Response Time |
|---|---|---|
| SEV-1 | FMS is completely unavailable or all evaluations are returning errors. Active customer impact. | Immediate (< 5 minutes) |
| SEV-2 | Significant degradation: p99 latency > 50ms, error rate > 5%, or flag changes not propagating for > 5 minutes. | < 15 minutes |
| SEV-3 | Minor degradation: elevated latency, low cache hit rate, non-critical alerts firing. No direct customer impact. | < 2 hours |
| SEV-4 | Informational: alerts that warrant investigation but are not causing customer impact. | Next business day |

### Escalation Chain

1. **On-call engineer** (primary) — Receives PagerDuty alert. Responsible for initial triage and remediation for SEV-2 and below. Escalates to SEV-1 channel if unable to resolve within 30 minutes.

2. **Platform team lead** — Escalate for SEV-1, or when the root cause is suspected to be infrastructure (Kubernetes node failures, PostgreSQL primary failover, Redis cluster split-brain).

3. **Database administrator (DBA)** — Escalate for any PostgreSQL-related incident: connection exhaustion, replication lag > 60 seconds, storage approaching capacity, or query performance regression.

4. **Redis/infrastructure specialist** — Escalate for Redis Cluster issues: shard failures, pub/sub delivery failures affecting all instances, or memory pressure causing aggressive eviction.

5. **Service owner (flagged team)** — Escalate if the root cause is in application-level flag configuration (e.g., a rule that is causing evaluation errors for a specific flag). The flagged team owns flag configuration, not the FMS platform team.

### Useful Commands for First Responders

```bash
# Check pod health across all instances
kubectl get pods -l app=fms -o wide

# Tail logs from all pods
kubectl logs -l app=fms -f --prefix

# Get a quick health summary
curl -s http://fms.internal.example.com/readyz | jq .

# Check current error rate
kubectl exec -it <fms-pod> -- \
  curl -s http://localhost:8080/metrics | grep fms_http_requests_total

# Check Redis connectivity from within a pod
kubectl exec -it <fms-pod> -- redis-cli -h $REDIS_HOST ping

# Check PostgreSQL connectivity from within a pod
kubectl exec -it <fms-pod> -- psql $DATABASE_URL -c "SELECT 1;"

# Force a rolling restart (clears L1 caches, re-establishes Redis subscriptions)
kubectl rollout restart deployment/fms
kubectl rollout status deployment/fms
```

### Post-Incident

After any SEV-1 or SEV-2 incident, a post-incident review (PIR) is required within 3 business days. The PIR template is in the internal wiki. At minimum, document:

- Timeline of the incident (detection, triage, mitigation, resolution)
- Root cause
- Contributing factors
- Action items to prevent recurrence (with owners and due dates)
