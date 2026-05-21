# GitLink CLI Agent Workflow Enhancement Suite

## 1. Background

GitLink CLI serves both human maintainers and AI Agents.
The competition focuses on intelligent open-source contribution workflows,
where structured analysis, stable output, and safe automation matter more than raw command count.

## 2. Problem

Open-source maintenance often suffers from:

- Issue backlog and delayed triage
- High PR review cost
- Repetitive release note preparation
- Lack of structured repository health evaluation
- AI Agents needing stable, machine-readable output

## 3. Solution

This project extends GitLink CLI with the **GitLink CLI Agent Workflow Enhancement Suite**.

Implemented now:

- `workflow +triage`
- `workflow +health`
- `workflow +pr-summary`
- `workflow +repo-report`
- read-only GitLink fetch layer for workflow triage and health
- read-only PR metadata, changed files, and commits fetch layer for PR summary
- partial read-only repository report aggregation for health, issues, and PR list metadata
- expanded fetch boundary tests for empty responses, label and author normalization,
  error-in-body handling, alternative activity timestamps, release shapes, and CI unavailability
- local-first analysis with no LLM dependency
- stable Agent-facing JSON / table / markdown output

Planned next:

- `workflow +release-notes`
- `workflow +stale`

## 4. Technical Route

- Go + Cobra + existing shortcut architecture
- rule-based analysis
- stable DTOs
- `json` / `table` / `markdown` renderers
- `en` / `zh-CN` message mapping
- no LLM dependency
- local-first, dry-run-safe workflow design

## 5. Implemented Features

| Command | Status | Main Value |
|---|---|---|
| `workflow +triage` | Done | Issue classification, priority, missing information, and actions |
| `workflow +health` | Done | Repository health score, risk level, and recommendations |
| `workflow +pr-summary` | Done | PR risk, review focus, test suggestions, and merge checklist |
| `workflow +repo-report` | Done | Aggregated repository workflow report for maintainers and Agents |
| `workflow +release-notes` | Planned | Release note generation from PR titles and commits |
| `workflow +stale` | Planned | Stale issue and PR analysis |

### workflow +triage

- issue type detection
- priority scoring
- confidence scoring
- missing information detection
- risk flags
- recommended action
- suggested comment
- reasoning and matched rules

### workflow +health

- health score
- risk level
- metrics
- scoring notes
- recommendations
- unknown metric tolerance

### workflow +pr-summary

- change type detection
- risk level analysis
- review focus generation
- test suggestion generation
- merge checklist generation
- read-only fetch of PR metadata, changed files, and commits

### workflow +repo-report

- one-command repository workflow report
- health, issue triage, and PR summary aggregation
- report score and overall risk level
- partial report behavior when optional remote sections fail
- markdown output for competition and maintainer reports
- JSON output for Agent consumption

## 6. Innovation Points

- Agent-native structured output
- rule-based intelligence without external LLM dependency
- explainable workflow decisions
- safety-first local analysis
- bilingual command output
- extensible workflow command design
- competition-friendly incremental PR path

## 7. Testing and Verification

- Unit tests cover triage, health scoring, messages, rendering, and command helpers.
- Fetch-layer tests cover issue normalization, repository health probing,
  and PR metadata/file/commit normalization with `httptest`.
- Boundary tests cover empty responses, label and author normalization,
  error-in-body handling, alternative activity timestamps, release response shapes,
  and CI unavailability.
- PR summary tests cover docs-only, workflow code, internal client,
  security-sensitive, mixed-file, zh-CN, render, command, and fetch-failure cases.
- Repo report tests cover aggregation, scoring, JSON/table/markdown rendering,
  command wiring, local JSON input, partial fetch behavior, and include flags.
- Local command examples were executed successfully.
- Full repository testing passed in the current environment.
- Automated tests use `httptest` and do not depend on real remote API availability.

## 8. Demonstration Plan

### Official repository

Use `Gitlink/gitlink-cli` as the reference repository:

1. `workflow +triage` with English table output
2. `workflow +triage` with security JSON output
3. `workflow +triage` with Chinese markdown output
4. `workflow +health` with table output
5. `workflow +health` with risky JSON output
6. `workflow +pr-summary` with markdown output
7. `workflow +repo-report` with markdown output for the full competition story
8. Explain how agents consume stable JSON

### Self-built test repository

Use a small demo repository to show:

- bug triage
- security triage
- docs triage
- healthy repo score
- risky repo score
- full repo report from `shortcuts/workflow/testdata/repo_report.json`

## 9. Roadmap

- Phase 1: local workflow prototype, completed
- Phase 2: API fetch and normalization, completed
- Phase 3: `pr-summary`, completed
- Phase 4: `repo-report`, completed
- Phase 5: `release-notes`, `stale`

## 10. PR Plan

- PR 1: workflow rule engine and local commands
- PR 2: documentation and tests
- PR 3: API fetch layer
- PR 4: `pr-summary`
- PR 5: `repo-report`
- PR 6: `release-notes` / `stale`

## 11. Evaluation Mapping

| Criterion | Evidence |
|---|---|
| 功能完整性 20% | Four implemented commands cover Issue triage, health scoring, PR summary, and repo report |
| 创新性 20% | Agent-native JSON, explainable rules, local-first safety model, repository workflow report |
| 实用价值 20% | Reduces maintainer triage/review overhead and creates copy-ready markdown reports |
| 文档与演示 20% | README, design doc, test report, competition write-up, demo script, defense Q&A |
| 成果落地 20% | Prepared for GitLink official PR, CI verification, and maintainer review iteration |

## 12. Landing Plan

- Push the implementation branch to the public repository.
- Create a GitLink official PR against `Gitlink/gitlink-cli`.
- Record CI result and PR URL in `docs/final-submission-checklist.md`.
- Respond to maintainer review over the expected 1-2 week review cycle.
- Keep `release-notes` and `stale` as follow-up work instead of expanding this PR further.
