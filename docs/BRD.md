# Counter Drop — Business Requirements Document

> Snapshot of the live doc: https://claude.ai/code/artifact/2f1f7583-83a1-4b12-9572-6fcb540c2de0

Sep 27, 2026 · @Sushil Pandey

## 0. Document control

This is version 1.0 of the Counter Drop BRD, a draft for review by the product owner, the tech lead and the pilot shop partners. Figures marked *proposal* are to be confirmed with pilot data.

| Field | Value |
| --- | --- |
| Product | Counter Drop |
| Document | Business Requirements Document (BRD) |
| Version | 1.0 (draft) |
| Status | For review |
| Author | Sushil Pandey |
| Reviewers | Tech lead, design lead, legal (DPDP), 3 pilot shop owners |
| Approvers | Product owner / founder |
| Related | Feature spec pack, `contracts/openapi.yaml`, `docs/ARCHITECTURE.md`, `docs/ROADMAP.md` |

**Change log**

| Version | Date | Change |
| --- | --- | --- |
| 1.0 | 27 Sep 2026 | Full BRD. Adds Print nearby and the Android app to Release 1. Splits Release 1 into R1a (walk-in pilot) and R1b (Print nearby and app). Adds money flow, retention split, edge cases and delight requirements. |
| 0.2 | Sep 2026 | Objectives section with Print nearby and app as top priorities. |
| 0.1 | Sep 2026 | Walk-in QR drop concept. |

**v1.1 (27 Sep 2026):** the customer app is a PWA instead of a native Flutter app. Scanning a shop QR opens an upload-only screen with payment at the counter. UPI prepay stays in Print nearby only.

**v1.2 (27 Sep 2026):** visual design adopted from the Stitch export and corrected to this BRD (see `design/DESIGN.md` in the repo). Spoken token announcements and PA audio moved to a later release.

**How to read this document.** Requirements use IDs: BO (business objective), UC-C (customer use case), UC-S (shop use case), UC-A (admin use case), FR (functional requirement), BR (business rule), EX (exception), UX, NFR, CT (compliance). Priority uses MoSCoW: **Must**, **Should**, **Could**, **Won't (this release)**.

## 1. Executive summary

Counter Drop replaces "send it on my WhatsApp" at print and document counters with a secure queue that customers and shopkeepers both enjoy using. It launches in Mumbai with 10 pilot shops, then opens Print nearby in one station cluster.

**The problem.** Customers queue twice, share personal numbers with strangers and can't see prices or wait times. Shops dig through chats, count pages by hand, argue over prices and keep customers' ID documents on their phones forever.

**The solution.** Two ways to send a job, one dashboard to run it:

- **Walk-in drop.** Scan the shop's QR, upload from the phone browser, get a token such as `A-07`, collect when it turns green. No app, no login.
- **Print nearby.** In the Counter Drop app (an installable PWA) or on the website, find an open, not-busy shop, upload, pay by UPI and walk in only to collect with a pickup code.
- **Shop dashboard.** Every job, walk-in or remote, in one live queue on a PC or phone, with auto page count, auto price, one-tap status and auto-delete of files after pickup.

**Why it wins.** At least 15 QR-print tools already exist in India. Counter Drop stands out on four things: live wait times across nearby shops, a rush-hour mode that keeps the counter calm, visible privacy (deletion receipts, Aadhaar masking) and a customer app that brings people back.

**Money.** Shops pay a monthly plan (proposal: ₹0 / ₹199 / ₹499). Customers pay a ₹2–5 convenience fee on remote orders only. 100% of print money settles to the shop. Counter Drop never holds shop funds.

**Release plan.**

- **R1a — Walk-in pilot (8 weeks):** QR drop, tokens, dashboard, live status, auto-delete, staff login.
- **R1b — Print nearby and app (next 10 weeks):** nearby search, UPI prepay with refunds, pickup codes, installable PWA with in-app QR scanner, minimal shop mode for accept/reject.
- **R2:** print agent, TV display, plans and billing, Aadhaar masking, reports.
- **R3:** business tier, Play Store listing (PWA wrapper), SwiftShare link.

**Decision needed.** This BRD assumes the R1a/R1b split so pilot data can set prices, fees and targets before Print nearby launches. See section 22.

## 2. Background and business problem

Most documents reach an Indian print counter through the shopkeeper's personal WhatsApp, and that habit is slow, error-prone and a growing legal risk.

**Market context**

- Every station area, college gate and government office in Mumbai has several xerox and print shops. Peak hours are 8–11 am and 5–8 pm, around office, college and form-filling deadlines.
- Typical jobs: ID copies (Aadhaar, PAN), forms, resumes, college assignments, ticket and booking printouts, photos, lamination and scanning.
- Customers carry files on their phones, not pen drives. WhatsApp is the default transfer channel.
- At least 15 QR-print tools exist in India. Most stop at "upload to one shop" and have no nearby discovery, live wait, rush handling or privacy promise.
- The DPDP Act 2023 and its Rules make anyone who keeps personal data responsible for it. Shops holding customers' ID copies indefinitely are exposed. The phased deadline this BRD plans for is **14 May 2027** (to be confirmed by legal review).

**Problems today**

| Stakeholder | Pain | Effect |
| --- | --- | --- |
| Customer | Doesn't know which nearby shop is open or busy | Wasted trips, long waits |
| Customer | Queues once to send the file, again to collect | 10–20 minutes lost at peak |
| Customer | Shares a personal number with a stranger | Spam, privacy worry, unwanted contact |
| Customer | No price or ready-time up front | Disputes, distrust |
| Customer | ID documents stay on the shop's phone | Fear of misuse |
| Shop owner | Searches chats for the right file and settings | Slow counter, wrong prints |
| Shop owner | Counts pages by hand | Under-charging, arguments |
| Shop owner | Jobs printed twice or missed at rush hour | Paper waste, angry customers |
| Shop owner | Personal WhatsApp flooded with strangers' files | Phone storage full, no work/life boundary |
| Shop owner | Depends only on walk-in footfall | Idle machines off-peak |
| Both | No record of who got which file, and when it was deleted | Legal and reputational risk |

**Root cause.** There is no shared, structured channel between the customer's phone and the shop's counter. WhatsApp is a chat tool used as a job queue.

## 3. Vision and product principles

**Vision.** Anyone in India can send a document to the nearest counter from anywhere, safely and quickly, without sharing a phone number and without leaving a copy behind.

**Promise to customers:** *"Drop it, see the price, walk in, pick it up. Your file is gone after."*

**Promise to shops:** *"Every job in one line, priced and counted for you. No more hunting through WhatsApp."*

**Product principles.** Every requirement in this BRD is checked against these ten rules. If a feature breaks one, it is redesigned.

1. **Faster than WhatsApp, or it fails.** Walk-in drop takes fewer taps than sending a WhatsApp file: scan, pick file, send. Target: under 30 seconds from scan to token.
2. **No app, no login, no form for walk-in.** Name is optional; a phone number is never required for walk-in.
3. **Show the price and the wait before asking for anything.** No surprises at the counter.
4. **One screen, one job.** Each screen has one primary action, a big thumb-reachable button, and plain words.
5. **Works on a ₹7,000 phone on patchy 4G.** Small pages, resumable uploads, no heavy animations.
6. **The shopkeeper never types.** Page count, price, token and status are automatic. Staff tap; they don't type.
7. **Calm at rush hour.** The dashboard gets simpler, not busier, when the queue grows.
8. **Privacy you can see.** Deletion receipts, masked IDs and a visible countdown, not a policy link.
9. **Speak the customer's language.** English, Hindi and Marathi from day one; icons and numbers carry meaning without reading.
10. **Never lose a job, never lose money.** Every failure has an automatic recovery path and a human-readable message.

**What "love to use" means, measurably**

| Experience goal | Measure | Target (proposal) |
| --- | --- | --- |
| Effortless walk-in | Median scan-to-token time | Under 30 s |
| Effortless remote order | Median find → pay time | Under 2 min |
| Customers come back | Customers with 2+ jobs in 30 days | 40% |
| Customers recommend it | Customer NPS (in-app, after pickup) | 50+ |
| Shops rely on it | Shops active on 5+ days a week | 80% of paying shops |
| Shops recommend it | Shop NPS (monthly) | 40+ |
| Low friction | Drop-off between file picked and job submitted | Under 10% |
| Low support load | Support tickets per 1,000 jobs | Under 5 |

## 4. Business objectives and KPIs

The first 6 months aim to prove that shops save time, that customers order remotely and that shops will pay. All targets are proposals to confirm after the R1a pilot.

**Business objectives**

| # | Objective | Measure | Target (proposal) |
| --- | --- | --- | --- |
| BO-1 | Cut counter time per job | Average counter time per job, before vs after | 50% lower in at least 5 of 10 pilot shops |
| BO-2 | Serve more customers at peak without more staff | Jobs completed per peak hour | Up 30% within 60 days of go-live |
| BO-3 | Make remote ordering a main channel | Share of jobs placed through Print nearby | 20% of jobs at remote-enabled shops within 90 days of R1b |
| BO-4 | Build a direct customer base | PWA installs; monthly active users (Mumbai) | 10,000 installs and 3,000 MAU within 6 months of R1b |
| BO-5 | Build a paying shop base | Paying shops | 25 after the pilot; 100 across 3 station clusters within 6 months |
| BO-6 | Recurring revenue with low churn | Monthly churn of paying shops | Under 3% |
| BO-7 | Second revenue stream | Convenience fee per remote order | ₹2–5 from R1b launch |
| BO-8 | Keep running costs low | Hosting + messaging + OTP cost | Under ₹2,000/month until 50 paying shops |
| BO-9 | Expand beyond print shops | Business-tier accounts | 2 colleges and 1 CSC network within 9 months |

