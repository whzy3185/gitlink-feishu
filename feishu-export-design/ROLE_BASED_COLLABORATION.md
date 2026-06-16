# Role-Based Feishu Collaboration Design

## Purpose

Extend the Feishu export workflow from a simple report sender into a role-aware collaboration layer.

The product should not notify every maintainer about every Pull Request event. It should separate the two communication needs:

```text
Owner / maintainer: periodic, summarized, prioritized project state.
Contributor: immediate, personal feedback for work they own.
```

This keeps Feishu useful as a collaboration surface instead of turning it into a noisy event stream.

## Role Model

### Owner View

Owners need batch summaries and decision support.

Default owner delivery:

```text
daily digest
weekly report
milestone status
review queue summary
high-risk PR summary
stale contribution summary
```

Owners should receive:

```text
repository status
new contributor activity
PRs grouped by review stage
PRs blocked by conflicts or required rebase
PRs close to merge
PRs needing owner decision
review coverage and stale review data
links to Feishu Doc / Wiki pages for full context
```

Owners should not receive by default:

```text
one message for every new PR
one message for every comment
one message for every patchset push
one message for every review reply
```

### Contributor View

Contributors need immediate feedback on their own work.

Default contributor delivery:

```text
review comment received
review status changed
changes requested
rebase required
merge conflict detected
CI or quality gate failed
PR approved
PR merged
PR refused or closed
maintainer requested more information
```

Contributor notifications should be personal and direct where Feishu identity mapping is available. If open_id mapping is not configured, the system should fall back to repository-level cards or dry-run output.

## Event Strategy

### Owner Events

Owner notifications are aggregation jobs, not raw events.

Recommended command shape:

```bash
gitlink-cli feishu +owner-digest \
  --owner <owner> \
  --repo <repo> \
  --period daily \
  --webhook-url "$FEISHU_WEBHOOK_URL" \
  --send
```

Alternative input-only flow:

```bash
gitlink-cli workflow +repo-report --owner <owner> --repo <repo> --format json > report.json
gitlink-cli feishu +owner-digest --from-workflow-json report.json --send
```

The first implementation should prefer the input-only flow. Direct GitLink collection can come later once the data model is stable.

### Contributor Events

Contributor notifications can be real-time if GitLink has webhooks, or near-real-time through polling.

Recommended command shape:

```bash
gitlink-cli feishu +contributor-notify \
  --from-event-json event.json \
  --send
```

Polling shape:

```bash
gitlink-cli feishu +contributor-watch \
  --owner <owner> \
  --repo <repo> \
  --interval 5m \
  --state-file .gitlink-feishu-state.json \
  --send
```

The state file records delivered event IDs so repeated polling does not resend old notifications.

## Feishu Channel Matrix

Different Feishu surfaces should not be used for the same job.

| Surface | Best Fit | Delivery Style | Current Status |
| --- | --- | --- | --- |
| Custom group bot webhook | Owner digest, weekly report, project status card | One-way group card | Implemented for current notify/report path |
| Self-built app IM API | Personal contributor notifications | Direct message or mention | Future, needs open_id mapping and IM scopes |
| DocX / Wiki | Long-form report, README mirror, project knowledge base | Persistent document | Experimental `+doc-export` exists |
| Bitable records | PR stage table, milestones, dashboard source | Structured rows | Dry-run records only |
| Bitable Gantt view | Milestone timeline | Visual project planning | Manual view first, OpenAPI later |
| Feishu AI summary / weekly report | Summarize generated docs and records | Feishu-side automation | Do not depend on private API |

Practical rule:

```text
Card = attention.
Doc / Wiki = context.
Bitable = structured state.
Gantt = milestone visualization.
AI summary = Feishu-side value added on top of structured content.
```

## Scheduling Model

Owner notifications should be scheduled. Contributor notifications should be event-driven where possible.

Owner cadence:

```text
daily: review queue and blocked items
weekly: contributor activity, milestone progress, risk trend
on demand: full report preview or manual send
```

