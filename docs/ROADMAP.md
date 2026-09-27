# Roadmap

The detailed, step-by-step plan is in [`build-plan/README.md`](build-plan/README.md). Requirements are in [`BRD.md`](BRD.md) and [`FSD.md`](FSD.md); visual design in [`../design/DESIGN.md`](../design/DESIGN.md).

| Release | Scope | Gate to move on |
| --- | --- | --- |
| **R1a — Walk-in pilot** (~8 weeks) | QR drop page (upload only, pay at counter), tokens, shop queue board, live status, auto-delete, staff login, admin console | 10 pilot shops live; counter time measured; zero lost jobs; 100% files deleted on time |
| **R1b — Print nearby + PWA app** (~10 weeks) | Nearby search, UPI prepay (Razorpay Route), accept/reject with auto refunds, pickup codes, installable PWA with in-app scanner, minimal shop mode, web push | 25 shops paying; remote orders paid and collected end to end |
| **R2 — Scale the counter** | Windows print agent, TV token display, full shop mode, plans and billing, owner reports, Aadhaar masking, malware scan | 100 paying shops; churn under 3% a month |
| **R3 — Expand** | Business tier (multi-branch, SSO, org audit), Play Store wrapper, SwiftShare link, DOCX conversion | — |
| **Later** | Spoken token announcements / PA audio (`design/later/`), OCR on photos, macOS agent | — |

## Current position (27 Sep 2026)

Planning and design are done. Code is at an early backend slice; see [`CODE-STATUS.md`](CODE-STATUS.md). Next: build-plan steps 01 → 02 → 03.
