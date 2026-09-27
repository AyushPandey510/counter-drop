# Counter Drop — Design System

Built from the Stitch "Station Utility" system and corrected to match `docs/BRD.md` and `docs/FSD.md`. This is what the PWA (build-plan steps 15–22) implements. Where this file and the Stitch export disagree, this file wins.

## 1. Principles

1. **Readable at a glance, in a crowd, in sunlight.** Solid colours, hard edges, no gradients, no blur, no soft shadows.
2. **The token is the hero.** Tokens (`A-07`) and pickup codes are the biggest thing on every screen that has one.
3. **One screen, one job.** One primary button, fixed in the bottom 35% of the screen (thumb zone).
4. **Three languages, no clipping.** English, Hindi and Marathi share layouts; line height ≥ 1.45 so Devanagari marks never clip.
5. **Honest, not impressive.** Show what the system really does. No compliance badges, no technical theatre (see §8).
6. **Calm at rush hour.** The shop board gets simpler as the queue grows (rush mode), never busier.

## 2. Colour

Light theme is the default for every surface. The shop dashboard and shop mode also offer a dark theme (owner setting, remembered per device).

### Light (default)

| Token | Hex | Use |
| --- | --- | --- |
| `ink` | `#0F172A` | Primary text, token digits, dark headers, focus ring |
| `ink-muted` | `#334155` | Secondary text |
| `ink-subtle` | `#64748B` | Hints, timestamps |
| `canvas` | `#F8FAFC` | Page background (level 0) |
| `surface` | `#FFFFFF` | Cards, inputs (level 1) |
| `surface-tint` | `#EFF4FF` | Selected tiles, info panels |
| `border` | `#CBD5E1` | Card and input borders (1.5 px) |
| `divider` | `#E2E8F0` | Row separators (1 px) |
| `action` | `#2563EB` | Primary buttons, links, selection outline, upload progress |
| `action-pressed` | `#1D4ED8` | Pressed primary |
| `ready` | `#15803D` | Ready for pickup, payment success, collected |
| `ready-bg` | `#DCFCE7` | Ready badges |
| `attention` | `#B45309` | Delays, busy shops, "pages to confirm", undo countdown |
| `attention-bg` | `#FEF3C7` | Attention badges |
| `danger` | `#DC2626` | Errors, reject, cancel, unpaid |
| `danger-bg` | `#FEE2E2` | Error panels |
| `scrim` | `rgba(15,23,42,0.72)` | Modal backdrop |

### Dark (shop option)

| Token | Hex |
| --- | --- |
| `ink` | `#E2E8F0` |
| `ink-muted` | `#94A3B8` |
| `canvas` | `#0B1220` |
| `surface` | `#111A2E` |
| `surface-tint` | `#172554` |
| `border` | `#334155` |
| `action` | `#3B82F6` |
| `ready` | `#22C55E` |
| `attention` | `#F59E0B` |
| `danger` | `#EF4444` |

### Job status colours (always colour + icon + words)

| State | Colour | Icon (Material Symbols) | Label |
| --- | --- | --- | --- |
| In line (`queued`) | `ink` on `surface-tint` | `schedule` | In line |
| Printing (`claimed`) | `action` | `print` | Printing |
| Ready (`ready`) | `ready` | `check_circle` | Ready |
| Collected | `ink-subtle` | `task_alt` | Collected |
| Waiting for shop (remote) | `attention` | `hourglass_top` | Waiting for the shop |
| Cancelled / rejected | `danger` | `cancel` | Cancelled |

Contrast: all text pairs meet WCAG 2.1 AA (4.5:1); tokens and primary buttons target AAA.

## 3. Typography

