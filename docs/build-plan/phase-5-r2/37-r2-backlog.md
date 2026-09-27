# 37 — R2 backlog: print agent, TV, billing, reports, Aadhaar masking

Start R2 only after R1b is live and Gate 2 is met. Each item below becomes its own step file when you start it; the prompts here are starting points.

| # | Item | Spec refs | Size |
| --- | --- | --- | --- |
| 37a | Windows print agent | FSD API-50, FR-6.3, UC-S21, SO-8 | L |
| 37b | TV token display | SCR-S16, UC-S22, FS-15.4 | S |
| 37c | Full shop mode in the PWA | FR-6.3 | M |
| 37d | Plans, billing and invoices | BRD §17, UC-S28, UC-A10 | L |
| 37e | Owner reports | UC-S27, BO-1, BO-2 | M |
| 37f | Aadhaar detection and masking on device | CT-3, UC-C26, FR-8.6 | L |
| 37g | Malware scan on upload | NFR-14 | S |
| 37h | Spoken token announcements and PA audio | design/later/, UC-S22 | M |

## 37a — Windows print agent

```text
Read docs/build-plan/00-common-context.md, docs/FSD.md API-50, FR-6.3 and agent/README.md. Build agent/ as a Go Windows service (golang.org/x/sys/windows/svc) that pairs with a shop using a one-time code from the dashboard (device token scoped to the shop), polls or subscribes (WebSocket) for claimed jobs assigned to "agent", downloads files via signed URLs to a temp folder, prints with exact settings (copies, duplex, colour, paper, page range) using SumatraPDF command-line printing bundled with the installer (or the Windows print spooler via a PDF-capable printer driver), reports printed/failed, deletes temp files immediately, and marks Ready when configured. Include a tray app for status, printer selection and pause; an MSI installer (WiX) with auto-update; logs without file names. Tests: settings → Sumatra arguments mapping; temp file cleanup on crash; pairing revocation.
```

## 37b — TV token display

```text
Read docs/build-plan/00-common-context.md and FSD SCR-S16, FS-15.4. Build /tv/:slug?key= as a full-screen, read-only page for a TV browser: "Now ready" tokens large, "Printing" tokens smaller, a chime on new ready tokens, no names ever. Owner creates a TV key in settings (revocable). Uses the shop WebSocket with a read-only TV scope that strips names server-side. Auto-reconnect and burn-in protection (subtle position shift every 10 min).
```

## 37c — Full shop mode

```text
Bring the full board (lanes, claim, pause, rush mode, settings) into /shop/mode with a phone-optimised layout, reusing step 21 components. Add gesture shortcuts: swipe right on a card = claim/ready, long-press = actions. Test on a ₹8,000 Android phone.
```

## 37d — Plans and billing

```text
Read BRD §17 and UC-S28, UC-A10. Implement plans Free/Pro/Plus/Business with limits (jobs/month, lanes, staff, features) enforced in the API; Razorpay Subscriptions for monthly/annual billing (confirm API in current docs), GST invoices (Counter Drop's GSTIN), pilot discounts (3 months free, then 50% for 3 months), dunning with grace period (7 days) and downgrade to Free (never block walk-in queues abruptly). Owner billing page and admin plan management.
```

## 37e — Owner reports

```text
Read UC-S27 and BRD §21. Daily and monthly reports per shop: jobs, pages, revenue split (walk-in recorded paid flags vs remote settled), peak hours heatmap, average counter time (claimed→collected), remote acceptance rate, ratings trend; CSV export; a monthly summary push/email ("212 jobs without a single WhatsApp file"). Build from cd_job_events and analytics tables with materialised daily aggregates.
```

## 37f — Aadhaar masking on device

```text
Read CT-3, UC-C26, FR-8.6 and BRD §16. On the ID-card capture (SCR-W07) and on uploaded images/PDF pages, run on-device OCR (Tesseract.js worker or a small ML Kit–equivalent WASM model) to detect 12-digit Aadhaar patterns (with Verhoeff checksum validation) and QR codes on the card; mask the first 8 digits with a solid box before upload; show the masked preview and let the customer confirm. Never send unmasked images to the server. Lazy-load models only in ID-card mode; target < 3 s on a mid-range phone. Legal review of wording before release.
```

## 37g — Malware scan

```text
Scan uploaded files with ClamAV (clamd in a sidecar container) after upload completion and before the job can be submitted; infected → reject with a clear message and delete immediately; scan time budget 3 s with fallback to "scan pending" that blocks printing but not submission. Metrics and alerts for detections.
```

## 37h — Spoken token announcements and PA audio

```text
Read docs/build-plan/00-common-context.md, design/DESIGN.md and the three screens in design/later/ (use them as inspiration only). Add a "Call token" feature for the shop board and TV: pressing a hotkey or the Call button speaks "Token A-07, counter 1" in the shop's language order (e.g. English then Marathi) using the browser Speech Synthesis API first, with pre-recorded number clips as a fallback where Hindi/Marathi voices are missing on the device. Settings per shop: chime style (3 options), language order, repeat rules (walk-in once, remote twice with 60 s gap), output device (default speaker; TV follows its own display). Per-desk key bindings (1–3 call the oldest ready tokens, F1–F3 quick-page presets with editable text). Log announcements without customer names. Drop from the Stitch screens: dB meters, frequency monitor, hardware claims, customer names in spoken text unless the customer opted in. Tests: voice fallback selection, repeat throttling, key binding conflicts with FS-6.3 shortcuts.
```
