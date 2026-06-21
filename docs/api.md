# API Reference

Complete REST API reference for the Feature Management Service.

---

## Table of Contents

- [Authentication](#authentication)
- [Request and Response Format](#request-and-response-format)
- [Error Codes](#error-codes)
- [Applications](#applications)
- [Environments](#environments)
- [Flags](#flags)
- [Rules](#rules)
- [Segments](#segments)
- [Evaluation API](#evaluation-api)
- [Audit Log](#audit-log)
- [System Endpoints](#system-endpoints)

---

## Authentication

All API endpoints (except `/healthz`, `/readyz`, and `/metrics`) require authentication via a Bearer token or an API key header.

### Bearer Token

```
Authorization: Bearer <token>
```

Tokens are issued by your identity provider and validated against the `API_TOKEN_SECRET` configured in the service.

### API Key

```
X-API-Key: <api-key>
```

API keys are created through the management UI and scoped to a specific application. SDK integrations should use API keys rather than Bearer tokens.

### Obtaining Credentials

For development, a static Bearer token can be configured via the `DEV_BEARER_TOKEN` environment variable. In production, integrate with your organization's token issuance system.

---

## Request and Response Format

### Content Type

All requests and responses use `application/json`. Always set `Content-Type: application/json` on requests with a body.

### Response Envelope

Successful responses return the resource directly at the top level (not wrapped in a `data` key):

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "checkout-service",
  "slug": "checkout",
  "created_at": "2026-06-16T14:00:00Z",
  "updated_at": "2026-06-16T14:00:00Z"
}
```

List responses include a `items` array and pagination metadata:

```json
{
  "items": [ ... ],
  "total": 42,
  "limit": 20,
  "offset": 0
}
```

### Error Format

All errors use a consistent structure:

```json
{
  "error": {
    "code": "flag_not_found",
    "message": "Flag 'checkout.new_ui' not found in environment 'production'",
    "request_id": "req-abc123"
  }
}
```

| Field | Description |
|---|---|
| `code` | Machine-readable error code (see [Error Codes](#error-codes)) |
| `message` | Human-readable description |
| `request_id` | Unique request identifier for tracing (also in `X-Request-ID` response header) |

### Pagination

List endpoints accept `limit` (default 20, max 100) and `offset` query parameters.

```
GET /v1/applications?limit=50&offset=100
```

### Timestamps

All timestamps are RFC3339 strings in UTC (e.g., `2026-06-16T14:44:22Z`).

### UUIDs

All IDs are UUIDs in standard hyphenated format (e.g., `550e8400-e29b-41d4-a716-446655440000`).

---

## Error Codes

| HTTP Status | Code | Meaning |
|---|---|---|
| 400 | `validation_error` | Request body failed validation. `message` describes the specific field. |
| 401 | `unauthorized` | Missing or invalid authentication credentials. |
| 403 | `forbidden` | Authenticated but not authorized to perform this action. |
| 404 | `application_not_found` | The specified application does not exist. |
| 404 | `environment_not_found` | The specified environment does not exist. |
| 404 | `flag_not_found` | The specified flag does not exist. |
| 404 | `rule_not_found` | The specified rule does not exist. |
| 404 | `segment_not_found` | The specified segment does not exist. |
| 404 | `audit_event_not_found` | The specified audit event does not exist. |
| 409 | `slug_conflict` | An application or environment with this slug already exists. |
| 409 | `flag_key_conflict` | A flag with this key already exists in this environment. |
| 409 | `version_conflict` | Optimistic concurrency check failed (If-Match header mismatch). |
| 422 | `invalid_rule_type` | `rule_type` must be `targeting` or `rollout`. |
| 422 | `invalid_operator` | The condition operator is not supported. |
| 422 | `rollout_pct_out_of_range` | `rollout_pct` must be between 0 and 100. |
| 422 | `variant_allocation_gap` | Variant allocations do not cover the full rollout range. |
| 429 | `rate_limited` | Too many requests. Retry after the duration in the `Retry-After` header. |
| 500 | `internal_error` | Unexpected server error. Contact the on-call team. |
| 503 | `service_unavailable` | Service is starting up or a required dependency is unavailable. |

---

## Applications

Applications are the top-level organizational unit. Each product or service that uses feature flags is typically one application.

---

### Create Application

```
POST /v1/applications
```

**Request Body**

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Human-readable display name |
| `slug` | string | Yes | URL-safe identifier, unique across all applications. Alphanumeric and hyphens only. |
| `description` | string | No | Optional description |

**Example Request**

```bash
curl -s -X POST http://localhost:8080/v1/applications \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Checkout Service",
    "slug": "checkout",
    "description": "Manages the end-to-end checkout flow"
  }' | jq .
```

**Example Response** `201 Created`

```json
{
  "id": "a1b2c3d4-0000-0000-0000-000000000001",
  "name": "Checkout Service",
  "slug": "checkout",
  "description": "Manages the end-to-end checkout flow",
  "created_at": "2026-06-16T14:00:00Z",
  "updated_at": "2026-06-16T14:00:00Z"
}
```

---

### List Applications

```
GET /v1/applications
```

**Query Parameters**

| Parameter | Type | Default | Description |
|---|---|---|---|
| `limit` | int | 20 | Max results per page (max 100) |
| `offset` | int | 0 | Pagination offset |

**Example Request**

```bash
curl -s "http://localhost:8080/v1/applications?limit=10" \
  -H "Authorization: Bearer dev-token" | jq .
```

**Example Response** `200 OK`

```json
{
  "items": [
    {
      "id": "a1b2c3d4-0000-0000-0000-000000000001",
      "name": "Checkout Service",
      "slug": "checkout",
      "created_at": "2026-06-16T14:00:00Z",
      "updated_at": "2026-06-16T14:00:00Z"
    }
  ],
  "total": 1,
  "limit": 10,
  "offset": 0
}
```

---

### Get Application

```
GET /v1/applications/{appID}
```

**Example Request**

```bash
curl -s "http://localhost:8080/v1/applications/a1b2c3d4-0000-0000-0000-000000000001" \
  -H "Authorization: Bearer dev-token" | jq .
```

**Example Response** `200 OK`

```json
{
  "id": "a1b2c3d4-0000-0000-0000-000000000001",
  "name": "Checkout Service",
  "slug": "checkout",
  "description": "Manages the end-to-end checkout flow",
  "created_at": "2026-06-16T14:00:00Z",
  "updated_at": "2026-06-16T14:00:00Z"
}
```

---

### Update Application

```
PATCH /v1/applications/{appID}
```

All fields are optional; only provided fields are updated.

**Request Body**

| Field | Type | Description |
|---|---|---|
| `name` | string | New display name |
| `description` | string | New description |

**Example Request**

```bash
curl -s -X PATCH \
  "http://localhost:8080/v1/applications/a1b2c3d4-0000-0000-0000-000000000001" \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{"description": "Updated description"}' | jq .
```

**Example Response** `200 OK` — updated application object.

---

### Delete Application

```
DELETE /v1/applications/{appID}
```

Deletes the application and all associated environments, flags, rules, segments, and audit events. This operation is irreversible.

**Example Request**

```bash
curl -s -X DELETE \
  "http://localhost:8080/v1/applications/a1b2c3d4-0000-0000-0000-000000000001" \
  -H "Authorization: Bearer dev-token"
```

**Example Response** `204 No Content`

---

## Environments

Environments isolate flag state within an application (e.g., production, staging, development).

---

### Create Environment

```
POST /v1/applications/{appID}/environments
```

**Request Body**

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Display name |
| `slug` | string | Yes | URL-safe identifier, unique within the application |
| `color` | string | No | Hex color for UI display (e.g., `#e53e3e`) |

**Example Request**

```bash
curl -s -X POST \
  "http://localhost:8080/v1/applications/${APP_ID}/environments" \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production",
    "slug": "production",
    "color": "#e53e3e"
  }' | jq .
```

**Example Response** `201 Created`

```json
{
  "id": "e1e2e3e4-0000-0000-0000-000000000001",
  "app_id": "a1b2c3d4-0000-0000-0000-000000000001",
  "name": "Production",
  "slug": "production",
  "color": "#e53e3e",
  "created_at": "2026-06-16T14:00:00Z",
  "updated_at": "2026-06-16T14:00:00Z"
}
```

---

### List, Get, Update, Delete Environment

```
GET    /v1/applications/{appID}/environments
GET    /v1/applications/{appID}/environments/{envID}
PATCH  /v1/applications/{appID}/environments/{envID}
DELETE /v1/applications/{appID}/environments/{envID}
```

Follow the same patterns as the Applications endpoints above.

---

## Flags

Flags live within an environment and carry the variants and rules that control behavior.

---

### Create Flag

```
POST /v1/applications/{appID}/environments/{envID}/flags/{flagKey}
```

The `flagKey` is part of the URL path. It must be unique within the environment. Use dot-notation to namespace flags (e.g., `checkout.new_ui`, `homepage.banner`).

**Request Body**

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Human-readable display name |
| `description` | string | No | Optional description |
| `flag_type` | string | Yes | `boolean`, `string`, `integer`, or `json` |
| `default_value` | string | Yes | Value returned when no rule matches. Must be a string representation of the type (e.g., `"false"`, `"control"`, `"42"`). |
| `variants` | array | Yes | At least one variant. See below. |
| `tags` | array of strings | No | Arbitrary tags for filtering in the management UI |

**Variant Object**

| Field | Type | Required | Description |
|---|---|---|---|
| `key` | string | Yes | Unique identifier for this variant within the flag |
| `value` | string | Yes | The value returned for this variant. Must be compatible with `flag_type`. |
| `description` | string | No | Optional description |

**Example Request**

```bash
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
      {"key": "on",  "value": "true",  "description": "New checkout UI enabled"},
      {"key": "off", "value": "false", "description": "Legacy checkout UI"}
    ],
    "tags": ["checkout", "ui"]
  }' | jq .
```

**Example Response** `201 Created`

```json
{
  "id": "f1f2f3f4-0000-0000-0000-000000000001",
  "app_id": "a1b2c3d4-0000-0000-0000-000000000001",
  "env_id": "e1e2e3e4-0000-0000-0000-000000000001",
  "flag_key": "checkout.new_ui",
  "name": "Checkout New UI Feature",
  "description": "Enables the redesigned checkout flow",
  "flag_type": "boolean",
  "default_value": "false",
  "status": "active",
  "version": 1,
  "variants": [
    {"key": "on",  "value": "true"},
    {"key": "off", "value": "false"}
  ],
  "tags": ["checkout", "ui"],
  "created_at": "2026-06-16T14:00:00Z",
  "updated_at": "2026-06-16T14:00:00Z"
}
```

---

### Get Flag

```
GET /v1/applications/{appID}/environments/{envID}/flags/{flagKey}
```

**Example Request**

```bash
curl -s \
  "http://localhost:8080/v1/applications/${APP_ID}/environments/${ENV_ID}/flags/checkout.new_ui" \
  -H "Authorization: Bearer dev-token" | jq .
```

Returns the flag object as shown above, including all variants and current rules.

---

### List Flags

```
GET /v1/applications/{appID}/environments/{envID}/flags
```

**Query Parameters**

| Parameter | Type | Description |
|---|---|---|
| `status` | string | Filter by status: `active`, `disabled`, `archived` |
| `tag` | string | Filter by tag |
| `limit` | int | Max results per page (default 20, max 100) |
| `offset` | int | Pagination offset |

---

### Update Flag

```
PUT /v1/applications/{appID}/environments/{envID}/flags/{flagKey}
```

Full replacement of the flag definition (name, description, default_value, variants, tags). Rules are managed separately via the Rules endpoints.

**Optimistic Concurrency**

Include `If-Match: <version>` to prevent lost updates:

```bash
curl -s -X PUT \
  "http://localhost:8080/v1/applications/${APP_ID}/environments/${ENV_ID}/flags/checkout.new_ui" \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -H "If-Match: 7" \
  -d '{...}' | jq .
```

Returns `409 Conflict` with code `version_conflict` if the current version does not match.

---

### Enable / Disable / Archive Flag

```
POST /v1/applications/{appID}/environments/{envID}/flags/{flagKey}/enable
POST /v1/applications/{appID}/environments/{envID}/flags/{flagKey}/disable
POST /v1/applications/{appID}/environments/{envID}/flags/{flagKey}/archive
```

These are convenience actions that set `status` without requiring a full PUT. A disabled flag immediately returns its `default_value` for all evaluations. An archived flag is excluded from the management UI by default.

**Example**

```bash
curl -s -X POST \
  "http://localhost:8080/v1/applications/${APP_ID}/environments/${ENV_ID}/flags/checkout.new_ui/disable" \
  -H "Authorization: Bearer dev-token"
# 204 No Content
```

---

### Delete Flag

```
DELETE /v1/applications/{appID}/environments/{envID}/flags/{flagKey}
```

Permanently deletes the flag and all its rules. Prefer archiving over deletion for production flags to preserve the audit trail.

---

## Rules

Rules define when a flag resolves to a non-default value. They are evaluated in ascending priority order.

---

### Create Rule

```
POST /v1/applications/{appID}/environments/{envID}/flags/{flagKey}/rules
```

**Request Body — Targeting Rule**

A targeting rule matches when all conditions are true and routes the entity to a single variant at 100%.

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Display name |
| `rule_type` | string | Yes | Must be `targeting` |
| `priority` | int | Yes | Evaluation order (ascending; 0 = highest priority) |
| `conditions` | array | Yes | At least one condition (see below) |
| `variant_key` | string | Yes | Variant to serve when all conditions match |
| `schedule_start` | string | No | RFC3339 timestamp; rule inactive before this time |
| `schedule_end` | string | No | RFC3339 timestamp; rule inactive after this time |

**Condition Object**

| Field | Type | Required | Description |
|---|---|---|---|
| `attribute` | string | Yes | Entity attribute path (e.g., `user.email`, `plan`, `country`) |
| `operator` | string | Yes | One of: `eq`, `neq`, `lt`, `lte`, `gt`, `gte`, `contains`, `not_contains`, `starts_with`, `ends_with`, `in`, `not_in`, `matches_regex`, `is_set`, `is_not_set` |
| `value` | string | Conditional | The value to compare against. Not required for `is_set` / `is_not_set`. For `in` / `not_in`, provide a JSON array string: `"[\"a\",\"b\"]"`. |

**Example — Targeting Rule**

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

**Request Body — Rollout Rule**

A rollout rule places each entity into a bucket (0-9999) using MurmurHash3 and assigns a variant based on the bucket range.

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Display name |
| `rule_type` | string | Yes | Must be `rollout` |
| `priority` | int | Yes | Evaluation order |
| `rollout_pct` | float | Yes | Percentage of entities in rollout (0-100). Entities with `bucket < rollout_pct * 100` are included. |
| `rollout_salt` | string | No | UUID salt for bucket computation. Auto-generated if omitted. Rotate to reassign all users. |
| `variant_allocations` | array | Yes | Bucket ranges per variant (see below). Ranges must be within [0, 10000). |
| `segment_ref` | string | No | If set, only entities in this segment are eligible for the rollout. |

**Variant Allocation Object**

| Field | Type | Description |
|---|---|---|
| `variant_key` | string | Variant to serve for this bucket range |
| `from` | int | Inclusive lower bound (0-9999) |
| `to` | int | Exclusive upper bound (1-10000) |

**Example — Rollout Rule**

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

**Example Response** `201 Created`

```json
{
  "id": "r1r2r3r4-0000-0000-0000-000000000001",
  "flag_id": "f1f2f3f4-0000-0000-0000-000000000001",
  "name": "Beta users rollout",
  "rule_type": "rollout",
  "priority": 1,
  "rollout_pct": 50,
  "rollout_salt": "7f3a9c21-aaaa-bbbb-cccc-000000000001",
  "variant_allocations": [
    {"variant_key": "on",  "from": 0,    "to": 5000},
    {"variant_key": "off", "from": 5000, "to": 10000}
  ],
  "created_at": "2026-06-16T14:00:00Z",
  "updated_at": "2026-06-16T14:00:00Z"
}
```

---

### List, Get, Update, Delete Rules

```
GET    /v1/applications/{appID}/environments/{envID}/flags/{flagKey}/rules
GET    /v1/applications/{appID}/environments/{envID}/flags/{flagKey}/rules/{ruleID}
PUT    /v1/applications/{appID}/environments/{envID}/flags/{flagKey}/rules/{ruleID}
DELETE /v1/applications/{appID}/environments/{envID}/flags/{flagKey}/rules/{ruleID}
```

---

### Reorder Rules

```
PUT /v1/applications/{appID}/environments/{envID}/flags/{flagKey}/rules/reorder
```

Atomically sets the priority of all rules for this flag.

**Request Body**

| Field | Type | Description |
|---|---|---|
| `rule_ids` | array of strings | Rule UUIDs in the desired priority order (first = priority 0) |

**Example Request**

```bash
curl -s -X PUT \
  "http://localhost:8080/v1/applications/${APP_ID}/environments/${ENV_ID}/flags/checkout.new_ui/rules/reorder" \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "rule_ids": [
      "111e8400-e29b-41d4-a716-446655440001",
      "550e8400-e29b-41d4-a716-446655440000"
    ]
  }' | jq .
```

**Example Response** `200 OK` — array of all rules with updated priorities.

---

## Segments

Segments are reusable groups of conditions that can be referenced by rollout rules. They belong to an application and are shared across all environments within it.

---

### Create Segment

```
POST /v1/applications/{appID}/segments
```

**Request Body**

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Display name |
| `description` | string | No | Optional description |
| `conditions` | array | Yes | List of condition objects (same format as rule conditions) |

**Example Request**

```bash
curl -s -X POST \
  "http://localhost:8080/v1/applications/${APP_ID}/segments" \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Pro Plan Users",
    "description": "Users on the pro subscription tier",
    "conditions": [
      {"attribute": "plan", "operator": "eq", "value": "pro"}
    ]
  }' | jq .
```

**Example Response** `201 Created`

```json
{
  "id": "s1s2s3s4-0000-0000-0000-000000000001",
  "app_id": "a1b2c3d4-0000-0000-0000-000000000001",
  "name": "Pro Plan Users",
  "description": "Users on the pro subscription tier",
  "conditions": [
    {"attribute": "plan", "operator": "eq", "value": "pro"}
  ],
  "created_at": "2026-06-16T14:00:00Z",
  "updated_at": "2026-06-16T14:00:00Z"
}
```

---

### List, Get, Update, Delete Segments

```
GET    /v1/applications/{appID}/segments
GET    /v1/applications/{appID}/segments/{segmentID}
PUT    /v1/applications/{appID}/segments/{segmentID}
DELETE /v1/applications/{appID}/segments/{segmentID}
```

---

## Evaluation API

The evaluation endpoints are the hot path: they are called by application code at request time to determine flag values.

---

### Evaluate Single Flag

```
POST /v1/evaluate
```

Returns the resolved value for a single flag, plus a complete explanation of how the decision was made.

**Request Body**

| Field | Type | Required | Description |
|---|---|---|---|
| `flag_key` | string | Yes | The flag key to evaluate |
| `app_id` | string | Yes | Application UUID |
| `environment` | string | Yes | Environment slug (e.g., `production`) |
| `context` | object | Yes | Evaluation context (see below) |

**Evaluation Context Object**

| Field | Type | Required | Description |
|---|---|---|---|
| `entity_id` | string | Yes | Stable identifier for the entity being evaluated (user ID, device ID, session ID, etc.) |
| `entity_type` | string | Yes | Type label for the entity (`user`, `device`, `session`, `org`) |
| `attributes` | object | No | Key-value map of entity attributes used by rule conditions. Values can be strings, numbers, or booleans. |

**Example Request**

```bash
curl -s -X POST http://localhost:8080/v1/evaluate \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "flag_key":    "checkout.new_ui",
    "app_id":      "a1b2c3d4-0000-0000-0000-000000000001",
    "environment": "production",
    "context": {
      "entity_id":   "user-456",
      "entity_type": "user",
      "attributes": {
        "email": "alice@acme.com",
        "plan":  "pro",
        "country": "US"
      }
    }
  }' | jq .
```

**Example Response** `200 OK`

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

**Response Fields**

| Field | Type | Description |
|---|---|---|
| `flag_key` | string | The evaluated flag key |
| `value` | string | The resolved value (always a string; parse according to `flag_type`) |
| `variant_key` | string | The key of the matched variant, or `"default"` |
| `explanation` | object | Full evaluation explanation (see Explainability section in README) |

---

### Batch Evaluate

```
POST /v1/evaluate/batch
```

Evaluates multiple flags for the same entity in a single request. The response includes all requested flags, with explanations for each. Use this for page-load or request-initialization scenarios where multiple flags are needed at once.

**Request Body**

| Field | Type | Required | Description |
|---|---|---|---|
| `flag_keys` | array of strings | Yes | List of flag keys to evaluate (max 50) |
| `app_id` | string | Yes | Application UUID |
| `environment` | string | Yes | Environment slug |
| `context` | object | Yes | Same evaluation context as single evaluate |

**Example Request**

```bash
curl -s -X POST http://localhost:8080/v1/evaluate/batch \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "flag_keys": [
      "checkout.new_ui",
      "checkout.express_pay",
      "homepage.banner"
    ],
    "app_id":      "a1b2c3d4-0000-0000-0000-000000000001",
    "environment": "production",
    "context": {
      "entity_id":   "user-456",
      "entity_type": "user",
      "attributes":  {"plan": "pro"}
    }
  }' | jq .
```

**Example Response** `200 OK`

```json
{
  "results": {
    "checkout.new_ui": {
      "flag_key": "checkout.new_ui",
      "value": "true",
      "variant_key": "on",
      "explanation": { "..." : "..." }
    },
    "checkout.express_pay": {
      "flag_key": "checkout.express_pay",
      "value": "false",
      "variant_key": "off",
      "explanation": { "default_served": true, "reason": "no rule matched" }
    },
    "homepage.banner": {
      "flag_key": "homepage.banner",
      "value": "control",
      "variant_key": "control",
      "explanation": { "..." : "..." }
    }
  },
  "evaluated_at": "2026-06-16T14:44:22Z"
}
```

Flags that don't exist return the client-supplied default value (or an empty string if none was provided) with `explanation.reason: "flag not found"`. The batch call never returns an error for missing flags — it always returns a result object for every requested flag key.

---

### Dry-Run Evaluate

```
POST /v1/evaluate/dry-run
```

Evaluates a flag exactly like the normal evaluate endpoint, but with two key differences:

1. Results are **never written to any cache**. This prevents the dry-run from warming L1 or L2 caches.
2. The request is logged at `debug` level and marked as `dry_run: true` in metrics, so it does not affect production dashboards.

Use dry-run to:
- Test new rule configurations before enabling them
- Debug why a hypothetical entity would receive a particular variant
- Validate rule logic in CI without side effects

**Request Body** — identical to single evaluate.

**Example Request**

```bash
curl -s -X POST http://localhost:8080/v1/evaluate/dry-run \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "flag_key":    "checkout.new_ui",
    "app_id":      "a1b2c3d4-0000-0000-0000-000000000001",
    "environment": "production",
    "context": {
      "entity_id":   "hypothetical-user-999",
      "entity_type": "user",
      "attributes":  {"plan": "free", "country": "CA"}
    }
  }' | jq .
```

**Example Response** — identical format to single evaluate, with an additional field:

```json
{
  "flag_key": "checkout.new_ui",
  "value": "false",
  "variant_key": "off",
  "dry_run": true,
  "explanation": { "..." : "..." }
}
```

---

## Audit Log

The audit log records every flag and segment mutation with actor, action, and before/after state.

---

### List Audit Events

```
GET /v1/applications/{appID}/audit
```

**Query Parameters**

| Parameter | Type | Description |
|---|---|---|
| `action` | string | Filter by action (e.g., `flag.created`, `flag.updated`, `rule.deleted`, `segment.updated`) |
| `resource_type` | string | Filter by resource type: `flag`, `rule`, `segment`, `environment` |
| `resource_id` | string | Filter by resource UUID |
| `actor_id` | string | Filter by actor (user ID or service account ID that made the change) |
| `from` | string | RFC3339 start of date range |
| `to` | string | RFC3339 end of date range |
| `limit` | int | Max results per page (default 20, max 100) |
| `offset` | int | Pagination offset |

**Example Request**

```bash
curl -s \
  "http://localhost:8080/v1/applications/${APP_ID}/audit?action=flag.updated&limit=5" \
  -H "Authorization: Bearer dev-token" | jq .
```

**Example Response** `200 OK`

```json
{
  "items": [
    {
      "id": "audit-uuid-001",
      "app_id": "a1b2c3d4-0000-0000-0000-000000000001",
      "action": "flag.updated",
      "resource_type": "flag",
      "resource_id": "f1f2f3f4-0000-0000-0000-000000000001",
      "actor_id": "user-admin-123",
      "before": {"status": "disabled"},
      "after":  {"status": "active"},
      "created_at": "2026-06-16T14:44:22Z"
    }
  ],
  "total": 1,
  "limit": 5,
  "offset": 0
}
```

---

### Get Audit Event

```
GET /v1/applications/{appID}/audit/{eventID}
```

Returns a single audit event by ID.

---

## System Endpoints

These endpoints do not require authentication.

---

### Liveness Check

```
GET /healthz
```

Returns `200 OK` if the process is alive and the HTTP server is accepting connections. Use this as the Kubernetes liveness probe.

```json
{"status": "ok"}
```

Returns `503 Service Unavailable` only if the server is shutting down.

---

### Readiness Check

```
GET /readyz
```

Returns `200 OK` if the service is ready to handle traffic — meaning it has healthy connections to PostgreSQL and Redis.

```json
{
  "status": "ok",
  "checks": {
    "database": "ok",
    "redis": "ok"
  }
}
```

Returns `503 Service Unavailable` with a failed check named if any dependency is unreachable. Use this as the Kubernetes readiness probe to prevent traffic from being routed to an instance that cannot reach its dependencies.

---

### Prometheus Metrics

```
GET /metrics
```

Returns metrics in the standard Prometheus text exposition format. See the Observability section in the README for the full metrics list.
