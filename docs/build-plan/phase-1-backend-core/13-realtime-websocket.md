# 13 — Realtime WebSocket hub

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 1 Backend core | R1a | M | 12 | FSD §15, FS-4.8, FS-6.1, FR-2.1, NFR-5 |

## Goal

Customers' ticket pages and shop dashboards update within 1 second through two WebSocket channels, with sequence numbers, snapshots on gaps, heartbeats and safe auth.

## Prompt

```text
First read docs/build-plan/00-common-context.md and docs/FSD.md §15 in full, FS-4.8 and FS-6.1. Read the outbox Publisher interface from step 12.

Task: implement api/internal/realtime.

1. Library: github.com/coder/websocket. Endpoints:
   - GET /api/v1/cd/ws/jobs/{id}: after upgrade, the first client message within 5 s must be {"auth":{"secret":"…"}} or {"auth":{"token":"…"}}; otherwise close 4401. Tickets are verified against secret_hash. Subscribes to topic job:{id}.
   - GET /api/v1/cd/ws/shop: first message {"auth":{"token":"…"}} (staff/owner session) → topic shop:{shopId}. A TV read-only key is added in R2.
   Tokens are not accepted in the URL query (they would leak into logs).
2. Hub: map topic → set of connections; Publish(topic, seq, eventType, payload) implements the outbox Publisher. Each connection has a buffered send channel (64); a slow client is disconnected (close 4408) rather than blocking the hub. Messages: {"seq":n,"type":"job.updated","data":{…}}.
3. Heartbeat: server ping every 25 s; close after 2 missed pongs. Max message size 4 KiB inbound. Origin check against CD_WEB_ORIGINS.
4. Snapshots: on connect after auth, send {"type":"hello","seq":currentSeq} so the client knows where it is; clients that detect a gap send {"type":"resync"} and the server replies {"type":"sync.required"} (the client then refetches over HTTP — FS-15.1). On server start, all clients get sync.required after reconnect naturally.
5. Scale note (not now): if CD_REDIS_URL is set in the future, publish via Redis pub/sub; leave an interface seam, no Redis dependency.
6. Metrics hooks: current connections by channel, messages sent, drops (exposed in step 24).
7. Tests: auth timeout closes; wrong secret closes 4401; publish reaches only subscribers of that topic; slow consumer is dropped without blocking others; seq increases; ping/pong keeps alive (use short intervals in test config). An end-to-end test: submit a job via HTTP and receive queue.job_added on a shop socket within 1 s.

Rules: no business logic in realtime; it only moves messages from the outbox dispatcher to sockets. Show the hub API first.
```

## Acceptance

- [ ] Shop socket receives `queue.job_added` < 1 s after submit.
- [ ] Ticket socket receives `job.updated` on claim/ready/collected.
- [ ] A stalled client doesn't delay others.

## Verify

```bash
make test-api
# manual: npx wscat -c ws://localhost:8080/api/v1/cd/ws/shop then send {"auth":{"token":"…"}}
```

## Commit

`feat(realtime): WebSocket hub for job and shop channels with auth, seq and heartbeats`
