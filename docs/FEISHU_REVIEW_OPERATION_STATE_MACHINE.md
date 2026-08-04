# Feishu Review operation state machine

## States

| State | Meaning | Automatic execution allowed |
| --- | --- | --- |
| `pending` | Durable intent is ready | yes |
| `leased` | A worker owns a time-bounded lease | only by lease owner |
| `writing` | Attempt began and a request may be in flight | only by lease owner |
| `succeeded` | Remote result and local state are confirmed | no |
| `unchanged` | Desired fingerprint already applies | no |
| `retry_scheduled` | A safe transient retry has a future time | when due |
| `failed_terminal` | Invalid input, authorization or another terminal failure | no |
| `unknown` | A remote side effect may have happened | no |
| `needs_reconciliation` | Recovery requires checking remote state | reconciliation only |
| `stale` | A newer desired fingerprint superseded this intent | no |
| `dead_letter` | Automatic attempt budget is exhausted or failure is terminal | explicit administration only |
| `cancelled` | Explicitly cancelled | no |
| `blocked` | Dependency is not satisfied | after dependency transition |

`unknown` is not `failed`. Treating it as a failure and retrying a non-idempotent request could duplicate a card, reply, document snapshot or task.

## Mutation status

| Value | Guarantee |
| --- | --- |
| `not_started` | No remote request is known to have started |
| `request_started` | The request crossed the local pre-write boundary |
| `remote_confirmed` | Reserved for a positively confirmed remote effect |
| `remote_unknown` | The remote effect may exist but is not safely identified |
| `local_confirmed` | Remote result and local durable state agree |

Operation status and mutation status must be read together. `unknown + remote_unknown` explicitly prevents an upper layer from reporting “no write occurred.”

## Normal transition

```text
pending/retry_scheduled
  -> leased
  -> writing
  -> succeeded | unchanged
```

Before execution, a dependency policy can prevent claiming the Operation. Before saving a newer Operation, older executable Operations for the same kind and work item become `stale`.

## Error transition

```text
writing
  -> retry_scheduled       safe transient or rate-limited request
  -> failed_terminal       non-retryable failure
  -> unknown               remote side effect is possible
  -> dead_letter           safe retry budget exhausted
  -> stale                 desired state is no longer current
```

An attempt is written before execution and finalized exactly once. Attempt completion records mutation status, typed error class and code, HTTP status, Retry-After and result fingerprint. Finished attempt rows cannot be updated or deleted.

## Lease recovery

An expired `leased` Operation can return to `retry_scheduled`. Recovery of expired `writing` depends on safety:

- idempotent: schedule a safe retry;
- reconcilable: enter `needs_reconciliation`;
- non-idempotent: enter `unknown`.

A stale worker cannot start or finish an attempt after another worker acquires the renewed lease.

## Dependency rule

The current production dependency is `reply_send -> canonical_card_upsert` with `success` policy. The reply becomes claimable only after the card is `succeeded` or `unchanged`. A card failure therefore does not silently produce an unanchored reply.

## Source-of-truth boundary

Operation rows record execution. They do not replace PR Presentation, Collaboration State or Chat Presentation. Resource State holds remote IDs; Projection Status is display-only and cannot be used as proof that GitLink or Feishu owns a fact.
