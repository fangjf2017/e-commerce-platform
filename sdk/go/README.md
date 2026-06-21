# Feature Flags SDK for Go

A lightweight Go client SDK for the Feature Management Service. It supports single-flag evaluation, batch evaluation, typed value helpers, and an in-memory TTL cache to reduce network round trips.

---

## Installation

```bash
go get github.com/ecommerce/featureflags-sdk-go
```

Requires Go 1.24 or later.

---

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    featureflags "github.com/ecommerce/featureflags-sdk-go"
)

func main() {
    client := featureflags.New(featureflags.Config{
        BaseURL:       "https://fms.internal",
        APIKey:        "your-api-key",
        ApplicationID: "550e8400-e29b-41d4-a716-446655440000",
        EnvironmentID: "7f6e5d4c-3b2a-1908-7654-321098765432",
        CacheTTL:      30 * time.Second,
        HTTPTimeout:   5 * time.Second,
    })

    ctx := context.Background()

    evalCtx := featureflags.EvalContext{
        EntityID:   "user-123",
        EntityType: "user",
        Attributes: map[string]interface{}{
            "user.role":  "beta",
            "user.email": "alice@example.com",
            "user.plan":  "pro",
        },
    }

    // Check a boolean flag
    if client.IsEnabled(ctx, "checkout.new_ui", evalCtx) {
        fmt.Println("New checkout UI is active")
    }

    // Get a string flag value with a fallback
    theme := client.StringValue(ctx, "ui.theme", "default", evalCtx)
    fmt.Println("Active theme:", theme)
}
```

---

## Configuration Options

| Field | Type | Default | Description |
|---|---|---|---|
| `BaseURL` | `string` | — | Base URL of the Feature Management Service (no trailing slash) |
| `APIKey` | `string` | — | Bearer token used for authentication |
| `ApplicationID` | `string` | — | UUID identifying your application |
| `EnvironmentID` | `string` | — | UUID of the target environment (e.g. production, staging) |
| `HTTPTimeout` | `time.Duration` | `5s` | Per-request HTTP timeout |
| `CacheTTL` | `time.Duration` | `30s` | Local in-memory cache TTL; set to `0` to use the default 30 s |

---

## Usage Examples

### BoolValue — read a boolean flag

```go
enabled := client.BoolValue(ctx, "payments.stripe_v3", false, evalCtx)
if enabled {
    // use Stripe v3 flow
}
```

### StringValue — read a string flag

```go
algo := client.StringValue(ctx, "search.ranking_algorithm", "bm25", evalCtx)
fmt.Println("Using ranking algorithm:", algo)
```

### NumberValue — read a numeric flag

```go
limit := client.NumberValue(ctx, "cart.max_items", 50, evalCtx)
fmt.Printf("Cart item limit: %.0f\n", limit)
```

### IsEnabled — boolean shorthand

`IsEnabled` is sugar for `BoolValue(..., false, ...)`.

```go
if client.IsEnabled(ctx, "feature.dark_mode", evalCtx) {
    renderDarkMode()
}
```

### Evaluate — full result with metadata

```go
result, err := client.Evaluate(ctx, "checkout.new_ui", evalCtx)
if err != nil {
    log.Println("flag evaluation failed:", err)
} else {
    fmt.Println("variant:", result.VariantKey)
    fmt.Println("cache hit:", result.CacheHit)
}
```

### EvaluateWithExplanation — bypass cache, get full rule trace

Always makes a live network request so the explanation reflects the current rule state.

```go
result, err := client.EvaluateWithExplanation(ctx, "checkout.new_ui", evalCtx)
if err != nil {
    log.Fatal(err)
}
exp := result.Explanation
fmt.Println("Reason:", exp.Reason)
fmt.Println("Matched rule:", exp.MatchedRuleName)
for _, step := range exp.Steps {
    fmt.Printf("  Rule %q (priority %d): %s\n", step.RuleName, step.Priority, step.Outcome)
}
```

### BatchEvaluate — evaluate multiple flags in one request

Useful at page/request load time to pre-fetch all flags needed for a session. Results are automatically written into the local cache.

```go
keys := []string{
    "checkout.new_ui",
    "payments.stripe_v3",
    "search.ranking_algorithm",
}

results, err := client.BatchEvaluate(ctx, keys, evalCtx)
if err != nil {
    log.Fatal(err)
}
for _, r := range results {
    fmt.Printf("%s = %s\n", r.FlagKey, string(r.Value))
}
```

---

## The Explanation Model

Every `EvaluationResult` carries an `Explanation` struct that describes exactly why the SDK received a particular value. Use it for debugging targeting rules or auditing decisions.

```
Explanation
├── FlagKey / FlagName / FlagVersion   — which flag was evaluated
├── EnvironmentSlug                    — which environment (e.g. "production")
├── EntityID / EntityType              — who was evaluated
├── DefaultServed                      — true when no rule matched and the default was returned
├── MatchedRuleID / MatchedRuleName    — the first rule that fired (nil if default served)
├── MatchedRuleType                    — "targeting" | "rollout" | "schedule"
├── BucketValue                        — the deterministic hash bucket (0–9999) used for rollouts
├── Reason                             — a human-readable one-liner
└── Steps                              — ordered list of every rule the engine inspected
    └── ExplanationStep
        ├── RuleName / RuleType / Priority
        ├── Outcome    — "matched" | "skipped" | "schedule_miss" | "disabled" | "flag_disabled"
        └── ConditionResults
            └── ConditionResult
                ├── Attribute / Operator / Value   — the rule condition
                ├── ActualValue                    — what the entity's attribute resolved to
                └── Matched                        — whether this condition passed
```

**Example: reading the explanation**

```go
result, _ := client.EvaluateWithExplanation(ctx, "checkout.new_ui", evalCtx)
exp := result.Explanation

if exp.DefaultServed {
    fmt.Println("No rule matched — default value served")
} else {
    fmt.Printf("Matched rule: %s (%s)\n", exp.MatchedRuleName, exp.MatchedRuleType)
    fmt.Println("Reason:", exp.Reason)
}

for _, step := range exp.Steps {
    if step.Outcome == "matched" {
        for _, cond := range step.ConditionResults {
            fmt.Printf("  %s %s %v → actual: %v (matched: %v)\n",
                cond.Attribute, cond.Operator, cond.Value,
                cond.ActualValue, cond.Matched,
            )
        }
    }
}
```

---

## Caching

The SDK keeps an in-memory TTL cache keyed by `{applicationID}:{environmentID}:{flagKey}:{entityID}`.

- **Default TTL:** 30 seconds (configurable via `Config.CacheTTL`).
- **BatchEvaluate** warms the cache for all returned flags automatically.
- **EvaluateWithExplanation** always bypasses the cache to guarantee a fresh result.
- A background goroutine sweeps expired entries every 60 seconds to bound memory usage.

### Invalidating a specific entry

```go
// Force the next Evaluate call for this flag + entity to go to the network
client.InvalidateCache("checkout.new_ui", "user-123")
```

### Disabling caching

Pass a very short TTL (e.g. `1 * time.Millisecond`) if you need every call to go to the network. A TTL of `0` in `Config.CacheTTL` falls back to the 30 s default rather than disabling the cache.