**Customer objectives**

| # | Objective | Success measure |
| --- | --- | --- |
| CO-1 | Find a nearby shop that is open and not busy | Nearby list shows distance, open status, prices and live wait for every listed shop |
| CO-2 | Order and pay from anywhere, then just collect | Find → upload → pay → pickup code in under 2 min (median) |
| CO-3 | A dedicated app for repeat use | Installable PWA with in-app QR scan, nearby, upload, UPI, live status, history and reorder |
| CO-4 | Walk-in needs no app | QR drop works fully in a mobile browser with no login |
| CO-5 | Never queue just to hand over a file | Share of walk-in jobs uploaded before reaching the counter |
| CO-6 | Know price, wait and ready-by time up front | Shown before submit or payment on every job |
| CO-7 | Be told when the job is ready | Push sent within 5 s of "Ready" (p95); open web page updates within 1 s |
| CO-8 | Trust that files are deleted | Deletion receipt for every collected job |
| CO-9 | Use it in their own language | English, Hindi, Marathi at launch |
| CO-10 | Keep ID numbers private | Aadhaar numbers detected and masked on the phone before upload (R2) |

**Shop objectives**

| # | Objective | Success measure |
| --- | --- | --- |
| SO-1 | See every job in one place | Zero jobs taken over WhatsApp at pilot shops after week 2 |
| SO-2 | Never print twice or lose a job | Zero duplicate claims and zero lost jobs in the audit log |
| SO-3 | No manual page counting or price disputes | Auto page count and price for 100% of PDF and image jobs |
| SO-4 | Handle rush hours calmly | Queue lanes, oldest-first claiming, pause intake, TV token screen (R2) |
| SO-5 | Get new customers from Print nearby | Remote orders per shop per week |
| SO-6 | Take remote orders without risk | Accept/reject within 5 min; auto refund on reject or timeout; money settles to the shop |
| SO-7 | Run the counter from a phone | Minimal shop mode in R1b; full shop mode in R2 |
| SO-8 | Print with one click | Windows print agent (R2) |
| SO-9 | Get started without help | Walk-in live within 10 min of signup; remote orders after payout KYC |
| SO-10 | Keep all the print money | 100% of print payments settle to the shop |
| SO-11 | Reduce legal risk | Auto-delete and audit log on every plan, including free |

## 5. Stakeholders and user personas

Counter Drop serves two sides that must both win: customers who want speed and privacy, and shopkeepers who want a calm, paid-up counter.

**Stakeholders**

| Stakeholder | Role | Interest | Involvement |
| --- | --- | --- | --- |
| Founder / product owner | Owns vision, scope, pricing | Paying shops, growth | Approves BRD and releases |
| Tech lead | Owns architecture and delivery | Reliability, cost | Reviews NFRs and feasibility |
| Design lead | Owns UX and copy | Ease of use | Reviews UX requirements |
| Shop owner | Pays for the plan, sets prices | Speed, revenue, fewer disputes | Pilot partner, feedback |
| Counter staff | Runs the queue daily | Simple, fast screens | Usability tests |
| Customer (walk-in) | Uploads at the shop | Speed, no app | Usability tests |
| Customer (remote / app) | Orders from anywhere | Nearby, prepay, pickup | Beta users |
| Business-tier admin | College, CSC, office | Control, audit, multi-branch | R3 design partner |
| Support / ops | Onboards shops, handles tickets | Low ticket volume, clear tools | Admin console users |
| Legal / DPO advisor | DPDP compliance | Minimal data, retention proof | Reviews section 16 |
| Razorpay (payment partner) | Payments, Route settlement, refunds | Compliant merchant flows | Integration, KYC |
| Firebase / SMS provider | Push and OTP | Volume | Integration |

**Personas**

| Persona | Who | Context | Needs | Frustrations today | What makes them love it |
| --- | --- | --- | --- | --- | --- |
| **Priya, 20, student** | Engineering student near Andheri | Prints assignments and forms before college, often late | Cheap, fast, no queue, Hindi/English | Queues at 8:30 am, shop loses her file in WhatsApp | Uploads from the train, pays by UPI, walks in and picks up in 30 s |
| **Ramesh, 45, office worker** | Commutes via Dadar | Needs ID copies and forms for bank or government work | Clear price, trust, simple steps | Hands his Aadhaar to a stranger's WhatsApp | Aadhaar masked automatically, deletion receipt on screen |
| **Sunita, 58, homemaker** | Low digital confidence, Marathi first | Occasional printouts for pension and school forms | Big text, own language, no signup | Doesn't know how to "send the file" | Scans QR, taps one big button, gets a token like a bank queue |
| **Imran, 34, shop owner** | Runs a 2-machine xerox shop near Thane station | Owner plus one helper, peak rush twice a day | Speed, fewer disputes, more customers | Phone full of strangers' files, arguments over pages | Auto price, one queue, new remote customers, WhatsApp back to personal |
| **Kavita, 26, counter staff** | Works at a busy Dadar shop | Handles 150+ jobs a day | Never lose a job, no typing | Double prints, shouting at rush hour | Big claim/ready buttons, oldest-first, lanes, customers watch the TV screen |
| **Mr. Joshi, 50, college admin** | Runs the college print room | Hundreds of students, audit needs | Control, reports, SSO | No record of who printed what | Business tier with org audit and branch reports (R3) |

## 6. Scope and release plan

Release 1 is split in two so the walk-in pilot proves the core and sets prices before money moves through Print nearby.

