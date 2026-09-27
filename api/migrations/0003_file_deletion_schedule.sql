ALTER TABLE cd_job_files
    ADD COLUMN IF NOT EXISTS delete_status TEXT NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS delete_after TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_cd_job_files_delete_status_after
    ON cd_job_files(delete_status, delete_after);
