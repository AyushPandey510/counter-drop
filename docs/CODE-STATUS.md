# Code status vs the plan

Snapshot: 27 Sep 2026. Reviewed by reading the code against `docs/FSD.md` and `docs/build-plan/`. The code could not be compiled or tested during this review (the Go module proxy was not reachable from the review environment), so run `cd api && go test ./...` to confirm.

**Summary:** a working backend slice exists (create job → presigned upload URL → submit → queue → claim/ready/collected/cancel/release, on Postgres or in memory). It is roughly the starting point of build-plan steps 04–10. Nothing in `web/`, `agent/`, `pkg/cdclient/`, `internal/realtime` or `internal/tasks` yet.

## Present, by build-plan step

| Step | What the plan needs | What exists | Status |
| --- | --- | --- | --- |
| 01 Dev env & CI | Makefile, lint, CI, test DB, `.gitignore` | `go.work`, docker-compose (Postgres + MinIO), `.env.example`; `.gitignore`, `.gitattributes`, `.editorconfig` added in this prep | Partial |
| 02 Storage (R2) | Presign PUT/GET, head, delete, key scheme, smoke test | Presign PUT only (`storage/s3.go`); key `cd/{shop}/{job}/{file}` | Partial |
| 03 OpenAPI | Full v1 contract | 7 operations sketched, no schemas | Early |
| 04 API foundation | Context, error mapping, envelope v2, middleware | Single `router.go`, envelope `{success,data,error}`, request logger; no context in store, no recover/CORS/request ID | Early |
| 05 Schema v2 | Migrations 0004–0008 | 0001–0003 (shops, token counters, jobs, files, upload + deletion columns) | Early |
| 06 State machine v2 | 10 states, T1–T19, effects | 5 states (`new`, `claimed`, `ready`, `collected`, `cancelled`), actions claim/ready/collected/cancel/release, 2 tests | Early |
| 07 Auth | OTP, PIN, sessions, shop scope | None | Not started |
| 08 Pricing | Quote engine | None (demo shop has 2 prices) | Not started |
| 09 Tokens & wait | Daily tokens per lane, wait engine | Token counter per shop (`A-NN`), no daily reset, no lanes | Early |
| 10 Walk-in job API | Create, upload confirm, page count, quote, submit, ticket, cancel | Create, submit, get, all without verification or page count | Early |
| 11 Shop API | Queue, claim-next, actions, lookup, settings | Queue + actions (no auth) | Early |
| 12 Outbox/audit/tasks | Outbox, audit log, scheduled tasks | None | Not started |
| 13 Realtime | WebSocket hub | Empty package | Not started |
| 14 Deletion worker | Worker, receipts, safety audit | `delete_after` is set on collected/cancel; no worker deletes anything | Early |
| 15–22 Web PWA | React PWA | `web/src/*` placeholders | Not started |
| 23+ | Admin, security, deploy, R1b | Dockerfile only | Not started |

## Issues found (fix in the step noted)

| # | Issue | Why it matters | Where | Fix in |
| --- | --- | --- | --- | --- |
| 1 | **Docker image had no migrations** — the API applies `./migrations` at startup but the runtime image only contained the binary | Container would fail on start | `api/Dockerfile` | **Fixed in this prep** |
| 2 | Shop routes have no authentication and take `?shop=` from the query | Anyone can view any queue and change any job | `httpapi/router.go` | 07 |
| 3 | Token is issued at **create**, and jobs enter the queue (`new`) before files are uploaded | Abandoned drafts burn tokens; staff see jobs with no files | `store/*.go CreateJob` | 06, 09, 10 |
| 4 | Submit marks every file `uploaded` without checking storage | A job can reach the counter with missing or wrong files | `SubmitJob` | 10 |
| 5 | Ticket secret stored in plain text, compared with `!=`, passed in the URL query | Leaks via logs/referrers; timing attack surface | store, router | 04, 05 |
| 6 | Token counter never resets | Tokens grow forever (`A-4821`) instead of `A-01` each day | `cd_token_counters` | 09 |
| 7 | Queue returns every job ever, with one files query per job | Slows down as history grows (N+1) | `ListShopQueue` | 11 |
| 8 | Undo window is 15 min | Docs say 10 min | `collectedDeleteGrace` | 14 |
| 9 | `customerName` is required | Docs make it optional (walk-in never needs identity) | `router.go` | 10 |
| 10 | No worker deletes files | Privacy promise (CT-1) not met yet | `internal/tasks` | 14 |
| 11 | Migrations run relative to the working directory and have no checksum | Breaks when started from another folder; edited migrations go unnoticed | `ApplyMigrations`, `main.go` | 05 |
| 12 | Store methods create their own `context.Background()` | Request cancellation and timeouts don't propagate | store | 04 |
| 13 | All `go.mod` requirements marked `// indirect` | Cosmetic; `go mod tidy` fixes it | `api/go.mod` | 01 |
| 14 | MinIO image uses `:latest` and wasn't pullable | Unreliable local setup | `deploy/docker-compose.yml` | 01–02 (switch to R2) |
| 15 | Go version 1.27.1 everywhere | Make sure your local Go and CI images match | `go.work`, `go.mod`, Dockerfile | 01 |

## Good to keep

- Clean module layout (`domain`, `store`, `httpapi`, `storage`) that matches the FSD.
- State changes run in transactions with `SELECT … FOR UPDATE`, so claims are already atomic.
- Object keys contain only IDs (no file names).
- Deletion columns (`delete_status`, `delete_after`, `deleted_at`) already in the schema.
- In-memory store for quick demos without Postgres.

## Housekeeping

- Not yet a git repository — see "Getting started" in the root `README.md`.
- `.gitkeep` files can be deleted from folders that now contain code (`api/cmd/api`, `api/internal/{config,domain,httpapi,storage,store}`, `api/migrations`).
