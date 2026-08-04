# Feishu Review worker model

## Goal

Slow GitLink reads, chat collaboration actions and Feishu resource writes must not share one blocking execution lane. Stage four introduces class-specific job workers, a persistent planner and separate Operation worker pools.

## Default concurrency

| Pool | Workers | Responsibility |
| --- | ---: | --- |
| event | 2 | normalize and route durable GitLink events |
| gitlink_read | 3 | GET-only PR queue/context refresh |
| collaboration | 1 | claim, release, deadline and subscription state |
| planner | 2 | convert completed job results into durable Operations |
| canonical_card | 2 | create or patch canonical PR cards |
| reply | 2 | send per-consumer replies after card success |
| resource | 2 | Base, Doc and Task Operations |
| controlled_write | 1 | separately gated GitLink common Review confirmation |
| agent | 0 | disabled by default |
| operation_reconciliation | 1 | remote-state checks for unknown effects |

Concurrency is validated within 0–32. Required pools cannot be configured to zero; Agent is intentionally zero until separately enabled and governed.

## Job lifecycle

Atomic admission writes capacity/rate decisions, the Job, its first consumer and the dedupe reservation in one SQLite transaction. Workers claim only their queue class and order ready Jobs by priority and creation time. A network call begins after the claim transaction commits.

When execution completes, the durable result sets `operation_plan_status=pending`. Planner workers claim completed results, create the full Operation set transactionally and mark the job `planned`. Process restart can therefore resume planning without replaying the GitLink read.

## Operation lifecycle

Operation workers are separated into `canonical_card`, `reply` and `resource`. A slow Base call cannot occupy a card worker; a slow Doc call cannot occupy a Task only in an absolute sense because Doc and Task share the two-worker resource pool, but with the default pool a single slow call does not block the other ready resource Operation.

Controlled GitLink writes do not run in a read or collaboration worker. They retain the pre-existing ActionPlan confirmation, installation/chat scope and reconciliation gates.

## Lease and restart rules

- Job, Planner, Operation and Reconciliation work use time-bounded leases.
- The lease owner must match to complete work.
- Expired safe work can be reclaimed.
- In-flight non-idempotent Operation recovery becomes unknown/reconciliation rather than a blind retry.
- Job and Operation payloads remain in SQLite across process restart.

## Isolation limits

This is in-process worker isolation over one SQLite database. It is not high availability, distributed queueing or cross-host leader election. Multi-instance production semantics, graceful shutdown, health endpoints, metrics, backup and WAL maintenance remain future work.
