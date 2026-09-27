# 05 — Schema v2 migrations (0004–0008)

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 1 Backend core | R1a | M | 04 | FSD §13, §8, §12, BR-D |

## Goal

The database has every R1a table and column from FSD §13, migrated safely from the current schema, with the demo shop reseeded to the new shape.

## Why now

The state machine, auth, pricing, tokens and deletion steps all read and write these columns.

## Scope

**In:** migrations 0004–0008, PostGIS extension, updated seed, migration runner improvements (checksum, one transaction per file), a schema test.
**Out:** payments tables (0009, step 30), ratings/disputes (0010, step 35).

## Migrations

| File | Contents |
| --- | --- |
| `0004_shop_profile_geo_hours.sql` | `CREATE EXTENSION IF NOT EXISTS postgis`; shop columns: status, online_state, pause_message, location geography(Point,4326), address, landmark, station, cluster_id, timezone default 'Asia/Kolkata', plan, remote_enabled, remote_max_files, remote_max_pages, payout_account_id, payout_status, gstin, rating_avg, rating_count, acceptance_rate_7d, avg_job_minutes_7d, min_charge_paise; migrate intake_paused → online_state; `cd_shop_hours`, `cd_shop_closures`, `cd_clusters` |
| `0005_lanes_prices_tokens.sql` | `cd_lanes`; `cd_price_lists` (version, rates jsonb, add_ons jsonb, min_charge_paise); migrate `cd_shops.prices` into a price list; new `cd_token_counters_v2(shop_id, lane_id, business_day date, last_no)`; keep old table until step 09 |
| `0006_users_members_devices_sessions.sql` | `cd_users`, `cd_shop_members`, `cd_devices`, `cd_sessions` (token_hash, user_id, device_id, kind, expires_at, revoked_at), `cd_otp_requests` (phone_hash, code_hash, expires_at, attempts, ip) |
| `0007_job_states_v2.sql` | jobs: lane_id, channel, customer_user_id, pickup_code, secret_hash, price_total_paise, fee_paise, price_version, pages_total, pages_to_confirm, ready_by, queued_at, claimed_by, cancelled_at, cancel_reason, reject_reason, business_day, files_deleted_at; data migration new→queued, secret→secret_hash; files: sha256, pages_status, settings jsonb, delete_attempts; indexes |
| `0008_events_outbox_tasks_audit.sql` | `cd_job_events`, `cd_outbox`, `cd_scheduled_tasks`, `cd_audit_log` (append-only via REVOKE UPDATE/DELETE from the app role where possible) |

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md §13 (data model), §8 (states) and §12 (retention), then api/migrations/*.sql and the ApplyMigrations, SeedDemoShop and scan functions in api/internal/store/postgres.go.

Task: write migrations 0004–0008 exactly as listed in docs/build-plan/phase-1-backend-core/05-schema-v2-migrations.md and update the store so the app still runs.

Details:
1. Every migration is idempotent where practical (IF NOT EXISTS), runs in one transaction, and never drops data. Use CHECK constraints for enums (state, channel, online_state, status, role, kind, delete_status, pages_status) instead of Postgres ENUM types.
2. Money columns are BIGINT and end in _paise. Timestamps are timestamptz. IDs are TEXT.
3. 0007 data migration: UPDATE cd_jobs SET state='queued' WHERE state='new'; set queued_at = created_at for queued rows; set secret_hash = encode(sha256(secret::bytea),'hex'); then drop NOT NULL on secret and stop writing it (column is removed in a later cleanup migration, not now). Set channel='walkin' for existing rows. Set business_day from created_at at Asia/Kolkata.
4. Indexes: jobs (shop_id, lane_id, state, queued_at); jobs (shop_id, business_day, token) unique where token is not null; jobs (shop_id, business_day, pickup_code) unique where pickup_code is not null; files (delete_status, delete_after); scheduled_tasks (run_at) where done_at is null; outbox (id) where sent_at is null; shops GIST(location); audit_log (shop_id, at), (job_id, at).
5. Migration runner: store a sha256 checksum per applied file in cd_schema_migrations and fail fast if an applied file's checksum changed. Apply each file in its own transaction.
6. Seed: SeedDemoShop creates shop demo-print (status live, online), a location near Dadar station (19.0178, 72.8478), Mon–Sat 09:00–21:30 hours, lane A "B/W" and lane B "Colour", a price list (B/W A4 one side 200 paise, both sides 300 per sheet, colour one side 1000, both 1800, Legal B/W 300, A3 B/W 1000 colour 2000, lamination add-on 2000, spiral 3000, stapling 0, min charge 500), and an owner user with phone from CD_DEMO_OWNER_PHONE if set.
7. Update the Go scan code so existing endpoints still work: JobState "queued" must be returned where "new" was (and "new" accepted in input as alias); job secret checks use secret_hash with constant-time compare.
8. Tests: an integration test that applies all migrations on an empty test database, then again (no-op), then checks key columns and indexes exist via information_schema/pg_indexes; a test that migrating a DB with a 'new' job and a plain secret ends with 'queued' and a matching secret_hash.

Rules: don't add payment or rating tables. Don't change HTTP responses except state name new→queued (keep a JSON alias if CD_LEGACY_ENVELOPE=true: include "legacyState":"new"). Show the SQL for 0007 before writing the rest.
```

## Acceptance

- [ ] Fresh DB: all migrations apply; second run is a no-op.
- [ ] Existing DB with old rows migrates without data loss.
- [ ] Demo shop has location, hours, two lanes and a price list.
- [ ] Changing an applied migration file makes startup fail with a clear message.

## Verify

```bash
make db-reset CONFIRM=1 && make api     # startup logs: migrations applied
psql "$CD_DATABASE_URL" -c '\d cd_jobs'
make test-api
```

## Commit

`feat(db): schema v2 — shop profile, geo, hours, lanes, prices, auth, job v2, outbox, audit`
