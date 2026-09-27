# 14 — Deletion worker and retention

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 1 Backend core | R1a | M | 12 | FSD §12 (all), BR-D1–D8, CT-1–CT-4, NFR-15, AT-11, AT-13 |

## Goal

Every file is deleted on schedule, proven with a receipt, with a daily safety check and the R2 lifecycle rule as a backstop.

## Prompt

```text
First read docs/build-plan/00-common-context.md and docs/FSD.md §12 in full (triggers table, worker algorithm, backstops FS-12.1–12.5, long-term records), docs/BRD.md BR-D1–D8 and CT-1–CT-4. Read api/internal/store/postgres.go scheduleFileDeletionTx and the storage ObjectStore from step 02.

Task: implement deletion and retention.

1. Fix the undo window: delete_after on collected = collected_at + CD_UNDO_WINDOW (default 10m). Remove the hard-coded 15 minutes. Cancel/expired walk-in → now. Rejected/expired remote → now + CD_REJECT_GRACE (15m). Every upload also gets delete_after = min(shop closing today, uploaded_at + 24h) at completion time; later triggers only move it earlier (except undo, which restores the upload-based cap).
2. api/internal/tasks/deletion.go worker, every CD_DELETION_INTERVAL (30 s):
   - SELECT … FROM cd_job_files WHERE delete_status IN ('active','pending','failed') AND delete_after <= now() AND (next_attempt_at IS NULL OR next_attempt_at <= now()) ORDER BY delete_after LIMIT 200 FOR UPDATE SKIP LOCKED.
   - ObjectStore.Delete(key) (missing object = success).
   - Success: delete_status deleted, deleted_at now, object_key NULL, filename replaced by "File N"; audit file.deleted.
   - Failure: attempts+1, status failed, next_attempt_at = now + backoff (1m, 5m, 15m); after 3 failures log error and increment a metric (alert wired in step 24).
   - When all files of a job are deleted: clear customer_name, set files_deleted_at, emit job.files_deleted via the outbox (drives the receipt).
   (Add next_attempt_at column in a small migration.)
3. Daily safety check (scheduled task deletion_audit at 03:00 IST): list R2 objects under cd/ older than 25 h (ListObjectsV2 with pagination); for each, find the DB row; delete; log critical with counts only. Add ListOlderThan to ObjectStore (fake + S3).
4. Receipt data on GET /jobs/{id}: filesDeletedAt, fileCount, per-file deletedAt, no names after deletion (FS-12.3).
5. Metrics hooks: deletion_lag_seconds histogram (deleted_at − delete_after), failed count, oldest undeleted file age.
6. Admin read model (used in step 23): store query DeletionHealth(ctx) → due now, deleted last 24 h, failed with last error, oldest undeleted age.
7. Tests: collected → deleted after 10 min (FixedClock) — AT-11; undo before the window cancels deletion — AT-12; closing-time deletion — AT-13; storage failure retries with backoff then succeeds; job-level clearing of name and files_deleted_at; safety check deletes an orphan object in FakeStore.

Rules: deletion must be idempotent and safe to run on several instances at once. Never log object keys together with job IDs and customer data (IDs only is fine). Show the plan first.
```

## Acceptance

- [ ] AT-11, AT-12, AT-13 pass.
- [ ] Deletion lag p99 under 5 minutes in a local soak (1,000 files).
- [ ] Orphan objects are found and deleted by the safety check.

## Verify

```bash
make test-api
```

## Commit

`feat(retention): deletion worker, receipts, safety audit and 10-minute undo window`
