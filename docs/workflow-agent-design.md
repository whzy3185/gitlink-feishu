# GitLink CLI Workflow Agent Design

## Background

`gitlink-cli` already provides low-level and shortcut operations for GitLink repositories,
issues, pull requests, releases, CI, organizations, search, and users.
The repository also includes `skills/gitlink-workflow/SKILL.md`, which describes
AI workflow patterns such as Issue triage, PR review, and Release Notes generation.

The current Go command tree did not include a `workflow` command group before this work.
The competition PR turns the documented workflow concept into concrete,
deterministic CLI commands that can be used by human maintainers and AI Agents
without calling an external LLM.

## Goals

First PR:
- Add `gitlink-cli workflow +triage`.
- Add `gitlink-cli workflow +health`.
- Keep write behavior dry-run by default.
- Produce stable JSON for Agents.
- Produce concise table output for terminal users.
- Produce markdown output for reports, PR comments, Issue comments, and competition materials.
- Support `--lang en` and `--lang zh-CN` with a lightweight message helper.

Additional workflow commands:
- `workflow +pr-summary`: done
- `workflow +repo-report`: done
- `workflow +release-notes`: planned
- `workflow +stale`: planned

Current implementation status:
- Rule engine: done
- Local command layer: done
- API fetch layer: done
- Boundary tests: expanded for empty responses, field normalization,
  unknown tolerance, and read-only error handling
- PR summary command: done with local JSON input, read-only fetch, rules, renderers, and tests
- Repo report command: done with local JSON input, partial read-only fetch aggregation,
  scoring, renderers, and tests

## Current Repository Findings

Command registration:
- `cmd/root.go` registers global flags and calls `shortcuts.RegisterAll(rootCmd)`.
- `shortcuts/register.go` maps command groups to shortcut slices.
- Each group exposes `Shortcuts() []*common.Shortcut`.
- `common.MountShortcut` maps a `Shortcut` into a Cobra command named `+<name>`.

Runtime and API calls:
- `common.NewRuntimeContext` creates `client.Client`, carries owner, repo, format, and command args.
- `ctx.ResolveOwnerRepo()` resolves `--owner` / `--repo` or Git remote context.
- `ctx.CallAPI` and `ctx.CallAPIWithQuery` call `internal/client`.
- `client.Do` appends `.json`, injects auth via transport, parses GitLink error-in-body responses, and returns `output.Envelope`.

Output:
- `internal/output` currently supports `json`, `yaml`, and generic `table`.
- Workflow requires `markdown`; the minimal-risk approach is a workflow-local renderer that prints stable workflow DTOs.
- A later cleanup can promote markdown support into `internal/output` if multiple command groups need it.
- Current workflow commands also expose workflow-local `json`, `table`, and `markdown` rendering without changing the global formatter.

Testing:
- Existing tests use pure unit tests plus `httptest.Server`.
- Shortcut tests instantiate `common.RuntimeContext` manually with a mocked `client.Client`.
- This pattern should be reused for workflow API tests.

## Command Design

### `workflow +triage`

Examples:

```bash
gitlink-cli workflow +triage --owner Gitlink --repo gitlink-cli --state open --limit 30 --dry-run --format json
gitlink-cli workflow +triage --owner Gitlink --repo gitlink-cli --state open --limit 30 --format table
gitlink-cli workflow +triage --owner Gitlink --repo gitlink-cli --state open --limit 30 --lang zh-CN --format markdown
```

Flags:
- `--state`: default `open`
- `--limit`: default `30`
- `--page`: default `1`
- `--dry-run`: default `true`
- `--from`: optional local JSON input
- `--title`, `--body`, `--number`, `--author`, `--url`, `--labels`: optional local single-issue input
- `--lang`: default `en`, allowed `en`, `zh-CN`

Stable JSON item fields:
- `issue_id`
- `number`
- `title`
- `url`
- `author`
- `state`
- `created_at`
- `updated_at`
- `detected_type`
- `priority`
- `confidence`
- `suggested_labels`
- `missing_information`
- `risk_flags`
- `recommended_action`
- `suggested_comment`
- `reasoning`

Rule categories:
- `bug`
- `feature`
- `question`
- `docs`
- `ci`
- `security`
- `performance`
- `refactor`
- `unknown`

Priority:
- `P0`: security incident, secret/token leak, auth bypass, repository unusable
- `P1`: core command unusable, install/login failure, CI/release blocker
- `P2`: normal bug, important feature, missing docs blocking usage
- `P3`: ordinary question, typo, minor improvement

Missing information for bug-like issues:
- reproduction steps
- expected behavior
- actual behavior
- version
- OS / platform
- command output
- logs

### `workflow +health`

Examples:

```bash
gitlink-cli workflow +health --owner Gitlink --repo gitlink-cli --format json
gitlink-cli workflow +health --owner Gitlink --repo gitlink-cli --format table
gitlink-cli workflow +health --owner Gitlink --repo gitlink-cli --lang zh-CN --format markdown
```

