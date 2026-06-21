CREATE TABLE rules (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    flag_id        UUID NOT NULL REFERENCES feature_flags(id) ON DELETE CASCADE,
    type           TEXT NOT NULL CHECK (type IN ('targeting', 'segment', 'rollout', 'schedule')),
    priority       INT NOT NULL DEFAULT 0,
    name           TEXT NOT NULL DEFAULT '',
    description    TEXT NOT NULL DEFAULT '',
    segment_id     UUID REFERENCES segments(id) ON DELETE SET NULL,
    rollout_pct    INT CHECK (rollout_pct IS NULL OR (rollout_pct BETWEEN 0 AND 100)),
    rollout_salt   UUID NOT NULL DEFAULT gen_random_uuid(),
    schedule_start TIMESTAMPTZ,
    schedule_end   TIMESTAMPTZ,
    enabled        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
