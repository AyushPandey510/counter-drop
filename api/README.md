# Counter Drop API

Go 1.27 service (`net/http` + pgx on Postgres) for the R1a walk-in flow: QR drop page, tickets, staff PIN sign-in, live queue, pricing, daily tokens per lane, and the deletion worker.

## Run

```bash
docker compose -f ../deploy/docker-compose.yml up -d postgres
CD_DATABASE_URL='postgres://counter_drop:counter_drop@localhost:55433/counter_drop?sslmode=disable' go run ./cmd/api
```

Migrations in `migrations/` run at start-up. In dev the `demo-print` shop is seeded (Owner PIN `1234`, Kavita PIN `1111`). Files go to local disk (`.data/files`) unless all `CD_STORAGE_*` R2 values are set. All settings: `../deploy/.env.example`.

Serve the PWA from the same process (one port, no CORS):

```bash
(cd ../web && npm install && npm run build)
CD_WEB_DIR=../web/dist CD_DATABASE_URL=... go run ./cmd/api     # open http://localhost:8080
```

## Onboarding shops (`cdadmin`)

Nobody hands out PINs. Creating or resetting an account prints a **one-time setup link** (valid 48 h) and a ready-to-send message; the person opens it and chooses their own PIN. Set `CD_PUBLIC_WEB_URL` (e.g. `https://counterdrop.in`) so links point at the right address.

```bash
go run ./cmd/cdadmin create-shop  -slug imran-xerox -name "Imran Xerox" -address "Station Rd, Pune" -owner Imran
go run ./cmd/cdadmin add-staff    -shop imran-xerox -name Sana [-role staff|owner]
go run ./cmd/cdadmin reset-pin    -shop imran-xerox -name Sana     # old PIN stops working, signed out everywhere
go run ./cmd/cdadmin remove-staff -shop imran-xerox -name Sana
go run ./cmd/cdadmin list-staff   -shop imran-xerox
```

After the owner has set their PIN they manage staff themselves in **Settings → Staff** (add, reset PIN, remove), and everyone can change their own PIN under **My account**.

Rules: PINs are 4 digits and obvious ones (1234, 1111, 1212, 2580…) are refused; a link works once, only the newest link for a person works, and its token lives in the URL fragment so it never reaches server logs; a reset clears the old PIN and signs the person out; nobody can reset or remove themselves; a shop always keeps at least one owner.

## Layout

| Package | What |
| --- | --- |
| `internal/domain` | Pure rules: job state machine and effects, pricing and page ranges, tokens, wait estimate, shop hours |
| `internal/store` | Postgres: shops, staff and sessions, jobs and files, queue, retention |
| `internal/httpapi` | Routes, auth middleware, `{data}` / `{error:{code,message}}` envelope, handlers, end-to-end tests |
| `internal/realtime` | Server-Sent Events hub (job and shop topics) |
| `internal/tasks` | Deletion worker and draft abandonment |
| `internal/storage` | `ObjectStore`: R2/S3 presign, or local disk with HMAC-signed links |
| `cmd/api`, `cmd/cdadmin` | Server and admin CLI |

## API (prefix `/api/v1/cd`)

Customer — the ticket secret goes in the `X-Ticket-Secret` header (or `?secret=` for the event stream only):

| Method | Path | |
| --- | --- | --- |
| GET | `/shops/{slug}` | Public shop: status, hours, prices, wait |
| POST | `/shops/{slug}/jobs` | Create a draft with file metadata → ticket, secret, upload targets |
| GET / PATCH | `/jobs/{id}` | Ticket; update name and per-file settings |
| POST / DELETE | `/jobs/{id}/files[/{fileId}]` | Add / remove files |
| POST | `/jobs/{id}/files/{fileId}/complete` | Confirm upload (server checks size and type) with page count |
| POST | `/jobs/{id}/submit` | Send to counter (needs the quoted `priceVersion`) → token |
| POST | `/jobs/{id}/cancel` | Cancel while in line |
| GET | `/jobs/{id}/events` | SSE stream |

Shop — `Authorization: Bearer <session>`:

| Method | Path | |
| --- | --- | --- |
| GET | `/shop/staff-names?shop=` | Names for the sign-in tiles |
| POST | `/shop/login`, `/shop/logout`; GET `/shop/me` | PIN sign-in (5 wrong tries → 15 min lock) |
| POST | `/shop/setup/info` `{token}` | Check a setup link (who, which shop, expiry) — no auth |
| POST | `/shop/setup` `{token, pin}` | Use a setup link: set PIN and sign in — no auth |
| PUT | `/shop/me/pin` `{currentPin, newPin}` | Change own PIN; signs out your other devices |
| GET / POST | `/shop/staff` `{name, role}` | Owner: list staff; add a person → one-time link |
| POST | `/shop/staff/{id}/link` | Owner: new setup link (reset clears the old PIN) |
| DELETE | `/shop/staff/{id}` | Owner: remove a person |
| GET | `/shop/queue` | Board snapshot |
| POST | `/shop/claim-next` | Claim the oldest job in a lane (`SKIP LOCKED`) |
| POST | `/shop/jobs/{id}/{claim,release,ready,collected,undo,cancel}` | Actions |
| GET | `/shop/jobs/{id}/files/{fileId}/url` | Short-lived download link |
| GET | `/shop/lookup?q=` | Find by token or name |
| PUT | `/shop/state` | Online / Paused / Offline |
| GET / PUT | `/shop/settings` | Owner: profile, hours, prices |
| GET | `/shop/deletion-health` | Owner: deletion backlog and failures |
| GET | `/shop/events?token=` | SSE stream |

## Tests

```bash
CD_TEST_DATABASE_URL='postgres://counter_drop:counter_drop@localhost:55433/counter_drop_test?sslmode=disable' go test ./...
```

Create the test database once: `docker compose -f ../deploy/docker-compose.yml exec postgres createdb -U counter_drop counter_drop_test`. The end-to-end tests are skipped when `CD_TEST_DATABASE_URL` is not set.
