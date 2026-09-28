-- R1a walk-in MVP: shop profile and prices, lanes, staff PIN login, job v2 columns,
-- daily tokens per lane, job event log. Safe to run on a database created by 0001–0003.

-- Shops -----------------------------------------------------------------------
ALTER TABLE cd_shops
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'live',
    ADD COLUMN IF NOT EXISTS online_state TEXT NOT NULL DEFAULT 'online',
    ADD COLUMN IF NOT EXISTS pause_message TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS address TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS timezone TEXT NOT NULL DEFAULT 'Asia/Kolkata',
    ADD COLUMN IF NOT EXISTS opens_at TEXT NOT NULL DEFAULT '09:00',
    ADD COLUMN IF NOT EXISTS closes_at TEXT NOT NULL DEFAULT '21:30',
    ADD COLUMN IF NOT EXISTS price_list JSONB NOT NULL DEFAULT
        '{"bwOnePaise":200,"bwBothPaise":300,"colourOnePaise":1000,"colourBothPaise":1800,"minChargePaise":0,"version":1}'::jsonb;

UPDATE cd_shops SET online_state = 'paused' WHERE intake_paused;

DO $$ BEGIN
    ALTER TABLE cd_shops ADD CONSTRAINT cd_shops_online_state_chk CHECK (online_state IN ('online','paused','offline'));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- Lanes -----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS cd_lanes (
    id TEXT PRIMARY KEY,
    shop_id TEXT NOT NULL REFERENCES cd_shops(id) ON DELETE CASCADE,
    letter TEXT NOT NULL,
    name TEXT NOT NULL,
    rule TEXT NOT NULL DEFAULT 'any' CHECK (rule IN ('bw','colour','any')),
    sort INTEGER NOT NULL DEFAULT 0,
    UNIQUE (shop_id, letter)
);

INSERT INTO cd_lanes (id, shop_id, letter, name, rule, sort)
SELECT 'lane_' || id || '_a', id, 'A', 'B/W', 'bw', 0 FROM cd_shops
ON CONFLICT DO NOTHING;
INSERT INTO cd_lanes (id, shop_id, letter, name, rule, sort)
SELECT 'lane_' || id || '_b', id, 'B', 'Colour', 'colour', 1 FROM cd_shops
ON CONFLICT DO NOTHING;

-- Staff and sessions (R1a: name + 4-digit PIN; phone OTP arrives with step 07) ----
CREATE TABLE IF NOT EXISTS cd_staff (
    id TEXT PRIMARY KEY,
    shop_id TEXT NOT NULL REFERENCES cd_shops(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'staff' CHECK (role IN ('owner','staff')),
    pin_hash TEXT NOT NULL,
    failed_attempts INTEGER NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (shop_id, name)
);

CREATE TABLE IF NOT EXISTS cd_sessions (
    token_hash TEXT PRIMARY KEY,
    staff_id TEXT NOT NULL REFERENCES cd_staff(id) ON DELETE CASCADE,
    shop_id TEXT NOT NULL REFERENCES cd_shops(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_cd_sessions_staff ON cd_sessions(staff_id);

-- Jobs ------------------------------------------------------------------------
ALTER TABLE cd_jobs
    ADD COLUMN IF NOT EXISTS channel TEXT NOT NULL DEFAULT 'walkin',
    ADD COLUMN IF NOT EXISTS lane_id TEXT REFERENCES cd_lanes(id),
    ADD COLUMN IF NOT EXISTS secret_hash TEXT,
    ADD COLUMN IF NOT EXISTS queued_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS cancelled_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS cancel_reason TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS claimed_by TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS price_total_paise BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS pages_total INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS pages_to_confirm BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS ready_by TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS paid_method TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS business_day DATE,
    ADD COLUMN IF NOT EXISTS files_deleted_at TIMESTAMPTZ;

ALTER TABLE cd_jobs ALTER COLUMN secret DROP NOT NULL;
ALTER TABLE cd_jobs ALTER COLUMN token DROP NOT NULL;
ALTER TABLE cd_jobs ALTER COLUMN customer_name DROP NOT NULL;

-- Legacy rows: 'new' → 'queued', plain secret → SHA-256 hash.
UPDATE cd_jobs SET state = 'queued' WHERE state = 'new';
UPDATE cd_jobs SET queued_at = created_at WHERE queued_at IS NULL AND state <> 'uploading';
UPDATE cd_jobs SET secret_hash = encode(sha256(secret::bytea), 'hex') WHERE secret_hash IS NULL AND secret IS NOT NULL;
UPDATE cd_jobs SET business_day = (created_at AT TIME ZONE 'Asia/Kolkata')::date WHERE business_day IS NULL;
UPDATE cd_jobs SET secret = NULL;


-- Tokens restart every business day, so uniqueness is per day.
ALTER TABLE cd_jobs DROP CONSTRAINT IF EXISTS cd_jobs_shop_id_token_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_cd_jobs_shop_day_token
    ON cd_jobs(shop_id, business_day, token) WHERE token IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_cd_jobs_shop_lane_queue
    ON cd_jobs(shop_id, lane_id, state, queued_at);

DO $$ BEGIN
    ALTER TABLE cd_jobs ADD CONSTRAINT cd_jobs_state_chk
        CHECK (state IN ('uploading','queued','claimed','ready','collected','cancelled'));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- Files -----------------------------------------------------------------------
ALTER TABLE cd_job_files
    ADD COLUMN IF NOT EXISTS settings JSONB NOT NULL DEFAULT '{"copies":1}'::jsonb,
    ADD COLUMN IF NOT EXISTS pages_status TEXT NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS delete_attempts INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS next_attempt_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_delete_error TEXT NOT NULL DEFAULT '';

ALTER TABLE cd_job_files ALTER COLUMN object_key DROP NOT NULL;

-- Daily token counters per lane --------------------------------------------------
CREATE TABLE IF NOT EXISTS cd_token_counters_v2 (
    shop_id TEXT NOT NULL REFERENCES cd_shops(id) ON DELETE CASCADE,
    lane_letter TEXT NOT NULL,
    business_day DATE NOT NULL,
    last_no INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (shop_id, lane_letter, business_day)
);

-- Job event log (timeline for ticket, board and audit) ------------------------------
CREATE TABLE IF NOT EXISTS cd_job_events (
    id BIGSERIAL PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES cd_jobs(id) ON DELETE CASCADE,
    shop_id TEXT NOT NULL,
    from_state TEXT NOT NULL,
    to_state TEXT NOT NULL,
    action TEXT NOT NULL,
    actor_type TEXT NOT NULL,
    actor_name TEXT NOT NULL DEFAULT '',
    reason TEXT NOT NULL DEFAULT '',
    at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_cd_job_events_job ON cd_job_events(job_id, id);
CREATE INDEX IF NOT EXISTS idx_cd_job_events_shop_at ON cd_job_events(shop_id, at);
