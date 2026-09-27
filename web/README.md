# Counter Drop Web (PWA)

One React + Vite + TypeScript PWA for every surface. Scaffolded in build-plan step 15; design system in `../design/DESIGN.md`.

| Route | Surface | Release |
| --- | --- | --- |
| `/s/:slug` | Walk-in drop page — upload only, pay at counter (opened by the shop QR) | R1a |
| `/t/:jobId#secret` | Ticket: live status and deletion receipt | R1a |
| `/` | Home of the installed PWA: Scan, Print nearby, My jobs | R1a (Nearby in R1b) |
| `/scan` | In-app shop QR scanner | R1a |
| `/nearby` | Print nearby (find, prepay by UPI, collect) | R1b |
| `/shop/*` | Shop dashboard: signup, PIN login, queue board, settings, QR kit | R1a |
| `/shop/mode` | Minimal shop mode for phones | R1b |
| `/tv/:slug` | TV token display | R2 |
| `/admin/*` | Internal admin console | R1a |

Not scaffolded yet — `src/*` folders hold `.gitkeep` placeholders.