Contributor cadence:

```text
immediate: review/comment/merge/rebase/conflict events
debounced: repeated comments in the same PR within a short window
digest fallback: if personal identity mapping is missing
```

Recommended debounce rule:

```text
If the same contributor receives multiple comments on the same PR within 10 minutes,
combine them into one notification with a count and latest link.
```

This avoids replacing owner spam with contributor spam.

## Pull Request Stage Colors

Feishu cards should use color as a stage signal, not as decoration.

Default stage rules:

| Stage | Card Color | Meaning | Typical Inputs |
| --- | --- | --- | --- |
| `new` | blue | New PR, not reviewed yet | no reviews, one patchset, recently opened |
| `active-review` | grey | Review is active, no clear risk yet | comments or common reviews exist |
| `near-ready` | green | Small gap to merge | approved or low-risk review, no conflict, checks pass |
| `needs-rebase` | yellow | Contributor action needed before review can continue | base branch changed, conflict, stale branch, merge check failed |
| `major-changes` | orange | Larger change request or high-risk delta | rejected review, large diff, missing tests, repeated review cycles |
| `blocked` | red | Owner or platform action needed | permission issue, failing required checks, unresolved dependency |
| `merged` | green | Completed | merged status |
| `closed` | grey | Closed without merge | closed/refused status |

The first stable implementation should support defaults only. User customization can be added as a config file after the stage model is proven.

Recommended config shape:

```yaml
feishu:
  pr_stages:
    near_ready:
      color: green
      max_unresolved_comments: 2
      require_no_conflicts: true
    needs_rebase:
      color: yellow
      require_mergeable: false
    major_changes:
      color: orange
      min_review_rounds: 2
      high_risk_labels:
        - missing-tests
        - large-diff
```

## Review Degree Model

The stage calculation should be explainable. A card should not only show a color; it should also show why the PR is in that stage.

Recommended derived fields:

```text
review_rounds
patchset_count
last_review_status
unresolved_comment_count
requested_changes_count
approved_count
changed_files_count
additions
deletions
mergeable
needs_rebase
ci_status
last_activity_at
```

GitLink anchors already available in the CLI ecosystem:

```text
pr +list
pr +view
pr +reviews
pr +versions
pr +files
```

The design should avoid scraping pages. It should use existing GitLink APIs or existing CLI JSON outputs.

## Stage Classification Order

The stage classifier should be deterministic. Later rules should not override higher-priority terminal or blocking states.

Recommended order:

```text
1. merged
2. closed
3. blocked
4. needs-rebase
5. major-changes
6. near-ready
7. active-review
8. new
```

Classification logic:

```text
merged:
  pull_request_status == merged

closed:
  pull_request_status == closed/refused

blocked:
  required check failed, permission issue, unresolved dependency, or owner-defined blocked label

needs-rebase:
  mergeable == false, conflict exists, stale base branch, or merge check says rebase is required

major-changes:
  last review rejected, requested_changes_count > 0, large diff threshold exceeded, or repeated review rounds

near-ready:
  approved_count > 0, no conflict, no requested changes, low remaining risk

active-review:
  common review/comment exists, patchset_count > 1, or maintainer has interacted

new:
  no review, no maintainer interaction, recently opened
```

Every stage output should include `reasons`:

```json
{
  "stage": "needs-rebase",
  "color": "yellow",
  "reasons": [
    "merge check failed",
    "base branch changed after latest patchset"
  ]
}
```

## Data Contracts

The role-aware extension should accept local JSON first. This keeps tests deterministic and avoids changing existing GitLink network behavior.

### Owner Digest Input

Recommended minimal JSON:

```json
{
  "repository": "Gitlink/gitlink-cli",
  "period": {
    "start": "2026-06-09",
    "end": "2026-06-16"
  },
  "pull_requests": [
    {
      "number": 123,
      "title": "feat: add export flow",
      "author": "contributor-a",
      "url": "https://www.gitlink.org.cn/org/repo/pulls/123",
      "status": "open",
      "stage": "needs-rebase",
      "color": "yellow",
      "reasons": ["merge check failed"],
      "review_rounds": 2,
      "patchset_count": 3,
      "last_activity_at": "2026-06-16T10:30:00+08:00"
    }
  ],
  "milestones": [],
  "contributors": []
}
```

### Owner Digest Output

Recommended output:

```json
{
  "repository": "Gitlink/gitlink-cli",
  "period_label": "2026-06-09 to 2026-06-16",
  "stage_counts": {
    "near-ready": 3,
    "needs-rebase": 2,
    "major-changes": 1,
    "new": 4
  },
  "top_actions": [
    "Review 4 new PRs",
    "Ask 2 contributors to rebase",
    "Merge 3 near-ready PRs"
  ],
  "doc_url": "https://example.feishu.cn/wiki/...",
  "dry_run": true
}
```

### Contributor Event Input

Recommended minimal event JSON:

```json
{
  "event_id": "repo-pr-123-review-456",
  "event_type": "review_comment",
  "repository": "Gitlink/gitlink-cli",
  "pr": {
    "number": 123,
    "title": "feat: add export flow",
    "url": "https://www.gitlink.org.cn/org/repo/pulls/123",
    "author": "contributor-a"
  },
  "actor": "maintainer-a",
  "recipient_gitlink_user": "contributor-a",
  "summary": "Maintainer requested changes in the export options.",
  "required_action": "Update the PR and push a new patchset.",
  "created_at": "2026-06-16T10:30:00+08:00"
}
```

### Contributor Notification Output

Recommended output:

```json
{
  "event_id": "repo-pr-123-review-456",
  "recipient_gitlink_user": "contributor-a",
  "recipient_feishu_id": "",
  "delivery_mode": "dry-run",
  "card_title": "PR feedback received",
  "required_action": "Update the PR and push a new patchset.",
  "dry_run": true
}
```

If `recipient_feishu_id` is empty, direct personal delivery must not be attempted.

## Owner Digest Card

Owner digest cards should be compact and action-oriented.

Recommended sections:

```text
1. Repository and period
2. Review queue by stage
3. Near-ready PRs
4. PRs needing rebase
5. High-risk or major-change PRs
6. New contributors
7. Stale PRs
8. Milestone progress
9. Link to Feishu Wiki / Doc full report
```

Example card semantics:

```text
Header: GitLink Owner Digest
Green section: 3 PRs close to merge
Yellow section: 2 PRs need rebase
Orange section: 1 PR needs major changes
Grey section: 4 new/unreviewed PRs
Button: Open Feishu report
Button: Open GitLink PR queue
```

The owner card should cap inline PR rows. A full report belongs in Feishu Doc / Wiki.

Recommended card limits:

```text
maximum stage groups shown: 5
maximum PR rows per stage: 3
maximum total inline PR rows: 10
always include full report link when available
```

If the owner digest exceeds the inline limits, the card should say how many rows are hidden and link to the Doc / Wiki report.

## Contributor Notification Card

Contributor cards should be immediate and specific.

Recommended sections:

```text
1. PR title and repository
2. Event type
3. Reviewer or actor
4. Required action
5. Short feedback summary
6. Link to PR
7. Link to Feishu reference doc if relevant
```

Example event mapping:

| Event | Card Intent |
| --- | --- |
| review comment | Read maintainer feedback |
| rejected review | Modify PR according to requested changes |
| approved review | Wait for merge or owner decision |
| merged | Contribution accepted |
| needs rebase | Rebase branch before further review |
| conflict | Resolve merge conflict |

Contributor delivery requires identity mapping:

```text
GitLink username -> Feishu open_id / union_id / email
```

Until that mapping exists, the CLI should generate dry-run records instead of attempting direct personal delivery.

## Identity Mapping

Identity mapping is a separate concern from PR analysis.

Supported mapping sources, in priority order:

