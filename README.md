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

**MVP (R1a walk-in pilot) works end to end:** QR drop page (upload only, pay at the counter) → price and wait shown before sending → daily token per lane (`A-01` B/W, `B-01` colour) → live ticket → staff PIN sign-in → live queue board (claim, ready, collected with cash/UPI, undo, cancel with reason) → files deleted automatically 10 min after pickup, with a deletion receipt on the customer's phone. Also: owner settings (hours, prices), printable QR poster, Online/Paused/Offline, English/Hindi/Marathi customer screens. Not in the MVP: phone OTP, shop self-signup (use `cdadmin`), Print nearby and UPI prepay (R1b), print agent, admin console. See [`docs/CODE-STATUS.md`](docs/CODE-STATUS.md).

## Project layout

```text
counter-drop/
├── api/                  # Go HTTP API (module counter-drop/api)
├── agent/                # Windows print agent (R2, placeholder)
├── pkg/cdclient/         # Shared Go client (placeholder)
├── web/                  # React/Vite PWA: drop page, ticket, shop board, settings, QR poster
├── contracts/openapi.yaml
├── deploy/               # docker-compose (DynamoDB Local; MinIO optional), .env.example, AWS files
├── design/               # Design system, reference screens, Stitch export
└── docs/                 # BRD, FSD, build plan, roadmap, architecture, code status
```

## Getting started

Needs Docker, Go 1.27 and Node 20+.

```bash
docker compose -f deploy/docker-compose.yml up -d dynamodb     # DynamoDB Local (data store)

# API (terminal 1)
cd api
CD_DYNAMODB_ENDPOINT=http://localhost:8000 go run ./cmd/api

# Web (terminal 2)
cd web
npm install
npm run dev -- --host
```

- Customer: open http://localhost:5173/s/demo-print (on a phone: `http://<your-laptop-ip>:5173/s/demo-print`).
- Shop: http://localhost:5173/shop/login → shop `demo-print` → **Kavita** PIN `1111`, or **Owner** PIN `1234` for settings.
- One-process mode: `cd web && npm run build`, then start the API with `CD_WEB_DIR=../web/dist` and open http://localhost:8080.

To onboard a real shop: `cd api && go run ./cmd/cdadmin create-shop -slug imran-xerox -name "Imran Xerox" -owner Imran` prints a one-time link; send it to the owner, who opens it and chooses their own PIN. See [`api/README.md`](api/README.md#onboarding-shops-cdadmin).

Files are stored on local disk by default (`api/.data/files`); set the `CD_STORAGE_*` R2 values to use Cloudflare R2. All settings: [`deploy/.env.example`](deploy/.env.example). API reference: [`api/README.md`](api/README.md).

## Tests

```bash
cd api
CD_TEST_DYNAMODB_ENDPOINT=http://localhost:8000 go test ./...   # needs DynamoDB Local running
cd ../web && npm run build      # type-check + build
```
