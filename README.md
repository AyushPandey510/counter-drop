# Counter Drop

Counter Drop replaces "send it on my WhatsApp" at print and xerox shops. A customer scans the shop's QR, uploads files from the phone browser, sees the price and wait, gets a token like `A-07` and collects when it turns green. Files are deleted automatically after pickup. Later, customers use **Print nearby** in the same PWA to find an open shop, prepay by UPI and collect with a pickup code. Shops run every job from one live queue.

This project is separate from SwiftShare: its own API, database, web app, storage prefix and print-agent path.

## Documents

| Document | What it is |
| --- | --- |
| [`docs/BRD.md`](docs/BRD.md) | Business requirements: goals, users, use cases, rules |
| [`docs/FSD.md`](docs/FSD.md) | Functional spec: screens, states, pricing, API, data model, tests |
| [`design/DESIGN.md`](design/DESIGN.md) | Design system and content rules; reference screens in `design/screens/` |
| [`docs/build-plan/README.md`](docs/build-plan/README.md) | Step-by-step build plan with a prompt per step |
| [`docs/CODE-STATUS.md`](docs/CODE-STATUS.md) | What exists in the code today vs the plan, and known issues |
| [`docs/ROADMAP.md`](docs/ROADMAP.md) | Releases R1a → R3 |

## Current status

A backend slice works: create job → presigned upload URL → submit → shop queue → claim / ready / collected / cancel / release, on Postgres or an in-memory store. Not built yet: auth, pricing, real upload verification, live updates, the deletion worker and the whole web app. See [`docs/CODE-STATUS.md`](docs/CODE-STATUS.md).

**Next:** build-plan steps 01 (dev environment and CI) → 02 (R2 storage) → 03 (OpenAPI contract).

## Project layout

```text
counter-drop/
├── api/                  # Go HTTP API (module counter-drop/api)
├── agent/                # Windows print agent (R2, placeholder)
├── pkg/cdclient/         # Shared Go client (placeholder)
├── web/                  # React/Vite PWA (scaffolded in step 15)
├── contracts/openapi.yaml
├── deploy/               # docker-compose (Postgres, MinIO), .env.example
├── design/               # Design system, reference screens, Stitch export
└── docs/                 # BRD, FSD, build plan, roadmap, architecture, code status
```

## Getting started

First time only — put the project under git:

```bash
cd ~/counter-drop
git init -b main
git add .
git commit -m "chore: initial import with docs, design and backend slice"
```

Start Postgres and run the API:

```bash
docker compose -f deploy/docker-compose.yml up -d postgres
cp deploy/.env.example deploy/.env      # edit values; deploy/.env is git-ignored
cd api
set -a; . ../deploy/.env; set +a
go run ./cmd/api
curl http://localhost:8080/health
```

Without `CD_DATABASE_URL` the API uses an in-memory demo store. Storage presigned URLs need `CD_STORAGE_*` values; local MinIO did not pull on this machine, so use Cloudflare R2 (build-plan step 02). The full curl walkthrough is in [`api/README.md`](api/README.md).

## Tests

Run from `api/` (the repo root is a `go.work` workspace, not a module):

```bash
cd api
go test ./...
```
