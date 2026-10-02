# Launch Runbook

> **Production on AWS (current plan):** one CDK stack, deployed from GitHub Actions — see [`deploy/cdk/README.md`](../deploy/cdk/README.md). AWS services only for the MVP (S3 for files, no Cloudflare R2). The single-server options below remain for local or self-hosted pilots.


This is the controlled-pilot launch path for Counter Drop R1a.

## Launch Definition

Ready for launch means one real shop can use Counter Drop for walk-in jobs with:

- HTTPS public URL for the PWA and API.
- DynamoDB table `cd-main` with point-in-time recovery on (`cdadmin create-table`).
- File storage with either Cloudflare R2/S3 or a persistent VM disk.
- Owner onboarding through `cdadmin` setup links.
- Customer QR upload from real phones.
- Shop queue actions: claim, ready, collected, cancel, copies deleted.
- Automatic deletion worker running continuously.

Not included in R1a launch: phone OTP, public self-signup, UPI prepay, Print Nearby, Windows print agent, admin console.

## Preflight

Run locally before every pilot deploy:

```bash
make launch-check
```

This runs API tests, the web production build, and the production Docker image build.

## Production Environment

Copy the template:

```bash
cp deploy/prod.env.example deploy/prod.env
```

Fill these first:

```text
CD_DYNAMODB_TABLE=cd-main            # created once with: cdadmin create-table
AWS_ACCESS_KEY_ID=...                # IAM user limited to the table and files bucket
AWS_SECRET_ACCESS_KEY=...
CD_DOMAIN=your-domain.example        # for the bundled HTTPS (Caddy)
ACME_EMAIL=you@your-domain.example
VITE_OPERATOR_NAME=Your Company Name # shown on /privacy and /terms
VITE_SUPPORT_EMAIL=support@your-domain.example
CD_PUBLIC_API_URL=https://your-domain.example
CD_PUBLIC_WEB_URL=https://your-domain.example
CD_WEB_ORIGINS=https://your-domain.example
CD_STORAGE_SIGNING_KEY=...
```

Generate the signing key with:

```bash
openssl rand -hex 32
```

Use `CD_ENV=prod` and `CD_DEMO_SEED=false` for real launch.

## Storage Choice

Recommended for pilot: Cloudflare R2 or another S3-compatible bucket. Set all of these:

```text
CD_STORAGE_ENDPOINT=
CD_STORAGE_BUCKET=
CD_STORAGE_ACCESS_KEY=
CD_STORAGE_SECRET_KEY=
```

If using local disk for a one-VM pilot, keep `/data/files` on a persistent volume and back it up. The signed links depend on `CD_STORAGE_SIGNING_KEY`; do not change it during a pilot.

## VM Docker Compose Deploy

From the server (Ubuntu with Docker; DNS A record for `CD_DOMAIN` pointing at it; ports 80 and 443 open):

```bash
git pull
cd deploy
docker compose --env-file prod.env -f docker-compose.prod.yml --profile https up -d --build
```

`--profile https` starts Caddy, which obtains and renews a Let's Encrypt certificate for `CD_DOMAIN` automatically and forwards to the app. The app itself listens only on `127.0.0.1:8080`. Keep `CD_TRUST_PROXY=true` so rate limits see real client IPs.

Check health:

```bash
curl -fsS https://your-domain.example/health
curl -fsS https://your-domain.example/ready
```

Using your own HTTPS layer instead (Nginx, Cloudflare Tunnel, an ALB)? Leave out `--profile https` and point it at `127.0.0.1:8080`. The browser-facing origin must match `CD_PUBLIC_API_URL` and `CD_PUBLIC_WEB_URL`.

**Quick HTTPS for phone testing without a server:** `cloudflared tunnel --url http://localhost:18080` prints a temporary `https://….trycloudflare.com` address; set `CD_PUBLIC_API_URL`/`CD_PUBLIC_WEB_URL` to it and restart the API. Camera scanning and app install need HTTPS.

**Backups:** the table is the only thing to back up (files are deleted by design). Point-in-time recovery keeps 7 days of continuous backups; `cdadmin create-table` turns it on. Restoring creates a new table you then point `CD_DYNAMODB_TABLE` at.

## Create First Real Shop

```bash
cd api
CD_DYNAMODB_TABLE=cd-main CD_DYNAMODB_REGION=ap-south-1 \
CD_PUBLIC_WEB_URL='https://your-domain.example' \
go run ./cmd/cdadmin create-shop \
  -slug shop-slug \
  -name "Shop Name" \
  -address "Shop Address" \
  -owner "Owner Name"
```

Send the printed setup link to the owner. The owner chooses their own PIN. Avoid weak PINs such as `1234`, `1111`, `0000`, or `1212`.

## Phone Pilot Script

1. Owner logs in at `/shop/login`.
2. Owner opens `/shop/qr` and prints or displays the QR.
3. Customer scans QR on Android Chrome and iPhone Safari.
4. Customer uploads one PDF and one image.
5. Customer adds another file before pressing Send.
6. Customer sends to counter and sees a token.
7. Customer taps ready-sound enable on the ticket.
8. Shop claims the job.
9. Shop opens/prints or downloads each file.
10. Shop marks ready; customer gets live update, vibration, and sound if enabled.
11. Shop marks collected with payment method.
12. Customer confirms deletion receipt after the undo window.
13. If the shop downloaded a file, customer requests copy deletion and shop confirms it.

## Go/No-Go Checklist

- API `/health` and `/ready` return 200 over HTTPS.
- PWA installs or opens standalone on Android Chrome; iPhone shows the "Add to Home Screen" hint.
- In-app **Scan shop QR** opens the camera and accepts only your shop codes.
- `/privacy` and `/terms` show your company name and support email (have them reviewed by a lawyer before a public launch).
- Security headers present: `curl -sI https://your-domain.example/ | grep -i -E "content-security|strict-transport|x-frame"`.
- Rate limits work behind the proxy: 11 quick wrong logins give HTTP 429.
- iPhone Safari can scan QR, upload, submit, and view ticket.
- Upload URLs use the public HTTPS API origin, not `localhost`.
- Queue live updates arrive without manual refresh.
- Ready sound works after the customer taps the sound button.
- Files are deleted after collection and deletion status is visible.
- `GET /api/v1/cd/shop/deletion-health` shows no failed backlog for owner.
- DynamoDB point-in-time recovery is on.
- File storage volume/bucket exists and is not temporary.
- Demo seed is off for production.

## Rollback

Keep the previous Docker image tag or commit SHA. To rollback on a VM:

```bash
docker compose --env-file prod.env -f deploy/docker-compose.prod.yml down
git checkout <previous-good-sha>
docker compose --env-file deploy/prod.env -f deploy/docker-compose.prod.yml up -d --build
```

Do not delete the DynamoDB table or file volumes during rollback. Data needs no migration step: old and new versions read the same items.