| Style | Font | Size / line | Weight | Use |
| --- | --- | --- | --- | --- |
| `token-xl` | JetBrains Mono | 72/76 | 700 | Ticket token, TV hero |
| `token-lg` | JetBrains Mono | 48/56 | 700 | Pickup code, board hero |
| `token-md` | JetBrains Mono | 28/32 | 700 | Board cards |
| `headline-xl` | Noto Sans | 28/36 (mobile), 36/44 (desktop) | 700 | Screen titles |
| `headline-md` | Noto Sans | 20/28 | 600 | Card titles |
| `body-lg` | Noto Sans | 16/24 | 500 | Main text (minimum for customers) |
| `body-md` | Noto Sans | 14/20 | 400 | Secondary text |
| `label` | Noto Sans | 12/16 | 600, +0.02em, uppercase in English only | Section labels, chips |
| `numeric` | JetBrains Mono | inherit | 500–700, tabular | Prices, page counts, timers |

- Fonts are self-hosted (Noto Sans, Noto Sans Devanagari, JetBrains Mono) — no Google Fonts calls on the drop page.
- Never uppercase Hindi or Marathi strings; `text-transform` only on `lang="en"`.
- Numbers use Western digits in all languages; money is `₹1,00,000` (en-IN grouping).

## 4. Layout, spacing, shape

- 8 px grid. Spacing tokens: 4, 8, 16, 24, 40.
- Mobile (≤ 640 px): single column, 16 px side padding, primary action fixed at the bottom.
- Tablet (641–1024): 6 columns — settings left, summary right.
- Desktop (≥ 1025): 12 columns, max 1280 px (shop board uses the full width).
- Touch targets ≥ 48 × 48 px, ≥ 8 px apart. Primary buttons ≥ 52 px tall.
- Radius: 4 px everywhere (chips and badges too); `full` only for status dots and the language switcher.
- Borders: 1.5 px for cards and inputs, 1 px for dividers, 2 px for selected/active.
- Elevation: none. Level 1 = white + 1.5 px border; level 2 = white + 2 px `action` border; modal = white + 2 px `ink` border + hard offset `4px 4px 0 #0F172A` over the scrim.

## 5. Components

| Component | Spec |
| --- | --- |
| **Primary button** | `action` fill, white 16/600 text, 52 px, pressed `action-pressed`, focus ring 2 px `ink` offset 2 px. |
| **Ready button** | `ready` fill — used for Mark ready, Collected, Accept. |
| **Secondary button** | White, 1.5 px `ink` border, `ink` text. |
| **Destructive button** | White, 1.5 px `danger` border, `danger` text. Only Cancel/Reject. |
| **Token card** | Dark `ink` panel, lane chip top-left, label "Your token", `token-xl` digits, status strip at the bottom in the state colour. |
| **Status stepper** | 3 steps In line → Printing → Ready with icons, time under completed steps. |
| **File row** | Type icon, file name (1 line, ellipsis), chips (pages, mode, sides), price right-aligned with a small unit line ("7 sheets"). Upload progress as a 4 px `action` bar under the row; errors turn the row border `danger` with one action. |
| **Settings tiles** | Segmented tiles: label + price hint ("₹2 / side", "₹3 / sheet"), selected = `surface-tint` + 2 px `action` border + check icon. |
| **Price breakdown** | Lines per file, add-ons, total in `numeric` bold. Walk-in footer: "Pay at the counter · cash or UPI". Remote adds a "Convenience fee" line. |
| **Chips/badges** | 4 px radius, `label` style, tinted backgrounds by meaning. |
| **Language switcher** | Segmented `EN · हिं · मरा` in the top bar, one place per screen. |
| **Install banner** | Slim, dismissible row under the header: icon, one line, "Install", ✕. Never modal; hidden during uploads and on the Ready screen. |
| **Deletion receipt card** | Shield icon, "Deletion receipt", countdown after pickup, then "3 files deleted at 6:57 pm". |
| **Customer bottom nav (installed PWA)** | Print nearby · **Scan** (raised centre button) · My jobs. Hidden on the upload-only flow. |
| **Shop board card** | Token (`token-md`), first name, channel badge (WALK-IN / PREPAID), files × pages, settings summary, price, age (amber at 10 min, red at 20). One primary action. |
| **Countdown banner** | Solid `danger` strip with a monospace timer — remote accept window only. |

## 6. Screens (reference HTML in `design/screens/`)