Flags:
- `--stale-days`: default `30`
- `--from`: optional local JSON input
- local metric flags such as `--repository`, `--open-issues`, `--open-prs`, `--has-readme`, `--has-license`, and `--agent-readiness-score`
- `--lang`: default `en`

Stable JSON fields:
- `repository`
- `open_issues`
- `open_prs`
- `stale_issues`
- `stale_prs`
- `recent_activity`
- `release_status`
- `ci_status`
- `documentation_status`
- `license_status`
- `contribution_status`
- `agent_readiness_score`
- `health_score`
- `risk_level`
- `recommendations`
- `scoring_notes`

Scoring:
- Issue backlog and response: 20
- PR backlog and merge state: 20
- Recent activity: 15
- Release status: 15
- Documentation completeness: 10
- License and contribution readiness: 10
- Agent readiness: 10

Unknown metric policy:
- Keep field present.
- Set status or score detail to `unknown`.
- Add one entry to `scoring_notes`.
- Either omit the metric from denominator or apply a conservative partial score; the first PR should prefer denominator adjustment to avoid fake precision.

Risk levels:
- `low`: 80-100
- `medium`: 60-79
- `high`: 40-59
- `critical`: 0-39

## Architecture

Proposed files:

```text
shortcuts/workflow/
  workflow.go          # Shortcuts() and command wiring
  types.go             # Stable DTOs
  triage_rules.go      # pure classifier, scoring, missing info detection
  triage_fetch.go      # GitLink issue fetching and response normalization
  triage_render.go     # json/table/markdown workflow rendering if needed
  health_score.go      # pure health scoring
  health_fetch.go      # repo, issue, PR, release, CI/doc/license probes
  health_render.go     # markdown/table rendering
  messages.go          # en and zh-CN strings
  *_test.go
```

Registration:
- Add `workflow` import in `shortcuts/register.go`.
- Add `"workflow": workflow.Shortcuts()` to `groups`.
- Add description `"AI agent workflow analysis"`.

No new dependency is needed for this PR.

## Data Normalization

GitLink responses vary by endpoint. Workflow code should not depend on a single raw shape. Add small extraction helpers:

- `stringField(map, keys...)`
- `numberField(map, keys...)`
- `timeField(map, keys...)`
- `sliceField(map, keys...)`
- `extractItems(env, candidateKeys...)`

Candidate issue list keys:
- `issues`
- `data`
- direct array after future client improvements

Candidate issue fields:
- ID: `id`, `issue_id`
- Number: `project_issues_index`, `number`, `index`, `id`
- Title: `subject`, `title`
- Body: `description`, `body`
- Author: `author.login`, `user.login`, `login`
- URL: `html_url`, `url`, `issue_url`

Health activity fields currently tolerated:
- `updated_at`
- `updatedAt`
- `last_updated_at`
- `lastUpdatedAt`
- `last_activity_at`
- `lastActivityAt`
- `merged_at`
- `mergedAt`
- `closed_at`
- `closedAt`

## Safety Strategy

- `+triage` only reads by default.
- `--dry-run` defaults true.
- A future explicit write flag for posting comments must require `--dry-run=false` in a later PR.
- Generated comments are output as data, not posted remotely in the first PR.
- Health checks never mutate remote state.
- If an API probe fails, health continues with `unknown`.
- The implemented prototype is local-first and has no LLM dependency.
- Remote fetch mode remains read-only and does not post comments, labels, merges, or close actions.
- API failures should fall back to `unknown` metrics or a clear fetch error instead of fabricating healthy data.

## Core Pseudocode

### Triage

```go
issues := fetchIssues(owner, repo, state, limit, page)
results := []TriageResult{}
for _, issue := range issues {
    text := normalize(issue.Title + "\n" + issue.Body)
    scores := scoreKeywords(text, keywordRules)
    detectedType := maxScoreType(scores)
    priority := scorePriority(text, detectedType)
    missing := detectMissingInfo(issue, detectedType)
    confidence := confidenceFromScores(scores, missing)
    result := TriageResult{
        IssueID: issue.ID,
        Number: issue.Number,
        DetectedType: detectedType,
        Priority: priority,
        SuggestedLabels: labelsFor(detectedType, priority, riskFlags),
        MissingInformation: missing,
        RiskFlags: detectRiskFlags(text),
        RecommendedAction: actionFor(detectedType, priority, missing, lang),
        SuggestedComment: commentFor(missing, lang),
        Reasoning: explainTopMatches(scores, priorityRules),
    }
    results = append(results, result)
}
render(results, format, lang)
```

### Health

```go
signals := collectHealthSignals(owner, repo)
score := NewWeightedScore(100)
score.Add("issues", 20, scoreIssueBacklog(signals.OpenIssues, signals.StaleIssues))
score.Add("prs", 20, scorePRBacklog(signals.OpenPRs, signals.StalePRs))
score.Add("activity", 15, scoreRecentActivity(signals.RecentActivity))
score.Add("release", 15, scoreReleaseStatus(signals.ReleaseStatus))
score.Add("docs", 10, scoreDocStatus(signals.DocumentationStatus))
score.Add("license", 10, scoreLicenseContribution(signals.LicenseStatus, signals.ContributionStatus))
score.Add("agent", 10, scoreAgentReadiness(signals))
result := HealthResult{
    HealthScore: score.Percent(),
    RiskLevel: riskLevel(score.Percent()),
    Recommendations: recommendations(signals, score),
    ScoringNotes: score.Notes(),
}
render(result, format, lang)
```

