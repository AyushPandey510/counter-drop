# 32 — Route settlement, shop payout KYC and refunds

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 4 Print nearby | R1b | L | 31 | FSD §10 FS-10.4–10.6, SCR-S13, SCR-S14, API-41, API-42, BR-M2–M6, BR-P6–P9, UC-S07, UC-S26, OI-2 |

## Goal

The shop's print amount is transferred to its own bank account via Razorpay Route, held until the shop accepts and reversed on reject; every refund is automatic within 60 seconds; Counter Drop never holds funds.

## Before you start

- Resolve open item OI-2 with Razorpay: Route enabled on your account, linked accounts for small proprietor shops, `on_hold` transfers supported, and the current UPI + Route charges (this sets the convenience fee).
- Read the current Razorpay Route docs (linked accounts / stakeholders / product configuration, transfers from payments, on_hold and on_hold_until, modify transfer, reversals) and Refunds docs.

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md §10 (FS-10.4 to FS-10.8), SCR-S13, SCR-S14, §14 API-41 and API-42, docs/BRD.md BR-M2–M6, BR-P6–P9 and UC-S07, UC-S26. Then read the CURRENT Razorpay Route and Refunds documentation and confirm endpoint paths and fields; don't rely on memory.

Task: implement shop payouts, transfers and refunds.

1. Shop onboarding (API-42 POST /shop/payout/onboard): create or resume a Razorpay linked account for the shop (legal business name, contact, business type proprietorship, bank account via the hosted/KYC flow Razorpay provides), store payout_account_id and payout_status (pending → active | needs_clarification | rejected | on_hold) updated from account webhooks. The owner UI (SCR-S13) shows status and "Continue KYC". Remote orders are orderable only when active (step 28 rule).
2. Transfers: executor for the effects from steps 30–31:
   - On payment captured: create a transfer of print_total_paise to the linked account with on_hold=true; ledger −transfer.
   - On accept: modify transfer on_hold=false (release) — BR-M3.
   - On reject/timeout/cancel before accept: reverse the transfer (reversal), then refund.
   - Handle transfer.processed / transfer.failed / reversal webhooks; retries with backoff; alert on failure.
3. Refund executor: CreateRefund(payment, amount per BR-P6 rule, notes {job_id, reason}); speed "normal"; within 60 s of the trigger (FS-10.6) via an outbox-driven worker; handle refund.processed and refund.failed (retry 3× then alert support); customer push/WS "Refund of ₹X sent (ref …)" and the ledger entry.
4. Settlements: consume settlement.processed webhooks (or a daily fetch) to mark per-order expected/actual settlement dates for the shop statement.
5. API-41 GET /shop/money?from=&to= (owner): remote orders with print amount, fee (informational), refunds, transfer status, settlement status and expected date; CSV export. SCR-S14 UI table with filters and totals.
6. Walk-in: nothing changes; walk-in money never touches Counter Drop (BR-M1).
7. Safety: all money operations idempotent by (job_id, operation) keys; a kill switch CD_PAYMENTS_ENABLED=false pauses new remote orders and shows the banner (FS-10.10) while letting refunds and reversals continue.
8. Tests with FakeGateway: capture → transfer on hold → accept → release; capture → reject → reversal → full refund; customer cancel after accept but before claim → reverse the released transfer and refund total − fee (the fee stays with Counter Drop, per BR-P6); refund failure retry; ledger sums to zero per job after settlement for each scenario.

Rules: never store bank details ourselves; Razorpay holds KYC. Show the money state table (event → transfer action → refund amount → ledger rows) first and wait for approval.
```

## Acceptance

- [ ] Every refund scenario from BR-P6 produces the right amount within 60 s (test mode).
- [ ] Shop money is released only after accept.
- [ ] Ledger balances to zero per job after settlement.
- [ ] Kill switch pauses new orders but not refunds.

## Commit

`feat(settlement): Razorpay Route linked accounts, on-hold transfers, reversals and automatic refunds`
