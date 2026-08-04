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
- Existing controlled common-Review write code remains behind its own ActionPlan,
  installation, source-chat, head and fingerprint gates. It is outside the
  Stage Six real acceptance scope.
- Automatic approve, reject, merge, close, reviewer mutation, line comments and
  thread resolution are not promised.

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