```text
1. explicit local mapping file
2. Bitable mapping table
3. email match from GitLink user profile and Feishu directory
4. no mapping, dry-run only
```

First implementation should only support the local file:

```yaml
contributors:
  contributor-a:
    feishu_open_id: ou_xxx
    display_name: Contributor A
  contributor-b:
    email: contributor-b@example.com
```

Validation rules:

```text
mapping file is optional
missing mapping downgrades to dry-run
mapping values are redacted in logs
the CLI does not call Feishu directory APIs in the first pass
```

## Feishu Docs / Wiki

Feishu Docs and Wiki should be treated as the long-form project artifact.

Recommended generated content:

```text
project overview
README summary
contribution guide summary
review policy
milestone plan
daily or weekly owner digest archive
PR stage table
high-risk change notes
```

README export should not replace the repository README. It should produce a Feishu-readable version for maintainers and contributors.

Recommended command shape:

```bash
gitlink-cli feishu +readme-doc \
  --owner <owner> \
  --repo <repo> \
  --wiki-url "<wiki_url>" \
  --send
```

Permission boundary:

```text
The owner configures Feishu app scopes and document permissions.
The CLI never changes Feishu document permissions automatically.
The CLI prints permission diagnostics when write access fails.
```

This matches the current `+doc-export` boundary and avoids hidden permission changes.

## README and Knowledge Base Export

The README export should be deterministic and conservative.

Recommended sections:

```text
1. Project title
2. Short repository summary
3. Quick start
4. Contribution workflow
5. Review policy
6. Current milestones
7. Current owner digest link
8. Source repository links
```

The command should accept local files before remote reads:

```bash
gitlink-cli feishu +readme-doc \
  --from-readme README.md \
  --from-contributing CONTRIBUTING.md \
  --wiki-url "<wiki_url>" \
  --format markdown
```

Later remote mode can use GitLink repository file APIs:

```bash
gitlink-cli feishu +readme-doc \
  --owner <owner> \
  --repo <repo> \
  --ref master \
  --wiki-url "<wiki_url>"
```

Doc write behavior:

```text
preview by default
--send required for document writes
append or update target must be explicit
do not change sharing settings
return permission diagnostics on 403
```

## Milestones and Gantt

Gantt-style planning belongs to milestone tracking, not raw notification cards.

Recommended data model:

```text
milestone_id
milestone_title
start_date
due_date
status
linked_issues
linked_prs
owner
progress_percent
risk_level
```

Feishu implementation options:

```text
Doc / Wiki: milestone narrative and current status.
Bitable records: structured milestone rows.
Bitable Gantt view: created manually by owner at first.
Later OpenAPI sync: update milestone rows after table IDs are configured.
```

The first implementation should only generate milestone-ready records and Doc content. Automatic Bitable view creation should remain out of scope until real Bitable writes are implemented.

Recommended Bitable milestone fields:

```text
milestone_key
repository
title
owner
start_date
due_date
status
progress_percent
risk_level
linked_prs
linked_issues
last_updated_at
```

Manual Gantt setup:

```text
1. Owner creates a Bitable table using generated schema.
2. Owner imports generated milestone records.
3. Owner creates a Gantt view from start_date and due_date.
4. Later CLI sync updates rows, not views.
```

## Feishu AI Summary Fit

The CLI should not depend on a private Feishu AI summary API for the first implementation.

Instead, the CLI should generate structured Feishu Docs and cards that are easy for Feishu-side summary, daily report, weekly report, and knowledge-base features to consume.

Practical split:

```text
gitlink-cli: collect, normalize, stage, render, send.
Feishu: summarize, archive, search, collaborate, display.
Owner: configure permissions, choose digest schedule, tune stage rules.
```

## Configuration File

A future config file should keep project policy out of command-line flags.

Recommended path:

```text
.gitlink-feishu.yaml
```

Recommended shape:

