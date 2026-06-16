CREATE TABLE IF NOT EXISTS variant_allocations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id       UUID NOT NULL REFERENCES rules(id) ON DELETE CASCADE,
    variant_id    UUID NOT NULL REFERENCES variants(id) ON DELETE CASCADE,
    rollout_from  INT NOT NULL CHECK (rollout_from >= 0 AND rollout_from < 10000),
    rollout_to    INT NOT NULL CHECK (rollout_to > 0 AND rollout_to <= 10000),
    CHECK (rollout_from < rollout_to),
    UNIQUE (rule_id, variant_id)
);
