# 03 — OpenAPI contract v1

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 0 Foundation | R1a (+ R1b stubs) | M | 01 | FSD §8, §9, §14, §15, §16 |

## Goal

`contracts/openapi.yaml` fully describes every R1a endpoint (and R1b endpoints marked `x-release: R1b`), with schemas, error codes and examples, so backend and frontend can work in parallel.

## Why now

The contract is the handshake between the Go API and the PWA. The frontend track (step 15 onward) mocks against it; the backend track implements it.

## Scope

**In:** full OpenAPI 3.1 document, shared schemas, error envelope, security schemes, examples, lint config, a generated TypeScript types file for the web.
**Out:** implementing endpoints.

## Files

- `contracts/openapi.yaml` (rewrite)
- `contracts/.redocly.yaml` (lint rules)
- `contracts/README.md` (how to lint, preview, generate types)
- `Makefile` (targets `contract-lint`, `contract-types`)

## Prompt

```text
First read docs/build-plan/00-common-context.md. Then read docs/FSD.md sections 2 (roles), 8 (job lifecycle), 9 (pricing, tokens, wait), 14 (API specification), 15 (real-time events) and 16 (validation and messages). Then read the current contracts/openapi.yaml and api/internal/httpapi/router.go.

Task: rewrite contracts/openapi.yaml as the complete v1 contract.

Requirements:

1. OpenAPI 3.1. servers: http://localhost:8080 (local), https://api.staging.counterdrop.in, https://api.counterdrop.in. Base path /api/v1/cd for all product routes; /health at root.
2. Envelope (new, replaces {success,data,error}):
   - Success: { "data": <payload> }
   - Error: { "error": { "code": string, "message": string, "details": object } }
   Define components/schemas/Error and a reusable response for each HTTP error (400, 401, 403, 404, 409, 422, 429, 500). List every error `code` from FSD §16 and §8 in an enum on Error.code with descriptions (e.g. invalid_transition, price_changed, uploads_incomplete, shop_paused, shop_closed, file_type, file_too_large, too_many_files, pdf_locked, pdf_corrupt, page_range, option_unavailable, name_invalid, phone_invalid, otp_invalid, otp_expired, slug_taken, hours_invalid, price_invalid, override_reason, pay_not_allowed, shop_unavailable, pages_unknown, rate_limited, unauthorized, forbidden, not_found).
3. Security schemes: bearerAuth (opaque session token), ticketSecret (apiKey in header X-Ticket-Secret), razorpaySignature (apiKey in header X-Razorpay-Signature). Apply per operation as in FSD §2 and §14.
4. Schemas (camelCase JSON): Shop, ShopPublic, ShopState (online|paused|offline), Hours, PriceList, Rate, AddOn, Lane, Job, JobState (uploading|payment_pending|awaiting_shop|queued|claimed|ready|collected|cancelled|rejected|expired; plus refunded as a paymentStatus field), JobChannel (walkin|remote), JobFile, FileSettings (copies 1–99, colour bw|colour, sides one|both, pageRange string, paper A4|Legal|A3, orientation auto|portrait|landscape, fit fit|actual, addOns[]), Quote (lines[], addOns[], printTotalPaise, feePaise, totalPaise, priceVersion, pagesToConfirm, readyBy), Ticket (id, token, lane, state, position, readyBy, pickupCode?, files, quote, timeline[], filesDeletedAt?), QueueSnapshot (lanes[{lane, jobs[]}], remotePending[], printing[], ready[], collectedTray[], shopState, wait), JobCard (the dashboard card fields from FSD SCR-S03), NearbyShop, Staff, Device, Session, Payment, Refund, LedgerEntry, Rating, Dispute, AuditEntry. Money fields end in `Paise` and are integers.
5. Paths: every endpoint in FSD §14 tables (API-00 to API-70), with operationId, summary, tags (public, customer, shop, owner, agent, admin, webhooks), parameters, request bodies, responses with examples. Add `x-release: R1a|R1b|R2` on each operation. Include the upload-only rule: POST /jobs/{id}/pay returns 409 pay_not_allowed for walk-in jobs.
6. Keep the existing routes (GET /shop/queue?shop=, POST /jobs/{id}/submit?secret=, GET /jobs/{id}?secret=) documented but marked deprecated: true with a note pointing to the new form (X-Ticket-Secret header, session-scoped queue). Accept `new` as a deprecated alias of `queued` in the JobState description.
7. WebSocket: document /ws/jobs/{id} and /ws/shop as GET operations with 101 response and describe every event from FSD §15 as schemas under components/schemas/Event* plus an `x-events` list.
8. Idempotency-Key header parameter on all POSTs that create jobs, pay or refund. Pagination (cursor, limit ≤ 100) on list endpoints. 429 with Retry-After.
9. contracts/.redocly.yaml with recommended rules; fix all errors. Add Makefile targets:
   - contract-lint: npx @redocly/cli lint contracts/openapi.yaml
   - contract-types: npx openapi-typescript contracts/openapi.yaml -o web/src/lib/api/schema.d.ts (create the folder if missing)
10. contracts/README.md explaining the conventions above in under 60 lines.

Rules: don't implement anything in Go in this step. Where FSD is silent, choose the simplest consistent shape and list your choices at the end of your message so I can confirm them. Show the list of schemas and paths you'll write before writing the full file.
```

## Acceptance

- [ ] `make contract-lint` passes with zero errors.
- [ ] Every API ID from FSD §14 appears as an operation with `x-release`.
- [ ] `make contract-types` produces `web/src/lib/api/schema.d.ts`.
- [ ] Deprecated routes are still present and marked.
- [ ] Error codes enum covers FSD §16.

## Verify

```bash
make contract-lint
make contract-types
grep -c "operationId" contracts/openapi.yaml
```

## Commit

`docs(contract): full OpenAPI v1 for R1a with R1b operations, errors and events`
