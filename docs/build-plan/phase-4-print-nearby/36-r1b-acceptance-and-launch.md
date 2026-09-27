# 36 — R1b acceptance tests and Dadar launch

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 4 Print nearby | R1b | M | 32, 33, 34, 35 | FSD §19 AT-15–AT-23, BRD §21 R1b acceptance criteria, §18 launch sequence, §6 Gate 2 |

## Goal

Print nearby goes live in Dadar with 15+ listed shops (8+ remote-enabled), all R1b acceptance tests green, payments in live mode and a launch plan.

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md §19 (AT-15 to AT-23), docs/BRD.md §21 (R1b acceptance criteria), §18 (launch sequence, customer acquisition) and §6 (Gate 2). Read the e2e suite from step 26.

Task: finish R1b quality and launch readiness.

1. Extend e2e/ with AT-15 to AT-23 (two browser contexts for customer phone + shop PC; Razorpay test mode with test UPI IDs for success/failure; a staging-only clock endpoint to fast-forward timers for AT-18). Add them to the nightly run.
2. Payment go-live checklist docs/launch/payments-live.md: live keys in the secret store, webhook URL registered with the live secret, Route live approval, linked-account KYC done for launch shops, test ₹1 order refunded end to end in live mode, reconciliation run clean, kill switch rehearsed.
3. Legal checklist docs/launch/legal.md: terms of use, privacy notice (EN/HI/MR) reflecting FSD §16 data inventory, shop DPA signed by launch shops, refund policy page matching BR-P6–P9, grievance officer contact (DPDP) — to be reviewed by the legal advisor (OI-3).
4. Density plan docs/launch/dadar.md: list of target shops with status (listed, remote-enabled, KYC), map screenshot, owner contacts kept outside the repo; go/no-go = 15 listed, 8 remote-enabled, median acceptance < 90 s in a dry-run week.
5. Customer launch: set VITE_FEATURE_NEARBY=true for the Dadar area only (feature flag by cluster from the API), first-order fee waiver on, in-shop posters "Order ahead, skip the line" (add a poster variant to the QR kit), college outreach one-pager.
6. Monitoring for launch week: dashboard panels for remote orders, acceptance time, refunds, payment failures, push latency; on-call rota and incident runbook links.
7. Post-launch review template docs/launch/review-week-1.md with the BRD metrics (BO-3 remote share, CO-2 median order time, refund rate, NPS).

Rules: no launch until every checklist item is ticked and Gate 2 criteria are met. Show the go/no-go checklist first.
```

## Acceptance

- [ ] AT-15–AT-23 green nightly on staging and once in production with live payments (₹1 test).
- [ ] Legal, payments and density checklists complete.
- [ ] Nearby enabled for Dadar only.

## Commit

`chore(launch): R1b acceptance suite, payments live checklist and Dadar launch plan`
