# 20 — Shop signup wizard and staff login

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 2 Web PWA | R1a | M | 07, 16 | FSD SCR-S01, SCR-S02, §2 roles, FR-9.1, SO-9, BRD §18 onboarding checklist |

## Goal

A shop owner goes from nothing to a live drop page in under 10 minutes without help, and staff sign in on their own devices with a PIN.

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md §6 SCR-S01 (wizard table) and SCR-S02, §2 roles/auth, §16 VAL-SH1–SH3 and VAL-P1–P2, docs/BRD.md FR-9.1, SO-9 and §18 onboarding checklist. Read the auth API from step 07 in contracts/openapi.yaml and web/src.

Task: build /shop/signup and /shop/login in the shop surface (light theme default, dark option — design/DESIGN.md).

1. Wizard at /shop/signup with a progress header "Step n of 6" and resumable state (sessionStorage):
   1 Phone: +91 input, Send OTP, 6-digit OTP input with WebOTP (navigator.credentials.get({otp:{transport:["sms"]}}) on Android Chrome) and autocomplete="one-time-code"; resend after 30 s.
   2 Shop: name (3–60), link name (auto-slug from name, editable, live availability check with debounce, suggestions on slug_taken), category (Xerox & print, Cyber café, Stationery, College print room, Other).
   3 Location: map with a draggable pin (Leaflet + OpenStreetMap tiles for now; provider decision OI-4 later), "Use my location", address line (reverse-geocode via Nominatim with a polite rate limit, editable), landmark, nearest station (select from a Mumbai stations list JSON).
   4 Hours: per weekday open/close with "Same every day" shortcut and closed toggles; VAL-SH2 inline.
   5 Prices: template "Typical Mumbai rates" prefilled (use the seed values) or custom — reuse the price editor from step 22 if it exists, otherwise a compact version with B/W A4 required; live preview sentence ("10 pages, B/W, both sides, 2 copies = ₹30").
   6 Done: set a 4-digit PIN (twice), then success screen with three actions: "Send a test job" (opens /s/{slug} in a new tab with a sample PDF), "Download QR kit" (step 22 link), "Invite staff". Show the onboarding checklist with progress.
   Note on screen: the shop is live for walk-in immediately; appearing on Print nearby needs approval (R1b).
2. /shop/login: if this device has a remembered device ID and member list, show name tiles → PIN pad (big digits, shuffle off, haptic on tap) → session. Otherwise phone OTP → choose shop (if member of several) → set PIN for this device. Lockout message after 5 failures with the remaining minutes.
3. Session storage: access token in memory + localStorage (try/catch) scoped per shop; automatic redirect to /shop/login on 401; "Switch staff" button in the top bar keeps the device and returns to name tiles.
4. Staff invite acceptance: /shop/join?shop={id} → OTP → PIN.
5. Tests: wizard validation per step, slug availability debounce, PIN pad input and lockout UI, Playwright end-to-end against the real API with CD_ENV=dev ConsoleSender (read the OTP from a test-only endpoint GET /dev/otp/last guarded by CD_ENV=dev — add it in the API if missing), measuring total time (target < 10 min, realistic typing).

Rules: all copy via i18n; big touch targets; never store PINs or OTPs in storage. Show the wizard state model first.
```

## Acceptance

- [ ] A new owner completes signup and sees their live drop page in under 10 minutes.
- [ ] Staff sign in with name tile + PIN in under 10 seconds on a known device.
- [ ] 401 anywhere returns to login without losing the device.

## Verify

```bash
cd web && pnpm test && pnpm exec playwright test shop-signup
```

## Commit

`feat(shop): self-serve signup wizard, staff PIN login and invite flow`
