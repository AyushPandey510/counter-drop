# Counter Drop — Functional Specification Document

> Snapshot of the live doc: https://claude.ai/code/artifact/02e52ebc-bf2c-407d-941e-7b0bae2df18d

Sep 27, 2026 · @Sushil Pandey

## 0. Document control

This FSD turns the Counter Drop BRD v1.0 into buildable behaviour: screens, states, rules, APIs, events and tests. Where the BRD says *what* and *why*, this document says *exactly how the system behaves*.

| Field | Value |
| --- | --- |
| Document | Functional Specification Document (FSD) |
| Version | 1.0 (draft) |
| Based on | Counter Drop BRD v1.0 (27 Sep 2026) |
| Author | Sushil Pandey |
| Audience | Engineering (Go API, React PWA), QA, design, ops |
| Status | For review |
| Codebase | `counter-drop/` monorepo: `api/` (Go), `web/` (React/Vite), `agent/` (Go, Windows), `contracts/openapi.yaml` |

**Conventions**

- IDs: `SCR-` screen, `FS-` functional spec item, `ST-` state, `EV-` event, `API-` endpoint, `VAL-` validation, `MSG-` message, `AT-` acceptance test. Each traces back to a BRD ID (UC, FR, BR, EX, UX, NFR, CT).
- Release tags: **R1a** (walk-in pilot), **R1b** (Print nearby + app), **R2**, **R3**, as in BRD section 6.
- Values marked *(config)* are server-side settings, not hard-coded, so pilot results can change them without a release.
- Money is stored in paise (integer). Times are stored in UTC and shown in IST.

**Differences from the current code (to fix)**

| Area | Current code | This spec |
| --- | --- | --- |
| Job states | `new, claimed, ready, collected, cancelled` | Adds `uploading`, `payment_pending`, `awaiting_shop`, `rejected`, `expired`, `refunded`; renames `new` → `queued` (API keeps `new` as an alias for one version) |
| Undo window | 15 min | 10 min *(config)* |
| Staff routes | No auth | Staff session + shop scope required |
| Shop table | No location, hours, payout fields | Adds geo, hours, remote settings, payout status |
| Tokens | One counter per shop/prefix, no daily reset | Per shop, per lane, per business day |

**v1.1 (27 Sep 2026):** customer app is a PWA, not Flutter. Scanning a shop QR opens an upload-only screen with payment at the counter; UPI prepay exists only in Print nearby (section 5.0).

**v1.2 (27 Sep 2026):** visual design and content rules live in `design/DESIGN.md`, with corrected reference screens in `design/screens/`. The shop board uses four columns with lane filter tabs (SCR-S03). Spoken announcements and PA audio are a later release (build-plan 37h); R1a keeps a simple chime.

## 1. System overview

Counter Drop is one Go API serving five clients over a single versioned contract, with Postgres as the source of truth and files kept in object storage for at most 24 hours.

