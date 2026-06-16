package featureflags

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client is the Feature Flags SDK client.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	cache      *localCache
	appID      string
	envID      string
}

// Config configures the SDK client.
type Config struct {
	BaseURL       string        // e.g. "https://fms.internal"
	APIKey        string        // API key
	ApplicationID string        // UUID of the application
	EnvironmentID string        // UUID of the environment
	HTTPTimeout   time.Duration // default 5s
	CacheTTL      time.Duration // default 30s (0 = disable cache)
}

// New creates a new Feature Flags SDK client.
func New(cfg Config) *Client {
	if cfg.HTTPTimeout == 0 {
		cfg.HTTPTimeout = 5 * time.Second
	}

	var c *localCache
	if cfg.CacheTTL > 0 {
		c = newLocalCache(cfg.CacheTTL)
	} else {
		c = newLocalCache(30 * time.Second)
	}

	return &Client{
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: cfg.HTTPTimeout,
		},
		cache: c,
		appID: cfg.ApplicationID,
		envID: cfg.EnvironmentID,
	}
}

// BoolValue evaluates a boolean flag and returns its value. Returns defaultValue on error.
func (c *Client) BoolValue(ctx context.Context, flagKey string, defaultValue bool, evalCtx EvalContext) bool {
	result, err := c.Evaluate(ctx, flagKey, evalCtx)
	if err != nil {
		return defaultValue
	}
	var v bool
	if err := json.Unmarshal(result.Value, &v); err != nil {
		return defaultValue
	}
	return v
}

// StringValue evaluates a string flag and returns its value. Returns defaultValue on error.
func (c *Client) StringValue(ctx context.Context, flagKey string, defaultValue string, evalCtx EvalContext) string {
	result, err := c.Evaluate(ctx, flagKey, evalCtx)
	if err != nil {
		return defaultValue
	}
	var v string
	if err := json.Unmarshal(result.Value, &v); err != nil {
		return defaultValue
	}
	return v
}

// NumberValue evaluates a numeric flag and returns its value. Returns defaultValue on error.
func (c *Client) NumberValue(ctx context.Context, flagKey string, defaultValue float64, evalCtx EvalContext) float64 {
	result, err := c.Evaluate(ctx, flagKey, evalCtx)
	if err != nil {
		return defaultValue
	}
	var v float64
	if err := json.Unmarshal(result.Value, &v); err != nil {
		return defaultValue
	}
	return v
}

// IsEnabled returns true if a boolean flag is active for the given context.
func (c *Client) IsEnabled(ctx context.Context, flagKey string, evalCtx EvalContext) bool {
	return c.BoolValue(ctx, flagKey, false, evalCtx)
}

// Evaluate evaluates a single flag with full explanation, using cache.
func (c *Client) Evaluate(ctx context.Context, flagKey string, evalCtx EvalContext) (*EvaluationResult, error) {
	cacheKey := c.cacheKey(flagKey, evalCtx.EntityID)
	if result, ok := c.cache.get(cacheKey); ok {
		return result, nil
	}

	result, err := c.evaluateRemote(ctx, flagKey, evalCtx)
	if err != nil {
		return nil, err
	}

	c.cache.set(cacheKey, result)
	return result, nil
}

// EvaluateWithExplanation evaluates a flag and returns the full explanation for debugging.
func (c *Client) EvaluateWithExplanation(ctx context.Context, flagKey string, evalCtx EvalContext) (*EvaluationResult, error) {
	return c.evaluateRemote(ctx, flagKey, evalCtx) // bypass cache for fresh explanation
}

// BatchEvaluate evaluates multiple flags in one round trip.
func (c *Client) BatchEvaluate(ctx context.Context, flagKeys []string, evalCtx EvalContext) ([]*EvaluationResult, error) {
	reqBody := map[string]interface{}{
		"application_id": c.appID,
		"environment_id": c.envID,
		"entity_id":      evalCtx.EntityID,
		"entity_type":    evalCtx.EntityType,
		"attributes":     evalCtx.Attributes,
		"flag_keys":      flagKeys,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/evaluate/batch", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server error: status %d", resp.StatusCode)
	}

	var response struct {
		Data []*EvaluationResult `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Warm cache
	for _, result := range response.Data {
		key := c.cacheKey(result.FlagKey, evalCtx.EntityID)
		c.cache.set(key, result)
	}

	return response.Data, nil
}

// InvalidateCache removes a flag from the local cache.
func (c *Client) InvalidateCache(flagKey, entityID string) {
	c.cache.delete(c.cacheKey(flagKey, entityID))
}

func (c *Client) evaluateRemote(ctx context.Context, flagKey string, evalCtx EvalContext) (*EvaluationResult, error) {
	reqBody := map[string]interface{}{
		"application_id": c.appID,
		"environment_id": c.envID,
		"flag_key":       flagKey,
		"entity_id":      evalCtx.EntityID,
		"entity_type":    evalCtx.EntityType,
		"attributes":     evalCtx.Attributes,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/evaluate", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server error: status %d", resp.StatusCode)
	}

	var response struct {
		Data *EvaluationResult `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return response.Data, nil
}

func (c *Client) cacheKey(flagKey, entityID string) string {
	return fmt.Sprintf("%s:%s:%s:%s", c.appID, c.envID, flagKey, entityID)
}
