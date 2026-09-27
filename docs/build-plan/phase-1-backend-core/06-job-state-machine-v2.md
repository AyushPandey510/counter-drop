# 06 — Job state machine v2

| Phase | Release | Size | Depends on | Spec refs |
| --- | --- | --- | --- | --- |
| 1 Backend core | R1a (R1b states included) | M | 05 | FSD §8 (T1–T19), FS-8.1–8.5, BR-Q, BR-P6 |

## Goal

`internal/domain` implements every state and transition from FSD §8 as pure, fully tested Go, and the store applies transitions atomically.

## Why now

Every endpoint that changes a job goes through this. Getting it exactly right, with exhaustive tests, prevents double prints, lost jobs and wrong refunds.

## Scope

**In:** states, events, guards, a transition table, side-effect descriptors (returned, not executed), store method `Transition` with optimistic state check, claim-next with SKIP LOCKED, exhaustive tests.
**Out:** executing side effects (tokens in step 09, events in step 12, refunds in step 32).

## Design

- `domain.Transition(job Job, ev Event, actor Actor, now time.Time) (Job, []Effect, error)` — pure.
- `Event` = {Kind: create|submit|pay|payment_captured|payment_failed|payment_timeout|accept|reject|accept_timeout|claim|release|ready|collected|undo|cancel|edit|abandon|files_expired, Reason, …}.
- `Actor` = {Type: guest|customer|staff|owner|agent|system|admin, ID, ShopID}.
- `Effect` = tagged values the caller executes: IssueToken, ReservePickupCode, ScheduleDeletion{At}, ClearDeletion, DeleteFilesNow, ScheduleTask{Kind, At}, CancelTask{Kind}, Refund{Amount, Reason}, ReleaseTransfer, ReverseTransfer, Notify{Topic}, Audit{Action}.

## Prompt

```text
First read docs/build-plan/00-common-context.md and docs/FSD.md §8 in full (states table, transitions T1–T19, implementation rules FS-8.1–8.5), plus BR-P6, BR-Q1–Q3 and BR-D1–D3 in docs/BRD.md §12. Then read api/internal/domain/job.go, job_test.go and the ApplyJobAction/SubmitJob code in api/internal/store/postgres.go.

Task: implement the v2 job state machine.

1. api/internal/domain/job_state.go: JobState constants (uploading, payment_pending, awaiting_shop, queued, claimed, ready, collected, cancelled, rejected, expired) and helpers IsTerminal(), IsVisibleInQueue(). ParseJobState accepts "new" as an alias of queued.
2. api/internal/domain/transition.go: Event, EventKind, Actor, ActorType, Effect types as described in the step file's Design section. Implement Transition(job, ev, actor, now, policy Policy) where Policy holds configurable durations (UndoWindow 10m, AcceptWindow 5m, PaymentWindow 15m, RejectGrace 15m, AbandonAfter 60m). Encode T1–T19 as a table: for each (from state, event kind) → guard func, to state, effects func. Unknown pairs return ErrInvalidTransition. Guards return typed errors: ErrWrongActor, ErrWrongChannel, ErrReasonRequired, ErrWindowExpired, ErrNotSameShop.
3. Refund amounts follow BR-P6 exactly: shop reject/timeout/shop cancel = full (print + fee); customer cancel in awaiting_shop = full; customer cancel in queued (remote) = total minus fee; after claim no customer cancel. Walk-in jobs never produce Refund effects.
4. Collected → ScheduleDeletion{At: now + UndoWindow}; Undo within window → state ready + ClearDeletion; after window → ErrWindowExpired. Release keeps the original queued_at (head of lane). Edit allowed only in queued and not claimed; for remote jobs a price increase is rejected (ErrPriceIncreaseNotAllowed).
5. Remove the old Apply(action) method but keep a thin adapter ApplyLegacyAction(action string, …) used by the existing /shop/jobs/{id}/{action} handler until step 11 replaces it.
6. Store: add Transition(ctx, jobID string, ev Event, actor Actor) (Job, []Effect, error) to PostgresStore and MemoryStore. In Postgres, within one tx: SELECT … FOR UPDATE the job; call domain.Transition; UPDATE cd_jobs SET … WHERE id=$1 AND state=$from; if 0 rows return ErrInvalidTransition; insert a cd_job_events row (from, to, event, actor, reason); commit; return effects for the caller. Effects are NOT executed here.
7. Store: ClaimNext(ctx, shopID, laneID string, actor Actor) (Job, error) using SELECT id FROM cd_jobs WHERE shop_id=$1 AND lane_id=$2 AND state='queued' ORDER BY queued_at, id LIMIT 1 FOR UPDATE SKIP LOCKED, then the claim transition in the same tx. Returns ErrNotFound when the lane is empty.
8. Tests:
   - Table test covering every row T1–T19 (happy path) and at least one rejected event per state (a full matrix test: for every state × every event kind, assert either the expected to-state or ErrInvalidTransition — generate the matrix from the table and compare to a hand-written expected map so a table typo is caught).
   - Refund amount tests for each BR-P6 case.
   - Undo inside and outside the window using FixedClock.
   - Concurrency integration test: 500 queued jobs, 3 goroutines calling ClaimNext until empty; assert 500 distinct claims and zero duplicates (AT-04).
9. Update docs/FSD.md only if you find a contradiction; list it in your final message instead of silently choosing.

Rules: domain package must not import store, httpapi or time.Now. Show me the transition table as a Go literal before implementing guards and effects.
```

## Acceptance

- [ ] Matrix test covers all states × events.
- [ ] Concurrency test: 500 jobs, 3 workers, zero duplicates.
- [ ] Every transition writes one `cd_job_events` row.
- [ ] Existing shop action endpoint still works via the legacy adapter.

## Verify

```bash
cd api && go test ./internal/domain/... -run Transition -v
make test-api
```

## Commit

`feat(domain): job state machine v2 with effects, atomic transitions and claim-next`
