# 07 — Auth: phone OTP, staff PIN, sessions, shop scope

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 1 Backend core | R1a | L | 05 | FSD §2, SCR-S01/S02, FS-2.1–2.4, FS-17.4–17.5, VAL-P1/P2 |

## Goal

Owners sign up and staff sign in with phone OTP plus a per-device PIN; every shop route is scoped to the session's shop; owners can revoke staff and devices instantly.

## Why now

Shop routes are open today. Nothing can go to a real shop until this lands.

## Design

- **Sessions are opaque tokens** (32 random bytes, base64url), stored as SHA-256 in `cd_sessions`. No JWT, so revocation is immediate.
- **Access token** lifetime: staff 12 h, owner 30 days per device, customer 15 min + refresh 90 days (customer part is used in step 27).
- **OTP sender** is an interface: `ConsoleSender` (dev, logs code to stdout only when `CD_ENV=dev`) and `MSG91Sender` or equivalent DLT-compliant provider (prod; configure in step 25).
- **PIN** hashed with Argon2id (`golang.org/x/crypto/argon2`, time=2, memory=64 MiB, threads=2); 5 failures → device locked 15 minutes.

## Endpoints (from FSD §14)

`POST /auth/otp`, `POST /auth/verify`, `POST /auth/refresh`, `POST /auth/logout`, `POST /shop/signup`, `POST /shop/login` (PIN on a known device), `POST /shop/devices/{id}/pin` (set PIN), `GET/POST/DELETE /shop/staff`, `GET/DELETE /shop/devices`.

## Prompt

```text
First read docs/build-plan/00-common-context.md, then docs/FSD.md §2 (roles, permission matrix, rules FS-2.1–2.4), §6 SCR-S01 and SCR-S02, §16 VAL-P1/P2 and §17 FS-17.4–17.5. Read migration 0006 and api/internal/httpapi/*.go.

Task: implement authentication, sessions and shop scoping.

1. Package api/internal/auth:
   - otp.go: RequestOTP(ctx, phone, ip) and VerifyOTP(ctx, phone, code) → user. Phone validated with ^[6-9]\d{9}$ and stored normalised as +91XXXXXXXXXX. Codes are 6 random digits, stored hashed (SHA-256 of phone+code+server pepper CD_OTP_PEPPER), valid 5 minutes, max 5 verify attempts, max 5 sends per phone per hour and 20 per IP per hour (count rows in cd_otp_requests). Return otp_invalid / otp_expired / rate_limited errors.
   - sender.go: type OTPSender interface { Send(ctx, phoneE164, code string) error }. ConsoleSender prints "OTP for ***1234: 123456" only when CD_ENV=dev; otherwise refuses to start. Stub MSG91Sender with config fields (API key, template ID, sender ID) and a TODO for the HTTP call; it must compile.
   - pin.go: HashPIN/VerifyPIN with Argon2id and a per-hash salt, encoded as $argon2id$v=19$m=65536,t=2,p=2$salt$hash. Lockout: 5 failures → locked_until = now+15m on cd_devices.
   - session.go: CreateSession(ctx, userID, deviceID, kind, ttl) returns the raw token once; Lookup(ctx, rawToken) returns Session{UserID, DeviceID, ShopID, Role, Kind, ExpiresAt} or ErrUnauthorized; Revoke by session, by device, by user+shop.
2. Middleware in httpapi/auth_middleware.go:
   - requireSession(kinds...) reads Authorization: Bearer <token>, looks up the session (cache for 5 seconds max in an LRU keyed by token hash so revocation propagates within 5 s — FS-2.4), puts Principal in context.
   - requireRole(owner|staff) for shop routes; the shop_id ALWAYS comes from the session (FS-2.1); any shop_id in the request body or query is ignored.
   - requireTicket for guest job routes: X-Ticket-Secret header (also accept ?secret= while CD_LEGACY_ENVELOPE=true), compared with secret_hash in constant time.
3. Handlers:
   - POST /api/v1/cd/auth/otp {phone} → 204.
   - POST /api/v1/cd/auth/verify {phone, code, deviceName, deviceKind} → {user, isNewUser, memberships[{shopId, shopName, role}]} and a short-lived "pre-session" token (kind=pre, 10 min) used only for signup or PIN setup.
   - POST /api/v1/cd/shop/signup (pre-session) {shopName, slug, category} → creates shop (status pending, online_state offline), owner membership, default lane A, empty price list; returns the shop and requires PIN setup next. Slug rules VAL-SH1 (3–30, a-z0-9-, unique, not in a reserved list: admin, api, shop, tv, s, t, nearby, www, app, help).
   - POST /api/v1/cd/shop/devices/pin (pre-session) {shopId, pin} → sets PIN for this device, returns a full session (owner 30 d, staff 12 h).
   - POST /api/v1/cd/shop/login {deviceId, userId, pin} → session; handles lockout.
   - POST /api/v1/cd/auth/logout → revokes current session.
   - Owner-only: GET/POST /shop/staff (invite by phone creates a pending membership; the staff completes OTP + PIN on their device), DELETE /shop/staff/{userId} (revokes all their sessions in this shop), GET /shop/devices, DELETE /shop/devices/{id}.
4. Protect existing shop routes: /shop/queue and /shop/jobs/{id}/{action} now require a staff/owner session and ignore the ?shop= parameter (use session shop). Keep a dev-only bypass CD_DEV_AUTH_BYPASS=true that injects the demo shop owner — refuse to start if set with CD_ENV != dev.
5. Every auth event writes cd_audit_log (otp_requested, otp_verified, login_ok, login_failed, device_locked, session_revoked, staff_added, staff_removed) with no phone numbers or codes in detail.
6. Tests: OTP rate limits and expiry with FixedClock; PIN hash/verify and lockout; session revoke propagates within the cache window; a staff session from shop A gets 404 on a job from shop B (AT-14); signup slug validation; middleware ignores a forged shop query param.
7. Update contracts/openapi.yaml for any shape you had to decide; list decisions.

Rules: never log OTP codes, PINs, tokens or phone numbers (log a phone hash prefix at most). Show the plan and the session table usage before coding.
```

## Acceptance

- [ ] Owner can sign up → set PIN → get a session → see the queue of their shop only.
- [ ] Staff invited by phone can sign in on their own device.
- [ ] Removing staff revokes their sessions within 5 seconds.
- [ ] 5 wrong PINs lock the device for 15 minutes.
- [ ] No OTPs, PINs or phone numbers in logs.

## Verify

```bash
make test-api
# manual: POST /auth/otp → check console code → /auth/verify → /shop/signup → /shop/devices/pin → GET /shop/queue with Bearer token
```

## Commit

`feat(auth): phone OTP, argon2id device PINs, opaque sessions and shop-scoped access`
