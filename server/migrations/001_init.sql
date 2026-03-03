CREATE TABLE IF NOT EXISTS devices (
    id           UUID PRIMARY KEY,
    hostname     TEXT NOT NULL,
    os           TEXT NOT NULL,
    ip           TEXT NOT NULL,
    agent_token  TEXT NOT NULL,
    enrolled_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS commands (
    id          UUID PRIMARY KEY,
    device_id   UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    type        TEXT NOT NULL,
    payload     JSONB,
    status      TEXT NOT NULL DEFAULT 'pending',
    result      JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
