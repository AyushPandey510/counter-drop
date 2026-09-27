# Counter Drop — Build Plan

Step-by-step plan to build Counter Drop from the current code to the R1a walk-in pilot, then R1b (Print nearby). Each step is one file with the goal, dependencies, files to touch, a **copy-paste prompt** for your coding agent (Claude Code or similar), acceptance checks and a commit message.

Source of truth: `docs/BRD.md` (what and why) and `docs/FSD.md` (exact behaviour). When a step and the FSD disagree, the FSD wins. Update the FSD first if you change a rule.

## How to use each step

1. Open the step file. Check that everything under **Depends on** is merged.
2. Start a fresh agent session in the repo root (`~/counter-drop`).
3. Paste the prompt from the step file. The prompt tells the agent to read `00-common-context.md` and the FSD sections it needs.
4. Review the plan the agent proposes before it writes code.
5. Run the **Verify** commands yourself. Tick the **Acceptance** checklist.
6. Commit with the suggested message, then move to the next step.

One step = one branch = one PR. Don't combine steps; small PRs are easier to review and roll back.

## Steps

### Phase 0 — Foundation (week 1)

| # | Step | Size | Depends on |
| --- | --- | --- | --- |
| 01 | [Dev environment, Makefile, CI](phase-0-foundation/01-dev-environment-and-ci.md) | S | — |
| 02 | [Object storage on Cloudflare R2 + upload verification](phase-0-foundation/02-storage-r2.md) | S | 01 |
| 03 | [OpenAPI contract v1](phase-0-foundation/03-openapi-contract.md) | M | 01 |

### Phase 1 — Backend core, R1a (weeks 1–3)

| # | Step | Size | Depends on |
| --- | --- | --- | --- |
| 04 | [API foundation refactor: context, errors, envelope, router split](phase-1-backend-core/04-api-foundation-refactor.md) | M | 03 |
| 05 | [Schema v2 migrations 0004–0008](phase-1-backend-core/05-schema-v2-migrations.md) | M | 04 |
| 06 | [Job state machine v2](phase-1-backend-core/06-job-state-machine-v2.md) | M | 05 |
| 07 | [Auth: phone OTP, staff PIN, sessions, shop scope](phase-1-backend-core/07-auth-otp-pin-sessions.md) | L | 05 |
| 08 | [Pricing engine and quotes](phase-1-backend-core/08-pricing-engine.md) | M | 05 |
| 09 | [Tokens, business day and wait engine](phase-1-backend-core/09-tokens-and-wait-engine.md) | M | 06 |
| 10 | [Walk-in job API](phase-1-backend-core/10-walkin-job-api.md) | L | 02, 06, 08, 09 |
| 11 | [Shop API: queue, claim, actions, settings](phase-1-backend-core/11-shop-api.md) | L | 07, 10 |
| 12 | [Outbox, job events, audit log, scheduled tasks](phase-1-backend-core/12-outbox-events-audit-tasks.md) | M | 06 |
| 13 | [Realtime WebSocket hub](phase-1-backend-core/13-realtime-websocket.md) | M | 12 |
| 14 | [Deletion worker and retention](phase-1-backend-core/14-deletion-worker.md) | M | 12 |

### Phase 2 — Web PWA, R1a (weeks 3–6)

| # | Step | Size | Depends on |
| --- | --- | --- | --- |
| 15 | [Web scaffold: Vite, React, PWA, API client, design tokens](phase-2-web-pwa/15-web-scaffold-pwa.md) | M | 03 |
| 16 | [i18n (en/hi/mr) and accessibility baseline](phase-2-web-pwa/16-i18n-accessibility.md) | S | 15 |
| 17 | [Customer drop flow (upload only)](phase-2-web-pwa/17-customer-drop-flow.md) | L | 10, 16 |
| 18 | [Ticket and deletion receipt](phase-2-web-pwa/18-ticket-and-receipt.md) | M | 13, 17 |
| 19 | [PWA home, in-app scanner, install banner](phase-2-web-pwa/19-pwa-home-scanner-install.md) | M | 17 |
| 20 | [Shop signup wizard and staff login](phase-2-web-pwa/20-shop-signup-and-login.md) | M | 07, 16 |
| 21 | [Shop queue board, job panel, collect, rush mode](phase-2-web-pwa/21-shop-queue-board.md) | L | 11, 13, 20 |
| 22 | [Shop settings: prices, hours, lanes, staff, QR kit](phase-2-web-pwa/22-shop-settings-and-qr-kit.md) | M | 11, 20 |

