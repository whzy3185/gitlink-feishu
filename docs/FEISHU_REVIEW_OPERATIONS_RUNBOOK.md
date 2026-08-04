# Feishu Review operations runbook

## 1. Install and configure

Build with `go build -o gitlink-cli .`. Start from
`docs/examples/feishu-review-bindings-v2.json`; use explicit installation and
repository allowlists. Put credentials in protected environment files or secret
references, never command arguments. Keep the state database and backup
directory on persistent storage.

## 2. Start and stop

- Linux: use the unit and environment examples under `deploy/linux`.
- Windows: use `deploy/windows/install-review-service.ps1`; run `-WhatIf` first.
- Stop through the service manager. Graceful shutdown closes admission, drains
  safe work, marks uncertain non-idempotent work for reconciliation, checkpoints
  WAL and releases the instance lock.

## 3. Health, readiness and metrics

- `GET /healthz`: process/database liveness.
- `GET /readyz`: admission and required component readiness.
- `GET /metrics`: authenticated loopback metrics only.
- Never expose `/metrics` or `/admin/v1/` through the public webhook proxy.

## 4. Read-only control plane

Use authenticated GET requests under `/admin/v1/summary`, `components`,
`operations`, `dead-letters` and `operation-reconciliations`. Non-GET methods are
rejected. Returned operation and remote identifiers are hashed.

## 5. Queue and operation triage

Inspect queue depth, oldest age, worker heartbeat, retry schedule and operation
error class. A transient or rate-limited idempotent operation may retry. An
unknown non-idempotent operation must not be reset to pending. Move exhausted
work to Dead Letter once and use an explicit, audited retry decision.

## 6. Reconciliation

- Base: search the stable unique key; repair the remote ID only when exactly one
  record matches.
- Card, Reply, Doc, Task: compare the hashed local reference with the named test
  or production resource manually; resolve or replace through an explicit plan.
- Never guess that a timeout means no remote side effect.

## 7. Backup, restore, retention and WAL

Run `verify-feishu-review-backup-restore.ps1` before deployment. Backups use a
consistent SQLite snapshot and checksum manifest. Restore first in verify-only
mode, refuse restore while the service is running, create a pre-restore backup,
then atomically replace and migrate. Retention uses bounded batches and preserves
Unknown operations, open Dead Letters, unresolved reconciliations and active
presentations. A busy WAL checkpoint is operational pressure, not corruption.

## 8. Webhook incidents

Confirm HTTPS reachability, signature mode, timestamp skew, body limit and
Content-Type. A correct handler returns 202 after Event Inbox persistence; job
creation is asynchronous. Use Replay with a new explicit request ID. Do not
manufacture GitLink activity merely to create a webhook event.

## 9. Feishu Channel incidents

Confirm exactly one intended connection set, published application version,
message event subscription, bot mention, chat allowlist and user policy. Trace
hashed message/chat IDs through admission, Job, Operation and reply state.

## 10. Rate limits and resource pressure

- GitLink 429: honor Retry-After and coalesce same-PR refreshes.
- Feishu 429: honor Retry-After per resource worker; do not block the canonical
  card on a slow Base/Doc/Task projection.
- SQLite Busy: inspect long transactions and disk latency; network calls must
  occur outside database transactions.
- Low disk: stop admission safely, preserve the database, free space, verify
  integrity, then resume.

## 11. Crash recovery

Restart the same binary with the same database. Expired leases are recovered;
stale workers cannot complete a newer lease. `writing` non-idempotent operations
become Unknown or needs-reconciliation, never blind retries.

## 12. Secret rotation

Rotate the referenced environment secret, restart through the service manager,
verify readiness, then run the Secret scan. Do not place old or new secret values
in logs, evidence, screenshots, SQLite or command lines.

## 13. Final evidence export

Run `scripts/run-feishu-review-final-acceptance.ps1 -OfflineOnly`. Real tests
require the two explicit switches, dedicated test resources and
`-ConfirmExternalWrites`. Re-run `scan-feishu-review-secrets.ps1` after evidence
generation. Evidence hashes remote identifiers and never auto-deletes resources.
