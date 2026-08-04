# Feishu Review dead letter and reconciliation

## Different responsibilities

Dead Letter and Reconciliation are not synonyms.

- A Dead Letter records work that automatic execution stopped handling.
- A Reconciliation task asks whether an uncertain remote side effect exists and whether local state can be repaired safely.

Dead Letter does not mean “ignore.” It is a durable administrative queue. Reconciliation does not mean “retry.” It must inspect before deciding.

## Dead Letter creation

A terminal failure or exhausted safe retry budget creates one idempotent dead-letter record for the Operation. Repeated finalization does not create duplicate records. The record contains redacted error data and hashes rather than full platform identifiers.

Automatic execution stops. Retrying a Dead Letter requires an explicit confirmation token through the local administrative command; merely viewing it changes no state.

## Reconciliation creation

An Operation with possible remote side effects creates one reconciliation task. Tasks use leases, bounded attempts, `next_attempt_at` and durable results. Resolved tasks are not automatically reopened by a repeated worker pass.

## Automatic boundary

Only Base currently has a safe automatic method:

```text
resource stable key
  -> Base search
  -> zero matches: unresolved/manual decision
  -> one match: save remote record ID and resolve
  -> multiple matches: manual required
```

Card, Reply, Doc and Task do not currently have sufficiently strong stable lookup contracts in this implementation. Their uncertain effects become `manual_required`; the reconciler does not create, resend or append anything.

## Failure handling

Reconciliation checks use their own attempt counter. A transient lookup error schedules another check. Exhausting that budget creates a Dead Letter for administrative handling. A failed check never changes the Operation to succeeded without positive remote evidence.

## Administration

Local read-oriented shortcuts expose Operation, Dead Letter and Reconciliation status. Mutating a Dead Letter requires explicit confirmation. Output redacts complete Chat IDs and remote IDs and never includes access tokens or desired payload secrets.

## Current boundary

The automatic Base lookup and all unsupported-resource outcomes are validated against fake clients. Real Base, card, reply, Doc and Task reconciliation has not yet been platform-validated. This stage therefore must not be described as recovering every unknown effect.
