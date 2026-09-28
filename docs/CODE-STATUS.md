# Code status vs the plan

Snapshot: 28 Sep 2026, after the MVP build. Verified by `go vet`, `go test ./...` against Postgres 17, `npm run build` (strict TypeScript), and a browser walkthrough (phone viewport for the customer, desktop for the board) with no console errors.

**Summary:** the R1a walk-in pilot slice is built end to end. A customer scans the QR, uploads, sees the price, gets a token and a live ticket; staff sign in with a PIN, work the live board and mark collected; the deletion worker removes files 10 minutes after pickup and the customer's phone shows the deletion receipt.

## Present, by build-plan step

| Step | Status | Notes |
| --- | --- | --- |
| 01 Dev env | Partial | compose (Postgres; MinIO optional profile), `.env.example`, git files. No Makefile or CI yet |
| 02 Storage | Done (MVP) | `ObjectStore`: R2/S3 presign PUT (length-signed) / GET / head / delete, plus **local disk** with HMAC-signed links (default) |
| 03 OpenAPI | Stale | `contracts/openapi.yaml` predates the MVP; `api/README.md` has the current routes |
| 04 API foundation | Done | recover, request ID, path-only logging, CORS, `{data}` / `{error}` envelope, strict JSON decode, context everywhere |
| 05 Schema v2 | Done (MVP) | `0004_mvp_walkin.sql`: lanes, staff, sessions, daily token counters, job events, file settings and deletion retries; upgrades old rows |
| 06 State machine | Done (MVP) | uploading → queued → claimed → ready → collected, cancelled; release, undo, abandon; effects; full matrix test |
| 07 Auth | Partial | Staff PIN (pbkdf2, lockout), hashed sessions, shop scope, owner role; **one-time setup links** (people choose their own PIN), change PIN, owner staff management, weak-PIN rule. **No phone OTP / self-signup yet** (shops created with `cdadmin`) |
| 08 Pricing | Done | Paise, per side / per sheet, colour, copies, page ranges, minimum charge, rupee rounding, price version |
| 09 Tokens & wait | Done | Daily tokens per lane (A B/W, B colour), IST business day, wait range and ready-by |
| 10 Walk-in job API | Done | Create, add/remove files, upload confirm with size/type check, page count, quote, submit, ticket, cancel |
| 11 Shop API | Done (MVP) | Queue, claim-next (`SKIP LOCKED`), actions, lookup, state, settings, signed file links |
| 12 Outbox/audit | Partial | `cd_job_events` audit trail; no outbox |
| 13 Realtime | Done (MVP) | **SSE** hub (not WebSocket), polling fallback |
| 14 Deletion worker | Done | Retries with backoff, clears names, drops file names, abandons drafts, deletion-health endpoint |
| 15–22 Web PWA | Done (MVP) | Home, drop page, ticket, shop login, board, settings, QR poster; EN/HI/MR customer screens; installable PWA |
| 23+ | Not started | Admin console, R1b (Print nearby, UPI prepay), print agent, deploy pipeline |

## Decisions taken while building (update BRD/FSD when convenient)

| Decision | Why |
| --- | --- |
| Server-Sent Events instead of WebSocket | One-way updates are all R1a needs; no extra dependency; works through proxies. Single API instance for now (hub is in memory) |
| Page count on the phone with pdf.js; server falls back to "pages confirmed at counter" | No PDF parsing on the server in R1a |
| Local-disk storage as the default | Pilot runs on one server without an R2 account; R2 switches on with env vars |
| Postgres required; in-memory store removed | One code path to test |
| Business day = local midnight (IST) | Tokens restart at A-01 each day |
| PIN-only staff sign-in, shops created with `cdadmin` | OTP and self-signup are the next auth step |
| One-time setup links instead of handing out PINs | Whoever creates an account never knows its PIN; works without SMS; becomes the "invite staff" step of the OTP signup later |
| Audio/voice features deferred | As agreed |

## Next

1. Deploy a pilot: one small VM (API + built web via `CD_WEB_DIR`) + managed Postgres, HTTPS, `CD_STORAGE_SIGNING_KEY` set; or R2.
2. Phone OTP for owners and shop self-signup (step 07 — setup links, change PIN and staff management are done) + admin console (step 23).
3. Refresh `contracts/openapi.yaml`; add Makefile and CI (step 01).
4. R1b: Print nearby and UPI prepay.
