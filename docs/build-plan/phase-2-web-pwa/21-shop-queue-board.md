# 21 — Shop queue board, job panel, collect and rush mode

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 2 Web PWA | R1a | L | 11, 13, 20 | FSD SCR-S03–S05, S07, FS-6.1–6.4, UX-S1–S8, EX-S01–S04, AT-04, AT-05, AT-09, AT-12 |

## Goal

Staff run the whole counter with four taps — Claim, Print, Ready, Collected — on a calm, dark, readable-from-a-metre board that updates live and never double-claims.

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md §6 SCR-S03, S04, S05, S07 and FS-6.1–6.4, docs/BRD.md UX-S1–S8, EX-S01–S04 and delight moments for shopkeepers. Read the shop API (API-30–38) in contracts/openapi.yaml and web/src.

Task: build the shop board at /shop.

1. Top bar: shop status segmented control Online · Paused · Offline (Paused asks for an optional message), search/scan box (focus with "/"), Rush mode toggle, today's count, settings menu (owner only), staff name + Switch staff.
2. Board (follow design/screens/06_shop_pc_queue_board and design/DESIGN.md §7): four columns In line · Printing · Ready · Collected (10-min undo), with lanes as filter tabs above the board (All lanes / Lane A / Lane B, with counts) and a lane chip on every card. "Claim oldest [N]" claims the oldest job in the selected lane (all lanes when "All" is selected). On phones the columns become tabs (design/screens/05_shop_mobile_queue). Job card: token (28 px, monospace), first name, channel badge (Walk-in), files × pages, settings summary, price (₹), age with amber at 10 min and red at 20 min. Light theme by default with a dark theme option (owner setting), per design/DESIGN.md. Apply the content rules in DESIGN.md §8: no racks/trays, no printer telemetry or page progress in R1a, no staff-created jobs.
3. Data: initial GET /shop/queue then /ws/shop events (queue.job_added, job_changed, job_removed, shop.state); apply events by seq, refetch on gap, full resync every 60 s and on reconnect (FS-6.1). Optimistic UI for claim/ready/collected with rollback on error.
4. Claim: "Claim next" → POST claim-next; tapping a card → POST claim; on already_claimed animate the card away and toast MSG-S01 (FS-6.2). Card opens the Job panel.
5. SCR-S04 Job panel (side sheet on PC, full screen on phone): file list with thumbnail, pages and per-file settings in bold, "Open" (API-36 signed URL in a new tab), "Print all" (opens /shop/jobs/{id}/print.pdf in a new tab so the browser print dialog shows; the cover page lists settings), buttons Ready (primary), Release, Ask customer (preset reasons), Edit (copies, add-ons, price override with reason — owner PIN prompt if > 3×), Cancel (reason required — the only confirmation dialog).
6. SCR-S05 Collect: search accepts A07 / a-07 / 4-digit code / first name (API-37); camera scan button for a customer's pickup QR (reuse the step 19 scanner component in "pickup" mode — R1b codes, harmless now). Result card: token, name, files, pages, "Collect ₹{price}" (walk-in). Buttons Collected + optional Paid cash / Paid UPI. After Collected, card moves to a Collected tray with an Undo chip and countdown (10 min).
7. Sounds (FS-6.4): soft tick on new walk-in job; unlock audio on first tap after sign-in; mute toggle.
8. Keyboard (FS-6.3): N claim next in focused lane, R ready, C collected, / search, Esc close panel, arrow keys move focus between cards. Show a shortcut help overlay on "?".
9. SCR-S07 Rush mode: only the next 3 cards per lane, 64 px buttons, settings summaries hidden; suggest enabling when queued > 15.
10. Offline (EX-S04): banner MSG-S02, board read-only with the last snapshot, actions queued and retried when back online, never assume a claim succeeded while offline.
11. Tests: Vitest for the event reducer (ordering, gaps, optimistic rollback) and keyboard handlers; Playwright against the real API (test DB) for AT-05 (release), AT-09 (pause), AT-12 (undo), and a two-browser-context race where both claim the same card (one wins, one toasts); visual check at 1366×768 and 360×800.

Rules: no confirmation dialogs except Cancel (UX-S2); every action one tap; numbers in Western digits. Show the component tree and the event reducer design first.
```

## Acceptance

- [ ] Two counters never claim the same job; the loser sees the toast.
- [ ] A new walk-in job appears on the board within 1 s.
- [ ] Undo restores a collected job within 10 minutes.
- [ ] Staff can learn the board in under 5 minutes (usability check with 2 people).

## Verify

```bash
cd web && pnpm test && pnpm exec playwright test shop-board
```

## Commit

`feat(shop): live queue board with lanes, job panel, print-all, collect, undo and rush mode`
