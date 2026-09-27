# 18 — Ticket and deletion receipt

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 2 Web PWA | R1a | M | 13, 17 | FSD §4 SCR-W04, W05, FS-4.8–4.9, §15, §11 (customer events), UC-C04–C08, AT-10, AT-11 |

## Goal

After sending, the customer watches a big token and live status, gets a loud, clear "Ready" moment, can edit/cancel before claim, sees "Ask customer" messages, and ends with a deletion receipt.

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md §4 SCR-W04 and SCR-W05 (and FS-4.8, FS-4.9), §15 (events, FS-15.1–15.2), §11 customer rows, docs/BRD.md UC-C04–C08 and §14 delight moments. Read web/src from steps 15–17 and the realtime server from step 13.

Task: build /t/:jobId.

1. Ticket secret handling (FS-4.9): the URL is /t/:jobId#<secret>. On load, read the fragment, move it to sessionStorage/localStorage (try/catch) and replace the URL without the fragment (history.replaceState) so it isn't shared by accident. If no secret is available show "Open this ticket from the phone that sent the files".
2. WebSocket client (src/lib/ws): connect to /ws/jobs/{id}, send {"auth":{"secret"}} first, handle hello/seq/sync.required, detect gaps and refetch GET /jobs/{id}, heartbeat, reconnect with backoff 1–30 s, fall back to polling every 10 s after 3 failures (FS-15.2). Pause when the page is hidden for > 5 min and resume on visibility.
3. SCR-W04 layout: TokenDisplay (72 px, lane name under it), status stepper In line → Printing → Ready → Collected with icon + text, position ("3 jobs ahead of you"), ready-by, price breakdown collapsed, Edit / Add files / Cancel only while queued and unclaimed (reuse step 17 components), "Notify me" (web push opt-in — show only if supported; wiring in step 34, leave a disabled hook now).
4. Ready moment: full-screen green state, "Ready — show A-07 at the counter", vibrate [300,150,300] where supported, one short chime (preloaded small audio, played only after a user gesture has unlocked audio — prompt "Tap to turn on sound" on first visit), aria-live assertive announcement, keep screen awake with the Wake Lock API while Ready (release on collected).
5. "Ask customer" event: banner with the shop's reason and a single action (re-upload file / come to counter). Re-upload replaces the file and keeps the token.
6. Cancelled by shop: reason text; for walk-in no refund text.
7. SCR-W05 receipt: "Collected at 6:47 pm", countdown to deletion (from server deleteAt), then "3 files deleted at 6:57 pm" with shield icon, job ID, and "Share receipt" (navigator.share with a text summary, fallback copy to clipboard). No file names after deletion.
8. Ticket restore: the drop page banner (step 17) links here; also keep a tiny "My tickets today" list in localStorage for the Home screen (step 19).
9. Tests: Vitest for the ws client (gap → refetch, reconnect, polling fallback) with a mock socket; Playwright with MSW websocket mocking: queued → claimed → ready (assert green + aria-live text) → collected → deleted receipt; fragment removed from URL; cancel before claim works, after claim hidden.

Rules: the page must work if WebSockets are blocked (polling fallback). Show the ws client API first.
```

## Acceptance

- [ ] Status changes appear within 1 s of the staff action (AT-10).
- [ ] Ready screen vibrates, chimes (after unlock) and is announced to screen readers.
- [ ] Receipt shows deletion time and no file names (AT-11 UI part).
- [ ] Secret is removed from the address bar after load.

## Verify

```bash
cd web && pnpm test && pnpm exec playwright test ticket
```

## Commit

`feat(customer): live ticket with WebSocket fallback, ready moment, ask-customer and deletion receipt`