```yaml
repository: Gitlink/gitlink-cli

owner_digest:
  enabled: true
  cadence: weekly
  webhook_env: FEISHU_WEBHOOK_URL
  doc_url_env: FEISHU_PROJECT_DOC_URL
  inline_limit: 10

contributor_notifications:
  enabled: true
  delivery: dry-run
  identity_mapping: .gitlink-feishu-users.yaml
  debounce_window: 10m

pr_stage_rules:
  near_ready:
    color: green
    require_approved: true
    require_no_conflicts: true
  needs_rebase:
    color: yellow
    require_mergeable: false
  major_changes:
    color: orange
    min_review_rounds: 2
    min_changed_files: 20

docs:
  wiki_url_env: FEISHU_PROJECT_WIKI_URL
  readme_sources:
    - README.md
    - CONTRIBUTING.md

milestones:
  enabled: true
  records_only: true
```

Rules:

```text
environment variable names may be stored
secret values must not be stored
unknown config keys should warn, not crash
invalid stage colors should fail validation
```

## Implementation Phases

### Phase A: Role-Aware Dry Run

Add local outputs only:

```text
owner digest model
contributor event model
PR stage model
default stage color rules
milestone record model
README-to-doc preview model
```

Commands:

```text
feishu +owner-digest --from-workflow-json
feishu +pr-stage-report --from-pr-json
feishu +contributor-events --from-event-json
feishu +readme-doc --from-readme
```

No GitLink writes. No Feishu writes by default.

Acceptance:

```text
owner digest JSON is stable
PR stage classification is deterministic
card color is derived from stage
missing optional fields do not panic
large input is capped in card preview
all tests use fixtures
```

### Phase B: Owner Digest Send

Enable bot cards for aggregated owner summaries:

```text
feishu +owner-digest --send
```

Use custom bot webhook, same safety model as current `+notify`.

Acceptance:

```text
--send is required for webhook delivery
--send without webhook URL fails
--send --dry-run fails
webhook URL is redacted
mock HTTP tests cover 200, 400, 429, and 500
```

### Phase C: Contributor Direct Notifications

Add contributor delivery after identity mapping exists:

```text
GitLink username -> Feishu user ID
```

Supported delivery modes:

```text
custom group bot mention
self-built app IM message
dry-run only if identity mapping is missing
```

Acceptance:

```text
missing identity mapping downgrades to dry-run
direct message mode requires explicit --send
recipient IDs are redacted in logs
event_id/state prevents duplicate delivery
debounce behavior is covered by tests
```

### Phase D: Docs / Wiki Project Space

Extend experimental document export:

```text
README summary
owner digest archive
milestone page
PR stage table
```

Keep all document writes behind `--send`.

Acceptance:

```text
README preview renders markdown
DocX/Wiki write requires app credentials and --send
403 errors include permission diagnostics
document permission changes are not attempted
```

### Phase E: Milestone / Gantt Data

Generate milestone-ready Bitable records first.

Real Bitable sync can follow only after:

```text
tenant token flow
table IDs
unique keys
upsert behavior
permission diagnostics
partial failure handling
```

Acceptance:

```text
milestone records include stable unique keys
records are usable for manual Bitable import
no Bitable OpenAPI calls happen without explicit send behavior
Gantt view creation remains manual in this phase
```

## Non-Goals

Do not implement in the first role-aware extension:

```text
automatic GitLink merge
automatic GitLink review
automatic GitLink comments
automatic Feishu permission changes
automatic Bitable view creation
direct dependency on BotBuilder
notification spam for every owner-visible event
```

## Design Verdict

This direction is stronger than a raw PR event notifier.

The product should be:

```text
Owner: Feishu digest and knowledge-base workspace.
Contributor: immediate personal feedback loop.
Project: Docs/Wiki for long-form context, Bitable/Gantt for milestone tracking.
```

The current implementation already supports the lowest-risk part:

```text
workflow JSON -> Feishu card / weekly report / Doc link / Bitable-ready records
```

The next useful design step is to add role-aware models and dry-run outputs before adding new Feishu or GitLink network behavior.
