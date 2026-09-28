# Counter Drop Web (PWA)

One React 18 + Vite + TypeScript + Tailwind PWA for customers and shops. Design system: `../design/DESIGN.md`.

```bash
npm install
npm run dev -- --host      # http://localhost:5173, proxies /api to http://localhost:8080
npm run build              # type-check + production build into dist/ (served by the API via CD_WEB_DIR)
```

Set `VITE_API_PROXY` to proxy to another API address in dev, or `VITE_API_BASE_URL` to call a separate API origin in production.

## Routes

| Route | Screen | Status |
| --- | --- | --- |
| `/` | Home: how it works, this phone's tickets, staff sign-in link | MVP |
| `/s/:slug` | Walk-in drop page — upload only, pay at the counter (shop QR opens this) | MVP |
| `/t/:jobId#secret` | Ticket: token, live status, ready moment, deletion receipt | MVP |
| `/shop/login` | Shop link name → staff name → 4-digit PIN | MVP |
| `/shop/setup#token` | One-time link: choose your own PIN, then signed in | MVP |
| `/shop/account` | My account: change PIN, sign out | MVP |
| `/shop` | Live queue board: lanes, claim, ready, collected, undo, cancel, search, Online/Paused/Offline | MVP |
| `/shop/qr` | Printable A4 counter poster (EN/HI/MR) | MVP |
| `/shop/settings` | Owner: name, address, hours, prices, staff (add, PIN links, remove) | MVP |
| `/scan` | In-app QR scanner | Later (phone camera works today) |
| `/nearby` | Print nearby: find, prepay by UPI, collect | R1b |
| `/shop/mode`, `/tv/:slug`, `/admin/*` | Phone shop mode, TV display, admin | R1b / R2 |

## Layout

```text
src/
├── App.tsx, main.tsx        # routes, providers, service worker
├── components/ui.tsx        # Button, Card, Chip, Segmented, Stepper, Banner…
├── lib/                     # api client, types, i18n (en/hi/mr), live (SSE), pdf page count, upload, tickets
├── customer/                # Home, DropPage, TicketPage
└── shop/                    # auth, LoginPage, SetupPage, AccountPage, BoardPage, QrPage, SettingsPage, StaffSection
```

Notes: page counts come from pdf.js on the phone (the server falls back to "pages confirmed at counter"). Live updates use Server-Sent Events with polling fallback. Customer screens are in English, Hindi and Marathi; shop screens are English only for now.
