# Architecture

Full system overview, data model and API are in [`FSD.md`](FSD.md) §1, §13 and §14. This page is the short version.

Counter Drop is separate from SwiftShare: its own repository, database, storage prefix and deployment.

## Boundaries

- One Go API serves every client under `/api/v1/cd` (one versioned contract: `contracts/openapi.yaml`).
- One React PWA (`web/`) serves the customer drop page, the installed customer app, the shop dashboard, the TV screen and the admin console.
- One DynamoDB table (`cd-main`) is the source of truth; the design (keys, indexes, TTL) is in ADR-001, *Counter-Drop-Serverless-Design*. Money in paise, times in UTC.
- Target runtime (ADR-001): Lambda behind API Gateway and CloudFront, WebSocket for live updates, EventBridge Scheduler for the deletion worker, all from a CDK stack.
- File bytes never pass through the API: clients upload with presigned PUT URLs; staff read with 5-minute signed GET URLs. Files live at most 24 hours under the `cd/` storage prefix.
- The print agent (R2) talks only to Counter Drop API endpoints.
- Razorpay (R1b) is the only external system that drives state, through verified webhooks.

## API modules (`api/internal/`)

| Module | Responsibility | Status |
| --- | --- | --- |
| `config` | `CD_*` environment config | exists |
| `domain` | Job state machine, pricing, tokens, wait (pure Go, no I/O) | basic state machine only |
| `store` | `Repository` interface, rules; `store/ddbstore` = DynamoDB | exists |
| `httpapi` | Routes, middleware, JSON envelope, error mapping | exists (single file, no auth) |
| `storage` | S3/R2 presigned URLs, head, delete | presign PUT only |
| `realtime` | WebSocket hub for job and shop channels | not started (step 13) |
| `tasks` | Deletion worker, timers, outbox dispatcher, reconciliation | not started (steps 12, 14) |
| `auth` | Phone OTP, staff PIN, sessions | not started (step 07) |
| `payments` | Razorpay orders, webhooks, Route, refunds, ledger | not started (R1b, steps 30–32) |
