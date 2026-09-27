# 09 — Tokens, business day and wait engine

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 1 Backend core | R1a | M | 06 | FSD §9.2–9.3 (FS-9.7–9.13), BR-T1–T4, AT-03 |

## Goal

Tokens like `A-07` reset every business day per lane and are issued atomically; pickup codes are unique per shop per day; every shop has a live wait estimate and ready-by time.

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md §9.2 and §9.3, docs/BRD.md BR-T1–T4, and the current token code in api/internal/store/postgres.go (CreateJob) plus migrations 0004/0005.

Task: implement business day, tokens, pickup codes and the wait engine.

1. api/internal/domain/businessday.go: BusinessDay(now time.Time, hours WeeklyHours, closures []Date, tz *time.Location) (day civil date, opensAt, closesAt time.Time, isOpen bool). Business day = from today's opening to the next opening (FS-9.7). A job created before today's opening belongs to the previous business day. Handle closed days and closures. Also NextOpening(now) and ClosingToday(now). Overnight hours are out of scope (validation already forbids them).
2. Token issue (store, inside the same tx as the submit transition): INSERT INTO cd_token_counters_v2 (shop_id, lane_id, business_day, last_no) VALUES (…, 1) ON CONFLICT DO UPDATE SET last_no = cd_token_counters_v2.last_no + 1 RETURNING last_no. Format: lane letter + "-" + two digits, three digits after 99 (A-07, A-100). Pure formatter in domain with tests. Drop the old cd_token_counters table in a small migration 0008b only after the code no longer uses it.
3. Pickup code (for R1b, implement now): 4 random digits from crypto/rand; unique among open remote jobs of the shop on the business day (rely on the unique partial index; retry up to 10 times on conflict). Never reuse within a day even after collection (BR-T2) — the unique index over business_day covers this.
4. Wait engine in api/internal/domain/wait.go:
   - Inputs: jobsAhead []JobSize{Pages int}, medianMinutesPerJob (from stats, default 3 via CD_WAIT_DEFAULT_MINUTES), activeCounters int (min 1).
   - EstimateMinutes = sum over jobs ahead of (median + pages/20 extra minutes) / activeCounters (FS-9.10–9.11).
   - Range display: round down and up to nearest 5 → {Low, High}; 0 ahead → NoWait (FS-9.12).
   - ReadyBy(now, wait, ownJob) = now + wait + own job minutes (FS-9.13).
5. Stats job in store: RefreshShopStats(ctx, shopID) computes median claimed→ready minutes over the last 7 days (≥20 jobs else null) with percentile_cont(0.5), and active counters = distinct claimed_by in the last 30 minutes. Store results on cd_shops (avg_job_minutes_7d). Run it every 5 minutes per active shop (a simple ticker in main.go for now; step 12 moves it to scheduled tasks).
6. Expose Wait in GET /shops/{slug} (waitLowMinutes, waitHighMinutes, jobsAhead, isOpen, opensAt, closesAt) and position in GET /jobs/{id} (jobs ahead in the same lane with earlier queued_at).
7. Tests: business day edges (just before opening, just after closing, closed Sunday, closure date, IST vs UTC around midnight), token formatting and daily reset (AT-03: last token yesterday A-58 → first today A-01), concurrent token issue (200 submits across 2 lanes → no duplicates, contiguous per lane), wait range rounding, pickup code retry on collision (inject a deterministic random source).

Rules: all time math through the shop's timezone (Asia/Kolkata); domain functions take now as a parameter. Show the BusinessDay function signature and test cases first.
```

## Acceptance

- [ ] AT-03 passes: tokens restart at `A-01` each business day.
- [ ] 200 concurrent submits give unique, gap-free tokens per lane.
- [ ] Shop info returns open state, opening/closing time and wait range.

## Verify

```bash
cd api && go test ./internal/domain/... -run 'BusinessDay|Token|Wait' -v
make test-api
```

## Commit

`feat(queue): business-day tokens per lane, pickup codes and wait estimates`
