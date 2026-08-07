# Feishu Review capability boundaries

## Current promise

The repository contains a single-instance Review collaboration service that
models multiple GitLink installations, multiple repositories and multiple
Feishu chats. It maintains one canonical PR card per chat, chat-scoped claim,
release and deadline state, durable event and operation queues, typed failure
handling, reconciliation, Base/Doc/Task projections, health/readiness/metrics,
read-only administration and SQLite maintenance.

These statements are code and offline-contract claims. A capability becomes a
real-platform claim only when its `evidence/final/real-*.json` file has
`passed=true` and `validation_mode=real_platform`. Stage Six did not execute
those external validators, so no new real-platform claim is made here.

## GitLink boundary

- Normal review discovery is GET-only.
- The Stage Six validator blocks every method except GET and HEAD before the
  network and records exact method counts.
- A common Review can be prepared in Feishu and confirmed from the local CLI.
  The write path is guarded by one ActionPlan, current head, complete source
  fingerprint, bound GitLink login, atomic lease and a persisted write boundary.
- `pr +review --status common` and Feishu confirmation use the same writer.
  One execution performs at most one POST and then a bounded GET read-back.
- Automatic approve, reject, merge, close, reviewer mutation, line comments and
  thread resolution are not promised.

## Common Review contract

The formal Feishu input is:

```text
review owner/repository#123 review body
```

Preparation creates a 15-minute ActionPlan containing the repository, PR,
expected head, source fingerprint, Feishu actor hash, bound GitLink login,
review body and a system-generated `RW-XXXXXX` request ID. The trusted footer
is written as `Ref: RW-XXXXXX`; look-alike references supplied in the body are
removed.

The local execution command is:

```text
gitlink-cli feishu +review-confirm-local \
  --plan-id <plan-id> \
  --state-db .local/review-gateway.db
```

It uses the current local CLI credential and checks `/users/me`; it does not
fall back to an Installation credential. `--dry-run` performs no POST, and
`--yes` only bypasses the terminal prompt. A cancelled, expired, stale,
completed, remotely uncertain, cross-actor or identity-mismatched plan cannot
start another POST.

Writer outcomes are `verified`, `duplicate`, `stale`, `failed` and `unknown`.
`commit_id=null` remains null with a warning. `unknown` requires reconciliation
and disables automatic retry. These are offline code-contract claims only;
real GitLink write acceptance still requires explicit user authorization and
separately preserved before/after evidence.

## Feishu boundary

- Remote test writes require both `FEISHU_REVIEW_REAL_VALIDATION=1` and
  `FEISHU_REVIEW_CONFIRM_EXTERNAL_WRITES=YES`.
- Only explicitly named test chats, Base table, Doc folder and Task assignee are
  valid test targets. Remote resources are not deleted automatically.
- Evidence stores hashes of chat, message, record, document and task IDs; it does
  not store the raw identifiers or credentials.
- A Base unknown can be reconciled by unique key. Card, Reply, Doc and Task
  unknown side effects normally require human reconciliation.

## Deployment boundary

- Supported topology: one active process and one SQLite database.
- Admin API and metrics are loopback-only and are not exposed by the example
  reverse proxy.
- Health and readiness proxy behavior is determined by the final Caddy/Nginx
  examples; operators must not expose Admin or metrics when changing them.
- Linux systemd and Windows Service assets avoid secrets in command arguments.

## Explicitly unsupported

Multi-instance high availability, distributed databases, Redis, Kafka,
zero-downtime schema migration, cross-region disaster recovery, complete user
OAuth, full Git mirroring, code search, automatic code modification, multi-Agent
execution and a production-validated Enterprise WeChat chain are not included.
Not every Unknown can be recovered automatically.
