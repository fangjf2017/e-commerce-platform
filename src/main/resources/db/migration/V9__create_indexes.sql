CREATE INDEX idx_flags_lookup ON feature_flags (application_id, environment_id, key) WHERE deleted_at IS NULL;
CREATE INDEX idx_flags_env_status ON feature_flags (environment_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_rules_flag_priority ON rules (flag_id, priority) WHERE enabled = TRUE;
CREATE INDEX idx_segment_rules_segment ON segment_rules (segment_id);
CREATE INDEX idx_audit_app_time ON audit_events (application_id, occurred_at DESC);
CREATE INDEX idx_audit_resource ON audit_events (resource_type, resource_id, occurred_at DESC);
CREATE INDEX idx_flags_tags ON feature_flags USING GIN (tags);
