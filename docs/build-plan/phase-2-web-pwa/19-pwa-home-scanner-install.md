# 19 — PWA home, in-app scanner and install banner

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 2 Web PWA | R1a | M | 17 | FSD §5.0 (entry rules, SCR-A00, SCR-A17, FS-5.0.1–5.0.6, PWA technical spec), UC-C28, UC-C29 |

## Goal

The installed PWA opens to a simple Home with a big **Scan shop QR** button; the in-app scanner only accepts Counter Drop shop codes and opens the upload-only flow; a non-blocking install banner appears on the drop page and after pickup.

## Prompt

```text
First read docs/build-plan/00-common-context.md and docs/FSD.md §5.0 in full (entry rules table, FS-5.0.1–5.0.6, SCR-A00, SCR-A17, PWA technical spec, known limits), plus docs/BRD.md §8 "App and entry model". Read web/src from steps 15–18.

Task: build Home, Scanner and the install experience.

1. SCR-A00 Home at "/": three large tiles — Scan shop QR (primary, full width), Print nearby (visible but shows "Coming soon" unless feature flag VITE_FEATURE_NEARBY=true), My jobs (today's tickets from localStorage, step 18). Active tickets appear as cards at the top ("A-07 at Imran Xerox · Ready") linking to /t/:jobId. Language switcher in the header.
2. SCR-A17 Scanner at "/scan":
   - getUserMedia({video:{facingMode:"environment"}}), full-screen video with a square viewfinder and torch toggle (ImageCapture/torch constraint where supported).
   - Decode with the native BarcodeDetector when available, else lazy-load the qr-scanner library (worker-based); ~10 fps.
   - Accept only https://cd.in/s/{slug}, https://<our web origin>/s/{slug} and cd:{slug} (configurable host list VITE_SHOP_LINK_HOSTS). Anything else → toast "This isn't a Counter Drop shop code." and keep scanning.
   - On success: vibrate 100 ms, stop the camera, navigate to /s/{slug}.
   - Permission denied or no camera: explanation with steps for Chrome and Safari plus a "Type shop code" input (slug printed under each QR) that navigates to /s/{slug} after validating via API-01.
   - Release the camera on unmount and when the page is hidden.
3. Install banner (src/lib/pwa/install.ts + InstallBanner component):
   - Android/Chrome: capture beforeinstallprompt, preventDefault, keep the event; show a slim banner "Install Counter Drop — scan faster next time" under the header on /s/:slug (SCR-W01) and on the receipt (SCR-W05). Tap → prompt(); record outcome.
   - iOS Safari (not standalone): one-line hint "Tap Share → Add to Home Screen" with the share icon; never a modal.
   - Never show during uploads in progress or on the Ready screen; dismissed → hidden 14 days (localStorage with try/catch); never show when already running standalone (matchMedia('(display-mode: standalone)') or navigator.standalone).
   - Events: install_banner_shown, install_accepted, install_dismissed, app_launched_standalone (FS-5.0.6).
4. Link capture: when installed on Android, /s/{slug} links from the camera app should open in the PWA (manifest handle_links: "preferred" and launch_handler client_mode "focus-existing"); document the Chrome behaviour and test manually.
5. Upload-only guarantee: add a Playwright test that navigates Home → Scan (mock BarcodeDetector) → /s/demo-print and asserts there is no payment, nearby or login element in the flow.
6. Tests: URL parsing unit tests (accepted and rejected codes); scanner component with a mocked BarcodeDetector; install banner visibility rules (standalone, dismissed, uploading); Playwright mobile emulation for Home.

Rules: camera code is lazy-loaded and not part of the /s/:slug budget. Show the scanner state machine (idle → requesting → scanning → found | denied | error) first.
```

## Acceptance

- [ ] Installed PWA opens to Home; Scan → shop QR → upload screen in under 3 s.
- [ ] Non-shop QR codes are rejected with a message.
- [ ] Banner never blocks upload and respects the 14-day dismissal.
- [ ] Works on Android Chrome and iPhone Safari (manual check on both).

## Verify

```bash
cd web && pnpm test && pnpm exec playwright test home scan
```

## Commit

`feat(pwa): home screen, in-app shop QR scanner and non-blocking install banner`
