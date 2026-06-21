CREATE TABLE IF NOT EXISTS audit_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id  UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    environment_id  UUID REFERENCES environments(id) ON DELETE SET NULL,
    action          TEXT NOT NULL,
    resource_type   TEXT NOT NULL,
    resource_id     UUID NOT NULL,
    resource_key    TEXT NOT NULL,
    actor_id        TEXT NOT NULL,
    actor_email     TEXT NOT NULL DEFAULT '',
    actor_type      TEXT NOT NULL DEFAULT 'user',
    before_state    JSONB,
    after_state     JSONB,
    metadata        JSONB NOT NULL DEFAULT '{}',
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
