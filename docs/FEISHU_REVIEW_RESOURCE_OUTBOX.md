# Feishu Review resource outbox

## Purpose

Stage four replaces synchronous Feishu side effects in the production Review path with a durable SQLite operation outbox. A completed Review job records desired effects first; dedicated workers execute them later. Process restart therefore does not erase the intent.

An Operation is an execution record, not a source of GitLink truth.

- PR Presentation remains the normalized source of GitLink PR facts.
- Collaboration State remains the source of truth for the chat team's owner, deadline and workflow state.
- Chat Presentation remains the source of truth for the one canonical card per chat and PR.
- Resource State stores the remote ID and last applied content fingerprint.
- Projection Status only describes the current synchronization state shown to users.

## Durable flow

```text
GitLink read / collaboration job
  -> durable Review result
  -> operation_plan_status=pending
  -> planner lease
  -> deterministic operations in SQLite
  -> operation worker lease
  -> remote request
  -> attempt + operation + resource/projection state committed
```

The planner performs no external write. `review_operations.idempotency_key` is derived from operation kind, work-item key and desired fingerprint. Re-planning the same desired state therefore creates no second Operation. A newer desired fingerprint marks an older executable Operation stale.

## Operation kinds

| Kind | Queue | Purpose | Retry safety |
| --- | --- | --- | --- |
| `canonical_card_upsert` | `canonical_card` | Create or patch the chat's canonical PR card | create non-idempotent; patch treated as idempotent |
| `reply_send` | `reply` | Reply independently to a requesting message | non-idempotent |
| `bitable_upsert` | `resource` | Search by stable key, then create or update a Base record | reconcilable |
| `doc_snapshot_upsert` | `resource` | Create a document when needed and append one new snapshot | non-idempotent |
| `task_upsert` | `resource` | Create, patch or complete a Review task | create non-idempotent; patch idempotent |

Reply Operations depend on successful canonical-card completion. Coalesced user requests keep separate Job Consumer rows, and the planner creates one reply Operation per consumer.

## Payload and identity constraints

- Desired JSON is limited to 256 KiB.
- Fields whose names indicate authorization, tokens, cookies or secrets are rejected.
- IDs and fingerprints are deterministic and do not include credentials.
- Attempts are append-only after completion.
- A worker must own the current lease to start or finish an attempt.
- Network calls run outside SQLite transactions.

## Resource-specific rules

### Card

The first send is non-idempotent. A timeout or missing message ID becomes `unknown`; it is not sent again automatically. Existing-card patches can retry after transient failures. Resource projection changes may enqueue a coalesced card refresh, but card completion cannot recursively enqueue itself.

### Reply

A reply is sent only after its card dependency succeeds. Timeout or missing message ID becomes `unknown`; the same Operation is not blindly resent.

### Base

Base searches by the stable resource key before creation. Zero matches permits create, one match permits update, and multiple matches require reconciliation. If the remote create succeeds but saving the local remote ID fails, the Operation becomes `unknown`.

### Doc

After document creation, the document ID is saved before blocks are appended. If append outcome is unknown, the fingerprint remains unapplied and automatic append is stopped. An already-applied fingerprint prevents a duplicate snapshot.

### Task

Task creation is non-idempotent and is not blindly retried after an uncertain outcome. Patches may retry. An archived PR with an existing task completes it; an archived PR without a task does not create one.

## Schema

Migrations 11 and 12 add the outbox core:

- `review_operations`
- `review_operation_attempts`
- `review_resource_projection_status`

Migrations 13–15 add reconciliation, worker admission and rate-limit support. All migrations are transactional and idempotent. Opening an old database applies them in order; opening a new database creates the final structure.

## Current validation boundary

The repository has offline contract validation using temporary SQLite, `httptest` and fake clients. This proves planning, state transitions, leases, dependency gates and simulated side-effect handling. It does not prove real Feishu Base, Doc, Task, card or reply behavior, and it performs no real GitLink POST.