### Phase 3 — Pilot readiness, R1a (weeks 6–8)

| # | Step | Size | Depends on |
| --- | --- | --- | --- |
| 23 | [Admin console (R1a)](phase-3-pilot-readiness/23-admin-console-r1a.md) | M | 12, 14 |
| 24 | [Security hardening and observability](phase-3-pilot-readiness/24-security-observability.md) | M | 11 |
| 25 | [Deploy: staging and production](phase-3-pilot-readiness/25-deploy.md) | M | 24 |
| 26 | [R1a acceptance tests and pilot runbook](phase-3-pilot-readiness/26-r1a-acceptance-and-pilot.md) | M | 21, 22, 25 |

### Phase 4 — Print nearby, R1b (weeks 9–18)

| # | Step | Size | Depends on |
| --- | --- | --- | --- |
| 27 | [Customer accounts (OTP) and /me](phase-4-print-nearby/27-customer-accounts.md) | S | 07 |
| 28 | [Geo search and nearby API](phase-4-print-nearby/28-nearby-api.md) | M | 09 |
| 29 | [Nearby UI and shop details](phase-4-print-nearby/29-nearby-ui.md) | M | 19, 28 |
| 30 | [Razorpay payments, webhooks, ledger](phase-4-print-nearby/30-razorpay-payments-ledger.md) | L | 12, 27 |
| 31 | [Remote order flow: pay, accept, reject, timeout](phase-4-print-nearby/31-remote-order-flow.md) | L | 30 |
| 32 | [Route settlement, shop KYC, refunds](phase-4-print-nearby/32-route-settlement-refunds.md) | L | 31 |
| 33 | [Reconciliation, disputes, admin money pages](phase-4-print-nearby/33-reconciliation-disputes.md) | M | 32 |
| 34 | [Web push and minimal shop mode](phase-4-print-nearby/34-web-push-shop-mode.md) | M | 31 |
| 35 | [History, reorder, ratings, share target, delete account](phase-4-print-nearby/35-history-ratings-share.md) | M | 31 |
| 36 | [R1b acceptance tests and launch](phase-4-print-nearby/36-r1b-acceptance-and-launch.md) | M | 32–35 |

### Phase 5 — R2 backlog

| # | Step | Depends on |
| --- | --- | --- |
| 37 | [R2 backlog: print agent, TV, billing, reports, Aadhaar masking, spoken announcements](phase-5-r2/37-r2-backlog.md) | R1b live |

Sizes: S ≈ half a day to 1 day, M ≈ 2–3 days, L ≈ 4–5 days for one developer working with an agent.

## Parallel tracks

Steps with no dependency between them can run in parallel on separate branches:

- **Backend track:** 04 → 05 → 06/07/08 → 09 → 10 → 11 → 12 → 13/14
- **Frontend track:** 15 → 16 can start right after 03, using mocked API responses from the OpenAPI contract.

## Definition of done (every step)

- [ ] `make test` passes (Go and web).
- [ ] `make lint` passes.
- [ ] New behaviour has tests (unit for rules, integration for SQL and HTTP).
- [ ] `contracts/openapi.yaml` matches any API change.
- [ ] No secrets, phone numbers or file names in logs.
- [ ] FSD section references in the PR description.
- [ ] README or step notes updated if run commands changed.
