CREATE TABLE IF NOT EXISTS conditions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id    UUID NOT NULL REFERENCES rules(id) ON DELETE CASCADE,
    attribute  TEXT NOT NULL,
    operator   TEXT NOT NULL,
    value      JSONB NOT NULL,
    negate     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
