export interface EvalContext {
  entityId: string;
  entityType?: string;
  attributes?: Record<string, unknown>;
}

export interface ConditionResult {
  attribute: string;
  operator: string;
  value: unknown;
  actualValue: unknown;
  matched: boolean;
}

export interface ExplanationStep {
  ruleId?: string;
  ruleName: string;
  ruleType: string;
  priority: number;
  outcome: 'matched' | 'skipped' | 'schedule_miss' | 'disabled' | 'flag_disabled';
  conditionResults?: ConditionResult[];
  segmentId?: string;
  segmentName?: string;
  rolloutBucket?: number;
  rolloutPct?: number;
}

export interface Explanation {
  flagKey: string;
  flagName: string;
  flagVersion: number;
  environmentSlug: string;
  evaluatedAt: string;
  entityId: string;
  entityType: string;
  steps: ExplanationStep[];
  matchedRuleId?: string;
  matchedRuleName?: string;
  matchedRuleType?: string;
  bucketValue?: number;
  defaultServed: boolean;
  reason: string;
}

export interface EvaluationResult<T = unknown> {
  flagKey: string;
  flagType: 'boolean' | 'string' | 'number' | 'json';
  value: T;
  variantKey?: string;
  enabled: boolean;
  explanation: Explanation;
  evaluatedAt: string;
  cacheHit: boolean;
  cacheLayer?: string;
}

export interface ClientConfig {
  baseUrl: string;
  apiKey: string;
  applicationId: string;
  environmentId: string;
  cacheTtlMs?: number;    // default 30000 (30s)
  httpTimeoutMs?: number; // default 5000 (5s)
}

export type FlagType = 'boolean' | 'string' | 'number' | 'json';
