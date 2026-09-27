# 31 — Remote order flow: pay, awaiting shop, accept, reject, timeout

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 4 Print nearby | R1b | L | 30 | FSD SCR-A07–A12, SCR-S06, T3–T9, FS-5.3–5.6, API-13, API-33, BR-R1–R4, UC-C13–C17, UC-S23–S24, AT-15, AT-17–AT-19, AT-21 |

## Goal

The full Print nearby order: pick files → settings → review with fee → pay by UPI → awaiting shop (5-minute countdown) → accepted with pickup code → ready → collected; with automatic handling of reject and timeout and a one-tap "send to another shop".

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md §5 SCR-A07 to SCR-A12 (FS-5.3–5.6), §6 SCR-S06, §8 transitions T3–T9, §14 API-13 and API-33, §11 remote rows, docs/BRD.md §8 UC-C13 main and alternate flows, UC-C14–C17, UC-S23–S24 and BR-R1–R4. Read steps 17 (upload engine), 21 (board), 28–30.

Task: implement remote ordering end to end (money movement to shops comes in step 32; for now captured payments are recorded and refunds are queued).

Backend
1. Create job with channel remote from the details page (POST /shops/{slug}/jobs with channel=remote, customer session required) — same upload engine and limits, plus shop remote limits (BR-R3).
2. On T4 (payment captured): state awaiting_shop, reserve pickup code, schedule accept_timeout at +5 min (BR-R1), outbox remote.pending to shop topic.
3. API-33 POST /shop/jobs/{id}/accept {readyByOffsetMinutes: 0|15|30|60} → T6: state queued in the lane by rule (or a "remote" lane if configured), ready_by computed, cancel the timeout task, outbox to customer (pickup code, ready-by). POST /shop/jobs/{id}/reject {reason: out_of_paper|machine_down|too_busy|cant_print_file|closing_soon} → T7: rejected, Refund effect full amount (queued for step 32 executor), files kept 15 min (reject grace).
4. Timeout T8: expired + full refund effect + shop hidden 30 min (hidden_until) + timeout counter; 3 in 7 days → owner nudge event; 5 → admin review flag (BR-R4).
5. Customer cancel (T9 / T15 remote): awaiting_shop → full refund; queued before claim → total minus fee (BR-P6).
6. API-13 POST /jobs/{id}/resend {shopSlug} → new remote job at the target shop reusing the same storage objects (copy objects server-side to new keys; don't re-upload), same settings, new quote; original job's grace deletion continues.
7. Walk-in collected flow reused; for remote, Collected requires the pickup code match (verifiedBy code) — 409 code_mismatch otherwise.

Web — customer
8. SCR-A07/A08/A09: reuse drop-flow components in a "remote" mode: review shows print subtotal, Convenience fee, Total, ready-by, shop name + distance, and "Pay ₹{total} with UPI" (AuthGate from step 27). Load Razorpay Checkout script only on this screen; UPI intent on Android shows the app chooser.
9. After Checkout success: "Confirming your payment…" (MSG-C09) until the websocket says awaiting_shop (never trust the client callback).
10. SCR-A10 Awaiting shop: 5:00 countdown ring, copy "You won't be charged if they can't take it", Cancel (full refund).
11. SCR-A11: pickup code (48 px) + QR (encodes cd:{slug}:{code}:{shortJobId}), ready-by, position; Ready state with Directions.
12. SCR-A12: rejected/timed out → reason, refund amount + status, primary "Send to {next best shop}" (calls API-13 then goes straight to pay), secondary "Choose another shop".

Web — shop
13. SCR-S06 remote order alert on the board: modal with loud chime repeated every 20 s until opened (FS-6.4), countdown, files, pages, settings, print amount (what the shop receives), ready-by adjusters +15/+30/+60, Accept (green) / Reject with one-tap reasons. Remote jobs show a Prepaid badge on cards; Collect requires code entry or QR scan (reuse scanner in pickup mode).

Tests
14. Go: T3–T9 integration with FakeGateway; resend copies objects; penalty and nudges. Playwright (two contexts: customer phone + shop PC) for AT-15 (happy path through collected), AT-17 (reject → refund queued → suggestion), AT-18 (timeout with fake clock endpoint on staging/test only), AT-19 (resend without re-upload), AT-21 (cancel after accept = total − fee).

Rules: a scan never reaches this flow; only Home → Print nearby → shop details → Order here. Show the backend changes and the customer screen flow first.
```

## Acceptance

- [ ] AT-15, AT-17, AT-18, AT-19, AT-21 pass (refund execution verified in step 32).
- [ ] Shop sees and hears a remote order within 2 s of payment capture.
- [ ] Resend reuses files without re-upload.

## Commit

`feat(remote): Print nearby ordering with UPI prepay, accept/reject/timeout, pickup codes and resend`
