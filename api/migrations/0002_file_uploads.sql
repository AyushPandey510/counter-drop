ALTER TABLE cd_job_files
    ADD COLUMN IF NOT EXISTS object_key TEXT,
    ADD COLUMN IF NOT EXISTS upload_status TEXT NOT NULL DEFAULT 'pending';

UPDATE cd_job_files
SET object_key = 'cd/legacy/' || job_id || '/' || id
WHERE object_key IS NULL;

ALTER TABLE cd_job_files
    ALTER COLUMN object_key SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_cd_job_files_object_key ON cd_job_files(object_key);
CREATE INDEX IF NOT EXISTS idx_cd_job_files_upload_status ON cd_job_files(upload_status);
