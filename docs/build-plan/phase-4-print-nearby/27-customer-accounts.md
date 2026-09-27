# 27 — Customer accounts (phone OTP) and /me

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 4 Print nearby | R1b | S | 07 | FSD SCR-A02, SCR-A16, §2 app customer, API-16–19, UC-C19, UC-C25 |

## Goal

Customers sign in with phone OTP only when they first pay for a Print nearby order; walk-in stays login-free forever.

## Prompt

```text
First read docs/build-plan/00-common-context.md, docs/FSD.md SCR-A02, SCR-A16, §2 (App customer role and sessions), §14 API-16 to API-19, and docs/BRD.md UC-C19 and UC-C25. Read the auth package from step 07.

Task: add customer accounts.

1. Reuse auth OTP. POST /auth/verify with deviceKind=customer creates or finds a cd_users row and returns an access token (15 min) + refresh token (90 days, rotated on every refresh, reuse detection revokes the family). Store refresh tokens hashed in cd_sessions with a family_id.
2. First sign-in asks for first name once (PATCH /me {name, language}).
3. Endpoints: GET /me, PATCH /me, DELETE /me (UC-C25: remove name, phone → hashed tombstone, push tokens, favourites; keep payment ledger rows linked by user ID only — BR-D5; explain on screen), POST /me/devices (push token, used in step 34), GET /me/jobs (step 35).
4. Web: src/customer/auth — an AuthGate component that wraps only the Pay action in Print nearby. OTP sheet with WebOTP autofill; stays on the same screen after sign-in; tokens in memory + refresh token in localStorage (try/catch) with silent refresh.
5. Guard: walk-in routes (/s/:slug, /t/:jobId, /scan) must never render AuthGate (add a Playwright assertion).
6. Tests: refresh rotation and reuse detection; delete account removes personal fields and revokes sessions; AuthGate only on pay.

Rules: no login wall anywhere else. Show the token lifecycle first.
```

## Acceptance

- [ ] Browsing Print nearby needs no login; paying asks for OTP once.
- [ ] Refresh token reuse revokes the session family.
- [ ] Delete account clears personal data and explains what's kept.

## Commit

`feat(accounts): customer phone OTP with rotating refresh tokens and /me`