## Output Protocol

JSON:
- Use stable struct tags.
- Include empty arrays as `[]` where useful for Agent consumption.
- Avoid prose outside JSON.

Table:
- Triage columns: `NUMBER`, `TYPE`, `PRIORITY`, `CONFIDENCE`, `MISSING`, `ACTION`
- Health rows: `METRIC`, `STATUS`, `SCORE`, `NOTE`

Markdown:
- Triage: one summary table with type, priority, confidence, action, and missing information.
- Health: repository score, metric table, recommendations, and scoring notes.
- `zh-CN` changes rule messages and recommendation text, not JSON field names.

## Test Plan

Unit tests:
- Issue type classification.
- Priority scoring.
- Missing information detection.
- Risk flag detection.
- Suggested comment generation.
- Health weighted score and risk level.
- Unknown metric denominator adjustment.
- Markdown headings and required sections.

Mock API tests:
- `workflow +triage` fetches issues and normalizes raw response.
- `workflow +health` tolerates failing CI/release/doc probes.

Command tests:
- `--dry-run` defaults to true.
- `--lang zh-CN` accepted.
- invalid `--lang` falls back to `en`.
- `--format markdown` routes to markdown renderer.

## Later Extensions

### `workflow +pr-summary`

Inputs:
- `--number`
- `--from`
- `--lang`
- `--format`
- optional `--include-files`
- optional `--include-commits`
- optional `--max-files`
- optional `--max-commits`

Default format:
- `table` for human review when `--format` is omitted

Data:
- PR details
- changed files
- commits

Output:
- `change_type`
- `risk_level`
- `review_focus`
- `test_suggestions`
- `merge_checklist`
- `reasoning`

Implementation status:
- read-only local JSON mode: done
- read-only GitLink fetch mode: done
- rules and renderers: done
- tests: rules, fetch boundary, render, and command wiring

Safety:
- no comments
- no approve/reject
- no merge
- no remote write operation

### `workflow +repo-report`

Inputs:
- `--owner`
- `--repo`
- `--from`
- `--lang`
- `--format`
- optional `--issue-limit`
- optional `--pr-limit`
- optional `--stale-days`
- optional `--include-issues`
- optional `--include-prs`
- optional `--include-health`

Default format:
- `markdown` for maintainer and competition reports when `--format` is omitted

Data:
- repository health input and score
- issue triage results aggregated by type, priority, risk, and missing information
- PR summary results aggregated by type, risk, and review focus

Output:
- `report_score`
- `risk_level`
- `health`
- `issue_summary`
- `pr_summary`
- `recommendations`
- `reasoning`

Partial report strategy:
- health, issue, and PR sections are fetched independently
- if at least one enabled section succeeds, the command returns a partial report
- failed sections are recorded in scoring notes or reasoning
- PR remote aggregation currently uses PR list metadata only;
  detailed changed files and commits remain available through `workflow +pr-summary --number`

Safety:
- read-only aggregation only
- no comments, labels, closes, approve/reject, or merge operations
- no LLM dependency

### `workflow +release-notes`

Inputs:
- `--from`
- `--to`
- optional `--tag`
- optional `--lang`

Data:
- PR titles
- commit messages

Markdown categories:
- Features
- Bug Fixes
- Documentation
- Tests
- Refactoring
- Chores
- Breaking Changes

### `workflow +stale`

Inputs:
- `--stale-days`
- `--state`
- `--dry-run`

Behavior:
- Identify stale issues and PRs.
- Generate suggested comments or labels.
- Do not mutate remote state by default.

## API Fetch Layer

The current fetch layer uses:

- `triage_fetch.go`
- `health_fetch.go`
- `pr_fetch.go`
- `repo_report_fetch.go`

Design goals already applied:

- tolerate unknown or partial API fields
- map GitLink response shapes into stable workflow DTOs
- continue operating when optional signals fail
- keep remote-write actions disabled until explicitly enabled later

Planned fetch-layer extension:

- `triage_fetch.go` and `health_fetch.go` remain the normalization boundary for remote mode.
- `pr_fetch.go` now reuses the same stable DTO and message patterns for read-only PR metadata, changed files, and commits.
- `repo_report_fetch.go` composes the existing fetch helpers and records partial failures instead of failing the whole report.
- Future `release-notes` should reuse the same normalization and renderer patterns.
- Unknown or missing fields should stay explicit in JSON output so Agents can decide how to proceed.

## Implementation Order

1. Pure DTOs and rule engine.
2. Pure health scoring.
3. Workflow renderers.
4. Command registration.
5. API fetch and normalization.
6. Tests.
7. README updates.
8. Competition docs and test report.
