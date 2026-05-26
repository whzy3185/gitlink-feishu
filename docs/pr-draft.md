# feat(workflow): add agent workflow commands for repository maintenance

## Summary

This PR adds four read-only workflow commands for repository maintenance:

- `workflow +triage`
- `workflow +health`
- `workflow +pr-summary`
- `workflow +repo-report`

The commands provide rule-based, explainable analysis with stable `json`, concise `table`,
and copy-friendly `markdown` output.

## Motivation

Open-source maintainers often spend time on repetitive information organization before
making actual decisions:

- Issue triage cost
- PR review cost
- repository health visibility
- Agent needs stable structured output

This PR adds workflow-level analysis on top of the existing GitLink CLI shortcut architecture
without introducing LLM dependencies or remote write behavior.

## Changes

### `workflow +triage`

- Classifies issues by type
- Scores priority and confidence
- Detects missing bug-report information
- Produces risk flags, recommended actions, suggested comments, and reasoning

### `workflow +health`

- Scores repository health
- Covers issue/PR backlog, activity, release, CI, docs, license, contributing, and Agent readiness signals
- Tolerates unknown metrics without failing the command

### `workflow +pr-summary`

- Summarizes PR metadata, changed files, and commits
- Produces change type, risk level, review focus, test suggestions, merge checklist, and reasoning
- Supports local JSON input and remote read-only PR fetch

### `workflow +repo-report`

- Aggregates health, issue triage, and PR summary signals
- Produces a repository workflow report with score, risk level, recommendations, and reasoning
- Supports partial read-only remote aggregation when optional sections are unavailable

## Safety

- Remote mode is read-only
- No LLM dependency
- No labels/comments/close operations
- No PR approve/reject/merge operations
- No `internal/output` change
- No new third-party dependency
- Test fixtures do not contain secrets or tokens

## Tests

```bash
gofmt -w shortcuts/workflow/*.go shortcuts/register.go
go test ./shortcuts/workflow
go test ./...
```

Coverage includes:

- triage rules
- health scoring
- PR summary rules
- repo report aggregation
- fetch normalization
- partial failure handling
- `json` / `table` / `markdown` rendering
- local `--from` fixtures
- command wiring tests

## Documentation

- `README.md`
- `docs/workflow-agent-design.md`
- `docs/workflow-agent-test-report.md`
- `skills/gitlink-workflow/SKILL.md`

## Known Limitations

- `workflow +release-notes` is not implemented.
- `workflow +stale` is not implemented.
- Real GitLink API shapes may require follow-up normalization.

## Examples

```bash
gitlink-cli workflow +triage --from shortcuts/workflow/testdata/issue_bug.json --format table
gitlink-cli workflow +health --from shortcuts/workflow/testdata/health_good.json --format markdown
gitlink-cli workflow +pr-summary --from shortcuts/workflow/testdata/pr_summary.json --format markdown
gitlink-cli workflow +repo-report --from shortcuts/workflow/testdata/repo_report.json --format markdown
```
