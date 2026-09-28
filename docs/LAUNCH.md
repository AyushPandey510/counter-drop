# Launch Runbook

This is the controlled-pilot launch path for Counter Drop R1a.

## Launch Definition

Ready for launch means one real shop can use Counter Drop for walk-in jobs with:

- HTTPS public URL for the PWA and API.
- Postgres with persistent storage and backups.
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
POSTGRES_PASSWORD=...
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

From the server:

```bash
git pull
make build-image
cd deploy
docker compose --env-file prod.env -f docker-compose.prod.yml up -d --build
```

Check health:

```bash
curl -fsS https://your-domain.example/health
curl -fsS https://your-domain.example/ready
```

Put Caddy, Nginx, Cloudflare Tunnel, Render, Fly, or another HTTPS layer in front of port `8080`. The browser-facing origin must match `CD_PUBLIC_API_URL` and `CD_PUBLIC_WEB_URL`.

## Create First Real Shop

```bash
cd api
CD_DATABASE_URL='postgres://counter_drop:<password>@<host>:5432/counter_drop?sslmode=require' \
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
- PWA installs or opens standalone on Android Chrome.
- iPhone Safari can scan QR, upload, submit, and view ticket.
- Upload URLs use the public HTTPS API origin, not `localhost`.
- Queue live updates arrive without manual refresh.
- Ready sound works after the customer taps the sound button.
- Files are deleted after collection and deletion status is visible.
- `GET /api/v1/cd/shop/deletion-health` shows no failed backlog for owner.
- Database volume/backup exists.
- File storage volume/bucket exists and is not temporary.
- Demo seed is off for production.

## Rollback

Keep the previous Docker image tag or commit SHA. To rollback on a VM:

```bash
docker compose --env-file prod.env -f deploy/docker-compose.prod.yml down
git checkout <previous-good-sha>
docker compose --env-file deploy/prod.env -f deploy/docker-compose.prod.yml up -d --build
```

Do not delete Postgres or file volumes during rollback.
