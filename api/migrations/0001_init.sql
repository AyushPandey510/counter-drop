CREATE TABLE IF NOT EXISTS cd_shops (
    id TEXT PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    intake_paused BOOLEAN NOT NULL DEFAULT FALSE,
    prices JSONB NOT NULL DEFAULT '{}'::jsonb,
    wait_minutes INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS cd_token_counters (
    shop_id TEXT NOT NULL REFERENCES cd_shops(id) ON DELETE CASCADE,
    prefix TEXT NOT NULL,
    last_no INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (shop_id, prefix)
);

CREATE TABLE IF NOT EXISTS cd_jobs (
    id TEXT PRIMARY KEY,
    shop_id TEXT NOT NULL REFERENCES cd_shops(id) ON DELETE CASCADE,
    token TEXT NOT NULL,
    secret TEXT NOT NULL,
    customer_name TEXT NOT NULL,
    settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    state TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    claimed_at TIMESTAMPTZ,
    ready_at TIMESTAMPTZ,
    collected_at TIMESTAMPTZ,
    UNIQUE (shop_id, token)
);

CREATE INDEX IF NOT EXISTS idx_cd_jobs_shop_state_created ON cd_jobs(shop_id, state, created_at);

CREATE TABLE IF NOT EXISTS cd_job_files (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES cd_jobs(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    mime TEXT NOT NULL,
    pages INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_cd_job_files_job_id ON cd_job_files(job_id);
