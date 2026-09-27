# 34 — Web push and minimal shop mode

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 4 Print nearby | R1b | M | 31 | FSD §11 (all rows, FS-11.1–11.5), §5.0 PWA push and known limits, SCR-M01, FR-6.2, FR-7.2–7.3, EX-S06 |

## Goal

Customers get "accepted", "ready" and "refunded" notifications even with the page closed (Android always; iPhone when installed), and shop phones get loud remote-order alerts through the installed PWA's minimal shop mode.

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md §11 (event table and FS-11.1–11.5), §5.0 (PWA technical spec push row, known limits), SCR-M01, docs/BRD.md FR-6.2, FR-7.2–7.3 and EX-S06. Read the outbox publisher (step 12) and service worker config (step 15).

Task: implement Web Push and shop mode.

1. Backend: VAPID keys (CD_VAPID_PUBLIC, CD_VAPID_PRIVATE, CD_VAPID_SUBJECT); github.com/SherClockHolmes/webpush-go. cd_push_subscriptions (id, user_id or ticket job_id, device_id, endpoint, p256dh, auth, created_at, last_success_at). POST /me/devices (customer) and POST /shop/devices/push (staff) store subscriptions; guest tickets may subscribe per job via POST /jobs/{id}/push (X-Ticket-Secret) — deleted when the job closes.
2. Push publisher: outbox topic push:* → send with TTL (ready: 1 h, remote order: 5 min), urgency high for remote.pending and job ready; 404/410 → delete subscription (FS-11.4); record latency metric (FS-11.1 p95 5 s).
3. Payloads: no file names or personal data beyond first name (FS-11.2); localized server-side using the recipient's language (FS-11.5) — share the message catalogue by exporting web/src/lib/i18n/locales/*/push.json into the API at build time.
4. Reminders (R1b): uncollected at +2 h and next opening; shop closing in 30 min with open remote jobs; quiet hours 21:00–08:00 except ready and remote-order alerts (FS-11.3).
5. Service worker: push event → showNotification with actions (Open, Directions for ready); notificationclick → focus or open /t/:jobId or /shop; for shop remote orders use requireInteraction and vibrate pattern; repeat alert at 2 and 4 minutes via server re-sends until accepted/rejected (EX-S06).
6. Permission UX: ask only after the first remote order is paid or when the customer taps "Notify me" on a ticket (never on first visit). iOS: show "Install the app to get notifications" when not standalone.
7. SCR-M01 minimal shop mode at /shop/mode (installable, add a second manifest shortcut "Counter Drop Shop"): Online/Offline toggle, pending remote orders full-screen with Accept/Reject and countdown, Ready list with tap → Collected (with code entry/scan), search by code. Keep the screen awake while online (Wake Lock).
8. Tests: webpush-go with a local test push service mock; subscription cleanup on 410; quiet hours; Playwright checks for permission prompt timing and notification click routing (Chromium supports push in tests with a mocked service).

Rules: push is best-effort; the WebSocket and board remain the source of truth. Show the subscription model and SW event handling first.
```

## Acceptance

- [ ] Android: ready and accepted notifications arrive with the browser closed.
- [ ] iPhone (installed PWA, iOS 16.4+): notifications arrive; not installed: install hint shown.
- [ ] Shop phone receives repeating remote-order alerts until actioned.

## Commit

`feat(push): Web Push for customers and shops, reminders, and minimal installable shop mode`