*Diagram: release roadmap · 4 phases, 3 gates — see the [live doc](https://claude.ai/code/artifact/2f1f7583-83a1-4b12-9572-6fcb540c2de0).*

Phase lengths are proposals. A gate that fails extends the phase; it does not skip ahead.

**In scope (all releases)**

| Area | Capability | Release | Priority |
| --- | --- | --- | --- |
| Walk-in | QR drop page (web, no login), token, live status page | R1a | Must |
| Queue | Shop dashboard with lanes, claim, ready, collected, cancel, release | R1a | Must |
| Queue | Auto page count and auto price for PDF and images | R1a | Must |
| Queue | Pause intake, wait-time estimate | R1a | Must |
| Privacy | Auto-delete, deletion receipt, audit log | R1a | Must |
| Access | Staff login (owner + staff roles), shop signup and QR kit | R1a | Must |
| Languages | English, Hindi, Marathi | R1a | Must |
| Remote | Print nearby: search, shop page, remote upload, pickup code | R1b | Must |
| Payments | UPI prepay via Razorpay, settlement to shop (Route), auto refunds, reconciliation | R1b | Must |
| App | Customer PWA: install prompt, in-app QR scan, nearby, upload, pay, status, push, history, reorder | R1b | Must |
| App | Minimal shop mode: new-order push, accept/reject, ready, collected | R1b | Must |
| Admin | Internal admin console: shops, KYC status, refunds, support | R1b | Must |
| Counter | Windows print agent | R2 | Should |
| Counter | TV token display | R2 | Should |
| App | Full shop mode (queue, lanes, settings) | R2 | Should |
| Revenue | Plans, billing, owner reports | R2 | Should |
| Privacy | Aadhaar detection and masking on device | R2 | Should |
| Expansion | Business tier: multi-branch, SSO, org audit | R3 | Could |
| Expansion | Play Store listing via Trusted Web Activity; link from SwiftShare | R3 | Could |
| Later | DOCX conversion, OCR on photos, macOS agent, spoken token announcements (multilingual PA audio) | Later | Could |

**Out of scope**

- Home delivery of printouts.
- Kiosks or hardware sales.
- Commission on shop print revenue.
- Document editing, design or typing services.
- Holding customer or shop funds (wallets, credit).
- Cash handling through Counter Drop (walk-in cash stays between customer and shop).
- Shops outside Mumbai until BO-5 is met.

## 7. Current-state vs future-state process

The future flow removes the first queue, the chat search and the manual pricing, and ends every job with a provable deletion.

**Current state (WhatsApp)**

1. Customer queues at the counter, asks for the shop's number.
2. Customer sends files on WhatsApp; the shopkeeper's phone now has the customer's number and files.
3. Shopkeeper scrolls chats, forwards files to the PC (or a WhatsApp Web session), asks "how many copies, colour or B/W?"
4. Shopkeeper counts pages by hand and quotes a price; sometimes the customer disputes it.
5. Job printed; at rush hour jobs are skipped or printed twice.
6. Customer pays cash or UPI to the shop's QR, often queuing again.
7. Files stay on the shop's phone and PC indefinitely.

**Future state (Counter Drop)**

*Diagram: future-state flow · walk-in and Print nearby into one queue — see the [live doc](https://claude.ai/code/artifact/2f1f7583-83a1-4b12-9572-6fcb540c2de0).*

Walk-in jobs enter the queue as soon as the upload finishes. Remote jobs enter only after payment succeeds and the shop accepts; a rejection or a 5-minute silence refunds the customer automatically.

**What changes for each side**

| Step | Before | After | Saved |
| --- | --- | --- | --- |
| Hand over file | Queue + WhatsApp | Upload from anywhere | One queue, no number shared |
| Settings | Asked verbally | Chosen on the phone | Fewer wrong prints |
| Page count and price | Counted by hand | Automatic | Disputes, under-charging |
| Finding the job | Scroll chats | Next job in the queue | Minutes per job at peak |
| Knowing it's ready | Ask or wait at counter | Push + live page | Crowding at the counter |
| Payment | Cash or UPI at counter | Walk-in: at counter; remote: prepaid UPI | Second queue for remote |
| After pickup | Files kept forever | Deleted, receipt shown | Legal risk, trust |

## 8. Customer use cases

Customers get 27 use cases across walk-in, Print nearby and the app; the three core flows are detailed below the catalogue.

**App and entry model (decided 27 Sep 2026).** One installable PWA serves every customer. How a customer arrives decides what they can do:

| Entry | Opens | Customer can | Payment |
| --- | --- | --- | --- |
| Scan shop QR with the phone camera (app not installed) | That shop's upload screen in the browser | Upload, choose settings, see price, get token; optional **Install Counter Drop** banner | At the counter |
| Tap **Scan** inside the installed PWA | The same upload screen, name and language remembered | Upload, settings, price, token | At the counter |
| Open the installed PWA from the home screen | Home: **Scan**, **Print nearby**, **My jobs** | Everything, including remote orders | UPI prepay (Print nearby only) |

- A scan is always **upload only**: no nearby list, no payment, no other shops in that flow.
- The install banner never blocks the upload. It shows again once on the ticket after pickup.
- New use cases: **UC-C28** scan a shop QR from inside the app (R1a, Must) and **UC-C29** install the app from the drop page banner (R1a, Should).

**Use case catalogue**

| ID | Use case | Channel | Release | Priority |
| --- | --- | --- | --- | --- |
| UC-C01 | Drop files at the shop by scanning its QR | Web | R1a | Must |
| UC-C02 | Choose settings per file: copies, B/W or colour, single or double-sided, page range, paper size, orientation | Web, app | R1a | Must |
| UC-C03 | See price, place in line and ready-by time before submitting | Web, app | R1a | Must |
| UC-C04 | Track the job live (queued, printing, ready) | Web, app | R1a | Must |
| UC-C05 | Collect the job by showing the token | Web, app | R1a | Must |
| UC-C06 | See a deletion receipt after pickup | Web, app | R1a | Must |
| UC-C07 | Cancel or change settings before the shop claims the job | Web, app | R1a | Must |
| UC-C08 | Add a forgotten file to a job before it is claimed | Web, app | R1a | Should |
| UC-C09 | Photograph an ID or document; auto-crop; print front and back on one page | Web, app | R1a | Should |
| UC-C10 | Switch language (English, Hindi, Marathi) | Web, app | R1a | Must |
| UC-C11 | Find nearby shops: list and map, open now, live wait, prices, services | App, web | R1b | Must |
| UC-C12 | View a shop page: prices, hours, services, wait, rating, directions | App, web | R1b | Must |
| UC-C13 | Place a remote order and prepay by UPI | App, web | R1b | Must |
| UC-C14 | Be told the shop accepted or rejected; get an automatic refund on reject or timeout | App, web | R1b | Must |
| UC-C15 | Receive pickup code and a "ready" push with directions | App, web | R1b | Must |
| UC-C16 | Collect a remote order with the pickup code | App, web | R1b | Must |
| UC-C17 | Cancel a remote order (refund per rule BR-P6) | App, web | R1b | Must |
| UC-C18 | Report a problem (wrong print, missing pages) and get a reprint or refund | App, web | R1b | Must |
| UC-C19 | Sign in with phone OTP (app and remote orders only) | App, web | R1b | Must |
| UC-C20 | See order history; reorder with the same shop and settings | App | R1b | Must |
| UC-C21 | Save favourite shops | App | R1b | Should |
| UC-C22 | Rate the shop after pickup (1–5 stars, one-tap tags) | App, web | R1b | Should |
| UC-C23 | Share a file to Counter Drop from WhatsApp, Gallery or Files (share sheet, installed PWA on Android) | App | R1b | Must |
| UC-C24 | Get a payment receipt (with shop GSTIN when provided) | App, web | R1b | Should |
| UC-C25 | Manage notifications; delete account and all data | App | R1b | Must |
| UC-C26 | Aadhaar numbers detected and masked before upload | App, web | R2 | Should |
| UC-C27 | Scan a paper document with the camera and print it | App | R2 | Could |

### UC-C01 — Walk-in drop (core flow)

**Actor:** walk-in customer. **Trigger:** customer scans the QR at the counter or on the shop's poster. **Precondition:** shop is open and intake is not paused.

**Main flow**

1. Customer scans the QR. The drop page opens in the phone browser in the shop's language setting, with the shop name, current wait ("6 jobs ahead, about 12 min") and a single button: **Choose files**.
2. Customer picks one or more files (PDF, JPG, PNG, HEIC; DOCX from R3). Uploads start immediately in the background.
3. For each file the page shows a thumbnail, detected page count and default settings (B/W, single-sided, 1 copy, all pages). Customer changes settings with large toggles.
4. Page shows the price breakdown and total, and the ready-by time.
5. Customer optionally types a first name (so staff can call it out). No phone number is asked.
6. Customer taps **Send to counter**. The page shows a large token (`A-07`), place in line and a live status bar.
7. Status updates in real time: *In line → Printing → Ready*. On Ready, the page vibrates (where supported), shows green and plays a short sound.
8. Customer shows the token at the counter and pays the shop directly (cash or the shop's UPI QR).
9. Staff marks it collected. The page shows "Collected" and a deletion countdown, then the deletion receipt.

**Alternate flows**

- *A1 — Intake paused:* page says "The counter is busy. Try again in a few minutes" and shows nearby shops with shorter waits (from R1b).
- *A2 — Shop closed:* page shows opening hours and nearby open shops (R1b).
- *A3 — Upload fails mid-way:* upload resumes automatically when the network returns; customer sees "Waiting for network", never an error code.
- *A4 — Unsupported or password-protected file:* clear message with what to do ("Remove the password and try again" or "Save as PDF").
- *A5 — Customer closes the browser:* reopening the same link restores the ticket (secret stored in the link and local storage).
- *A6 — Customer wants to change something:* **Edit** or **Cancel** available until the job is claimed.

### UC-C13 — Remote order through Print nearby (core flow)

**Actor:** signed-in customer. **Trigger:** customer opens the app (or website) or shares a file into the app. **Precondition:** at least one remote-enabled shop within 10 km.

**Main flow**

1. App asks for location once (or accepts a typed area or station). It lists nearby shops sorted by a "best pick" score: open now, shortest wait, distance, rating.
2. Each card shows distance, open/closing time, live wait, B/W and colour price per page, and badges (Colour, Lamination, Spiral binding).
3. Customer picks a shop (or picks files first, then shop, when arriving from the share sheet).
4. Customer sets options per file, sees the price breakdown: print price + convenience fee = total, and "Ready by 6:40 pm if you pay now".
5. Customer pays by UPI intent (GPay, PhonePe, Paytm, BHIM), without leaving the flow for more than one tap.
6. Payment succeeds. The order goes to the shop as **Awaiting acceptance**; customer sees a 5-minute countdown: "Waiting for Imran Xerox to accept."
7. Shop accepts. Customer receives a push with the pickup code (4 digits + QR) and ready-by time.
8. Shop prints and marks Ready. Customer gets a push "Ready at Imran Xerox, 350 m away" with a Directions button.
9. At the counter, customer shows the pickup code; staff enters or scans it; job marked Collected. Deletion receipt appears in the app.
10. App asks for a one-tap rating.

**Alternate flows**

- *A1 — Shop rejects (out of paper, machine down, too busy):* instant full refund initiated, push explains why, app suggests the next best shop with files already attached: "Send to Sai Print instead?" (one tap, pays again).
- *A2 — No response in 5 minutes:* auto-rejected, auto-refunded, same suggestion as A1. The shop's listing is hidden for 30 minutes (BR-R4).
- *A3 — Payment pending or failed:* order is not sent to the shop until payment is confirmed by webhook. If UPI stays pending over 10 minutes, order is cancelled and any debit is refunded automatically.
- *A4 — Shop closes before customer arrives:* customer notified 30 min before closing; uncollected printouts are kept by the shop until next opening (BR-P8).
- *A5 — Customer never collects:* see BR-P8 (no refund after printing; reminder pushes at ready, +2 h and next morning).

### UC-C16 — Pickup (both channels)

1. Customer arrives and shows the token (walk-in) or pickup code / QR (remote).
2. Staff searches by token or code, or scans the QR with the dashboard camera.
3. Dashboard shows name, file count, pages and whether it's **Prepaid** or **Collect payment ₹X**.
4. Staff hands over, taps **Collected**. For walk-in, staff can also tap **Paid cash** or **Paid UPI** to keep counts.
5. Customer's screen shows "Collected" and the deletion countdown, then the receipt: *"3 files deleted from Counter Drop at 6:52 pm. The shop never received a copy on WhatsApp."*

## 9. Shopkeeper and staff use cases

Shops get 30 use cases covering setup, the daily counter, remote orders, money and growth; the counter loop and remote acceptance are detailed below.

**Use case catalogue**

| ID | Use case | Actor | Release | Priority |
| --- | --- | --- | --- | --- |
| **Setup** |  |  |  |  |
| UC-S01 | Sign up with phone OTP, shop name, address pin on map, hours | Owner | R1a | Must |
| UC-S02 | Set price list: B/W and colour per side, paper sizes, double-sided, add-ons (lamination, spiral, stapling, scan) | Owner | R1a | Must |
| UC-S03 | Download and print the QR kit (counter standee, window poster, token slips) | Owner | R1a | Must |
| UC-S04 | Add staff with a PIN; assign roles (owner, staff) | Owner | R1a | Must |
| UC-S05 | Set counters/lanes (e.g. Lane A: B/W, Lane B: colour and photos) | Owner | R1a | Should |
| UC-S06 | Set holidays and special hours | Owner | R1a | Should |
| UC-S07 | Enable remote orders: bank/UPI payout KYC via Razorpay | Owner | R1b | Must |
| UC-S08 | Choose which services appear on Print nearby; set max files and pages per remote order | Owner | R1b | Should |
| **Daily counter** |  |  |  |  |
| UC-S09 | Open the shop (go online) and close it (go offline) | Staff | R1a | Must |
| UC-S10 | See the live queue by lane, oldest first, with page count, settings and price | Staff | R1a | Must |
| UC-S11 | Claim the next job; preview and open files | Staff | R1a | Must |
| UC-S12 | Release a claimed job back to the queue | Staff | R1a | Must |
| UC-S13 | Mark Ready (customer notified) | Staff | R1a | Must |
| UC-S14 | Find a job by token, pickup code, name or QR scan; mark Collected | Staff | R1a | Must |
| UC-S15 | Record walk-in payment (cash or UPI) at collection | Staff | R1a | Should |
| UC-S16 | Cancel a job with a reason; customer notified | Staff | R1a | Must |
| UC-S17 | Adjust price for extra work (e.g. lamination added at the counter), with reason | Staff | R1a | Should |
| UC-S18 | Pause intake at rush hour; set a wait message | Staff | R1a | Must |
| UC-S19 | Rush mode: bigger buttons, only next jobs shown, auto-refresh | Staff | R1a | Should |
| UC-S20 | Undo an accidental Collected within 10 minutes (before deletion) | Staff | R1a | Must |
| UC-S21 | Print with one click via the Windows print agent | Staff | R2 | Should |
| UC-S22 | Show tokens on a TV ("Now ready: A-07, B-03") | Staff | R2 | Should |
| **Remote orders** |  |  |  |  |
| UC-S23 | Get an alert (sound, push) for a new remote order | Staff | R1b | Must |
| UC-S24 | Accept or reject within 5 minutes, with a reason | Staff | R1b | Must |
| UC-S25 | Handle a customer complaint: reprint or approve a refund | Owner | R1b | Must |
| **Money and growth** |  |  |  |  |
| UC-S26 | See settlements, refunds and fees for remote orders | Owner | R1b | Must |
| UC-S27 | Daily and monthly report: jobs, pages, revenue, peak hours, average counter time | Owner | R2 | Should |
| UC-S28 | Choose and pay for a plan; download invoices | Owner | R2 | Should |
| UC-S29 | Reply to ratings; see rating trend | Owner | R2 | Could |
| UC-S30 | Manage several branches from one login | Business owner | R3 | Could |

### UC-S10 to UC-S14 — The counter loop

**Actor:** counter staff. **Precondition:** staff signed in on the dashboard (PC browser or shop mode in the app); shop online.

1. Dashboard shows lanes as columns. Each card: token, name, file count, total pages, settings summary ("12 pp, B/W, 2-sided, ×2"), price, age ("3 min"), and a badge **Prepaid** for remote jobs.
2. Staff taps **Claim next** (claims the oldest unclaimed job in that lane). If another counter claimed it a moment earlier, the next one is claimed instead, silently. No double claim is possible.
3. Claimed card opens: file previews, a **Print** button (opens the PDF in the browser print dialog, or sends to the print agent in R2) and the exact settings to use.
4. Staff prints and taps **Ready**. The customer is notified; the card moves to the Ready column.
5. Customer arrives; staff searches or scans the token/code; taps **Collected** (and **Paid cash/UPI** for walk-in).
6. Files are scheduled for deletion; the card shows a 10-minute **Undo** before deletion is final.

**Alternate flows**

- *A1 — Printer jam or wrong settings:* **Release** returns the job to the head of its lane with the same token.
- *A2 — Customer asks for changes at the counter:* staff edits copies or add-ons; price updates with an audit reason.
- *A3 — File won't open:* staff taps **Ask customer** with a preset reason ("File is corrupt, please upload again"); customer's page shows a re-upload button; token is kept.
- *A4 — Dashboard loses internet:* last queue stays visible and read-only with an "Offline" banner; actions retry when back online; claims are never assumed.

### UC-S24 — Accepting a remote order

1. New remote order arrives: loud chime, flashing card, push to the owner's and staff phones (shop mode).
2. Card shows files, pages, settings, price, the customer's first name and a 5:00 countdown.
3. Staff taps **Accept** (optionally adjusts ready-by: +15, +30, +60 min) or **Reject** with a one-tap reason: *Out of paper / Machine down / Too busy / Can't print this file / Closing soon*.
4. On Accept: job joins the queue in the remote lane (or its service lane), marked Prepaid. On Reject: customer is refunded automatically; shop sees "Refund sent, no charge to you."
5. Shops with more than 3 timeouts in 7 days get a nudge to set shorter hours or pause remote orders (BR-R4).

## 10. Platform admin and business-tier use cases

The internal team needs just enough tooling to onboard shops, resolve money issues and prove compliance, without ever opening customer files.

| ID | Use case | Actor | Release | Priority |
| --- | --- | --- | --- | --- |
| UC-A01 | Approve or suspend a shop listing (fake address, repeated timeouts, complaints) | Ops | R1a | Must |
| UC-A02 | View shop KYC and payout status (Razorpay linked account) | Ops | R1b | Must |
| UC-A03 | Search an order by ID, token or payment ID; see its timeline (no file content) | Support | R1b | Must |
| UC-A04 | Trigger a manual refund with reason, within policy | Support | R1b | Must |
| UC-A05 | Daily payment reconciliation: payments vs orders vs refunds vs settlements; flag mismatches | Finance | R1b | Must |
| UC-A06 | View deletion health: files due, deleted, failed; retry failed deletions | Ops | R1a | Must |
| UC-A07 | Export an audit trail for a shop or order (for a legal or customer request) | Ops | R1a | Must |
| UC-A08 | Handle a data request: customer asks what data is held, or to delete it | Ops | R1b | Must |
| UC-A09 | Manage cities, station clusters and the "best pick" ranking weights | Ops | R1b | Should |
| UC-A10 | Set plans, prices, convenience fee and promo codes | Ops | R2 | Should |
| UC-A11 | Broadcast a message to shops (downtime, new feature) | Ops | R1a | Should |
| UC-A12 | Monitor KPIs: jobs, remote share, counter time, deletion SLA, churn | Founder | R1a | Must |
| UC-B01 | Business admin: add branches, counters and staff under one organisation | Org admin | R3 | Could |
| UC-B02 | Sign in with Google Workspace or Microsoft SSO | Org admin | R3 | Could |
| UC-B03 | Org-wide audit log and usage reports by branch, department or user | Org admin | R3 | Could |
| UC-B04 | Set org retention rules (shorter than default, never longer than 24 h) | Org admin | R3 | Could |
| UC-B05 | Allow only verified members (e.g. college email) to drop files | Org admin | R3 | Could |

**Guardrail.** No admin role can view or download customer files. Admin tools show metadata only (file count, pages, sizes, timestamps). This is enforced in the API, not only the UI.

## 11. Functional requirements

The system has ten modules; every requirement below traces to a use case and is testable.

**Job lifecycle**

*Diagram: job states · 10 states, one end — see the [live doc](https://claude.ai/code/artifact/2f1f7583-83a1-4b12-9572-6fcb540c2de0).*

A remote job enters the queue only after payment is confirmed and the shop accepts. Uncollected jobs also move to Files deleted at shop closing or after 24 hours, whichever is first (BR-D2).

**FR-1 Walk-in drop (web)**

| ID | Requirement | Traces to | Release |
| --- | --- | --- | --- |
| FR-1.1 | Each shop has a permanent QR and short URL (`cd.in/s/<slug>`) that opens its drop page | UC-C01 | R1a |
| FR-1.2 | Drop page loads without login, shows shop name, open status and live wait | UC-C01 | R1a |
| FR-1.3 | Accept PDF, JPG, PNG, HEIC; up to 20 files and 50 MB per job (proposal) | UC-C01 | R1a |
| FR-1.4 | Upload directly to storage via presigned URL; resumable on network loss | UC-C01 | R1a |
| FR-1.5 | Detect page count for PDFs; one page per image; show thumbnails | UC-C02 | R1a |
| FR-1.6 | Per-file settings: copies, B/W or colour, single/double-sided, page range, paper size, orientation, fit-to-page | UC-C02 | R1a |
| FR-1.7 | Show price breakdown, total and ready-by before submit | UC-C03 | R1a |
| FR-1.8 | Issue a token atomically per shop, per lane and per day (e.g. `A-01` resets daily) | UC-C01 | R1a |
| FR-1.9 | Ticket link with an unguessable secret restores the job on any device | UC-C04 | R1a |
| FR-1.10 | Edit, add files or cancel until the job is claimed | UC-C07, C08 | R1a |
| FR-1.11 | ID-card mode: photograph front and back, auto-crop, place both on one A4 page | UC-C09 | R1a |

**FR-2 Queue and dashboard**

| ID | Requirement | Traces to | Release |
| --- | --- | --- | --- |
| FR-2.1 | Live queue by lane, oldest first, updated within 1 s over WebSocket | UC-S10 | R1a |
| FR-2.2 | Claim is atomic: two counters can never claim the same job | UC-S11 | R1a |
| FR-2.3 | Actions: claim, release, ready, collected, cancel, undo-collected (10 min) | UC-S11–S20 | R1a |
| FR-2.4 | Search by token, pickup code, first name; scan pickup QR with the device camera | UC-S14 | R1a |
| FR-2.5 | Pause and resume intake; custom wait message | UC-S18 | R1a |
| FR-2.6 | Rush mode toggles a simplified layout with only next jobs and big buttons | UC-S19 | R1a |
| FR-2.7 | Price override with mandatory reason, recorded in audit log | UC-S17 | R1a |
| FR-2.8 | "Ask customer" message with preset reasons and re-upload link | UC-S11 | R1a |
| FR-2.9 | Works in Chrome on Windows PC and Android; offline banner and safe retry | UC-S10 | R1a |

**FR-3 Print nearby**

| ID | Requirement | Traces to | Release |
| --- | --- | --- | --- |
| FR-3.1 | Search shops within 10 km of GPS or a chosen area/station; results in under 500 ms | UC-C11 | R1b |
| FR-3.2 | Filters: open now, colour, services, max wait, price; sort by best pick, distance, wait, price, rating | UC-C11 | R1b |
| FR-3.3 | Only shops that are online, remote-enabled, KYC-complete and not paused are orderable; others shown greyed with reason | UC-C11 | R1b |
| FR-3.4 | Shop page with prices, hours, services, photos, rating and map directions | UC-C12 | R1b |
| FR-3.5 | Live wait = jobs ahead × shop's rolling average minutes per job (last 7 days), shown as a range | UC-C11 | R1b |
| FR-3.6 | Remote order flow: files → settings → price + fee → UPI → awaiting shop → pickup code | UC-C13 | R1b |
| FR-3.7 | Pickup code: 4 digits + QR, unique per shop per day | UC-C15 | R1b |
| FR-3.8 | On reject/timeout, suggest the next best shop and re-send with files attached in one tap | UC-C14 | R1b |

**FR-4 Payments**

| ID | Requirement | Traces to | Release |
| --- | --- | --- | --- |
| FR-4.1 | UPI intent and collect via Razorpay; cards optional later | UC-C13 | R1b |
| FR-4.2 | Order moves to Awaiting shop only on a verified payment webhook, never on client callback alone | UC-C13 | R1b |
| FR-4.3 | Razorpay Route transfer to the shop's linked account, held until the shop accepts, released on accept | UC-S26 | R1b |
| FR-4.4 | Automatic full refund on reject, timeout, payment-after-cancel or shop-cancel | UC-C14 | R1b |
| FR-4.5 | Idempotent webhooks; every payment, refund and transfer has one ledger entry | UC-A05 | R1b |
| FR-4.6 | Daily reconciliation job; mismatches alert ops | UC-A05 | R1b |
| FR-4.7 | Customer receipt; shop statement of settlements, refunds and fees | UC-C24, S26 | R1b |

**FR-5 Customer app (PWA)**

| ID | Requirement | Traces to | Release |
| --- | --- | --- | --- |
| FR-5.1 | React PWA, installable (manifest + service worker), app shell under 300 KB; Android Chrome and iOS Safari 16.4+ | UC-C11 | R1b |
| FR-5.2 | Phone OTP sign-in; stays signed in 90 days | UC-C19 | R1b |
| FR-5.3 | Web Share Target (installed PWA on Android): accept files from any app and start an order | UC-C23 | R1b |
| FR-5.4 | Web push for accept, reject, ready, refund, reminders | UC-C15 | R1b |
| FR-5.5 | Order history (metadata only), reorder with same shop and settings | UC-C20 | R1b |
| FR-5.6 | Favourites, ratings, language, notification settings, delete account | UC-C21–C25 | R1b |
| FR-5.7 | Shop QR is a plain link that opens the upload-only screen, in the installed PWA if present, otherwise in the browser; the app's Scan button opens the same screen | UC-C01 | R1b |

**FR-6 Shop mode (app)**

| ID | Requirement | Traces to | Release |
| --- | --- | --- | --- |
| FR-6.1 | Same app, switch to shop mode after staff sign-in | UC-S23 | R1b |
| FR-6.2 | R1b minimal: new-order alert, accept/reject, ready, collected, go online/offline | UC-S24 | R1b |
| FR-6.3 | R2 full: lanes, claim, pause, price list, reports | UC-S10 | R2 |

**FR-7 Notifications**

| ID | Requirement | Release |
| --- | --- | --- |
| FR-7.1 | Web: live status over WebSocket; sound and vibration on Ready while the page is open | R1a |
| FR-7.2 | Web push (optional opt-in) where the browser supports it | R1a |
| FR-7.3 | App push for all remote order events | R1b |
| FR-7.4 | No SMS or WhatsApp to customers by default (cost and privacy); OTP SMS only | R1b |

**FR-8 Privacy and deletion**

| ID | Requirement | Release |
| --- | --- | --- |
| FR-8.1 | Deletion worker deletes storage objects on schedule and records deleted\_at; retries failed deletes | R1a |
| FR-8.2 | Deletion receipt shown to the customer with file count and time | R1a |
| FR-8.3 | Clear customer name on the job at file deletion | R1a |
| FR-8.4 | Audit log of every state change and file access (who, when, which job) | R1a |
| FR-8.5 | Files served to staff only via short-lived signed URLs (5 min); no download link in admin tools | R1a |
| FR-8.6 | On-device Aadhaar detection and masking (first 8 digits) before upload | R2 |

**FR-9 Shop onboarding and settings**

| ID | Requirement | Release |
| --- | --- | --- |
| FR-9.1 | Self-serve signup to live drop page in under 10 minutes, with a guided checklist | R1a |
| FR-9.2 | Printable QR kit (PDF) generated per shop | R1a |
| FR-9.3 | Price list editor with preview ("a 10-page B/W double-sided copy costs ₹X") | R1a |
| FR-9.4 | Remote-order enablement with Razorpay linked-account KYC status | R1b |

**FR-10 Admin console**

| ID | Requirement | Release |
| --- | --- | --- |
| FR-10.1 | Shop list with status, KYC, plan, activity, complaints | R1a |
| FR-10.2 | Order timeline (metadata only), manual refund within policy | R1b |
| FR-10.3 | Reconciliation and deletion-health dashboards | R1a/R1b |
| FR-10.4 | Role-based access for ops, support, finance; every admin action audited | R1a |

## 12. Business rules

These rules settle every money, time and data question in advance so that neither side ever has to argue at the counter. Values are proposals to confirm in the R1a pilot.

**Tokens and codes**

| ID | Rule |
| --- | --- |
| BR-T1 | Walk-in tokens are `<lane letter>-<number>`, starting at 01 each day per shop; 2 digits, rolling to 3 after 99. |
| BR-T2 | Remote jobs use a 4-digit pickup code plus a QR, unique per shop per day; never reused on the same day. |
| BR-T3 | Token and code issuing is atomic; no two jobs at the same shop share one on the same day. |
| BR-T4 | A token or code alone does not reveal files; staff must be signed in to open them. |

**Queue**

| ID | Rule |
| --- | --- |
| BR-Q1 | Jobs are served oldest first within a lane. Remote jobs join the lane at the time of acceptance, not payment. |
| BR-Q2 | A released job returns to the head of its lane. |
| BR-Q3 | Customers can edit or cancel until claimed; after claim only staff can change the job. |
| BR-Q4 | A claimed job with no action for 20 minutes shows a warning to the owner. |
| BR-Q5 | When intake is paused, new walk-in drops are blocked and the shop is hidden from Print nearby; existing jobs continue. |

**Pricing**

| ID | Rule |
| --- | --- |
| BR-P1 | Price = Σ per file (billable sides × per-side rate for mode and paper × copies) + add-ons. |
| BR-P2 | Shops set their own rates; Counter Drop never sets print prices. |
| BR-P3 | A minimum job charge may be set by the shop (e.g. ₹5). |
| BR-P4 | Price shown to the customer before submit is the price charged, unless staff record an override with a reason visible to the customer. |
| BR-P5 | Convenience fee (₹2–5) applies to remote orders only and is shown as a separate line; walk-in has no fee. |
| BR-P6 | Customer cancels a remote order: before accept, full refund; after accept but before claim, refund minus the convenience fee; after claim, no refund unless the shop agrees. |
| BR-P7 | Shop rejects, times out or cancels: full refund including the fee. |
| BR-P8 | Not collected: printout kept by the shop for 3 days (proposal); no refund after printing; files are still deleted on schedule. |
| BR-P9 | Wrong print (settings not followed): shop reprints free or approves a refund of that job; ops decides disputes within 48 h. |

**Payments and settlement**

| ID | Rule |
| --- | --- |
| BR-M1 | Walk-in payment happens directly between customer and shop (cash or shop UPI); Counter Drop does not touch it. |
| BR-M2 | Remote payments go through Razorpay; print amount is transferred to the shop's linked account via Route; the fee stays with Counter Drop. |
| BR-M3 | The transfer is held until the shop accepts; a reject reverses it before release. |
| BR-M4 | Settlement to the shop follows Razorpay's cycle (T+2 default, proposal); shops see expected date per order. |
| BR-M5 | Refunds are initiated automatically within 60 seconds of the triggering event; the customer sees the refund reference. |
| BR-M6 | Counter Drop never keeps a wallet balance for any customer or shop. |

**Remote-order reliability**

| ID | Rule |
| --- | --- |
| BR-R1 | Shop must accept or reject within 5 minutes; otherwise it auto-rejects. |
| BR-R2 | A remote order is only offered when the shop will be open for at least 30 minutes after the ready-by time. |
| BR-R3 | Max files and pages per remote order are set by the shop (default 20 files, 200 pages). |
| BR-R4 | A timeout hides the shop from Print nearby for 30 minutes; 3 timeouts in 7 days trigger an owner nudge; 5 trigger a review. |
| BR-R5 | Acceptance rate and on-time-ready rate feed the "best pick" ranking. |

**Data retention**

| ID | Rule |
| --- | --- |
| BR-D1 | Files are deleted 10 minutes after Collected (undo window). |
| BR-D2 | Files not collected are deleted at shop closing or 24 hours after upload, whichever is first. |
| BR-D3 | Files of cancelled, rejected or refunded jobs are deleted immediately. |
| BR-D4 | Customer name on a job is cleared when its files are deleted. |
| BR-D5 | Order and payment records (amounts, times, shop, masked phone) are kept for 8 years for tax and payment-law compliance; they never contain file content. |
| BR-D6 | Customer location is used for search only and never stored against the customer. |
| BR-D7 | Reorder repeats shop and settings only; the customer picks the files again from the phone. |
| BR-D8 | Audit log entries contain IDs and actions, never file content; kept for 1 year (proposal, to confirm with legal). |

## 13. Edge cases and exception handling

Every failure has an automatic recovery and a plain-language message; no one should ever see an error code or lose money or a job.

**Customer side**

| ID | Situation | System behaviour | Customer sees |
| --- | --- | --- | --- |
| EX-C01 | Network drops mid-upload | Resume from last chunk; job not submitted until all files confirmed in storage | "Waiting for network — we'll continue automatically" |
| EX-C02 | File too large | Block before upload; suggest compressing or splitting | "This file is 80 MB. The limit is 50 MB." |
| EX-C03 | Password-protected or corrupt PDF | Detected on upload; job not sent | "This PDF is locked. Remove the password and try again." |
| EX-C04 | Unsupported format (DOCX in R1) | Blocked with a tip | "Save it as PDF from Word or Google Docs, then upload." |
| EX-C05 | Page count can't be read | Job accepted with "pages to confirm"; staff confirm price at counter | "Price will be confirmed at the counter." |
| EX-C06 | Customer submits the same file twice in 2 min | Warn before creating a duplicate | "You already sent this file (A-07). Send again?" |
| EX-C07 | Browser closed or phone restarted | Ticket restored from link or local storage | Same ticket, same status |
| EX-C08 | QR scanned for a shop that's closed | Show hours; suggest open nearby shops | "Opens at 9:00 am. 3 shops open near you." |
| EX-C09 | UPI app debits but callback is lost | Order waits for webhook; poll status; auto-refund if not captured in 15 min | "Confirming your payment…" then success or refund |
| EX-C10 | Double payment for one order | Second payment refunded automatically | "You were charged twice. ₹X refunded." |
| EX-C11 | Customer arrives before Ready | Staff see the job state; can prioritise if nearly done | Status shows position and estimate |
| EX-C12 | Customer lost pickup code | Staff search by first name + last 4 phone digits (remote only) | — |
| EX-C13 | Someone else shows the token | Token plus first name shown to staff; remote needs the code | — |
| EX-C14 | GPS denied | Ask for area or station; list works without GPS | "Choose your area" |
| EX-C15 | No shops nearby or all busy | Show nearest with wait; invite to notify when one opens | "No open shops within 2 km. Nearest: 3.4 km, 10 min wait." |
| EX-C16 | Refund delayed at the bank | Show refund reference and bank timeline; support link | "Refund sent on 3 Oct. Banks take up to 5 working days." |
| EX-C17 | Very old phone or browser | Basic upload page without live updates; refresh button | Works, simpler |

**Shop side**

| ID | Situation | System behaviour | Staff sees |
| --- | --- | --- | --- |
| EX-S01 | Two counters tap Claim at the same moment | Atomic claim; loser gets the next job | Always a different job |
| EX-S02 | Accidental Collected | Undo within 10 min; deletion waits | Undo button with countdown |
| EX-S03 | Printer jam or wrong print | Release; job returns to head of lane | "Returned to top of Lane A" |
| EX-S04 | Internet down at the shop | Dashboard shows last queue read-only; actions queue and retry; remote intake pauses automatically after 2 min offline | "Offline — remote orders paused" |
| EX-S05 | Shop forgets to go offline at closing | Auto-offline at set closing time; uncollected jobs flagged | Closing summary |
| EX-S06 | Staff phone misses the remote-order push | Dashboard chime, repeat push at 2 and 4 min, then timeout | Countdown on card |
| EX-S07 | Remote order with a file the shop can't print | Reject with "Can't print this file"; auto refund | "Refund sent, no charge to you" |
| EX-S08 | Customer disputes the price | Show the price the customer accepted and the per-file breakdown | Breakdown screen |
| EX-S09 | Customer wants extra service at counter | Price override with reason; walk-in pays at counter; remote pays difference at counter | Audit entry |
| EX-S10 | Staff leaves the job | Owner removes staff; sessions revoked at once | — |
| EX-S11 | Suspicious volume (spam uploads) | Rate-limit per device and IP; owner can block a ticket | "Blocked" tag |
| EX-S12 | Shop KYC rejected or payouts on hold | Remote orders disabled; walk-in continues | Banner with next step |

**Platform side**

| ID | Situation | System behaviour |
| --- | --- | --- |
| EX-P01 | Storage delete fails | Retry with backoff; alert after 3 failures; appears in deletion health |
| EX-P02 | Payment webhook delayed or duplicated | Idempotent processing by payment ID; reconciliation catches gaps |
| EX-P03 | Razorpay outage | Remote ordering paused with a banner; walk-in unaffected |
| EX-P04 | FCM outage | Web status still live; app polls when opened |
| EX-P05 | Region or database failover | Queue state restored from database; no job lost; tokens continue |
| EX-P06 | Abuse: illegal or harmful content reported by a shop | Shop can refuse and report; job cancelled; files deleted; metadata kept for the report only |

## 14. UX requirements and delight moments

People will love Counter Drop if it feels faster than WhatsApp, never makes them feel stupid and rewards them at the moments that used to hurt.

**Customer UX requirements**

| ID | Requirement |
| --- | --- |
| UX-C1 | Walk-in happy path in 3 taps after scan: **Choose files → Send to counter → (done)**. Settings have smart defaults and are optional. |
| UX-C2 | Primary button always at the bottom, thumb-reachable, at least 48 px tall, one per screen. |
| UX-C3 | Price, wait and ready-by visible on the same screen as the Send/Pay button. |
| UX-C4 | Token shown huge (at least 64 px) so staff can read it from across the counter. |
| UX-C5 | Status uses colour, icon and words together (for colour-blind users and low literacy). |
| UX-C6 | Language picker on the first screen; remembers the choice; auto-detects from phone settings. |
| UX-C7 | All copy in plain words, reviewed by native Hindi and Marathi speakers; no jargon ("duplex" → "both sides"). |
| UX-C8 | Errors say what happened and what to do next, in one sentence, with a single action. |
| UX-C9 | Accessible: WCAG 2.1 AA contrast, screen-reader labels, scalable text up to 200%. |
| UX-C10 | Loading states show progress per file; nothing freezes. |
| UX-C11 | Never ask for a phone number on walk-in; ask for it in the app only when paying. |
| UX-C12 | Remote flow in one screen per step: shop → files → pay → wait → collect; back button never loses files. |

**Shop UX requirements**

| ID | Requirement |
| --- | --- |
| UX-S1 | Dashboard readable from 1 metre: large tokens, high contrast, dark theme option for dim shops. |
| UX-S2 | Every counter action is one tap; no confirmation dialogs except Cancel. Undo instead of "Are you sure?". |
| UX-S3 | Keyboard shortcuts on PC: `N` claim next, `R` ready, `C` collected, `/` search. |
| UX-S4 | Distinct sounds: new walk-in (soft), new remote order (loud, repeating until seen). |
| UX-S5 | Rush mode reduces the screen to the next 3 jobs and the Ready column. |
| UX-S6 | Setup checklist with progress ("3 of 5 done") and a test job button that shows the whole flow in 1 minute. |
| UX-S7 | Staff can learn the dashboard in under 5 minutes without training (verified in usability tests). |
| UX-S8 | Hindi and Marathi dashboard; numbers always in Western digits for speed. |

**Delight moments**

| Moment | Pain it replaces | What we do |
| --- | --- | --- |
| Share from WhatsApp straight to Counter Drop | Forwarding files to a stranger | Share sheet (installed PWA on Android): tap Share → Counter Drop → nearest shop pre-selected |
| "Ready" buzz while still on the train | Waiting at the counter | Push with walking distance and one-tap directions |
| Price before you pay, always | Arguments at the counter | Breakdown the shop has already agreed to |
| Deletion receipt | Fear about Aadhaar copies | "3 files deleted at 6:52 pm" with a shield icon; shareable |
| ID-card mode | Two copies, cut and paste | Front + back on one A4 automatically |
| "Send to another shop" in one tap | Starting over after a rejection | Files and settings carried over |
| Walk-in with no app | Installing yet another app | Works in the browser in 30 seconds |
| Shopkeeper's first rush hour | Chaos, shouting | Rush mode + TV screen; customers watch their token, not the staff |
| Shopkeeper's WhatsApp back | Phone full of strangers' files | "You got 212 jobs this month without a single WhatsApp file" in the monthly summary |
| New customers | Only walk-in footfall | "14 new customers found you on Print nearby this week" |

**Research and validation**

- 5 usability sessions per persona before R1a (walk-in) and before R1b (remote), in the customer's language.
- Shadow 3 shops for one peak hour each before and after go-live to measure counter time (BO-1).
- In-app NPS after the 2nd collected job; shop NPS monthly.

## 15. Non-functional requirements

The system must feel instant on a budget phone, never lose a job and stay under ₹2,000 a month until 50 paying shops.

| ID | Category | Requirement | Target |
| --- | --- | --- | --- |
| NFR-1 | Availability | Monthly uptime of API, drop page and dashboard | 99.5% on paid plans (about 3.6 h/month) |
| NFR-2 | Performance | Drop page interactive on 4G, mid-range Android | Under 2 s; page weight under 200 KB |
| NFR-3 | Performance | API p95 latency for queue and job actions | Under 300 ms |
| NFR-4 | Performance | Nearby search within 10 km | Under 500 ms p95 |
| NFR-5 | Real-time | Queue and status updates to open clients | Under 1 s p95 |
| NFR-6 | Real-time | Push sent after state change | Within 5 s p95 |
| NFR-7 | Concurrency | Atomic claims and token issue | Tested: 3 counters, 500 jobs, zero duplicates |
| NFR-8 | Scale | Year-one capacity without re-architecture | 1,000 shops, 20,000 live connections, 50,000 jobs/day |
| NFR-9 | Upload | Large file handling | 50 MB per job, resumable, direct to storage |
| NFR-10 | Durability | No job lost after submit | Persisted before acknowledgement; RPO 5 min, RTO 1 h |
| NFR-11 | Security | Transport and storage | TLS 1.2+; storage encrypted at rest; signed URLs expire in 5 min |
| NFR-12 | Security | Authentication | Staff: phone OTP + 4-digit PIN per device; owner sessions revocable; customers: OTP for app/remote |
| NFR-13 | Security | Authorisation | Shop-scoped access on every query; no cross-shop data; admin cannot read files |
| NFR-14 | Security | Abuse protection | Rate limits per IP/device; malware scan on upload (R2); OWASP ASVS L1 |
| NFR-15 | Privacy | Deletion SLA | 99.9% of files deleted within 5 min of the scheduled time; 100% within 1 h |
| NFR-16 | Compatibility | Customer web | Chrome/Android 8+, Safari/iOS 14+, Samsung Internet |
| NFR-17 | Compatibility | Dashboard | Chrome/Edge on Windows 10+, Chrome on Android |
| NFR-18 | App | Size and start | Shell under 300 KB, cached by the service worker; launch under 2 s |
| NFR-19 | Localisation | Languages | English, Hindi, Marathi; all strings externalised |
| NFR-20 | Accessibility | Standard | WCAG 2.1 AA |
| NFR-21 | Observability | Monitoring | Health checks, error tracking, SLO dashboards, alert on deletion or payment failures |
| NFR-22 | Cost | Running cost | Under ₹2,000/month until 50 paying shops; cost per job tracked |
| NFR-23 | Architecture | One contract | Web, app and print agent use the same versioned API (`/api/v1/cd`) |
| NFR-24 | Independence | Separation from SwiftShare | Separate repo, database, storage prefix and deployment |

**Architecture notes (for the solution design)**

- Go API with Postgres; job state machine in the domain layer; atomic claim via `UPDATE … WHERE state='queued' … RETURNING` or `SELECT … FOR UPDATE SKIP LOCKED`.
- Object storage (Cloudflare R2 or S3) with presigned uploads and lifecycle rules as a backstop to the deletion worker.
- WebSocket hub per shop for queue events; Redis pub/sub only when running more than one API instance.
- PostGIS (or a geohash index) on shop locations for nearby search.
- Razorpay Route for split settlement; webhook consumer with an idempotency table.
- Firebase Cloud Messaging for app push; web push for opted-in browsers.

## 16. Data, privacy and compliance

Counter Drop keeps file content for minutes, personal details for hours and money records for years, and keeps these three layers separate.

**Data inventory and retention**

| Data | Source | Purpose | Retention | Who can see it |
| --- | --- | --- | --- | --- |
| Uploaded files | Customer | Printing | Deleted per BR-D1 to D3 (max 24 h) | Signed-in staff of that shop, via 5-min signed URL |
| Job details (settings, pages, price, token) | Customer, system | Queue, pricing | Kept; name cleared at file deletion | Shop staff, customer (own), admin (metadata) |
| Customer first name (walk-in) | Customer, optional | Calling out the job | Cleared at file deletion | Shop staff |
| Customer phone (app/remote) | Customer | Sign-in, receipts | Until account deletion; masked on shop screens | Customer; admin (masked) |
| Payment records | Razorpay | Settlement, refunds, tax | 8 years (proposal, confirm with legal) | Finance, shop (own), customer (own) |
| Location | Customer device | Nearby search | Not stored against the customer | No one |
| Audit log | System | Accountability | 1 year (proposal) | Owner (own shop), admin |
| Shop KYC | Owner via Razorpay | Payouts | Held by Razorpay; we store status only | Admin (status) |

**Compliance objectives**

| ID | Objective | Measure |
| --- | --- | --- |
| CT-1 | Keep files only as long as needed | Deleted 10 min after pickup; at closing if not collected; 24 h maximum |
| CT-2 | Minimise personal data | Name cleared at file deletion; no location stored; phone only for app users |
| CT-3 | Unmasked Aadhaar never reaches the shop or our servers (R2) | On-device detection and masking |
| CT-4 | Record every access and deletion | 100% of state changes and file accesses logged |
| CT-5 | Be DPDP-ready before 14 May 2027 | Legal review, privacy notice, consent text and a data processing agreement for shops before the paid launch |
| CT-6 | Handle payments safely | All payments via Razorpay (regulated); Route for splits; no payment aggregator licence needed; no stored funds |
| CT-7 | Honour data rights | Access and deletion requests answered within 30 days (proposal); in-app delete account |
| CT-8 | Breach readiness | Incident runbook; notify affected users and the Data Protection Board as the Rules require |

**Roles under DPDP (to confirm with legal).** For uploaded files, the shop decides why and how files are used (printing), so the shop is likely the *Data Fiduciary* and Counter Drop its *Data Processor* under the shop DPA. For app accounts and payments, Counter Drop is the Data Fiduciary. Consent text on the drop page states the purpose ("to print at \<shop>") and the deletion promise.

**Security controls summary**

- Files never pass through WhatsApp, email or the shop's personal phone storage.
- No file previews are cached on shop devices beyond the browser session.
- Signed URLs are single-shop and short-lived; every open is logged.
- Staff PINs per device; owner can revoke a device instantly.
- Annual external security review before the business tier launches.

## 17. Revenue model and pricing

Revenue comes from shop subscriptions and a small customer fee on remote orders; Counter Drop never takes a cut of print money. All prices are proposals to confirm in the pilot.

**Shop plans (proposal)**

| Plan | Price / month | For | Includes |
| --- | --- | --- | --- |
| Free | ₹0 | Trying it out | Walk-in drop, 1 counter, up to 300 jobs/month, auto-delete, audit log, Print nearby listing |
| Pro | ₹199 | Most shops | Unlimited jobs, 3 lanes, rush mode, remote orders, shop mode app, reports |
| Plus | ₹499 | Busy station shops | Everything in Pro, print agent, TV display, 10 staff, priority support, featured placement rules |
| Business | Custom | Colleges, CSC networks, offices | Multi-branch, SSO, org audit, custom retention, invoicing |

- Pilot shops get Pro free for 3 months, then 50% off for 3 months.
- Annual billing: 2 months free.
- Auto-delete and the audit log are on every plan, including Free (SO-11).

**Customer convenience fee (remote only)**

| Order total | Fee (proposal) |
| --- | --- |
| Up to ₹50 | ₹2 |
| ₹51–200 | ₹3 |
| Above ₹200 | ₹5 |

The fee must cover Razorpay charges on the order and leave a margin; exact numbers to be set after confirming Razorpay pricing for UPI and Route.

**Unit economics to validate in the pilot**

- Average jobs per shop per day, pages per job and job value.
- Share of remote orders and average fee revenue per shop.
- Payment processing cost per remote order vs fee.
- Hosting, storage, OTP and push cost per job (target under ₹0.10).
- Willingness to pay: % of pilot shops converting at ₹199.

## 18. Go-to-market, onboarding and support

Shops are signed up one station cluster at a time, because Print nearby only works when a customer sees several open shops within walking distance.

**Launch sequence**

1. **R1a pilot:** 10 shops, split across Dadar, Andheri and Thane, recruited in person. Measure counter time before and after.
2. **Density push (during R1a):** list at least 15 shops in one cluster (Dadar first) on the Free plan so Print nearby has coverage on day one of R1b.
3. **R1b launch in Dadar:** app published; posters in listed shops ("Order ahead, skip the line"); student outreach at nearby colleges.
4. **Expand** to Andheri and Thane once Dadar hits 20% remote share or 60 days, whichever is first.

**Shop acquisition**

- Field visits with a 2-minute demo on the owner's own phone: scan, upload, see the job appear.
- Referral: a shop that refers another gets 1 month of Pro free.
- Free QR kit printed and delivered to the first 50 shops.

**Customer acquisition**

- Every walk-in drop page shows "Next time, order ahead with the app" after pickup (never before).
- QR posters in shops and colleges; student ambassadors near campuses.
- First remote order: convenience fee waived.

**Shop onboarding checklist (target: live in under 10 minutes)**

- [ ] Sign up with phone OTP
- [ ] Pin shop location and set hours
- [ ] Set prices (templates for typical Mumbai rates)
- [ ] Print the QR kit
- [ ] Send a test job from your own phone
- [ ] (For remote orders) Complete payout KYC

**Support model**

| Channel | For | Target response |
| --- | --- | --- |
| In-app help and FAQ (3 languages) | Everyone | Self-serve |
| WhatsApp Business support number | Shops | Within 30 min, 9 am–9 pm |
| In-app "Report a problem" on an order | Customers | Within 4 h; refunds within policy automatic |
| Phone callback | Paid-plan shops during pilot | Same day |

Support never asks customers or shops to send documents; issues are resolved from order metadata.

## 19. Assumptions, constraints and dependencies

The plan depends on shops having a smartphone, customers using Android and UPI, and Razorpay onboarding shops quickly enough.

**Assumptions**

| ID | Assumption | How we validate |
| --- | --- | --- |
| AS-1 | Customers have a smartphone with mobile data; most use Android | Pilot device analytics |
| AS-2 | Shops have a smartphone; most also have a Windows PC for printing | Shop survey at signup |
| AS-3 | Shops will pay about ₹99–499 a month, based on competitor pricing | Pilot conversion (BO-5) |
| AS-4 | Customers will pay ₹2–5 to skip the line | Remote order conversion with and without fee |
| AS-5 | Shops can respond to remote orders within 5 minutes | Pilot acceptance times |
| AS-6 | UPI is acceptable for prepayment for most customers | Payment success rate |
| AS-7 | A cluster needs 15+ listed shops for Print nearby to feel useful | Search-to-order conversion by cluster density |

**Constraints**

- Launch city: Mumbai only until BO-5 is met.
- Budget: running cost under ₹2,000/month until 50 paying shops.
- Team: small; Release scope must fit 8 weeks (R1a) and 10 weeks (R1b).
- One PWA for Android and iPhone from R1b; a Play Store wrapper in R3.
- Counter Drop must not hold funds or need a payment aggregator licence.
- Separate from SwiftShare in code, data and deployment.

**Dependencies**

| Dependency | Needed for | Risk if late |
| --- | --- | --- |
| Razorpay account, Route and linked-account KYC | Remote payments | R1b blocked |
| Object storage (Cloudflare R2 or S3) | Uploads | R1a blocked |
| Firebase Cloud Messaging | App push | Ready alerts weaker |
| SMS OTP provider (DLT-registered templates) | App sign-in | App sign-in blocked |
| Google Play developer account (R3 wrapper only) | App release | Play listing slips; PWA unaffected |
| Maps / geocoding provider | Nearby search, directions | Search by area only |
| Legal review (DPDP, terms, shop DPA) | Paid launch | Paid launch slips |
| Hindi and Marathi translators | Launch copy | English-only launch |

## 20. Risks and mitigations

The biggest risks are shops slipping back to WhatsApp and Print nearby launching without enough shops; both are addressed by sequencing, not features.

| ID | Risk | Likelihood | Impact | Mitigation | Owner |
| --- | --- | --- | --- | --- | --- |
| RK-1 | Shops keep taking WhatsApp files out of habit | High | High | Walk-in faster than WhatsApp; "No WhatsApp files" counter sign in the QR kit; weekly usage check-ins in the pilot | Product |
| RK-2 | Too few shops per cluster; Print nearby feels empty | High | High | Density push in one cluster before R1b; free listing; launch in Dadar only | Growth |
| RK-3 | Shops miss or ignore remote orders | Medium | High | Loud alerts, repeat push, 5-min auto-refund, ranking penalty, pause option | Product |
| RK-4 | Razorpay KYC delays for small shops | Medium | High | Start KYC during R1a; walk-in works without KYC; ops helps with documents | Ops |
| RK-5 | Customers unwilling to pay a fee | Medium | Medium | Waive first fee; test ₹0 vs ₹2 vs ₹3 in pilot | Product |
| RK-6 | Competitor copies features | High | Medium | Win on density, rush mode, privacy and shop relationships | Founder |
| RK-7 | File deletion fails silently | Low | High | Worker + storage lifecycle rule backstop; deletion health alerts; daily audit | Tech |
| RK-8 | Payment and refund mismatches | Medium | High | Idempotent webhooks, ledger, daily reconciliation, alerts | Tech |
| RK-9 | DPDP interpretation changes or date moves | Medium | Medium | Legal review before paid launch; design already minimises data | Legal |
| RK-10 | Shop internet is unreliable | High | Medium | Offline banner, safe retry, auto-pause remote orders, phone as backup | Tech |
| RK-11 | Misuse: printing illegal or forged documents | Low | High | Shop can refuse and report; terms of use; metadata kept for reports | Legal |
| RK-12 | Running cost exceeds budget | Low | Medium | Direct-to-storage uploads, short retention, no SMS notifications, cost per job tracked | Tech |
| RK-13 | Scope too large for the team | High | High | R1a/R1b split; strict MoSCoW; defer shop mode extras, agent and TV to R2 | Product |

## 21. Success metrics and acceptance criteria

Each release ships only when its acceptance criteria pass; the metrics below are tracked weekly from the first pilot day.

**Metrics and how they are measured**

| Metric | Definition | Source | Cadence |
| --- | --- | --- | --- |
| Counter time per job | Customer reaching the counter to leaving with prints | Shadowing (before/after) + claimed→collected time | Pilot weeks 0, 4, 8 |
| Jobs per peak hour | Jobs collected 8–11 am and 5–8 pm | Job events | Weekly |
| Scan-to-token time | QR page load to token shown | Web analytics | Weekly |
| Submit drop-off | Files picked but job not submitted | Web analytics | Weekly |
| Remote share | Remote jobs ÷ all jobs at remote-enabled shops | Job events | Weekly |
| Acceptance rate and time | Accepted ÷ remote orders; median seconds to accept | Job events | Weekly |
| Refund rate | Refunded ÷ paid remote orders | Ledger | Weekly |
| Deletion SLA | Files deleted within 5 min of schedule | Deletion worker | Daily |
| Repeat customers | Customers with 2+ jobs in 30 days | App accounts + ticket links | Monthly |
| NPS | Customer and shop NPS | In-app survey | Monthly |
| Paying shops, churn | Active paid subscriptions; cancellations ÷ start-of-month | Billing | Monthly |
| Cost per job | Infra + messaging cost ÷ jobs | Cloud bills | Monthly |

**R1a acceptance criteria (walk-in pilot)**

- [ ] 10 pilot shops live and taking jobs daily.
- [ ] Walk-in drop works on Chrome/Android and Safari/iOS with no login.
- [ ] Token issue and claim tested with 3 counters and 500 jobs: zero duplicates.
- [ ] Zero lost jobs and zero double prints in the audit log over 4 weeks.
- [ ] 100% of files deleted per BR-D1 to D3; deletion receipts shown.
- [ ] Counter time measured before and after in all 10 shops.
- [ ] English, Hindi and Marathi copy reviewed by native speakers.
- [ ] Staff login, audit log and admin deletion-health view live.

**R1b acceptance criteria (Print nearby and app)**

- [ ] At least 15 listed shops in Dadar, with at least 8 remote-enabled (KYC complete).
- [ ] PWA installable on Android and iPhone; in-app QR scan and install banner working.
- [ ] Remote orders paid, accepted, printed and collected end to end in production.
- [ ] Reject and timeout refunds verified automatically within 60 s.
- [ ] Daily reconciliation shows zero unexplained mismatches for 2 weeks.
- [ ] Nearby search p95 under 500 ms; push sent within 5 s p95.
- [ ] Legal review of terms, privacy notice and shop DPA complete.
- [ ] 25 shops paying after the free period (Gate 2).

## 22. Open decisions, glossary and approvals

Seven decisions need an owner before development of R1b starts; the first one shapes the whole plan.

**Open decisions**

| # | Decision | Options | Recommendation | Decide by |
| --- | --- | --- | --- | --- |
| D-1 | Release 1 shape | One big R1 vs R1a pilot then R1b | R1a then R1b (this BRD) | Before R1a starts |
| D-2 | When the shop's money is released | On accept vs on collection | On accept (simpler, shop-friendly) | Before R1b build |
| D-3 | No-show policy for prepaid orders | Shop keeps money / partial refund / full refund | Shop keeps print amount after printing (BR-P8) | Pilot week 6 |
| D-4 | Customer app login | Phone OTP vs Google sign-in vs both | Phone OTP (needed for receipts anyway) | Before R1b build |
| D-5 | Undo window before deletion | 10 min (BRD) vs 15 min (current code) | 10 min; update code | Now |
| D-6 | Convenience fee level | ₹0 / ₹2 / ₹3 / tiered | Test in pilot; start tiered | Pilot week 6 |
| D-7 | First Print nearby cluster | Dadar / Andheri / Thane | Dadar (highest footfall) | Pilot week 4 |

**Glossary**

| Term | Meaning |
| --- | --- |
| Walk-in drop | Customer uploads at the shop by scanning its QR, in the browser, with no login |
| Print nearby | Remote ordering: find a shop, upload and prepay, then collect |
| Token | Walk-in job number such as `A-07`, reset daily |
| Pickup code | 4-digit code + QR for a remote order |
| Lane | A sub-queue in a shop, e.g. B/W or colour |
| Claim | Staff takes a job to print; only one counter can claim it |
| Rush mode | Simplified dashboard for peak hours |
| Pause intake | Stop new jobs temporarily |
| Deletion receipt | Proof shown to the customer that files were deleted |
| Route | Razorpay feature that splits a payment to a shop's linked account |
| DPDP | Digital Personal Data Protection Act 2023 and its Rules |
| DPA | Data processing agreement between Counter Drop and a shop |
| Station cluster | Shops around one railway station area |

**Approvals**

| Role | Name | Decision | Date |
| --- | --- | --- | --- |
| Product owner / founder |  |  |  |
| Tech lead |  |  |  |
| Design lead |  |  |  |
| Legal advisor |  |  |  |
