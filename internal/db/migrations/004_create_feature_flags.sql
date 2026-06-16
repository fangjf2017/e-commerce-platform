CREATE TABLE IF NOT EXISTS feature_flags (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    key            TEXT NOT NULL,
    name           TEXT NOT NULL,
    description    TEXT NOT NULL DEFAULT '',
    type           TEXT NOT NULL CHECK (type IN ('boolean', 'string', 'number', 'json')),
    status         TEXT NOT NULL DEFAULT 'inactive'
                       CHECK (status IN ('active', 'inactive', 'archived')),
    default_value  JSONB NOT NULL,
    tags           TEXT[] NOT NULL DEFAULT '{}',
    version        BIGINT NOT NULL DEFAULT 1,
    deleted_at     TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (application_id, environment_id, key)
);

CREATE TABLE IF NOT EXISTS variants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    flag_id     UUID NOT NULL REFERENCES feature_flags(id) ON DELETE CASCADE,
    key         TEXT NOT NULL,
    value       JSONB NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    UNIQUE (flag_id, key)
);
