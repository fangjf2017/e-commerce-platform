# Feature Flags SDK for TypeScript / JavaScript

A lightweight TypeScript client for the Feature Management Service. Works in Node.js 18+ and modern browsers. Supports typed flag evaluation, batch fetching, in-memory TTL caching, and full rule-trace explanations.

---

## Installation

```bash
npm install @ecommerce/featureflags-sdk
```

Or with yarn / pnpm:

```bash
yarn add @ecommerce/featureflags-sdk
pnpm add @ecommerce/featureflags-sdk
```

---

## Quick Start

### TypeScript

```typescript
import { FeatureFlagsClient } from '@ecommerce/featureflags-sdk';

const client = new FeatureFlagsClient({
  baseUrl: 'https://fms.internal',
  apiKey: 'your-api-key',
  applicationId: '550e8400-e29b-41d4-a716-446655440000',
  environmentId: '7f6e5d4c-3b2a-1908-7654-321098765432',
  cacheTtlMs: 30_000,
  httpTimeoutMs: 5_000,
});

const ctx = {
  entityId: 'user-123',
  entityType: 'user',
  attributes: {
    'user.role': 'beta',
    'user.email': 'alice@example.com',
    'user.plan': 'pro',
  },
};

// Boolean flag check
const showNewUI = await client.isEnabled('checkout.new_ui', ctx);
if (showNewUI) {
  renderNewCheckout();
}

// String flag
const theme = await client.stringValue('ui.theme', 'default', ctx);
console.log('Active theme:', theme);
```

### JavaScript (CommonJS)

```javascript
const { FeatureFlagsClient } = require('@ecommerce/featureflags-sdk');

const client = new FeatureFlagsClient({
  baseUrl: 'https://fms.internal',
  apiKey: process.env.FEATURE_FLAGS_API_KEY,
  applicationId: process.env.APP_ID,
  environmentId: process.env.ENV_ID,
});

const ctx = { entityId: req.user.id, entityType: 'user', attributes: req.user };

const enabled = await client.isEnabled('new_checkout', ctx);
```

---

## API Reference

All evaluation methods are async and return a Promise. On network or server error, the typed value helpers (`boolValue`, `stringValue`, `numberValue`, `isEnabled`) return the provided `defaultValue` rather than throwing, making them safe to call without try/catch in hot paths.

### `boolValue(flagKey, defaultValue, ctx)`

Evaluates a boolean flag. Returns `defaultValue` on any error.

```typescript
const enabled = await client.boolValue('payments.stripe_v3', false, ctx);
if (enabled) {
  // use Stripe v3 flow
}
```

### `stringValue(flagKey, defaultValue, ctx)`

Evaluates a string flag. Returns `defaultValue` on any error.

```typescript
const algo = await client.stringValue('search.ranking_algorithm', 'bm25', ctx);
console.log('Ranking algorithm:', algo);
```

### `numberValue(flagKey, defaultValue, ctx)`

Evaluates a numeric flag. Returns `defaultValue` on any error.

```typescript
const maxItems = await client.numberValue('cart.max_items', 50, ctx);
console.log('Cart limit:', maxItems);
```

### `isEnabled(flagKey, ctx)`

Shorthand for `boolValue(flagKey, false, ctx)`.

```typescript
if (await client.isEnabled('feature.dark_mode', ctx)) {
  applyDarkMode();
}
```

### `evaluate(flagKey, ctx)`

Returns the full `EvaluationResult` including metadata. Results are served from cache when available.

```typescript
const result = await client.evaluate('checkout.new_ui', ctx);
console.log('Variant:', result.variantKey);
console.log('Cache hit:', result.cacheHit);
console.log('Cache layer:', result.cacheLayer);
```

### `evaluateWithExplanation(flagKey, ctx)`

Same as `evaluate` but always bypasses the local cache, ensuring a fresh response with a current rule trace. Use for debugging targeting rules.

```typescript
const result = await client.evaluateWithExplanation('checkout.new_ui', ctx);
const { explanation } = result;

console.log('Reason:', explanation.reason);
console.log('Matched rule:', explanation.matchedRuleName);
console.log('Default served:', explanation.defaultServed);

for (const step of explanation.steps) {
  console.log(`  Rule "${step.ruleName}" [${step.ruleType}] priority ${step.priority}: ${step.outcome}`);
  for (const cond of step.conditionResults ?? []) {
    console.log(`    ${cond.attribute} ${cond.operator} ${JSON.stringify(cond.value)}`);
    console.log(`    actual: ${JSON.stringify(cond.actualValue)} matched: ${cond.matched}`);
  }
}
```

### `batchEvaluate(flagKeys, ctx)`

Evaluates multiple flags in a single HTTP request. All results are written into the local cache automatically. Ideal for pre-fetching all flags needed at page load or request start.

```typescript
const results = await client.batchEvaluate([
  'checkout.new_ui',
  'payments.stripe_v3',
  'search.ranking_algorithm',
], ctx);

for (const result of results) {
  console.log(result.flagKey, '=', result.value);
}
```

### `dryRun(flagKey, ctx)`

Evaluates a flag without recording any impression or analytics event on the server side. Useful for unit tests or pre-production rule validation. Always bypasses the local cache.

```typescript
const result = await client.dryRun('checkout.new_ui', {
  entityId: 'test-user-999',
  attributes: { 'user.role': 'beta' },
});
console.log('Dry-run value:', result.value);
```

### `invalidateCache(flagKey, entityId)`

Removes a single flag+entity combination from the local cache. The next `evaluate` call for that combination will go to the network.

```typescript
client.invalidateCache('checkout.new_ui', 'user-123');
```

### `clearCache()`

Removes all entries from the local cache.

