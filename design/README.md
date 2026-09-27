# Counter Drop — Design

Visual reference for the PWA. Built from the Stitch export (27 Sep 2026) and corrected to follow `docs/BRD.md` and `docs/FSD.md`.

| Folder | What it is |
| --- | --- |
| `DESIGN.md` | **The design system to implement** — colours, type, spacing, components and content rules. |
| `screens/` | Corrected reference screens (`code.html` + `screen.png` + `CHANGES.md` listing every edit). |
| `later/` | Screens kept for a later release (spoken announcements). Not built in R1a/R1b. |
| `stitch-export/` | The original Stitch export, untouched, for reference only. Don't build from it. |

## Screens

| Screen | FSD | Release | Keep from Stitch | Fixed to follow the docs |
| --- | --- | --- | --- | --- |
| `00_logo` | — | R1a | Paper sheet + drop arrow on navy | — |
| `01_walkin_drop` | SCR-W01–W03 | R1a | One-page drop flow, live price, optional name, sticky "Send to Counter ₹23", privacy note | Removed account icon and duplicate language row; added मरा; prices follow the pricing rules (₹2/side, ₹3/sheet); "both sides" wording; non-blocking install banner; token `A-07` |
| `02_ticket_and_receipt` | SCR-W04–W05 | R1a | Dark token card, 3-step status, haptic/sound notice, order summary, deletion countdown, install card | Removed phone number, staff name, rack bay; removed "DPDP verified", hash seal and spooler claims; Edit/Cancel replaced by "Changes are closed" while printing; prices corrected (₹24) |
| `03_home_print_nearby` | SCR-A00, A03 | R1b | Station cluster tabs, filters, active-token strip, Scan QR card, best-pick cards, bottom nav with centre Scan | Removed "PWA" badge and rack bay; "/side" pricing; R2 features (Aadhaar masking) removed from shop tags |
| `04_shop_remote_order_alert` | SCR-S06 | R1b | 5-minute red countdown, payout, file manifest, ready-by buttons, one-tap reject reasons | Removed customer profiling ("4th order", "verified commuter") and train/location tracking; "Prepaid · paid to you when you accept" (no escrow); no GSM paper options; reasons match FSD; refund within 60 s; timeout penalty explained; no rack token |
| `05_shop_mobile_queue` | SCR-M01 | R1b (full R2) | Claim next oldest, Queued/Printing/Ready tabs, remote accept/reject card, collected cash/UPI | WALK-IN badge; remote source is Print nearby (no WhatsApp bot); no trays or shelves; prices corrected; honest deletion footer |
| `06_shop_pc_queue_board` | SCR-S03–S05 | R1a | Four-column board with lane tabs, keyboard legend, printing panel, ready cards with paid flags, 10-min undo column, day stats incl. "WhatsApp saved" | Removed audio modal, racks, device telemetry, printer list, page progress and staff-created jobs; "Print all (opens print dialog)"; Release / Ask customer actions; pickup code field; search by token, code or name (not mobile) |
| `07_tv_token_display` | SCR-S16 | R2 | Ready hero, printing and up-next panels, rush banner, "Skip WhatsApp Queue" QR | No customer names or printer data anywhere; no racks; remote shows order number + "show your pickup code"; plain chime text |

## Dropped from the Stitch export

- Customer location, train and profile details on shop screens (privacy, BR-D6).
- WhatsApp as an ordering channel.
- Compliance badges, hashes and claims about shop hardware.
- Racks, trays, shelves and bay numbers.
- Printer models, toner, page-by-page progress and device telemetry (possible only with the R2 print agent).
- Staff-created walk-in orders.
- Paper weight (GSM) options.

## Later release (`later/`)

Spoken token announcements in English/Hindi/Marathi, PA speaker routing, quick-page presets, audio monitor and desk key bindings (3 screens). Added to the R2+ backlog as step 37h. R1a keeps a simple chime and the keyboard shortcuts from FSD FS-6.3.

## Still to design (next Stitch round)

Customer: closed/paused shop (W06), ID-card camera (W07), in-app scanner (A17), upload error states, full-screen Ready state, Hindi and Marathi versions.
Shop: signup wizard (S01), PIN login (S02), collect search/scan result (S05), prices (S08), hours/lanes/staff (S09–S11), printable QR kit (S12).
Print nearby (R1b): OTP sign-in (A02), shop details (A06), customer review & pay (A09), waiting for shop (A10), pickup code (A11), rejected + refunded (A12).
Admin console (R1a).

## Rendering the reference HTML

The HTML loads Tailwind and fonts from CDNs, so open it in a browser with internet access. `screen.png` in each folder is a pre-rendered snapshot.
