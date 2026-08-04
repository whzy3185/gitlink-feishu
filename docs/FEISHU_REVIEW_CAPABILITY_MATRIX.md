# Feishu Review capability matrix

## Reading the matrix

`pass` means evidence exists for that layer. `pending` means a hosted CI run is
still required. `not run` means no real platform claim is made. Code and
offline tests are not treated as substitutes for real GitLink or Feishu proof.

| Capability | Code | Unit | Offline integration | Linux CI | Windows CI | Race | Real GitLink | Real Feishu | Fault | Current claim |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| PR read | pass | pass | pass | pending | pending | n/a | not run | n/a | pass | implementation verified offline |
| Review read | pass | pass | pass | pending | pending | n/a | not run | n/a | pass | implementation verified offline |
| Thread read | pass | pass | pass | pending | pending | n/a | not run | n/a | pass | implementation verified offline |
| PR Presentation | pass | pass | pass | pending | pending | pass | not run | not run | pass | implementation verified offline |
| canonical card create | pass | pass | pass | pending | pending | pass | n/a | not run | pass | no Stage Six real proof |
| canonical card patch | pass | pass | pass | pending | pending | pass | n/a | not run | pass | idempotent retry boundary verified |
| lightweight reply | pass | pass | pass | pending | pending | pass | n/a | not run | pass | non-idempotent uncertainty verified |
| claim | pass | pass | pass | pending | pending | pass | n/a | not run | pass | chat-local only |
| release | pass | pass | pass | pending | pending | pass | n/a | not run | pass | chat-local only |
| deadline | pass | pass | pass | pending | pending | pass | n/a | not run | pass | date contract verified |
| dual-chat isolation | pass | pass | pass | pending | pending | pass | n/a | not run | pass | real two-chat observation outstanding |
| subscription | pass | pass | pass | pending | pending | pass | n/a | not run | pass | stored revisioned configuration |
| webhook | pass | pass | pass | pending | pending | pass | not run | n/a | pass | public proxy and GitLink delivery outstanding |
| Event Inbox | pass | pass | pass | pending | pending | pass | n/a | n/a | pass | durable and idempotent offline |
| replay | pass | pass | pass | pending | pending | pass | n/a | n/a | pass | explicit request ID required |
| reconciliation | pass | pass | pass | pending | pending | pass | n/a | n/a | pass | Base automatic; other resources manual |
| Base projection | pass | pass | pass | pending | pending | pass | n/a | not run | pass | real test table outstanding |
| Doc projection | pass | pass | pass | pending | pending | pass | n/a | not run | pass | real test folder outstanding |
| Task projection | pass | pass | pass | pending | pending | pass | n/a | not run | pass | real test task outstanding |
| Operation Outbox | pass | pass | pass | pending | pending | pass | n/a | n/a | pass | remote boundary persisted |
| Dead Letter | pass | pass | pass | pending | pending | pass | n/a | n/a | pass | explicit retry confirmation |
| worker isolation | pass | pass | pass | pending | pending | pass | n/a | n/a | pass | single-instance workers |
| rate limit | pass | pass | pass | pending | pending | pass | n/a | n/a | pass | 429/Retry-After verified |
| refresh coalescing | pass | pass | pass | pending | pending | pass | n/a | n/a | pass | same-scope reads coalesce |
| healthz | pass | pass | pass | pending | pending | n/a | n/a | n/a | pass | local service contract |
| readyz | pass | pass | pass | pending | pending | n/a | n/a | n/a | pass | stale worker degrades readiness |
| metrics | pass | pass | pass | pending | pending | n/a | n/a | n/a | pass | authenticated; cached on SQLite busy |
| Admin API | pass | pass | pass | pending | pending | n/a | n/a | n/a | pass | loopback and read-only |
| backup | pass | pass | pass | pending | pending | n/a | n/a | n/a | pass | WAL-aware snapshot |
| restore | pass | pass | pass | pending | pending | n/a | n/a | n/a | pass | refuses running service |
| retention | pass | pass | pass | pending | pending | n/a | n/a | n/a | pass | Unknown and open DLQ retained |
| Linux service | pass | pass | pass | pending | n/a | n/a | n/a | n/a | pass | deployment contract only |
| Windows service | pass | pass | pass | n/a | pending | n/a | n/a | n/a | pass | WhatIf/static contract only |

The committed evidence source commit is the parent of the final documentation
commit. Hosted results for the final SHA are recorded in the Actions artifact
and the final handoff, not retroactively invented in this file.
