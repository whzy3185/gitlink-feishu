# Feishu Integration Task Chain Assessment

## Scope Reviewed

This assessment reviews the local task chain file:

`C:\Users\zyc\OneDrive\文档\# Codex 任务链：gitlink-cli 飞书协同导出工作流.txt`

It also checks the current local `gitlink-cli` repository layout under:

`E:\GitLinkCLI-Competition\gitlink-cli`

## Repository Fit

The task chain is mostly aligned with the current repository:

- The repository is a Go CLI project using `cobra`.
- Shortcut commands are mounted through `shortcuts.RegisterAll` in `shortcuts/register.go`.
- Existing shortcut groups live under `shortcuts/<group>/`.
- `workflow` already exists and provides `+triage`, `+health`, `+pr-summary`, and `+repo-report`.
- `workflow` already supports local fixtures and read-only GitLink fetches.
- Existing AI Agent skills live under `skills/`.
- Documentation and examples directories already exist.

One mismatch: the task chain expects `.trustie-pipeline.yml`, but the current local repository does not contain that file. The repo does contain `.github`, `Makefile`, and Go tests.

## Completeness Judgment

The task chain is directionally complete but too broad for one clean PR.

Strong parts:

- Clear safety boundary: one-way export from GitLink data to Feishu.
- Clear exclusions: no Feishu tasks, approval flows, callback server, GitLink remote writes, issue closure, comments, or merge actions.
- Good default safety: dry-run first and mock-based CI tests.
- Good implementation path: scan first, then skeleton, then security/options, then client/card/mapper, then docs and skill.
- Good repo fit: `shortcuts/feishu` and `skills/gitlink-feishu` match the existing architecture.

Missing or underspecified parts:

- Feishu Bitable writes require authentication beyond `app_token` and `table_id`. A practical implementation needs either an app credential flow (`app_id` + `app_secret` to obtain `tenant_access_token`) or an explicit user-provided tenant token. The task chain mentions tenant token leakage but does not define how the token is supplied or refreshed.
- The plan does not specify Feishu API request shapes for Bitable create/list/update records, especially auth headers, error response parsing, pagination, and batch behavior.
- The plan does not define a strict payload contract for `+notify`, `+weekly-report`, and `+sync-bitable`; it lists fields but not stable JSON schemas.
- `+bot-test` behavior is slightly ambiguous: it says default dry-run, but also says the command semantics may send a test card. For safety, real sends should require an explicit flag such as `--to-feishu` plus `--dry-run=false`.
- The plan assumes `go test ./...` is currently clean. Current local baseline is not clean on Windows due existing auth/token-store tests, while `shortcuts/workflow` passes.
- The plan includes 15 phases, which is useful as a roadmap but too large for a single maintainable contribution.

## Practical Utility Judgment

The idea has real practical value if scoped correctly.

Useful real-world outcomes:

- Send GitLink repository workflow summaries to a Feishu group as cards.
- Export repository health, issue triage, pull request summaries, and weekly reports from existing `workflow` commands.
- Generate Bitable-ready records for issues, pull requests, contributors, and reports.
- Provide an Agent Skill that tells agents how to use the CLI safely.

The strongest MVP is:

1. `feishu +bot-test`
2. `feishu +weekly-report --from-workflow-json ... --dry-run`
3. `feishu +notify --from-workflow-json ... --dry-run`
4. Feishu custom bot card construction and signing
5. Mocked HTTP tests

Bitable sync is practical, but it should be a second PR unless authentication and record upsert behavior are designed first.

## Current Test Baseline

Command run:

```bash
go test ./...
```

Result:

- Many packages passed, including `shortcuts/workflow`.
- Current failures are in existing auth/token-store tests:
  - `cmd/auth`
  - `internal/auth`
- Failures are related to Windows credential path and missing credential file behavior, not Feishu work.

This baseline should be recorded before starting implementation. Early Feishu work should at least pass:

```bash
go test ./shortcuts/feishu
go test ./shortcuts/workflow
go test ./shortcuts
```

Full `go test ./...` remains a desired final check, but existing unrelated failures must be handled separately.

## External Feishu Feasibility Notes

Feishu custom bots support webhook-based group messages and signature verification. Feishu card messages can be sent through custom bot webhooks. Bitable record writes require app/table identifiers and proper app authentication.

Relevant official docs:

- Custom bot usage guide: https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot
- Send message cards with custom bot: https://open.feishu.cn/document/feishu-cards/quick-start/send-message-cards-with-custom-bot?lang=zh-CN
- Create Bitable record: https://open.feishu.cn/document/server-docs/docs/bitable-v1/app-table-record/create?lang=zh-CN
- Get tenant access token for internal app: https://open.feishu.cn/document/server-docs/authentication-management/access-token/tenant_access_token_internal?lang=zh-CN

## Recommendation

Use the task chain as a roadmap, but reduce the first implementation to a small, verifiable contribution:

- Phase A: scan and architecture note.
- Phase B: command skeleton and safe options.
- Phase C: bot card/signing/client with mock tests.
- Phase D: workflow JSON mapper and weekly report card.
- Phase E: docs, examples, and Agent Skill.

Move Bitable real writes to a later phase after authentication and upsert semantics are explicit.
