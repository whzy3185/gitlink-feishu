# Feishu Review capability matrix

## Reading the matrix

`pass` means evidence exists for that layer. `pending` means a hosted CI run is
still required. `not run` means no real platform claim is made. `blocked-env`
means the required explicitly designated test resource was unavailable. Code and
offline tests are not treated as substitutes for real GitLink or Feishu proof.

| Capability | Code | Unit | Offline integration | Linux CI | Windows CI | Race | Real GitLink | Real Feishu | Fault | Current claim |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| PR read | pass | pass | pass | pending | pending | n/a | pass | pass | pass | real PR #1-#5 reads complete |
| Review read | pass | pass | pass | pending | pending | n/a | pass | pass | pass | real common/approved/rejected read-back |
| Thread read | pass | pass | pass | pending | pending | n/a | pass | pass | pass | real context thread section loaded |
| PR Presentation | pass | pass | pass | pending | pending | pass | pass | pass | pass | real terminal states observed |
| canonical card create | pass | pass | pass | pending | pending | pass | n/a | pass | pass | dedicated create returned 200 |
| canonical card patch | pass | pass | pass | pending | pending | pass | n/a | pass | pass | same canonical message reused |
| lightweight reply | pass | pass | pass | pending | pending | pass | n/a | pass | pass | dedicated and workflow replies succeeded |
| claim | pass | pass | pass | pending | pending | pass | n/a | pass | pass | real chat-local claim persisted |
| release | pass | pass | pass | pending | pending | pass | n/a | pass | pass | real release cleared owner and deadline |
| deadline | pass | pass | pass | pending | pending | pass | n/a | pass | pass | real date persisted |
| dual-chat isolation | pass | pass | pass | pending | pending | pass | n/a | blocked-env | pass | only one designated test chat |
| subscription | pass | pass | pass | pending | pending | pass | n/a | not run | pass | stored revisioned configuration |
| webhook | pass | pass | pass | pending | pending | pass | blocked-env | n/a | pass | public callback and subscription unavailable |
| Event Inbox | pass | pass | pass | pending | pending | pass | n/a | n/a | pass | durable and idempotent offline |
| replay | pass | pass | pass | pending | pending | pass | n/a | n/a | pass | explicit request ID required |
| reconciliation | pass | pass | pass | pending | pending | pass | n/a | n/a | pass | Base automatic; other resources manual |
| Base projection | pass | pass | pass | pending | pending | pass | n/a | blocked-env | pass | explicit test table unavailable |
| Doc projection | pass | pass | pass | pending | pending | pass | n/a | blocked-env | pass | explicit test folder unavailable |
| Task projection | pass | pass | pass | pending | pending | pass | n/a | blocked-env | pass | explicit test assignee unavailable |
| common Review | pass | pass | pass | pending | pending | pass | pass | pass | pass | Review 130 verified; duplicate zero |
| approve Review | pass | pass | pass | pending | pending | pass | pass | pass | pass | Review 131 approved; duplicate zero |
| reject Review | pass | pass | pass | pending | pending | pass | pass | pass | pass | Review 132 rejected; PR stayed open |
| reject and close | pass | pass | pass | pending | pending | pass | pass | pass | pass | native close; zero Review create |
| controlled merge | pass | pass | pass | pending | pending | pass | pass | pass | pass | disposable base only; CI unknown |
| head stale protection | pass | pass | pass | pending | pending | pass | pass | pass | pass | old plan produced zero mutation |
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

## 飞书用户命令

正式命令统一使用中文，并要求 PR 级操作显式包含 `owner/repo`：

```text
查看 owner/repo PR #123
领取 owner/repo PR #123
取消领取 owner/repo PR #123
设置 owner/repo PR #123 审查截止 YYYY-MM-DD
提交审查意见 owner/repo PR #123 <意见>
批准 owner/repo PR #123 <说明>
要求修改 owner/repo PR #123 <原因>
拒绝并关闭 owner/repo PR #123 <原因>
合并 owner/repo PR #123
```

`要求修改`只提交需修改审查结论，PR 保持开放。英文 `review / approve /
reject / refuse / merge`、`释放`和旧截止日期语法仅作为兼容入口保留。
所有受控写操作仍然只在飞书生成操作计划，最终执行必须由绑定的 GitLink
身份在本地确认，并经过身份、版本新鲜度、单次写入和 GET 回读门禁。