| Folder | FSD screens | Release |
| --- | --- | --- |
| `01_walkin_drop` | SCR-W01, W02, W03 | R1a |
| `02_ticket_and_receipt` | SCR-W04, W05 | R1a |
| `03_home_print_nearby` | SCR-A00, A03 | R1b (Home R1a without Nearby) |
| `04_shop_remote_order_alert` | SCR-S06 | R1b |
| `05_shop_mobile_queue` | SCR-M01 (full version R2) | R1b / R2 |
| `06_shop_pc_queue_board` | SCR-S03, S04, S05 | R1a |
| `07_tv_token_display` | SCR-S16 | R2 |

## 7. Shop board layout (updates FSD SCR-S03)

The desktop board uses four columns — **In line · Printing · Ready · Collected (10-min undo)** — with lanes as filter tabs above the board (All lanes / Lane A / Lane B) and a lane chip on every card. "Claim oldest [N]" claims the oldest job in the selected lane (all lanes when "All" is selected). This replaces "one column per lane" in the FSD because it stays readable with 2–5 lanes on a 1366 px screen.

## 8. Content rules (from the design review)

Every screen must follow these; they come from the BRD and FSD.

| Rule | Source |
| --- | --- |
| Tokens look like `A-07`; remote orders show an order number `#4921`; the 4-digit pickup code is shown only to the customer and typed/scanned by staff. | BR-T1–T4 |
| Walk-in screens have no login, no account icon, no payment and no nearby links. | FS-5.0.1 |
| Never show a customer's phone number, location, travel plans or order history to the shop. First name only. | BR-D6, CT-2 |
| The TV shows tokens, order numbers and counters only — never names or prices. | FS-15.4 |
| No compliance badges ("DPDP verified"), hashes or claims about shop PCs and printer spoolers. Say what we do: "files deleted from Counter Drop at 6:57 pm". | CT-1–CT-5 |
| Say "both sides", not "duplex". Prices are "₹2 / side" (one side) and "₹3 / sheet" (both sides). | FS-9.1, UX-C7 |
| Remote money copy: "Prepaid · paid to you when you accept". Never "escrow" or "wallet". | BR-M2–M6 |
| Refunds happen "within 60 seconds". Accept window is 5 minutes; ready-by adjusters are On time / +15 / +30 on screen, with +60 under "More" (API-33 allows 0, 15, 30, 60). | FS-10.6, BR-R1 |
| No WhatsApp ordering channel. Channels are Walk-in and Print nearby. | BRD scope |
| No racks, trays, shelves, printer models, toner, page progress or device telemetry until the R2 print agent exists. | FSD R2 |
| Edit, Add files and Cancel disappear once a job is being printed. | BR-Q3 |
| Spoken announcements, PA speakers and audio routing are a later release (see `design/later/`). A simple chime is fine now. | Decision 27 Sep 2026 |

## 9. Tailwind theme (for build-plan step 15)

```js
// web/tailwind.config.ts — theme.extend
colors: {
  ink: { DEFAULT: 'var(--ink)', muted: 'var(--ink-muted)', subtle: 'var(--ink-subtle)' },
  canvas: 'var(--canvas)', surface: { DEFAULT: 'var(--surface)', tint: 'var(--surface-tint)' },
  border: 'var(--border)', divider: 'var(--divider)',
  action: { DEFAULT: 'var(--action)', pressed: 'var(--action-pressed)' },
  ready: { DEFAULT: 'var(--ready)', bg: 'var(--ready-bg)' },
  attention: { DEFAULT: 'var(--attention)', bg: 'var(--attention-bg)' },
  danger: { DEFAULT: 'var(--danger)', bg: 'var(--danger-bg)' },
},
fontFamily: {
  sans: ['"Noto Sans"', '"Noto Sans Devanagari"', 'sans-serif'],
  mono: ['"JetBrains Mono"', 'monospace'],
},
borderRadius: { DEFAULT: '4px', sm: '2px', full: '9999px' },
boxShadow: { modal: '4px 4px 0 0 #0F172A' },
```

CSS variables hold the light values on `:root` and the dark values on `[data-theme="dark"]` (shop only).
