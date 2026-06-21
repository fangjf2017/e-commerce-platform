import { ClientConfig, EvalContext, EvaluationResult } from './types';
import { LocalCache } from './cache';

interface ApiResponse<T> {
  data: T;
  meta?: {
    request_id?: string;
  };
}

export class FeatureFlagsClient {
  private config: Required<ClientConfig>;
  private cache: LocalCache<EvaluationResult>;

  constructor(config: ClientConfig) {
    this.config = {
      cacheTtlMs: 30_000,
      httpTimeoutMs: 5_000,
      ...config,
    };
    this.cache = new LocalCache(this.config.cacheTtlMs);
  }

  // Evaluate a single flag with caching
  async evaluate(flagKey: string, ctx: EvalContext): Promise<EvaluationResult> {
    const cacheKey = this.cacheKey(flagKey, ctx.entityId);
    const cached = this.cache.get(cacheKey);
    if (cached) return cached;

    const result = await this.evaluateRemote(flagKey, ctx);
    this.cache.set(cacheKey, result);
    return result;
  }

  // Get a boolean flag value, returns defaultValue on error
  async boolValue(flagKey: string, defaultValue: boolean, ctx: EvalContext): Promise<boolean> {
    try {
      const result = await this.evaluate(flagKey, ctx);
      return result.value as boolean;
    } catch {
      return defaultValue;
    }
  }

  // Get a string flag value, returns defaultValue on error
  async stringValue(flagKey: string, defaultValue: string, ctx: EvalContext): Promise<string> {
    try {
      const result = await this.evaluate(flagKey, ctx);
      return result.value as string;
    } catch {
      return defaultValue;
    }
  }

  // Get a numeric flag value, returns defaultValue on error
  async numberValue(flagKey: string, defaultValue: number, ctx: EvalContext): Promise<number> {
    try {
      const result = await this.evaluate(flagKey, ctx);
      return result.value as number;
    } catch {
      return defaultValue;
    }
  }

  // Returns true if a boolean flag is enabled
  async isEnabled(flagKey: string, ctx: EvalContext): Promise<boolean> {
    return this.boolValue(flagKey, false, ctx);
  }

  // Evaluate with full explanation, bypasses cache
  async evaluateWithExplanation(flagKey: string, ctx: EvalContext): Promise<EvaluationResult> {
    return this.evaluateRemote(flagKey, ctx);
  }

  // Batch evaluate multiple flags in one request
  async batchEvaluate(flagKeys: string[], ctx: EvalContext): Promise<EvaluationResult[]> {
    const body = {
      application_id: this.config.applicationId,
      environment_id: this.config.environmentId,
      entity_id: ctx.entityId,
      entity_type: ctx.entityType ?? 'user',
      attributes: ctx.attributes ?? {},
      flag_keys: flagKeys,
    };

    const response = await this.fetch<EvaluationResult[]>('/v1/evaluate/batch', body);

    // Warm cache
    for (const result of response) {
      this.cache.set(this.cacheKey(result.flagKey, ctx.entityId), result);
    }

    return response;
  }

  // Dry run for testing rules (bypasses cache, no side effects)
  async dryRun(flagKey: string, ctx: EvalContext): Promise<EvaluationResult> {
    const body = {
      application_id: this.config.applicationId,
      environment_id: this.config.environmentId,
      flag_key: flagKey,
      entity_id: ctx.entityId,
      entity_type: ctx.entityType ?? 'user',
      attributes: ctx.attributes ?? {},
    };

    return this.fetchOne<EvaluationResult>('/v1/evaluate/dry-run', body);
  }

  // Invalidate a specific flag from local cache
  invalidateCache(flagKey: string, entityId: string): void {
    this.cache.delete(this.cacheKey(flagKey, entityId));
  }

  // Clear all cached values
  clearCache(): void {
    this.cache.clear();
  }

  // Clean up resources
  destroy(): void {
    this.cache.destroy();
  }

  private async evaluateRemote(flagKey: string, ctx: EvalContext): Promise<EvaluationResult> {
    const body = {
      application_id: this.config.applicationId,
      environment_id: this.config.environmentId,
      flag_key: flagKey,
      entity_id: ctx.entityId,
      entity_type: ctx.entityType ?? 'user',
      attributes: ctx.attributes ?? {},
    };

    return this.fetchOne<EvaluationResult>('/v1/evaluate', body);
  }

  private async fetch<T>(path: string, body: unknown): Promise<T> {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), this.config.httpTimeoutMs);

    try {
      const resp = await globalThis.fetch(this.config.baseUrl + path, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${this.config.apiKey}`,
        },
        body: JSON.stringify(body),
        signal: controller.signal,
      });

      if (!resp.ok) {
        const errBody = await resp.text();
        throw new Error(`Feature Flags API error ${resp.status}: ${errBody}`);
      }

      const json = (await resp.json()) as ApiResponse<T>;
      return json.data;
    } finally {
      clearTimeout(timeout);
    }
  }

  private async fetchOne<T>(path: string, body: unknown): Promise<T> {
    return this.fetch<T>(path, body);
  }

  private cacheKey(flagKey: string, entityId: string): string {
    return `${this.config.applicationId}:${this.config.environmentId}:${flagKey}:${entityId}`;
  }
}
