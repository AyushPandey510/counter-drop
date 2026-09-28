-- 0006: files a customer removes before sending are hidden from the job everywhere
-- (board, ticket, receipts) while the deletion worker erases them.
ALTER TABLE cd_job_files ADD COLUMN IF NOT EXISTS removed_at TIMESTAMPTZ;
-- Backfill: drafts' files already marked for deletion while the job was still uploading.
UPDATE cd_job_files f SET removed_at = COALESCE(f.delete_after, now())
FROM cd_jobs j
WHERE j.id = f.job_id AND f.removed_at IS NULL AND f.delete_status <> 'active'
  AND j.queued_at IS NOT NULL AND f.delete_after IS NOT NULL AND f.delete_after < j.queued_at;