```typescript
client.clearCache();
```

### `destroy()`

Stops the background cache-cleanup timer and clears all entries. Call this when the client is no longer needed (e.g. in test teardown or server shutdown) to avoid keeping the Node.js event loop alive.

```typescript
client.destroy();
```

---

## EvalContext

`EvalContext` describes the entity being evaluated and the attributes the rule engine uses for targeting.

```typescript
interface EvalContext {
  entityId: string;              // Required. Unique identifier for the entity (e.g. user ID, device ID)
  entityType?: string;           // Optional. Defaults to "user" on the server side
  attributes?: Record<string, unknown>; // Key-value pairs used by targeting rules
}
```

Attribute keys follow a dot-notation convention by default (e.g. `user.role`, `account.plan`, `device.platform`), but the exact names are determined by how rules are configured in the Feature Management Service.

```typescript
const ctx: EvalContext = {
  entityId: 'user-123',
  entityType: 'user',
  attributes: {
    'user.role': 'beta',
    'user.country': 'US',
    'account.plan': 'enterprise',
    'device.platform': 'web',
  },
};
```

---

## The Explanation Model

Every `EvaluationResult` carries an `explanation` field that documents exactly why the SDK received the value it did. The explanation is most useful when retrieved via `evaluateWithExplanation` (which bypasses the cache for a live trace) or via `evaluate` when the response is already fresh.

```
EvaluationResult.explanation (Explanation)
├── flagKey / flagName / flagVersion    which flag, which version
├── environmentSlug                     e.g. "production"
├── entityId / entityType               who was evaluated
├── defaultServed                       true when no rule matched
├── matchedRuleId / matchedRuleName     first rule that fired (undefined if default)
├── matchedRuleType                     "targeting" | "rollout" | "schedule"
├── bucketValue                         deterministic hash bucket 0–9999 (rollouts only)
├── reason                              human-readable one-liner
└── steps                               every rule the engine inspected, in priority order
    └── ExplanationStep
        ├── ruleName / ruleType / priority
        ├── outcome   "matched" | "skipped" | "schedule_miss" | "disabled" | "flag_disabled"
        └── conditionResults
            └── ConditionResult
                ├── attribute     the attribute key from the rule definition
                ├── operator      e.g. "in", "eq", "gte", "contains"
                ├── value         the rule's configured value
                ├── actualValue   what the entity's attribute resolved to at evaluation time
                └── matched       whether this individual condition passed
```

**Reading the explanation**

```typescript
const result = await client.evaluateWithExplanation('checkout.new_ui', ctx);
const exp = result.explanation;

if (exp.defaultServed) {
  console.log('No rule matched; default value was served');
} else {
  console.log(`Matched: "${exp.matchedRuleName}" (${exp.matchedRuleType})`);
  console.log('Reason:', exp.reason);
}

// Walk every rule step to see what the engine considered
for (const step of exp.steps) {
  console.log(`[${step.priority}] ${step.ruleName}: ${step.outcome}`);
  if (step.outcome === 'matched') {
    for (const cond of step.conditionResults ?? []) {
      console.log(
        `  ${cond.attribute} ${cond.operator} ${JSON.stringify(cond.value)}`,
        `→ got ${JSON.stringify(cond.actualValue)}`,
        cond.matched ? '✓' : '✗',
      );
    }
  }
}
```

---

## Caching

The SDK keeps a per-instance in-memory TTL cache. The cache key is `{applicationId}:{environmentId}:{flagKey}:{entityId}`, so each unique combination of application, environment, flag, and entity is cached independently.

| Method | Cache behaviour |
|---|---|
| `evaluate` | Read-through: serves from cache if available, otherwise fetches and stores |
| `boolValue` / `stringValue` / `numberValue` / `isEnabled` | Delegates to `evaluate`, same behaviour |
| `evaluateWithExplanation` | Always bypasses cache (live network request) |
| `batchEvaluate` | Always fetches; warms cache for all returned flags |
| `dryRun` | Always bypasses cache |
| `invalidateCache(flagKey, entityId)` | Removes one entry |
| `clearCache()` | Removes all entries |

**Default TTL:** 30,000 ms (30 seconds). Configure via `cacheTtlMs` in `ClientConfig`.

A background `setInterval` sweeps expired entries every 60 seconds. In Node.js the timer is `unref`'d so it does not prevent the process from exiting. Call `client.destroy()` to stop the timer immediately.

---

## Browser vs Node.js Compatibility

The SDK uses the standard `fetch` API (`globalThis.fetch`) for all HTTP requests, which is available in:

- **Node.js 18+** — native `fetch` is included without any polyfill
- **Modern browsers** — Chrome 66+, Firefox 57+, Safari 12.1+, Edge 79+

For older Node.js versions (14–16) you can polyfill `fetch` globally before creating the client:

```javascript
const fetch = require('node-fetch');
globalThis.fetch = fetch;
```

The `AbortController`-based timeout is also part of the standard Web APIs and is available in the same environments.

**Note for browser usage:** avoid embedding your `apiKey` in client-side code that is served to end users. Either proxy evaluation requests through your backend or use a restricted read-only key scoped to the specific application and environment.

---

## ClientConfig Reference

| Field | Type | Default | Description |
|---|---|---|---|
| `baseUrl` | `string` | — | Base URL of the Feature Management Service (no trailing slash) |
| `apiKey` | `string` | — | Bearer token used for authentication |
| `applicationId` | `string` | — | UUID identifying your application |
| `environmentId` | `string` | — | UUID of the target environment |
| `cacheTtlMs` | `number` | `30000` | Local cache TTL in milliseconds |
| `httpTimeoutMs` | `number` | `5000` | Per-request HTTP timeout in milliseconds |