*Diagram: system context · 5 clients, 6 API modules, 5 dependencies — see the [live doc](https://claude.ai/code/artifact/02e52ebc-bf2c-407d-941e-7b0bae2df18d).*

File bytes never pass through the API: clients upload with presigned URLs and staff read with 5-minute signed URLs. Razorpay is the only two-way dependency, because its webhooks drive payment and refund states.

**Components**

| Component | Tech | Responsibility | Release |
| --- | --- | --- | --- |
| Customer web | React + Vite, PWA | Walk-in drop page, ticket page, Print nearby on the web | R1a / R1b |
| Installed PWA | React PWA (same codebase) | Customer mode (nearby, order, pay, history) and shop mode | R1b |
| Shop dashboard | React (same web app, `/shop`) | Queue, lanes, actions, settings, reports | R1a |
| TV display | React (`/tv/<shop>`) | Ready and in-progress tokens, full screen | R2 |
| Print agent | Go, Windows service | Pulls claimed jobs, prints with settings, reports result | R2 |
| Admin console | React (`/admin`) | Shops, KYC, orders, refunds, reconciliation, deletion health | R1a / R1b |
| API | Go, `net/http`, sqlc | All business logic and state | R1a |
| Database | Postgres 16 + PostGIS | Shops, jobs, files, tokens, payments, ledger, audit | R1a |
| Object storage | Cloudflare R2 (S3 API) | Uploaded files; lifecycle rule deletes anything older than 26 h as a backstop | R1a |
| Workers | Go goroutines in the API (`internal/tasks`) | Deletion, remote-order timeouts, reconciliation, auto-offline | R1a |

## 2. Actors, roles and permissions

There are seven roles; every API call is checked against this matrix on the server, and no role can read file content outside its own shop.

**Roles and authentication**

| Role | Who | Authentication | Session |
| --- | --- | --- | --- |
| Guest customer | Walk-in customer | None; job access by `ticket secret` (128-bit, in the ticket URL) | Ticket valid until job closed + 24 h |
| App customer | Signed-in customer (app or web) | Phone OTP | Refresh token 90 days, access token 15 min |
| Shop owner | Registered owner | Phone OTP + 4-digit PIN per device | 30 days per device, revocable |
| Shop staff | Staff added by owner | Owner-issued invite; PIN per device | 12 h, re-enter PIN each day |
| Print agent | Windows service | Device token created by owner, scoped to one shop | Until revoked |
| Platform admin | Ops, support, finance | Google Workspace SSO + TOTP | 8 h |
| Org admin (R3) | Business tier | SSO (Google / Microsoft) | Per org policy |

**Permission matrix** (✓ allowed · own = only own records · — not allowed)

| Action | Guest | App customer | Staff | Owner | Agent | Admin |
| --- | --- | --- | --- | --- | --- | --- |
| Create walk-in job | ✓ | ✓ | — | — | — | — |
| Create remote order and pay | — | ✓ | — | — | — | — |
| View / edit / cancel own job | own | own | — | — | — | — |
| View shop queue | — | — | ✓ | ✓ | own jobs | metadata |
| Open file (signed URL) | own (preview) | own (preview) | ✓ | ✓ | claimed jobs | — |
| Claim / release / ready / collected | — | — | ✓ | ✓ | ready only | — |
| Cancel job, price override | — | — | ✓ | ✓ | — | — |
| Accept / reject remote order | — | — | ✓ | ✓ | — | — |
| Pause intake, go online/offline | — | — | ✓ | ✓ | — | — |
| Edit prices, hours, lanes | — | — | — | ✓ | — | — |
| Manage staff and devices | — | — | — | ✓ | — | — |
| View settlements, reports | — | — | — | ✓ | — | ✓ |
| Approve / suspend shop | — | — | — | — | — | ✓ |
| Manual refund | — | — | — | request | — | ✓ (within policy) |
| Export audit trail | — | — | — | own shop | — | ✓ |

**Rules**

- FS-2.1 Every shop-scoped query filters by `shop_id` from the session, never from the request body.
- FS-2.2 Admin endpoints never return `object_key` or a file URL.
- FS-2.3 Each allowed action writes an audit entry: actor type, actor ID, shop, job, action, before/after state, IP, device, time.
- FS-2.4 Removing a staff member revokes all their device sessions within 5 seconds (session check on each request).

## 3. Screen inventory

The product has 42 screens across four surfaces; each is specified in sections 4–7 under its SCR ID.

| ID | Screen | Surface | Route / location | Role | Release |
| --- | --- | --- | --- | --- | --- |
| SCR-W01 | Shop landing (drop page) | Customer web | `/s/{slug}` | Guest | R1a |
| SCR-W02 | File settings | Customer web | `/s/{slug}` step 2 | Guest | R1a |
| SCR-W03 | Review and send | Customer web | `/s/{slug}` step 3 | Guest | R1a |
| SCR-W04 | Ticket (live status) | Customer web | `/t/{jobId}#{secret}` | Guest | R1a |
| SCR-W05 | Deletion receipt | Customer web | `/t/{jobId}` final state | Guest | R1a |
| SCR-W06 | Shop closed / paused | Customer web | `/s/{slug}` | Guest | R1a |
| SCR-W07 | ID-card capture | Customer web | `/s/{slug}` modal | Guest | R1a |
| SCR-W08 | Print nearby (web) | Customer web | `/nearby` | App customer | R1b |
| SCR-A01 | Onboarding and language | App | first launch | — | R1b |
| SCR-A02 | Phone OTP sign-in | App | on first order | App customer | R1b |
| SCR-A03 | Home / Nearby list | App | tab: Nearby | App customer | R1b |
| SCR-A04 | Map view | App | Nearby toggle | App customer | R1b |
| SCR-A05 | Filters | App | Nearby sheet | App customer | R1b |
| SCR-A06 | Shop details | App | from list/map | App customer | R1b |
| SCR-A07 | Pick files | App | order step 1 / share sheet | App customer | R1b |
| SCR-A08 | File settings | App | order step 2 | App customer | R1b |
| SCR-A09 | Review and pay | App | order step 3 | App customer | R1b |
| SCR-A10 | Awaiting shop | App | order status | App customer | R1b |
| SCR-A11 | Order status and pickup code | App | order status | App customer | R1b |
| SCR-A12 | Rejected / refund, try another shop | App | order status | App customer | R1b |
| SCR-A13 | Rate and receipt | App | after collected | App customer | R1b |
| SCR-A14 | Orders (history) | App | tab: Orders | App customer | R1b |
| SCR-A15 | Report a problem | App | order menu | App customer | R1b |
| SCR-A16 | Profile and settings | App | tab: Profile | App customer | R1b |
| SCR-S01 | Shop sign-up wizard | Dashboard | `/shop/signup` | Owner | R1a |
| SCR-S02 | Staff sign-in (PIN) | Dashboard / app | `/shop/login` | Staff, owner | R1a |
| SCR-S03 | Queue board | Dashboard | `/shop` | Staff, owner | R1a |
| SCR-S04 | Job detail / print | Dashboard | `/shop/jobs/{id}` panel | Staff, owner | R1a |
| SCR-S05 | Collect (search / scan) | Dashboard | `/shop` search bar | Staff, owner | R1a |
| SCR-S06 | Remote order alert | Dashboard / app | overlay | Staff, owner | R1b |
| SCR-S07 | Rush mode | Dashboard | `/shop?rush=1` | Staff, owner | R1a |
| SCR-S08 | Prices | Dashboard | `/shop/settings/prices` | Owner | R1a |
| SCR-S09 | Shop profile and hours | Dashboard | `/shop/settings/profile` | Owner | R1a |
| SCR-S10 | Lanes | Dashboard | `/shop/settings/lanes` | Owner | R1a |
| SCR-S11 | Staff and devices | Dashboard | `/shop/settings/staff` | Owner | R1a |
| SCR-S12 | QR kit | Dashboard | `/shop/qr` | Owner | R1a |
| SCR-S13 | Remote orders and payouts | Dashboard | `/shop/settings/remote` | Owner | R1b |
| SCR-S14 | Settlements | Dashboard | `/shop/money` | Owner | R1b |
| SCR-S15 | Reports | Dashboard | `/shop/reports` | Owner | R2 |
| SCR-S16 | TV display | TV browser | `/tv/{slug}?key=` | Device key | R2 |
| SCR-M01 | Shop mode (minimal) | App | mode switch | Staff, owner | R1b |
| SCR-X01 | Admin: shops, orders, refunds, reconciliation, deletion health | Admin | `/admin/*` | Admin | R1a / R1b |

**Navigation**

- Customer web is linear: W01 → W02 → W03 → W04 → W05. Back never loses uploaded files.
- App has three tabs: **Nearby** (A03), **Orders** (A14), **Profile** (A16). Sharing a file into the app opens A07 with the best-pick shop preselected.
- Dashboard has a top bar: shop status toggle (Online/Paused/Offline), search/scan, rush mode, settings menu (owner only).

## 4. Customer web — walk-in drop

A walk-in customer goes from QR scan to token in three screens and under 30 seconds, with uploads running in the background from the first tap. Traces to UC-C01–C10, FR-1, UX-C1–C12.

### SCR-W01 Shop landing

| Element | Spec |
| --- | --- |
| Header | Shop name, area, open/closing time ("Open · closes 9:30 pm") |
| Wait chip | "{n} jobs ahead · about {m} min"; hidden when n = 0 ("No wait") |
| Language | EN / हिं / मरा; defaults to browser language, remembered in local storage |
| Primary button | **Choose files** (opens OS picker; `accept=.pdf,.jpg,.jpeg,.png,.heic`; multiple) |
| Secondary | **Copy of ID card** (opens SCR-W07) |
| Footer | "Files are deleted after pickup" with a shield icon; link to privacy notice |

Behaviour

- FS-4.1 On load, `GET /shops/{slug}` returns shop state. If `state = paused` or `closed`, render SCR-W06 instead.
- FS-4.2 If a ticket for this shop exists in local storage and is not closed, show a banner "You have job A-07 in line — View".
- FS-4.3 After files are picked, create the job draft (`POST /shops/{slug}/jobs`, state `uploading`) and start uploads in parallel (max 3 concurrent) using presigned PUT URLs; move to SCR-W02 immediately.

### SCR-W02 File settings

| Field | Type | Default | Rules |
| --- | --- | --- | --- |
| Copies | Stepper | 1 | 1–99 |
| Colour | Toggle B/W · Colour | B/W | Colour hidden if shop has no colour price |
| Sides | Toggle One side · Both sides | One side | Both sides hidden for images |
| Pages | All · Range | All | Range like `1-3,5`; validated against page count |
| Paper | A4 · Legal · A3 | A4 | Only sizes the shop prices |
| Orientation | Auto · Portrait · Landscape | Auto | — |
| Fit | Fit to page · Actual size | Fit to page | Images always fit |
| Add-ons | Checkboxes | none | From shop's add-on list (lamination, spiral, stapling) with price |
| Apply to all | Link | — | Copies settings of this file to every file |

Each file row shows thumbnail, name, upload progress, page count ("12 pages") and the file's subtotal. **Add more files** is available. Remove file = swipe or ✕.

- FS-4.4 Page count comes from the server after upload (`pages_status = counted`); until then the row shows "Counting pages…" and price shows "—".
- FS-4.5 Settings are saved to the draft on every change (debounced 500 ms, `PATCH /jobs/{id}`).

### SCR-W03 Review and send

| Element | Spec |
| --- | --- |
| Summary | "{files} files · {sides} printed sides" |
| Price | Line per file + add-ons + total, in ₹; "Pay at the counter" |
| Ready by | "Ready by about 6:40 pm" from the wait engine |
| Name | Optional text, 1–20 chars, placeholder "Your first name (so we can call you)" |
| Primary | **Send to counter** — disabled until all uploads are complete and all page counts known (or flagged, EX-C05) |

- FS-4.6 Send calls `POST /jobs/{id}/submit`. Server verifies each object exists in storage with the declared size, assigns the token atomically and moves the job to `queued`.
- FS-4.7 Duplicate check: if the same file hash was submitted to the same shop in the last 2 minutes, show MSG-C06 first.

### SCR-W04 Ticket

| Element | Spec |
| --- | --- |
| Token | 72 px, e.g. **A-07**; lane name under it |
| Status | Stepper: In line → Printing → Ready → Collected; colour + icon + text |
| Position | "3 jobs ahead of you" (queued only) |
| Actions | **Edit** and **Cancel** while `queued` and unclaimed; **Add files** while `queued` |
| Ready state | Full-screen green, "Ready — show A-07 at the counter", vibrate 2× 300 ms, chime once |
| Notify me | Opt-in web push button (where supported) |

- FS-4.8 Ticket subscribes to `ws /ws/jobs/{id}?secret=`; falls back to polling every 10 s after 3 failed reconnects.
- FS-4.9 Ticket URL contains the secret in the fragment (`#`) so it is not sent in referrers or server logs.

### SCR-W05 Deletion receipt

Shows "Collected at 6:47 pm", a countdown "Files will be deleted in 9:12", then "3 files deleted at 6:57 pm" with a shield, job ID and a **Share receipt** button. After pickup, a single card: "Next time, order ahead with the Counter Drop app" (R1b).

### SCR-W06 Closed or paused

- Closed: "Opens at 9:00 am" + weekly hours; R1b adds "3 shops open near you" linking to `/nearby`.
- Paused: shop's wait message ("Very busy, back in 10 min") + retry button that re-checks every 30 s.

### SCR-W07 ID-card capture

Camera view with a card-shaped frame. Steps: Front → Back → Preview (both on one A4, B/W default). Auto-crop and perspective correction run on the device; only the composed A4 PDF is uploaded. R2 adds Aadhaar masking on this screen before upload.

## 5. Customer app — Print nearby

The app lets a customer find an open shop, prepay by UPI and walk in only to collect, in under 2 minutes median. Traces to UC-C11–C25, FR-3, FR-5, BR-P, BR-R.

### 5.0 PWA shell, install and in-app scan

One React PWA serves walk-in, Print nearby and shop mode. A scan always leads to the upload-only flow of section 4; the home screen is the only way into Print nearby and prepay.

**Entry rules**

| Entry | Route | Flow | Payment |
| --- | --- | --- | --- |
| Camera scan, app not installed | `/s/{slug}` in the browser | SCR-W01–W05 (upload only) + install banner | At the counter |
| Camera scan, app installed | `/s/{slug}` opened in the PWA (Android link capture) | SCR-W01–W05 | At the counter |
| **Scan** button in the PWA | SCR-A17 → `/s/{slug}` | SCR-W01–W05 | At the counter |
| Open PWA from home screen | `/` → SCR-A00 Home | Scan, Print nearby, My jobs | UPI prepay in Print nearby only |

- FS-5.0.1 The upload-only flow never shows nearby shops, payment, other shops or history. It uses the guest ticket (no login).
- FS-5.0.2 Walk-in jobs created by a scan always have `channel = walkin` and cannot call `POST /jobs/{id}/pay` (API returns `409 pay_not_allowed`).

**SCR-A00 Home (installed PWA)**

Three large tiles: **Scan shop QR** (primary), **Print nearby**, **My jobs**. Active jobs appear as cards on top ("A-07 at Imran Xerox · Ready").

**SCR-A17 In-app scanner**

| Element | Spec |
| --- | --- |
| Camera | Rear camera, full screen, square viewfinder; torch toggle |
| Decoder | `BarcodeDetector` API where available, else the `qr-scanner` library; 10 fps |
| Accepted codes | `https://cd.in/s/{slug}` and `cd:{slug}` only; anything else → "This isn't a Counter Drop shop code." |
| On success | Vibrate 100 ms, open `/s/{slug}` |
| Permission denied | "Allow camera to scan" with steps, plus a **Type shop code** field (the short slug printed under the QR) |

**Install banner**

- FS-5.0.3 Android Chrome: capture `beforeinstallprompt`; show a slim banner **Install Counter Drop** on SCR-W01 under the header and on SCR-W05 after pickup. Tapping calls `prompt()`.
- FS-5.0.4 iPhone Safari: show a one-line hint "Tap Share → Add to Home Screen"; never a modal.
- FS-5.0.5 A dismissed banner stays hidden for 14 days (local storage). Never shown during upload progress or on the Ready screen.
- FS-5.0.6 Metrics: `install_banner_shown`, `install_accepted`, `install_dismissed`, `app_launched_standalone`.

**PWA technical spec**

| Item | Spec |
| --- | --- |
| Manifest | `name` Counter Drop, `display: standalone`, `start_url: /?src=pwa`, `scope: /`, icons 192/512 + maskable |
| Service worker | Workbox: precache app shell (under 300 KB); network-first for API; never caches uploaded files or signed URLs |
| Share target | Manifest `share_target` (POST, multipart, PDF and images) → opens SCR-A07 with files attached (installed, Android) |
| Push | Web Push with VAPID keys; permission asked only after the first remote order or when the customer taps **Notify me** |
| Link capture | `handle_links: preferred` so scanned `/s/{slug}` links open in the installed PWA on Android |
| Offline | Home and My jobs show cached data with an offline banner; upload and pay need network |
| Shop mode | Same PWA at `/shop`, installable on the shop phone; remote-order alert uses web push plus a repeating in-page chime while open |

**Known limits (accepted)**

- iPhone push works only when the PWA is installed (iOS 16.4+); otherwise the ticket page must stay open.
- Web Share Target is Android-only.
- Remote-order alerts on a shop phone are less reliable when the browser is closed; shops keep the dashboard open on a PC or the PWA open during working hours (EX-S06 still applies).

### SCR-A01 Onboarding

Three swipe cards ("Find a shop that's open", "Pay by UPI, skip the line", "Files deleted after pickup") with **Skip**. Language picker on card 1. Location permission is asked on SCR-A03, not here.

### SCR-A02 Phone OTP sign-in

- Asked only when the customer first taps **Pay** (browsing is anonymous).
- Phone field (+91, 10 digits), **Send OTP**; 6-digit OTP with WebOTP auto-fill on Android Chrome and one-time-code autofill on iPhone; resend after 30 s; max 5 sends per phone per hour.
- On success, store refresh token in encrypted storage; ask for first name once (used on pickup).

### SCR-A03 Nearby list

| Element | Spec |
| --- | --- |
| Location bar | "Near Dadar West" with change; GPS or pick area/station (EX-C14) |
| Sort | Best pick (default) · Nearest · Shortest wait · Cheapest · Top rated |
| Filter chip row | Open now (on by default) · Colour · Lamination · Spiral · Wait under 15 min |
| Shop card | Name, distance ("350 m · 5 min walk"), Open · closes 9:30 pm, wait chip, B/W and colour price per side, rating, service badges |
| Card state | Orderable; or greyed with reason: Closed · Paused · Not taking online orders |
| Empty | MSG-C15 with nearest open shop and radius expand to 10 km |

- FS-5.1 List calls `GET /nearby?lat&lng&radius=2000` first, expanding to 5 km then 10 km if fewer than 5 orderable shops.
- FS-5.2 List refreshes wait chips every 30 s while visible (lightweight `GET /nearby/waits?ids=`).

### SCR-A06 Shop details

Photos (optional), full price list, hours per day, services, rating with the last 5 comments, **Directions** (opens Google Maps intent) and primary **Order here**.

### SCR-A07 – A09 Order flow

1. **Pick files** (A07): system picker or share-sheet input; same file types and limits as web (VAL-F1–F4).
2. **Settings** (A08): identical fields to SCR-W02; per-shop limits from BR-R3 enforced.
3. **Review and pay** (A09): lines per file, add-ons, print subtotal, **Convenience fee**, **Total**; ready-by; shop name and distance; **Pay ₹{total} with UPI**.

- FS-5.3 Uploads start on A07 (job draft with `channel = remote`, state `uploading`).
- FS-5.4 **Pay** calls `POST /jobs/{id}/pay`, which re-validates shop state (open, not paused, remote enabled) and price, creates a Razorpay order and returns checkout params. If the price changed since display, show MSG-C18 and the new total before continuing.
- FS-5.5 Payment uses Razorpay Checkout with UPI intent (app chooser for GPay, PhonePe, Paytm, BHIM) and UPI collect as fallback.

### SCR-A10 Awaiting shop

Animated 5:00 countdown ring, "Waiting for {shop} to accept", "You won't be charged if they can't take it." Options: **Cancel** (full refund, BR-P6).

### SCR-A11 Order status and pickup code

| State | Shows |
| --- | --- |
| Accepted / queued | Pickup code (4 digits, 48 px) + QR; "Ready by 6:40 pm"; position |
| Printing | Same + "Being printed now" |
| Ready | Green, "Ready at {shop}, 350 m" + **Directions** |
| Collected | Deletion countdown, then receipt (as SCR-W05) |

### SCR-A12 Rejected or timed out

"{shop} couldn't take this order ({reason}). ₹{total} refunded — ref {refund\_id}." Primary **Send to {next shop}** (next best pick, with price and wait) which creates a new order with the same files and settings and goes straight to A09.

- FS-5.6 Files of the rejected job are not deleted until the customer chooses or 15 min pass, so the re-send needs no re-upload; they are then deleted (BR-D3 with a 15-min grace, *config*).

### SCR-A13 Rate and receipt

1–5 stars, one-tap tags (Fast, Good quality, Friendly, Wrong settings, Long wait), optional comment 200 chars. Receipt: order ID, shop name and GSTIN if set, items, fee, total, payment and refund references.

### SCR-A14 Orders

List of past orders (shop, date, pages, total, status). Tap for detail. **Reorder** opens A07 with shop and settings prefilled; files must be picked again (BR-D7).

### SCR-A15 Report a problem

Reasons: Wrong settings · Missing pages · Poor quality · Charged wrongly · Shop closed · Other; photo optional (deleted after 30 days). Creates a dispute visible to the shop owner (UC-S25) and support; SLA 48 h (BR-P9).

### SCR-A16 Profile

Name, phone (masked), language, notification toggles, favourites, **Delete account** (removes profile and push tokens; payment records retained per BR-D5, explained on screen).

## 6. Shop dashboard and shop mode

Staff run the whole counter with four taps — Claim, Print, Ready, Collected — on a board that stays readable from a metre away. Traces to UC-S01–S30, FR-2, FR-6, FR-9, UX-S1–S8.

### SCR-S01 Sign-up wizard (owner)

| Step | Fields | Rules |
| --- | --- | --- |
| 1 Phone | Mobile number, OTP | One owner account per phone |
| 2 Shop | Shop name (3–60), short name for URL (auto-slug, editable, unique), category | Slug: lowercase, 3–30, `a-z0-9-` |
| 3 Location | Map pin (drag), address line, landmark, nearest station | Pin required; reverse-geocoded address editable |
| 4 Hours | Per weekday open/close, closed days; "Same every day" shortcut | Close > open; overnight not supported in R1 |
| 5 Prices | Template "Typical Mumbai rates" or custom (SCR-S08) | At least B/W A4 single side |
| 6 Done | Test job button, download QR kit, invite staff | — |

Shop goes live for walk-in right after step 6. It appears on Print nearby only after admin approval (UC-A01) and, for ordering, payout KYC (SCR-S13).

### SCR-S02 Staff sign-in

First time on a device: owner or staff enters phone + OTP, then sets a 4-digit PIN. Daily: pick your name tile, enter PIN. 5 wrong PINs lock the device session for 15 minutes.

### SCR-S03 Queue board

| Area | Spec |
| --- | --- |
| Top bar | Status toggle **Online · Paused · Offline**; search/scan box; **Rush mode**; today's count ("86 jobs"); settings (owner) |
| Columns | **Four columns: In line, Printing, Ready and Collected (10-minute undo). Lanes are filter tabs above the board, with a lane chip on every card; Claim oldest works on the selected lane (all lanes when All is selected). On phones the columns become tabs** |
| Job card | Token (28 px), first name, channel badge (**Walk-in** / **Prepaid**), files × pages, settings summary, price, age (turns amber at 10 min, red at 20 min) |
| Lane header | Lane name, count, **Claim next** button |
| Remote strip | Pending remote orders with countdown, above the columns (R1b) |

Behaviour

- FS-6.1 Board state comes from `GET /shop/queue` then live `ws /ws/shop` events; full resync every 60 s and on reconnect.
- FS-6.2 **Claim next** calls `POST /shop/lanes/{lane}/claim-next`; the server returns the claimed job. Tapping a specific card calls `POST /shop/jobs/{id}/claim`; on 409 the card animates away and a toast says "Taken by Counter 2".
- FS-6.3 Keyboard: `N` claim next in the focused lane, `R` ready, `C` collected, `/` focus search, `Esc` close panel.
- FS-6.4 Sounds: new walk-in = soft tick; new remote order = loud chime repeated every 20 s until opened (UX-S4). Browser autoplay unlock on first tap after sign-in.

### SCR-S04 Job detail and print

Side panel with file list (thumbnail, name, pages, per-file settings in bold), **Open** (signed URL in a new tab), **Print all** (R1: opens a merged PDF in the browser print dialog with settings printed at the top of the panel; R2: sends to the print agent), **Ready**, **Release**, **Ask customer** (preset reasons), **Edit** (copies, add-ons, price override with reason), **Cancel** (reason required, confirmation dialog — the only one).

### SCR-S05 Collect

Search accepts token (`A07`, `a-07`), 4-digit code, or first name. Camera scan reads the customer's pickup QR. Result card shows token/code, name, files, pages and **Prepaid** or **Collect ₹{price}**. Buttons: **Collected** and, for walk-in, **Paid cash** / **Paid UPI** (optional, for reports). After Collected, a 10-minute **Undo** chip stays on the card in a "Collected" tray.

### SCR-S06 Remote order alert

Modal (dashboard) or full-screen (shop mode): customer first name, files, pages, settings, print amount, ready-by with **+15 / +30 / +60 min** adjusters, countdown. Buttons **Accept** (green) and **Reject** (reasons: Out of paper, Machine down, Too busy, Can't print this file, Closing soon).

### SCR-S07 Rush mode

Hides settings summaries and the Ready column's history; shows only the next 3 cards per lane with buttons 64 px tall; auto-enabled suggestion when queued > 15 jobs.

### SCR-S08 Prices

| Item | Unit | Example |
| --- | --- | --- |
| B/W A4 one side | per side | ₹2 |
| B/W A4 both sides | per sheet | ₹3 |
| Colour A4 one side | per side | ₹10 |
| Colour A4 both sides | per sheet | ₹18 |
| Legal / A3 | per side, B/W and colour | ₹3 / ₹20 |
| Photo print 4×6 | per photo | ₹15 |
| Add-ons | per job or per item | Lamination ₹20 each, spiral ₹30, stapling ₹0 |
| Minimum charge | per job | ₹5 |

Live preview: "10 pages, B/W, both sides, 2 copies = ₹30".

### SCR-S10 – S14 Settings

- **Lanes** (S10): up to 3 on Pro, 5 on Plus; each with letter, name, routing rule (by colour, by channel, or manual).
- **Staff and devices** (S11): list, invite by phone, remove, revoke device, view last active.
- **QR kit** (S12): A4 counter standee, A3 window poster, small sticker sheet; each with shop QR, short URL and three-language instructions.
- **Remote orders** (S13): toggle, max files/pages, services on Print nearby, payout KYC status and "Continue KYC" (Razorpay hosted onboarding).
- **Settlements** (S14): list of remote orders with print amount, fee, refunds, settlement status and expected date; CSV export.

### SCR-M01 Shop mode in the app (R1b minimal)

Four elements: Online/Offline toggle, pending remote orders (SCR-S06 full screen on push), Ready list (tap → Collected), search by code. Full queue and lanes arrive in R2 (FR-6.3).

## 7. Admin console

The admin console shows metadata only and every action is audited; it has six pages. Traces to UC-A01–A12, FR-10.

| Page | Contents | Actions | Roles |
| --- | --- | --- | --- |
| Shops | Table: name, cluster, status (pending, live, suspended), plan, KYC, remote enabled, jobs 7d, acceptance rate, timeouts 7d, complaints | Approve, suspend (reason), unsuspend, change cluster, impersonate **read-only** view of the board (no file open) | Ops |
| Orders | Search by order ID, token + shop, pickup code, Razorpay payment/refund ID; order timeline of every state change with actor and time | Manual refund (full/partial, reason, within BR-P6–P9), add note | Support |
| Disputes | Open disputes with SLA timer (48 h), shop response, customer reason | Decide: reprint / refund / reject; message both sides | Support |
| Reconciliation | Daily table: payments captured, transfers created/released/reversed, refunds, fees; mismatches list | Mark resolved with note; re-run for a date | Finance |
| Deletion health | Files due now, deleted in last 24 h, failed with error, oldest undeleted file age | Retry failed; force delete | Ops |
| KPIs | Jobs/day, remote share, acceptance rate, median counter time, deletion SLA, paying shops, churn, cost per job | Export CSV | Founder, ops |

- FS-7.1 Admin API never returns `object_key` or signed URLs (FS-2.2); the read-only board view masks names.
- FS-7.2 Manual refunds above ₹500 need a second admin to approve (four-eyes).
- FS-7.3 Every admin action writes an audit entry with the admin's email and reason.

## 8. Job lifecycle

A job has 11 states and 19 allowed transitions; any other transition returns `409 invalid_transition` and changes nothing. The state diagram is in BRD section 11.

**States**

| State | Meaning | Terminal | Visible in queue |
| --- | --- | --- | --- |
| `uploading` | Draft; files uploading, settings editable | No | No |
| `payment_pending` | Remote: Razorpay order created, awaiting capture | No | No |
| `awaiting_shop` | Remote: paid, waiting for accept (5 min) | No | Remote strip |
| `queued` | In a lane, waiting to be claimed | No | Yes |
| `claimed` | A counter is printing it | No | Printing column |
| `ready` | Printed, waiting for pickup | No | Ready column |
| `collected` | Handed over; undo window open | Yes (after undo window) | Collected tray |
| `cancelled` | Cancelled by customer or staff | Yes | No |
| `rejected` | Remote: shop rejected | Yes | No |
| `expired` | Remote: no accept in time, or payment not captured, or draft abandoned | Yes | No |
| `refunded` | Sub-status on cancelled/rejected/expired remote jobs once refund is processed | Yes | No |

**Transitions**

| # | From | Event | Actor | Guard | To | Side effects |
| --- | --- | --- | --- | --- | --- | --- |
| T1 | — | create | Customer | Shop exists; walk-in: shop online and not paused | `uploading` | Presigned URLs issued |
| T2 | `uploading` | submit | Guest/customer | Walk-in; all files uploaded; pages known or flagged; shop not paused | `queued` | Token issued; `EV-job.queued`; wait recalculated |
| T3 | `uploading` | pay | App customer | Remote; shop orderable; price revalidated | `payment_pending` | Razorpay order created |
| T4 | `payment_pending` | payment.captured webhook | System | Amount = order amount | `awaiting_shop` | Route transfer created **on hold**; pickup code reserved; shop alerted; 5-min timer starts |
| T5 | `payment_pending` | payment.failed or 15 min elapsed | System | — | `expired` | Any captured amount refunded; files deleted |
| T6 | `awaiting_shop` | accept | Staff | Within timer | `queued` | Transfer hold released; pickup code sent; ready-by set |
| T7 | `awaiting_shop` | reject | Staff | Reason required | `rejected` → `refunded` | Transfer reversed; full refund incl. fee; files kept 15 min for re-send, then deleted |
| T8 | `awaiting_shop` | timer 5 min | System | — | `expired` → `refunded` | As T7; shop hidden 30 min (BR-R4) |
| T9 | `awaiting_shop` | cancel | Customer | — | `cancelled` → `refunded` | Full refund incl. fee |
| T10 | `queued` | claim | Staff | Not already claimed (atomic) | `claimed` | `claimed_by`, `claimed_at`; `EV-job.claimed` |
| T11 | `claimed` | release | Staff | Same shop | `queued` | Keeps token and original `queued_at` (head of lane) |
| T12 | `claimed` | ready | Staff/agent | — | `ready` | Customer notified (push/WS) |
| T13 | `ready` | collected | Staff | Walk-in: token match; remote: code match | `collected` | Files scheduled for deletion at +10 min |
| T14 | `collected` | undo | Staff | Within 10 min | `ready` | Deletion schedule cleared |
| T15 | `queued` | cancel | Customer | Walk-in or remote before claim | `cancelled` | Walk-in: files deleted now; remote: refund minus fee (BR-P6) |
| T16 | `queued`, `claimed`, `ready` | cancel | Staff | Reason required | `cancelled` | Remote: full refund; files deleted now |
| T17 | `queued` | edit | Customer | Not claimed | `queued` | Price recalculated; remote: price may only go down (refund difference) |
| T18 | `uploading` | abandon | System | No activity for 60 min | `expired` | Files deleted |
| T19 | `queued`, `claimed`, `ready` | shop closing / 24 h | System | — | unchanged | Files deleted (BR-D2); job marked `files_deleted`; still collectable if printed |

**Implementation rules**

- FS-8.1 Transitions run in one database transaction: `UPDATE cd_jobs SET state=$to, ... WHERE id=$id AND state=$from RETURNING *`. Zero rows → 409.
- FS-8.2 Claim-next uses `SELECT id FROM cd_jobs WHERE shop_id=$1 AND lane=$2 AND state='queued' ORDER BY queued_at LIMIT 1 FOR UPDATE SKIP LOCKED`.
- FS-8.3 Every transition inserts a `cd_job_events` row and publishes a realtime event after commit (outbox pattern, so events never go out for rolled-back changes).
- FS-8.4 Timers (T5, T8, T18, collected +10 min) are rows in `cd_scheduled_tasks` polled every 10 s; they survive restarts.
- FS-8.5 The existing `domain.Job.Apply` is extended to this table; `new` is accepted as an alias of `queued` in API input for one version.

## 9. Pricing, token and wait-time engines

Price, token and wait are computed only on the server, from the shop's own settings, so every client shows the same numbers. Traces to BR-P1–P5, BR-T1–T3, FR-3.5.

### 9.1 Pricing

For each file *f* with selected pages *P* (after page range), copies *c* and mode *m* (B/W or colour), paper *s*:

```latex
\text{sides}_f = |P| \qquad \text{sheets}_f = \lceil |P| / 2 \rceil
```

```latex
\text{price}_f = c \times \begin{cases} \text{sides}_f \times r_{m,s,\text{single}} & \text{one side} \\ \text{sheets}_f \times r_{m,s,\text{double}} & \text{both sides} \end{cases}
```

```latex
\text{print total} = \max\Big(\text{min charge},\ \sum_f \text{price}_f + \sum \text{add-ons}\Big)
```

- FS-9.1 If the shop has no double-sided rate, double-sided is charged as 2 × single-sided per sheet.
- FS-9.2 Images: one side each; photo sizes (4×6) use the photo rate.
- FS-9.3 Amounts in paise; round only the final total to the nearest rupee (half up).
- FS-9.4 Remote total = print total + convenience fee, where fee = ₹2 if print total ≤ ₹50, ₹3 if ≤ ₹200, else ₹5 *(config)*; first remote order per customer fee = ₹0 *(config)*.
- FS-9.5 A price quote carries `price_version` (hash of rates + settings). `submit` and `pay` reject a stale version with `409 price_changed` and the new quote.
- FS-9.6 Unknown page count (EX-C05): quote shows `pages_to_confirm = true`; walk-in allowed; remote blocked ("We couldn't count pages in this file").

**Worked example.** 12-page PDF, pages 1–10, B/W, both sides, 2 copies, rate ₹3/sheet; plus one lamination ₹20. Sheets = 5; file price = 2 × 5 × ₹3 = ₹30; total = ₹50. Remote fee ₹2 → customer pays ₹52; shop receives ₹50.

### 9.2 Tokens and pickup codes

- FS-9.7 Business day = shop's opening time to next opening time, in IST.
- FS-9.8 Token = lane letter + counter from `cd_token_counters(shop_id, lane, business_day)`, incremented with `UPDATE … SET last_no = last_no + 1 RETURNING last_no` inside the submit transaction. Format `A-07`; three digits after 99.
- FS-9.9 Pickup code = random 4 digits, unique among the shop's open remote jobs that business day; retried on collision; QR encodes `cd:{shop_slug}:{code}:{job_id_short}`.

### 9.3 Wait time

```latex
\text{wait (min)} = \text{jobs ahead} \times \bar{t}_{\text{job}} \div \text{active counters}
```

- FS-9.10 *t̄* = median minutes from `claimed` to `ready` over the shop's last 7 days (minimum 20 jobs), else a default of 3 min *(config)*, plus 1 min per 20 pages of the job's size.
- FS-9.11 Active counters = distinct staff sessions that claimed a job in the last 30 min (minimum 1).
- FS-9.12 Shown as a range rounded to 5 minutes ("10–15 min"); "No wait" when 0 jobs ahead.
- FS-9.13 Ready-by = now + wait + own job time; remote accept can add +15/30/60 min.
- FS-9.14 Best pick score for Print nearby (weights *config*): 0.35 × wait score + 0.25 × distance score + 0.15 × price score + 0.15 × acceptance rate + 0.10 × rating score, each normalised 0–1 across the result set.

## 10. Payments, settlement and refunds

Only a verified Razorpay webhook moves an order to paid, and the shop's money is released only when it accepts. Traces to FR-4, BR-M1–M6, BR-P6–P9, CT-6.

*Diagram: remote payment sequence · 10 steps, 1 refund branch — see the [live doc](https://claude.ai/code/artifact/02e52ebc-bf2c-407d-941e-7b0bae2df18d).*

The app's own success callback only shows "Confirming payment…"; state changes wait for step 5.

**Specification**

- FS-10.1 One Razorpay order per job; `receipt = job_id`; `notes = {shop_id, job_id}`; amount = remote total in paise.
- FS-10.2 Webhooks handled: `payment.captured`, `payment.failed`, `refund.processed`, `refund.failed`, `transfer.processed`, `transfer.failed`, `settlement.processed`. Signature verified with the webhook secret; unverified requests → 401 and logged.
- FS-10.3 Idempotency: `cd_webhook_events(event_id PK)`; a repeated event ID returns 200 and does nothing.
- FS-10.4 Transfer: on capture, create a Route transfer of the print amount to the shop's linked account with `on_hold = true`; on accept, set `on_hold = false`; on reject/timeout/cancel, reverse the transfer before refunding.
- FS-10.5 Refund amounts by rule: shop reject/timeout/shop cancel = full; customer cancel before accept = full; after accept before claim = total minus fee; after claim = none (support may override).
- FS-10.6 Refunds are created within 60 s of the trigger; `refund.failed` retries 3 times then alerts support.
- FS-10.7 Ledger: each payment, transfer, reversal, refund and fee is one row in `cd_ledger` with direction, amount, reference and job. Sum per job must equal zero after settlement (payment = transfer + fee − refunds).
- FS-10.8 Reconciliation (daily 02:00 IST): compare ledger with Razorpay payments, refunds, transfers and settlements for the previous day; mismatches go to the admin Reconciliation page and alert finance.
- FS-10.9 Double payment for one job (EX-C10): second capture refunded automatically.
- FS-10.10 Razorpay unavailable (EX-P03): `GET /nearby` returns `remote_ordering = false`; app shows a banner and hides **Order here**.
- FS-10.11 Walk-in payments are not processed; staff may record **Paid cash / Paid UPI** as a flag for reports only.

## 11. Notifications

Every state change that matters to a person reaches them within 5 seconds through the cheapest channel that works: WebSocket first, push second, no SMS except OTP. Traces to FR-7, CO-7, BRD BO-8.

| Event | Recipient | Channels | Message (EN; HI/MR from string catalogue) | Release |
| --- | --- | --- | --- | --- |
| Job queued (walk-in) | Customer | WS | "You're in line — A-07. 3 jobs ahead." | R1a |
| Job claimed | Customer | WS, push (remote) | "Your job is being printed." | R1a |
| Job ready | Customer | WS + vibrate/chime, web push (opt-in), app push | "Ready at {shop} — show A-07 / code 4821." | R1a |
| Collected | Customer | WS, app push | "Collected. Your files will be deleted in 10 min." | R1a |
| Files deleted | Customer | WS, in-app receipt (no push) | "{n} files deleted at {time}." | R1a |
| Ask customer | Customer | WS, app push | "{shop}: {reason}. Tap to fix." | R1a |
| Job cancelled by shop | Customer | WS, app push | "{shop} cancelled your job: {reason}." (+ refund line for remote) | R1a |
| New walk-in job | Shop | WS (soft sound) | Card appears | R1a |
| New remote order | Shop | WS (loud chime), push to owner + staff devices, repeat at 2 and 4 min | "New order ₹{amount} — accept within 5 min." | R1b |
| Remote accepted | Customer | App push | "{shop} accepted. Pickup code 4821, ready by 6:40 pm." | R1b |
| Remote rejected / timed out | Customer | App push | "{shop} couldn't take it. ₹{total} refunded. Try {next shop}?" | R1b |
| Refund processed | Customer | App push | "Refund of ₹{amount} sent (ref {id})." | R1b |
| Uncollected reminder | Customer | App push at +2 h and next opening | "Your prints are waiting at {shop}." | R1b |
| Shop closing soon with open remote jobs | Customer | App push 30 min before close | "{shop} closes at 9:30 pm." | R1b |
| Timeout nudge | Owner | App push, dashboard banner | "You missed 3 orders this week. Pause online orders when busy?" | R1b |
| Dispute opened | Owner | App push, dashboard banner | "A customer reported a problem with order {id}." | R1b |
| Settlement done | Owner | Dashboard, weekly summary | "₹{amount} settled to your bank." | R1b |

**Rules**

- FS-11.1 Push is sent from the outbox after commit; target p95 5 s from state change to FCM accept.
- FS-11.2 Push payloads carry no file names or personal data beyond first name; tapping opens the order.
- FS-11.3 Quiet hours do not apply to Ready and remote-order alerts; reminders respect 9 pm–8 am quiet hours.
- FS-11.4 Stale FCM tokens (unregistered) are deleted on first failure.
- FS-11.5 Messages use the recipient's language setting; placeholders are filled server-side.

## 12. Deletion and retention engine

A worker deletes every file on schedule, proves it with a receipt, and a storage lifecycle rule deletes anything the worker misses after 26 hours. Traces to BR-D1–D8, FR-8, CT-1, NFR-15.

**When a file gets a `delete_after`**

| Trigger | `delete_after` | Also clears |
| --- | --- | --- |
| Job collected (T13) | collected\_at + 10 min *(config)* | Customer name (at deletion) |
| Undo (T14) | cleared (null) | — |
| Cancelled walk-in, expired draft (T15, T18) | now | Customer name |
| Rejected / expired remote (T7, T8) | now + 15 min (re-send grace) | Customer name |
| Upload for any job | min(shop closing time today, uploaded\_at + 24 h) | — |

The earliest applicable time wins. `delete_status` moves `active → pending → deleted` (or `failed`).

**Worker algorithm (every 30 s)**

1. Select up to 200 files `WHERE delete_status IN ('active','pending','failed') AND delete_after <= now() ORDER BY delete_after FOR UPDATE SKIP LOCKED`.
2. For each: `DeleteObject(object_key)`; a "not found" response counts as success.
3. On success: `delete_status = 'deleted'`, `deleted_at = now()`, `object_key = NULL`; audit entry `file.deleted`.
4. On failure: `delete_attempts += 1`, `delete_status = 'failed'`, retry with backoff (1, 5, 15 min); after 3 failures raise an alert.
5. When all files of a job are deleted: clear `customer_name`, set `files_deleted_at`, emit `EV-job.files_deleted` (drives the receipt).

**Backstops and proof**

- FS-12.1 Bucket lifecycle rule: delete objects older than 26 hours under the `cd/` prefix.
- FS-12.2 Daily check at 03:00 IST: list objects older than 25 hours; any found → critical alert and deletion.
- FS-12.3 Receipt data = job ID, file count, `deleted_at` per file; no file names after deletion (stored as "File 1, File 2").
- FS-12.4 Deletion SLA metric: `deleted_at − delete_after`; target p99.9 under 5 min.
- FS-12.5 The current code sets 15 min and has no worker; update to 10 min and add `internal/tasks/deletion.go`.

**Long-term records**

| Record | Kept | Contents after file deletion |
| --- | --- | --- |
| Job row | Indefinitely (anonymised) | Shop, token, pages, price, times, channel; no name, no file names |
| Payment ledger | 8 years *(confirm with legal)* | Amounts, references, job ID |
| Audit log | 1 year *(confirm)* | Actor, action, IDs, IP, time |
| Customer account | Until deleted by customer | Phone, name, language, favourites |

## 13. Data model

Twenty-two Postgres tables, all prefixed `cd_`, extend the three that exist today (`cd_shops`, `cd_jobs`, `cd_job_files`). IDs are ULIDs (text); money in paise (bigint); times `timestamptz` UTC.

| Table | Key fields | Notes |
| --- | --- | --- |
| `cd_shops` | id, slug (unique), name, status (`pending`, `live`, `suspended`), online\_state (`online`, `paused`, `offline`), pause\_message, location `geography(Point)`, address, landmark, station, cluster\_id, timezone, plan, remote\_enabled, remote\_max\_files, remote\_max\_pages, payout\_account\_id, payout\_status (`none`, `pending`, `active`, `on_hold`), gstin, rating\_avg, rating\_count, acceptance\_rate\_7d, avg\_job\_minutes\_7d | Extends existing table; GIST index on location |
| `cd_shop_hours` | shop\_id, weekday (0–6), opens\_at, closes\_at, closed | One row per weekday; plus `cd_shop_closures(date)` |
| `cd_price_lists` | shop\_id, version, rates JSONB, add\_ons JSONB, min\_charge, updated\_at | `rates` keyed by mode × paper × sides; version drives `price_version` |
| `cd_lanes` | id, shop\_id, letter, name, rule (`bw`, `colour`, `remote`, `manual`), sort | Default lane A created at signup |
| `cd_users` | id, phone (unique), name, language, created\_at, deleted\_at | Customers and shop people share this table |
| `cd_shop_members` | shop\_id, user\_id, role (`owner`, `staff`), status | — |
| `cd_devices` | id, user\_id, shop\_id, kind (`dashboard`, `app`, `agent`, `tv`), pin\_hash, fcm\_token, last\_seen\_at, revoked\_at | Session anchor for staff PIN |
| `cd_jobs` | id, shop\_id, lane\_id, channel (`walkin`, `remote`), customer\_user\_id (nullable), customer\_name, token, pickup\_code, secret\_hash, state, settings JSONB, price\_total, fee, price\_version, pages\_total, pages\_to\_confirm, ready\_by, queued\_at, claimed\_at, claimed\_by, ready\_at, collected\_at, cancelled\_at, cancel\_reason, reject\_reason, files\_deleted\_at, created\_at, updated\_at | Replace plain `secret` with `secret_hash` (SHA-256); unique (shop\_id, business\_day, token) |
| `cd_job_files` | id, job\_id, filename, mime, size\_bytes, sha256, pages, pages\_status, settings JSONB, object\_key, upload\_status, delete\_status, delete\_after, deleted\_at, delete\_attempts | Extends existing table |
| `cd_token_counters` | shop\_id, lane\_id, business\_day, last\_no | PK (shop\_id, lane\_id, business\_day) |
| `cd_job_events` | id, job\_id, from\_state, to\_state, event, actor\_type, actor\_id, reason, at | Timeline for customer, shop and admin |
| `cd_payments` | id, job\_id, razorpay\_order\_id, razorpay\_payment\_id, amount, fee, status, captured\_at | One per job |
| `cd_transfers` / `cd_refunds` | id, payment\_id, razorpay id, amount, status, on\_hold, reason | — |
| `cd_ledger` | id, job\_id, kind (`payment`, `transfer`, `reversal`, `refund`, `fee`), amount (signed), ref, at | Sum per job = 0 when settled |
| `cd_webhook_events` | event\_id PK, type, received\_at, processed\_at | Idempotency |
| `cd_outbox` | id, topic, payload JSONB, created\_at, sent\_at | Realtime + push fan-out after commit |
| `cd_scheduled_tasks` | id, kind, ref\_id, run\_at, done\_at | Timers T5, T8, T18, deletion |
| `cd_ratings` / `cd_disputes` | job\_id, stars, tags, comment / reason, status, decision, decided\_by | — |
| `cd_audit_log` | id, actor\_type, actor\_id, shop\_id, job\_id, action, detail JSONB, ip, device\_id, at | Append-only; no file content |

**Migrations to add (after existing 0001–0003)**

- `0004_shop_profile_geo_hours.sql` — shop fields, hours, closures, PostGIS.
- `0005_lanes_tokens_daily.sql` — lanes, token counter per business day.
- `0006_users_members_devices.sql` — auth tables.
- `0007_job_states_v2.sql` — new job columns, `new → queued` data migration, `secret → secret_hash`.
- `0008_events_outbox_tasks_audit.sql`.
- `0009_payments_ledger.sql` — payments, transfers, refunds, ledger, webhooks.
- `0010_ratings_disputes.sql`.

## 14. API specification

All clients use one REST API under `/api/v1/cd` with a JSON envelope; the table extends the endpoints already in `contracts/openapi.yaml`, which must be updated to match.

**Conventions**

- Envelope: `{"data": …}` on success; `{"error": {"code": "price_changed", "message": "…", "details": {}}}` on failure.
- Auth: `Authorization: Bearer <token>` for app/staff/admin; `X-Ticket-Secret` header for guest job access (replaces `?secret=` query, which leaks to logs).
- Idempotency: all POSTs that create or pay accept `Idempotency-Key`; repeats within 24 h return the first response.
- Pagination: `?cursor=&limit=` (max 100). Rate limits return 429 with `Retry-After`.
- Versioning: breaking changes go to `/api/v2/cd`; additive changes are allowed in v1.

**Public and customer**

| ID | Method and path | Auth | Purpose | Key responses |
| --- | --- | --- | --- | --- |
| API-01 | `GET /shops/{slug}` | none | Shop info, state, prices, wait | 200, 404 |
| API-02 | `POST /shops/{slug}/jobs` | none / customer | Create draft job with file list; returns job ID, secret (once), presigned PUT URLs | 201, 409 `shop_paused`, 422 |
| API-03 | `POST /jobs/{id}/files` | ticket / customer | Add file to draft or queued job | 201, 409 |
| API-04 | `DELETE /jobs/{id}/files/{fileId}` | ticket / customer | Remove file before submit/claim | 204, 409 |
| API-05 | `POST /jobs/{id}/files/{fileId}/complete` | ticket / customer | Confirm upload; server checks object, counts pages | 200 |
| API-06 | `PATCH /jobs/{id}` | ticket / customer | Update settings, name; returns new quote | 200, 409 `claimed` |
| API-07 | `GET /jobs/{id}/quote` | ticket / customer | Price breakdown, `price_version`, ready-by | 200 |
| API-08 | `POST /jobs/{id}/submit` | ticket | Walk-in submit (T2); returns token | 200, 409 `price_changed`, `uploads_incomplete`, `shop_paused` |
| API-09 | `GET /jobs/{id}` | ticket / customer | Job status, position, timeline, receipt | 200, 403 |
| API-10 | `POST /jobs/{id}/cancel` | ticket / customer | T9 / T15 | 200, 409 |
| API-11 | `GET /nearby` | none / customer | `lat,lng,radius,filters,sort` → shops with distance, wait, prices, orderable flag | 200 |
| API-12 | `POST /jobs/{id}/pay` | customer | Revalidate, create Razorpay order (T3) | 200, 409 `price_changed`, `shop_unavailable` |
| API-13 | `POST /jobs/{id}/resend` | customer | Copy a rejected job's files + settings to another shop | 201 |
| API-14 | `POST /jobs/{id}/rating` | customer | Stars, tags, comment | 201 |
| API-15 | `POST /jobs/{id}/disputes` | customer | Report a problem | 201 |
| API-16 | `GET /me/jobs` | customer | Order history | 200 |
| API-17 | `POST /auth/otp` · `POST /auth/verify` · `POST /auth/refresh` | none | Phone OTP sign-in | 200, 429 |
| API-18 | `DELETE /me` | customer | Delete account | 204 |
| API-19 | `POST /me/devices` | customer / staff | Register FCM token | 204 |

**Shop (staff and owner)**

| ID | Method and path | Role | Purpose |
| --- | --- | --- | --- |
| API-30 | `GET /shop/queue` | staff | Snapshot of lanes, remote strip, ready, collected tray (replaces `?shop=` with session scope) |
| API-31 | `POST /shop/lanes/{laneId}/claim-next` | staff | Atomic claim of oldest queued job |
| API-32 | `POST /shop/jobs/{id}/{action}` | staff | `claim`, `release`, `ready`, `collected`, `undo`, `cancel` (reason) — existing route, extended |
| API-33 | `POST /shop/jobs/{id}/accept` · `/reject` | staff | Remote accept (with ready-by offset) or reject (reason) |
| API-34 | `PATCH /shop/jobs/{id}` | staff | Edit copies, add-ons, price override with reason |
| API-35 | `POST /shop/jobs/{id}/ask` | staff | Ask customer (preset reason) |
| API-36 | `GET /shop/jobs/{id}/files/{fileId}/url` | staff / agent | 5-min signed GET URL; audited |
| API-37 | `GET /shop/lookup?q=` | staff | Find by token, code or name |
| API-38 | `PUT /shop/state` | staff | `online`, `paused` (message), `offline` |
| API-39 | `GET/PUT /shop/profile` · `/hours` · `/prices` · `/lanes` | owner | Settings |
| API-40 | `GET/POST/DELETE /shop/staff` · `/devices` | owner | Staff and device management |
| API-41 | `GET /shop/money` | owner | Remote orders, settlements, refunds, fees; CSV |
| API-42 | `POST /shop/payout/onboard` | owner | Start or resume Razorpay linked-account KYC |
| API-43 | `POST /shop/signup` · `POST /shop/login` | none | Owner signup, staff PIN login |
| API-44 | `GET /shop/qr-kit.pdf` | owner | QR kit |

**Agent, admin and webhooks**

| ID | Method and path | Auth | Purpose |
| --- | --- | --- | --- |
| API-50 | `GET /agent/jobs?state=claimed` · `POST /agent/jobs/{id}/printed` | agent token | Print agent pull and report (R2) |
| API-60 | `GET /admin/shops` · `POST /admin/shops/{id}/approve` · `/suspend` | admin | Shop management |
| API-61 | `GET /admin/jobs?q=` · `POST /admin/jobs/{id}/refund` | admin | Order search, manual refund |
| API-62 | `GET /admin/reconciliation?date=` · `GET /admin/deletion-health` · `GET /admin/kpis` | admin | Ops dashboards |
| API-63 | `GET /admin/audit?shop=&job=` | admin | Audit export |
| API-70 | `POST /webhooks/razorpay` | signature | Payment, refund, transfer, settlement events |
| API-00 | `GET /health` | none | Liveness (exists) |

**Example — API-08 submit**

```json
POST /api/v1/cd/jobs/01J9Z…/submit
X-Ticket-Secret: 7f3c…
{ "priceVersion": "p_8a1c", "customerName": "Priya" }

200 OK
{ "data": { "id": "01J9Z…", "state": "queued", "token": "A-07", "lane": "A",
  "position": 3, "readyBy": "2026-10-12T13:10:00Z", "priceTotal": 5000 } }
```

## 15. Real-time events

Two WebSocket channels carry every live update: one per job for customers and one per shop for staff; both deliver within 1 s p95. Traces to FR-2.1, FR-7.1, NFR-5.

| Channel | Path | Auth | Subscribers |
| --- | --- | --- | --- |
| Job | `wss://…/api/v1/cd/ws/jobs/{id}` | first message `{"auth":{"secret":"…"}}` or bearer token | Customer ticket page, app order screen |
| Shop | `wss://…/api/v1/cd/ws/shop` | bearer token (staff session) | Dashboard, shop mode, TV (read-only key) |

**Events**

| Event | Channel | Payload (fields) | Sent when |
| --- | --- | --- | --- |
| `job.updated` | job | state, position, readyBy, token, pickupCode (remote), message | Any transition or position change |
| `job.files_deleted` | job | deletedAt, fileCount | All files deleted |
| `job.ask` | job | reason, action (`reupload`, `contact_counter`) | Staff asks customer |
| `queue.job_added` | shop | job card | T2, T6 |
| `queue.job_changed` | shop | job card | Claim, release, ready, edit, cancel, collected, undo |
| `queue.job_removed` | shop | jobId, reason | Leaves the board |
| `remote.pending` | shop | job card, expiresAt | T4 |
| `remote.resolved` | shop | jobId, outcome (`accepted`, `rejected`, `expired`, `cancelled`) | T6–T9 |
| `shop.state` | shop | onlineState, pauseMessage, wait | State or wait change |
| `sync.required` | both | — | Server restart or missed sequence |

**Rules**

- FS-15.1 Every event has a per-channel `seq`. A client that sees a gap requests a snapshot (`GET /shop/queue` or `GET /jobs/{id}`).
- FS-15.2 Server pings every 25 s; client reconnects with exponential backoff 1–30 s; after 3 failures the customer page polls every 10 s.
- FS-15.3 Events are published from `cd_outbox` after commit; with more than one API instance, fan-out goes through Redis pub/sub.
- FS-15.4 TV channel receives only tokens and states, never names.

## 16. Validation rules and message catalogue

Every validation runs on the server, and clients mirror them for speed; every message says what happened and what to do next. Traces to BRD section 13 (EX-) and UX-C8.

**Validation rules**

| ID | Field / object | Rule | Error code | Message |
| --- | --- | --- | --- | --- |
| VAL-F1 | File type | PDF, JPG, JPEG, PNG, HEIC by MIME sniffing, not extension | `file_type` | MSG-C04 |
| VAL-F2 | File size | ≤ 25 MB per file, ≤ 50 MB per job *(config)* | `file_too_large` | MSG-C02 |
| VAL-F3 | File count | ≤ 20 per job; remote ≤ shop limit | `too_many_files` | "Up to 20 files per job." |
| VAL-F4 | PDF | Not encrypted, parseable; ≤ 500 pages | `pdf_locked`, `pdf_corrupt` | MSG-C03 |
| VAL-S1 | Copies | Integer 1–99 | `copies_range` | "Copies must be 1 to 99." |
| VAL-S2 | Page range | Pattern `^\d+(-\d+)?(,\d+(-\d+)?)*$`, within 1..pages, ascending | `page_range` | "Pages must be like 1-3,5 and within 1–12." |
| VAL-S3 | Mode / paper / sides | Must exist in shop price list | `option_unavailable` | "This shop doesn't offer colour." |
| VAL-N1 | Customer name | Optional; 1–20 chars; letters, spaces, `.'-`; profanity filter | `name_invalid` | "Please use letters only." |
| VAL-P1 | Phone | Indian mobile `^[6-9]\d{9}$` | `phone_invalid` | "Enter a 10-digit mobile number." |
| VAL-P2 | OTP | 6 digits; 5 attempts; valid 5 min | `otp_invalid`, `otp_expired` | "That code didn't match. 3 tries left." |
| VAL-J1 | Submit | All files `uploaded`; shop online and not paused; price\_version current | `uploads_incomplete`, `shop_paused`, `price_changed` | MSG-C01, MSG-C08, MSG-C18 |
| VAL-J2 | Pay | Remote enabled, KYC active, shop open ≥ 30 min after ready-by, pages known | `shop_unavailable`, `pages_unknown` | MSG-C19 |
| VAL-SH1 | Slug | 3–30, `a-z0-9-`, unique, not reserved | `slug_taken` | "That link name is taken. Try {suggestion}." |
| VAL-SH2 | Hours | close > open; at least one open day | `hours_invalid` | "Closing time must be after opening time." |
| VAL-SH3 | Prices | ₹0.50–₹500 per side/sheet; B/W A4 single required | `price_invalid` | "Enter a price between ₹0.50 and ₹500." |
| VAL-SH4 | Price override | Reason required; ≤ 3× original without owner PIN | `override_reason` | "Add a reason for the new price." |

**Customer messages (EN master; Hindi and Marathi in the string catalogue)**

| ID | When | Text |
| --- | --- | --- |
| MSG-C01 | Network drop during upload | "Waiting for network — we'll continue automatically." |
| MSG-C02 | File too large | "This file is {size} MB. The limit is 25 MB — try compressing it." |
| MSG-C03 | Locked or corrupt PDF | "This PDF is locked. Remove the password and upload again." / "We can't open this file. Save it again as PDF." |
| MSG-C04 | Unsupported type | "We can print PDF and photos. Save your document as PDF first." |
| MSG-C05 | Pages unknown | "We'll confirm the price at the counter." |
| MSG-C06 | Duplicate | "You already sent this file ({token}). Send it again?" |
| MSG-C08 | Shop paused | "{shop} is very busy right now. {pause message}" |
| MSG-C09 | Payment confirming | "Confirming your payment… this can take a few seconds." |
| MSG-C10 | Double charge | "You were charged twice. ₹{amount} is on its way back." |
| MSG-C15 | No shops nearby | "No open shops within {radius} km. Nearest: {shop}, {distance} away." |
| MSG-C16 | Refund pending at bank | "Refund sent on {date}. Banks take up to 5 working days." |
| MSG-C18 | Price changed | "The price changed from ₹{old} to ₹{new}. Continue?" |
| MSG-C19 | Shop can't take online orders | "{shop} isn't taking online orders right now. Try {next shop}?" |

**Shop messages**

| ID | When | Text |
| --- | --- | --- |
| MSG-S01 | Claim lost | "Taken by {counter}. Here's the next one." |
| MSG-S02 | Offline | "You're offline — showing the last queue. Online orders are paused." |
| MSG-S03 | Reject done | "Refund sent to the customer. No charge to you." |
| MSG-S04 | Undo available | "Collected by mistake? Undo ({mm:ss})." |
| MSG-S05 | KYC needed | "Finish bank details to take online orders." |

## 17. Security, localisation, accessibility and observability

These specs turn the BRD's non-functional targets (NFR-1–24, CT-1–8) into concrete settings the team can build and test.

**Security**

| ID | Spec |
| --- | --- |
| FS-17.1 | TLS 1.2+ everywhere; HSTS; secure, HttpOnly, SameSite=Lax cookies for the dashboard. |
| FS-17.2 | Ticket secret: 128-bit random, shown once, stored as SHA-256; compared in constant time. |
| FS-17.3 | Storage keys: `cd/{shop_id}/{job_id}/{file_id}` with random IDs; bucket private; presigned PUT valid 15 min, GET valid 5 min, both bound to content length/type. |
| FS-17.4 | Staff PIN stored with Argon2id; 5 failures → 15-min lock; owner can revoke any device. |
| FS-17.5 | OTP: 6 digits, 5-min validity, 5 sends per phone per hour, 20 per IP per hour; DLT-registered template. |
| FS-17.6 | Rate limits: job creation 10/min per IP and device; nearby 60/min; staff actions 120/min per device. |
| FS-17.7 | Uploaded PDFs are never rendered server-side with scripts enabled; page counting uses a sandboxed parser with a 5-s timeout. Malware scan in R2. |
| FS-17.8 | Secrets (DB, Razorpay, FCM) in a managed secret store; rotated yearly and on staff change. |
| FS-17.9 | Content-Security-Policy on web; no third-party scripts on the drop page except Razorpay Checkout on pay screens. |
| FS-17.10 | Admin access via SSO + TOTP; IP allowlist optional; all admin actions audited. |

**Localisation**

- FS-17.11 Languages: `en`, `hi`, `mr`; strings in ICU MessageFormat, one catalogue shared by web and app (one React codebase).
- FS-17.12 Fallback order: user choice → device language → `en`. Numbers use Western digits; currency `₹` with Indian grouping (1,00,000).
- FS-17.13 Dates and times in IST, 12-hour clock ("6:40 pm").
- FS-17.14 Fonts: Noto Sans + Noto Sans Devanagari, subset and self-hosted for speed.

**Accessibility**

- FS-17.15 WCAG 2.1 AA: contrast 4.5:1, touch targets 48 px, visible focus, labels on every control.
- FS-17.16 Status never relies on colour alone (icon + text). Ready screen announced via `aria-live="assertive"`.
- FS-17.17 Supports text scaling to 200% without horizontal scroll on 360 px screens.

**Observability**

| Signal | Spec |
| --- | --- |
| Logs | JSON structured logs with request ID, shop ID, job ID; never file names, phone numbers or secrets |
| Metrics | Request rate/latency/errors per route; WS connections; queue depth per shop; deletion lag; webhook lag; push latency; payment success rate |
| Traces | OpenTelemetry on API; sample 10%, 100% for errors |
| SLO alerts | API 5xx > 1% for 5 min; deletion lag p99 > 5 min; any file older than 25 h; webhook backlog > 50; reconciliation mismatch |
| Status page | Public page for shops showing API, payments and push status |

## 18. Analytics events

Twenty product events feed every BRD metric (section 21); they carry IDs and numbers only, never names, phone numbers or file names. Server events come from `cd_job_events`; client events go to a self-hosted analytics endpoint (no third-party trackers on the drop page).

| Event | Source | Properties | Feeds metric |
| --- | --- | --- | --- |
| `drop_page_viewed` | Web | shop\_id, lang, returning | Scan-to-token, funnel |
| `files_picked` | Web, app | job\_id, count, total\_mb | Submit drop-off |
| `upload_completed` | Web, app | job\_id, ms, retries | Performance |
| `settings_changed` | Web, app | job\_id, field | UX tuning |
| `job_submitted` | Server | job\_id, shop\_id, channel, pages, price, ms\_since\_view | Scan-to-token, jobs |
| `job_claimed` / `job_ready` / `job_collected` | Server | job\_id, shop\_id, lane, ms\_in\_state, counter\_id | Counter time, jobs per peak hour |
| `job_cancelled` | Server | job\_id, by, reason, state | Quality |
| `nearby_searched` | App, web | radius, results, orderable, sort, filters | Search conversion, density |
| `shop_viewed` | App, web | shop\_id, rank | Ranking |
| `pay_started` / `pay_succeeded` / `pay_failed` | App, server | job\_id, amount, method, error | Payment success |
| `remote_accepted` / `remote_rejected` / `remote_expired` | Server | job\_id, shop\_id, secs\_to\_decision, reason | Acceptance rate |
| `refund_processed` | Server | job\_id, amount, reason, secs\_to\_refund | Refund rate |
| `files_deleted` | Server | job\_id, lag\_secs | Deletion SLA |
| `rating_submitted` | App, web | job\_id, stars, tags | NPS, quality |
| `nps_answered` | App, dashboard | score, role | NPS |
| `resend_to_other_shop` | App | from\_shop, to\_shop | Recovery rate |
| `shop_state_changed` | Server | shop\_id, state, by | Uptime, rush behaviour |
| `rush_mode_toggled` | Dashboard | shop\_id, on, queued | UX |

- FS-18.1 Counter time proxy = `job_claimed` → `job_collected` per job; the pilot also records shadowed before/after times (BRD section 21).
- FS-18.2 Peak hours = 8–11 am and 5–8 pm IST *(config)*.

## 19. Acceptance test scenarios

These 24 scenarios are the release gate for R1a and R1b; each maps to requirements and runs as an automated API test plus a manual device check.

| ID | Scenario | Given / When / Then (short) | Traces | Release |
| --- | --- | --- | --- | --- |
| AT-01 | Walk-in happy path | Shop online → scan, pick 2 PDFs, send → token A-01 shown in < 30 s; card appears on dashboard in < 1 s | UC-C01, FR-1, FR-2.1 | R1a |
| AT-02 | Price correctness | 12-page PDF, pages 1–10, both sides, 2 copies, ₹3/sheet + lamination ₹20 → total ₹50 on web, dashboard and API | FS-9.1–9.5 | R1a |
| AT-03 | Token daily reset | Last token yesterday A-58 → first job today is A-01 | FS-9.7–9.8 | R1a |
| AT-04 | Concurrent claim | 3 counters call claim-next 500 times on 500 jobs → 500 distinct claims, zero duplicates, zero errors | FR-2.2, NFR-7 | R1a |
| AT-05 | Release | Claimed job released → returns to head of lane with same token | T11 | R1a |
| AT-06 | Edit before claim | Customer changes copies 1 → 3 → price updates on both screens; after claim, Edit disabled | T17, BR-Q3 | R1a |
| AT-07 | Upload resume | Cut network at 50% upload for 20 s → upload resumes, job submits, no duplicate files | EX-C01 | R1a |
| AT-08 | Locked PDF | Upload encrypted PDF → MSG-C03, job cannot be sent | VAL-F4 | R1a |
| AT-09 | Pause intake | Staff pauses → new scan shows SCR-W06 paused; existing jobs continue | FR-2.5, BR-Q5 | R1a |
| AT-10 | Ready notification | Staff taps Ready → ticket page turns green, vibrates, within 1 s | FR-7.1 | R1a |
| AT-11 | Collected and deletion | Collected at 18:00 → at 18:10 files gone from storage, receipt shows count and time, name cleared | BR-D1, FS-12 | R1a |
| AT-12 | Undo | Collected by mistake → Undo at +4 min → back to Ready, deletion cancelled | T14 | R1a |
| AT-13 | Closing deletion | Job queued, never collected, shop closes 21:30 → files deleted by 21:35 | BR-D2 | R1a |
| AT-14 | Access control | Staff of shop A requests job of shop B → 404; admin requests file URL → 403 | FS-2.1–2.2 | R1a |
| AT-15 | Remote happy path | Signed-in customer, orderable shop → pay UPI → webhook → shop accepts → pickup code push → ready → collected with code; ledger sums to zero after settlement | UC-C13, FR-4 | R1b |
| AT-16 | Callback without webhook | Client reports success but no webhook → order stays "Confirming"; no shop alert; auto-refund/expire at 15 min if not captured | FR-4.2, T5 | R1b |
| AT-17 | Reject | Shop rejects "Out of paper" → full refund created < 60 s; customer sees reason and next-shop suggestion | T7, FS-10.5 | R1b |
| AT-18 | Timeout | No response for 5 min → expired, refunded, shop hidden 30 min | T8, BR-R4 | R1b |
| AT-19 | Resend | After reject, tap "Send to Sai Print" → new order with same files, no re-upload, pays again | API-13, FS-5.6 | R1b |
| AT-20 | Duplicate webhook | Same `payment.captured` delivered 3 times → one state change, one transfer | FS-10.3 | R1b |
| AT-21 | Customer cancel after accept | Cancel before claim → refund = total − fee | BR-P6 | R1b |
| AT-22 | Nearby search | 30 shops seeded around Dadar → results < 500 ms p95; closed/paused shown greyed; best pick order matches formula | FR-3.1–3.5 | R1b |
| AT-23 | Share sheet | Share a PDF from WhatsApp to the app → order screen opens with file attached and best-pick shop selected | FR-5.3 | R1b |
| AT-24 | Languages | Switch to Marathi → every customer screen and push shows Marathi strings; no missing keys | FS-17.11 | R1a / R1b |

**Example in Gherkin (AT-17)**

```text
Feature: Remote order rejected by shop
  Scenario: Shop is out of paper
    Given a customer paid ₹52 for a remote order at "Imran Xerox"
    And the order is awaiting the shop
    When staff reject it with reason "Out of paper"
    Then the Route transfer is reversed
    And a refund of ₹52 is created within 60 seconds
    And the customer sees "Imran Xerox couldn't take this order (Out of paper). ₹52 refunded"
    And the app offers the next best shop with files attached
```

## 20. Traceability and open items

Every BRD functional area maps to FSD sections and acceptance tests; five items need a decision before R1b build starts.

**Traceability matrix**

| BRD requirement | FSD sections | Screens | APIs | Tests |
| --- | --- | --- | --- | --- |
| FR-1 Walk-in drop | 4, 8, 9 | W01–W07 | API-01–10 | AT-01–03, 06–08 |
| FR-2 Queue and dashboard | 6, 8, 15 | S02–S07 | API-30–38 | AT-04, 05, 09, 12 |
| FR-3 Print nearby | 5, 9.3 | A03–A12, W08 | API-11–13 | AT-19, 22 |
| FR-4 Payments | 10 | A09–A12, S14 | API-12, 41, 42, 70 | AT-15–18, 20, 21 |
| FR-5 Customer app | 5, 11 | A01–A16 | API-14–19 | AT-23 |
| FR-6 Shop mode | 6 (M01) | M01, S06 | API-30–38 | AT-15 |
| FR-7 Notifications | 11, 15 | W04, A10–A11, S06 | WS | AT-10 |
| FR-8 Privacy and deletion | 12, 17 | W05 | — | AT-11, 13, 14 |
| FR-9 Shop onboarding | 6 (S01, S08–S13) | S01, S08–S13 | API-39–44 | — (usability test) |
| FR-10 Admin console | 7 | X01 | API-60–63 | AT-14 |
| BR-P Pricing | 9.1 | W02–W03, A08–A09, S08 | API-06–07 | AT-02, 21 |
| BR-T Tokens | 9.2 | W04, S05 | API-08 | AT-03 |
| BR-D Retention | 12, 13 | W05 | — | AT-11, 13 |
| NFR / CT | 15, 17, 18 | — | — | AT-04, 14, 22 |

**Open items**

- [ ] OI-1 Confirm R1a/R1b split (BRD D-1); this FSD assumes it.
- [ ] OI-2 Confirm Razorpay Route supports `on_hold` transfers for this account type, and the fee needed to cover UPI + Route charges (sets FS-9.4).
- [ ] OI-3 Legal review of retention periods (ledger 8 years, audit 1 year) and the shop DPA.
- [ ] OI-4 Choose maps provider (Google Maps Platform vs open alternatives) for geocoding and directions; cost check against BO-8.
- [ ] OI-5 Decide PDF page-count library and sandbox approach for the Go API (FS-17.7).

**Next steps for the codebase**

- [ ] Update `contracts/openapi.yaml` to section 14.
- [ ] Add migrations 0004–0010 (section 13).
- [ ] Extend `domain/job.go` to the section 8 transition table; change undo window to 10 min.
- [ ] Add staff auth middleware before any pilot.
- [ ] Build `internal/tasks/deletion.go` (section 12) and the outbox publisher.
- [ ] Rewrite `docs/ROADMAP.md` to the R1a/R1b/R2/R3 plan.
