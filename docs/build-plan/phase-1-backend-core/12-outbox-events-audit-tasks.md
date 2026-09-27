# 12 — Outbox, job events, audit log and scheduled tasks

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 1 Backend core | R1a | M | 06 (and 11 for effects wiring) | FSD FS-8.3–8.4, FS-2.3, §11 FS-11.1, §15 FS-15.3 |

## Goal

Side effects happen reliably: realtime and push events are written to an outbox in the same transaction as the state change and dispatched after commit; timers (abandon, closing, deletion, stats) survive restarts; every action is audited.

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md FS-8.3, FS-8.4, FS-2.3, §11 rules and §15 rules, and migration 0008. Read api/internal/app (step 11) and the EffectExecutor interface.

Task: implement the outbox, audit logging and a durable scheduled-task runner.

1. Outbox:
   - In the same transaction as each transition, insert cd_outbox rows for the Notify effects: topic (job:{id}, shop:{id}, push:user:{id}), event type (job.updated, queue.job_added, queue.job_changed, queue.job_removed, shop.state, job.files_deleted, job.ask) and payload (the JobCard or ticket view — no file names, no phone).
   - Dispatcher goroutine (api/internal/tasks/outbox.go): LISTEN cd_outbox_new (the insert fires pg_notify via trigger) and also polls every 1 s as a fallback; SELECT … FOR UPDATE SKIP LOCKED LIMIT 100 WHERE sent_at IS NULL ORDER BY id; hand each row to a Publisher interface (realtime hub in step 13; push in step 34); mark sent_at; on error increment attempts with backoff and alert after 10.
   - Per-topic sequence numbers (seq) assigned at insert time from a per-topic counter so clients can detect gaps (FS-15.1).
2. Audit: api/internal/audit with Log(ctx, tx, Entry{ActorType, ActorID, ShopID, JobID, Action, Detail map, IP, DeviceID}). Use it for every transition (via effects), file open, settings change, auth event and admin action. Detail must never include file names, phone numbers or names — add a unit test that rejects entries whose Detail contains keys in a denylist (phone, name, filename, secret, pin, otp, token).
3. Scheduled tasks (api/internal/tasks/scheduler.go):
   - cd_scheduled_tasks(id, kind, ref_id, run_at, attempts, done_at, last_error). Kinds now: job_abandon (T18), shop_auto_offline (EX-S05), shop_closing_files (BR-D2 / T19), shop_stats_refresh (step 09), and placeholders for payment_timeout, accept_timeout, reject_grace (R1b).
   - Runner polls every 10 s: claims due tasks with SKIP LOCKED, executes a handler by kind, marks done or reschedules with backoff. Idempotent handlers (safe to run twice).
   - ScheduleTask / CancelTask effects write and cancel rows in the same tx as the transition.
   - shop_closing_files: at each shop's closing time, set delete_after = now for files of non-collected jobs (FS-12 "at closing"), and create tomorrow's task. Created for every live shop by a daily planner task at 00:05 IST.
4. Move the stats ticker from step 09 into shop_stats_refresh tasks (every 5 min per shop with activity in the last hour).
5. Graceful shutdown: runners stop claiming on SIGTERM and finish in-flight work within 10 s.
6. Tests (integration): transition + outbox row committed together (and neither on rollback); dispatcher delivers to a fake publisher in order with increasing seq; a task survives a restart (insert, stop runner, start new runner, executes once); abandon task expires a draft after 60 min using FixedClock; audit denylist test.

Rules: no Redis; Postgres LISTEN/NOTIFY + polling only. Show the table usage and the Publisher interface first.
```

## Acceptance

- [ ] Killing the API mid-flight loses no events or timers.
- [ ] Outbox rows exist only for committed transitions.
- [ ] Audit entries contain no personal data (denylist test).

## Verify

```bash
make test-api
psql "$CD_DATABASE_URL" -c 'select kind, count(*) from cd_scheduled_tasks group by 1'
```

## Commit

`feat(tasks): transactional outbox, audit log, durable scheduled task runner`
