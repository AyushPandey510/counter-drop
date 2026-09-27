# 30 — Razorpay payments, webhooks and ledger

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 4 Print nearby | R1b | L | 12, 27 | FSD §10 (FS-10.1–10.3, 10.7, 10.9), §13 payments tables, API-12, API-70, BR-M1–M6, CT-6, AT-16, AT-20 |

## Goal

Prepay by UPI through Razorpay with a verified-webhook-only source of truth and a double-entry-style ledger per job. (Route transfers and refunds come in step 32.)

## Before you start

- Razorpay account in test mode with API key/secret and a webhook secret.
- Read the current Razorpay docs for Orders, Payments, Webhooks and Checkout (web). APIs change; the agent must confirm field names against the docs, not memory.

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md §10 (diagram notes and FS-10.1–10.11), §13 (cd_payments, cd_transfers, cd_refunds, cd_ledger, cd_webhook_events), §14 API-12 and API-70, docs/BRD.md BR-M1–M6 and CT-6. Then read the current official Razorpay documentation for Orders API, Payment capture (auto-capture), Webhooks (signature verification and event list) and Standard Checkout for web. Confirm every field name against the docs.

Task: implement the payments module without Route (step 32) and without the remote order UI (step 31).

1. Migration 0009_payments_ledger.sql: cd_payments (job_id unique, razorpay_order_id unique, razorpay_payment_id unique nullable, amount_paise, fee_paise, currency, status created|attempted|captured|failed|refunded_partial|refunded, captured_at, method), cd_transfers, cd_refunds (shape now, used in step 32), cd_ledger (id, job_id, kind payment|transfer|reversal|refund|fee, amount_paise signed, ref, at), cd_webhook_events (event_id PK, type, received_at, processed_at, payload_hash).
2. api/internal/payments/razorpay: small HTTP client (no SDK needed) with basic auth, timeouts, retries on 5xx with idempotency (Razorpay receipt = job ID). Interface Gateway { CreateOrder; FetchPayment; CreateRefund; CreateTransfer; PatchTransfer; ReverseTransfer } with a FakeGateway for tests.
3. API-12 POST /jobs/{id}/pay (customer session, remote jobs only; walk-in → 409 pay_not_allowed): revalidate shop orderability and price (priceVersion) → create order (amount = total, notes {job_id, shop_id}) → transition pay (T3) → return Checkout params (key_id, order_id, amount, currency INR, name "Counter Drop", description "{shop} · {pages} pages", prefill contact from the signed-in user, method preference upi, theme color). Idempotency-Key honoured.
4. API-70 POST /webhooks/razorpay: verify X-Razorpay-Signature (HMAC-SHA256 of the raw body with the webhook secret, constant-time) → 401 on failure (logged, no body). Insert cd_webhook_events by event ID (conflict → 200 no-op, FS-10.3). Handle payment.captured (amount must equal order amount → transition T4 payment_captured, ledger +payment, −fee split recorded), payment.failed (mark attempted/failed; T5 only on timeout), and store other events for step 32. Process synchronously but fast (< 2 s); heavy work via outbox.
5. Payment timeout: T5 scheduled task (15 min after order creation) → if still not captured, FetchPayment status via the order; if captured late, process as capture; else expire the job (and refund if any capture appears later — EX-C09).
6. Double capture (FS-10.9): a second captured payment for the same order/job → record and emit a Refund effect (executed in step 32; leave queued).
7. Client callback: POST /jobs/{id}/pay/confirm {razorpay_payment_id, razorpay_order_id, razorpay_signature} only verifies the signature and returns "confirming"; it never changes state (FS-10 note, AT-16).
8. Ledger invariants: helper LedgerBalance(job) and a test asserting payment = transfer + fee − refunds once settled (full check in step 33).
9. Config: CD_RAZORPAY_KEY_ID, CD_RAZORPAY_KEY_SECRET, CD_RAZORPAY_WEBHOOK_SECRET, CD_PAYMENTS_ENABLED.
10. Tests with FakeGateway and recorded webhook payload fixtures (from the docs' examples): signature valid/invalid; duplicate event (AT-20); amount mismatch → alert and no transition; callback without webhook keeps state (AT-16); timeout expiry; late capture.

Rules: money in paise; never trust client amounts; never log card/UPI details (store method type only). Show the webhook handling state table first.
```

## Acceptance

- [ ] AT-16 and AT-20 pass.
- [ ] Invalid signatures are rejected; duplicates are no-ops.
- [ ] Walk-in jobs can't be paid.

## Commit

`feat(payments): Razorpay orders, verified webhooks, idempotency and job ledger`
