-- 0007: the shop prints or downloads each file; downloads are shown to the customer;
-- the customer can ask for downloaded copies to be deleted and the shop confirms it.
-- Active jobs keep their files until they finish; uncollected jobs close after the shop's hold days.

ALTER TABLE cd_job_files
    ADD COLUMN IF NOT EXISTS printed_at TIMESTAMPTZ,            -- first opened for printing
    ADD COLUMN IF NOT EXISTS print_opens INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS downloaded_at TIMESTAMPTZ,         -- first downloaded to the shop's device
    ADD COLUMN IF NOT EXISTS downloaded_by TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS downloads INTEGER NOT NULL DEFAULT 0;

ALTER TABLE cd_jobs
    ADD COLUMN IF NOT EXISTS copies_delete_requested_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS copies_deleted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS copies_deleted_by TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_cd_jobs_copies_pending ON cd_jobs (shop_id)
    WHERE copies_deleted_at IS NULL AND state IN ('collected', 'cancelled');

ALTER TABLE cd_shops ADD COLUMN IF NOT EXISTS hold_days INTEGER NOT NULL DEFAULT 7;
ALTER TABLE cd_shops DROP CONSTRAINT IF EXISTS cd_shops_hold_days_check;
ALTER TABLE cd_shops ADD CONSTRAINT cd_shops_hold_days_check CHECK (hold_days BETWEEN 1 AND 7);

-- Jobs in line, printing or ready no longer lose their files at closing time.
UPDATE cd_job_files f SET delete_after = NULL
FROM cd_jobs j
WHERE j.id = f.job_id AND j.state IN ('queued', 'claimed', 'ready')
  AND f.deleted_at IS NULL AND f.removed_at IS NULL AND f.delete_status = 'active';
