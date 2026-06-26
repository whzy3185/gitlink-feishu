# GitLink CLI Capability Boundary

Date: 2026-06-26

## Purpose

This document defines the current `gitlink-cli` capability boundary before the Feishu integration grows from export-only reporting into a permissioned collaboration gateway.

The key rule is:

```text
Feishu can become an entry point, but gitlink-cli remains the authoritative GitLink data reader and action executor.
```

Any Feishu-triggered GitLink action must respect this boundary:

```text
Feishu callback -> validate Feishu identity -> validate repo binding -> map to gitlink-cli action -> dry-run -> confirm -> execute -> audit
```

## Capability Levels

Use these levels when deciding whether a GitLink command can be exposed through Feishu.

| Level | Name | Meaning | Feishu Gateway Policy |
| --- | --- | --- | --- |
| Level 0 | Read / Local Analysis | Reads GitLink data or analyzes local input | Safe for Stage 1 cards, docs, Bitable-ready records |
| Level 1 | Low-Risk Write | Adds reversible or additive collaboration data | Stage 3 only, requires identity mapping, dry-run, confirmation, audit |
| Level 2 | Medium-Risk Write | Changes project state but is usually recoverable | Planned only, disabled by default |
| Level 3 | High-Risk Write | Merge, delete, close, permission, or membership changes | Do not expose by default; requires explicit dangerous-action opt-in |
| Admin | Credential / Raw API | Auth, config, arbitrary API calls | Do not expose through Feishu cards |

## Current Command Surface

Top-level command groups currently available:

```text
api
auth
branch
ci
compare
config
dataset
doctor
feishu
health
ignore
issue
label
license
member
milestone
org
pipeline
pr
profile
release
repo
search
user
webhook
workflow
```

## Level 0: Read and Local Analysis

These commands are appropriate inputs for Feishu reports, Bitable records, project dashboards, and owner / contributor digest cards.

### Repository Read

```text
repo +list
repo +info
repo +readme
repo +tree
repo +languages
repo +contributors
repo +contributor-stats
repo +code-stats
repo +watchers
repo +stargazers
```

Feishu use:

```text
project overview
README / Wiki mirror
contributor dashboard
repository health report
organization cockpit
```

### Issue Read

```text
issue +list
issue +view
issue +authors
issue +assigners
issue +priorities
issue +statuses
issue +tags
```

Feishu use:

```text
issue risk summary
triage queue
personal task panel
dashboard by priority / status / stale age
```

### Pull Request Read

```text
pr +list
pr +view
pr +files
pr +diff
pr +versions
pr +version-diff
pr +reviews
```

Feishu use:

```text
PR stage cards
review queue
contributor feedback digest
rebase / conflict / review-round tracking
near-ready merge list
```

### CI and Pipeline Read

```text
ci +builds
ci +logs
pipeline +list
pipeline +view
pipeline +runs
pipeline +logs
pipeline +results
pipeline +save-yaml
```

Feishu use:

```text
CI failure digest
release readiness report
pipeline health dashboard
PR risk enrichment
```

### Webhook Read

```text
webhook +list
webhook +view
webhook +tasks
```

Feishu use:

```text
integration diagnostics
delivery failure summary
owner operational report
```

### Member / Organization Read

```text
member +list
member +invite-info
org +list
org +info
org +members
```

Feishu use:

```text
maintainer roster
reviewer capacity view
repository permission audit preview
```

### Milestone / Release / Dataset / Label Read

```text
milestone +list
milestone +view
release +list
release +view
release +edit
dataset +list
dataset +view
label +list
license +list
```

Feishu use:

```text
milestone Gantt source
release calendar
dataset inventory
issue label taxonomy
```

### Search / User / Profile / Compare

```text
search +repos
search +users
user +me
user +info
profile +ability
profile +activity
profile +contribution
profile +major
profile +role
compare +view
compare +files
```

Feishu use:

```text
contributor profile enrichment
organization talent view
release diff summary
project discovery
```

### Workflow and Health Analysis

```text
workflow +triage
workflow +health
workflow +pr-summary
workflow +repo-report
health +fetch
doctor
version
```

Feishu use:

```text
weekly report
owner digest
issue triage card
PR review summary
repository health dashboard
```

Boundary:

```text
Workflow commands are the safest first-class source for Feishu Stage 1.
They should remain read-only or local-analysis commands.
```

## Level 1: Low-Risk Write

These actions add collaboration information but do not normally destroy project state.

Candidates:

```text
issue +comment
issue +create
pr +comment
pr +review with common/comment
pr +review with approved
pr +review with rejected/request changes
```

Feishu Gateway policy:

```text
Stage 3 only
requires self-built app callback validation
requires Feishu user -> GitLink user mapping
requires repo binding
requires GitLink token and permission check
requires dry-run preview
requires explicit confirmation
requires audit log
```

Why these are lower risk:

```text
comments and reviews are additive
issue creation is visible and reversible by later close/edit
approval/request changes affects review state but does not merge code
```

Still not safe for Stage 1:

```text
These are GitLink writes. They must not be exposed from custom bot cards or unauthenticated webhooks.
```

## Level 2: Medium-Risk Write

These actions change workflow state and can disrupt project management, but they are usually recoverable.

Candidates:

```text
issue +update
pr +create
pr +reopen
ci +restart
ci +stop
pipeline +run
pipeline +enable
pipeline +disable
milestone +create
milestone +update
milestone +close
milestone +reopen
release +create
release +update
dataset +create
dataset +update
label +create
label +update
repo +follow
repo +unfollow
repo +like
repo +unlike
webhook +test
member +accept-invite
```

Feishu Gateway policy:

```text
planned only
disabled by default
requires stronger confirmation than Level 1
requires allowlist by action type and repository
requires audit log
should support dry-run where the underlying command supports it
```

Design note:

```text
Some Level 2 actions can move to Level 1 only after the project has clear policy and tests.
For example, creating a milestone may be low-risk in one organization but not in another.
```

## Level 3: High-Risk Write

These actions should not be exposed by default through Feishu.

Actions:

```text
pr +merge
pr +refuse
issue +close
issue +batch-close
branch +delete
branch +protect
branch +unprotect
release +delete
dataset +delete-attachment
label +delete
repo +delete
member +add
member +batch-add
member +remove
member +role
webhook +create
webhook +update
webhook +delete
pipeline +delete
org +create
repo +create
repo +fork
branch +create
```

Why high risk:

```text
merge changes code history and release state
close/refuse can stop contributor work
delete actions can remove project assets
member actions change access control
webhook actions can exfiltrate or disrupt events
branch protection changes affect repository safety
repo/org creation can create governance and ownership issues
```

Feishu Gateway policy:

```text
do not implement in the main Stage 3 path
only design as experimental
requires --enable-dangerous-actions or equivalent server config
requires maintainer / owner role check
requires repository allowlist
requires action-specific second confirmation
requires audit log with before/after payloads when available
requires rate limiting
requires rollback guidance where possible
```

## Admin and Raw API Surface

These surfaces should not be exposed as Feishu card actions.

```text
auth login
auth logout
config set
api arbitrary METHOD PATH
api --batch-file without strict allowlist
```

Reason:

```text
They operate on credentials, local configuration, or arbitrary GitLink API requests.
They are too broad for a safe Feishu action gateway.
```

Allowed Feishu use:

```text
show auth status diagnostics
show required setup steps
run app-check style read-only environment diagnostics
```

Not allowed:

```text
collect GitLink passwords
display tokens
write credentials from Feishu payloads
execute arbitrary raw API requests from card callbacks
```

## Dry-Run and Confirmation Requirements

Current gitlink-cli has uneven dry-run coverage. Some write commands support dry-run, some do not.

Action Gateway must not assume all commands are safe to preview.

Required gateway behavior:

```text
Level 0:
  can run read commands directly after repo binding validation

Level 1:
  must construct a dry-run preview
  if native dry-run exists, use it
  if native dry-run does not exist, render a gateway-level preview and stop before execution

Level 2:
  dry-run preview plus explicit confirmation
  repository and action allowlist required

Level 3:
  disabled by default
  dangerous-action opt-in required
  second confirmation required
```

## Identity Boundary

GitLink CLI and Feishu identities are different permission domains.

Required mapping:

```text
Feishu open_id / union_id / email -> GitLink username -> GitLink token or allowed service identity
```

Do not assume:

```text
Feishu display name == GitLink username
Feishu email always exists
one Feishu user maps to exactly one GitLink user
group chat actor is authorized for all repository actions
```

Stage policy:

```text
Stage 1:
  no personal write actions, identity mapping optional

Stage 2:
  mapping can be used for dashboards and personal panels

Stage 3:
  mapping is mandatory before any GitLink write

Stage 4:
  mapping plus maintainer role verification is mandatory
```

## Feishu Integration Implications

### Safe First Implementation

Use Level 0 commands to produce:

```text
owner digest
contributor digest preview
PR stage summary
Issue risk summary
CI status summary
Bitable-ready records
Doc/Wiki markdown
```

### Low-Risk Action Gateway

Only after self-built app integration exists:

```text
issue comment
PR comment
PR review comment
PR approve
PR request changes
create issue
```

### Explicitly Not First-Line Feishu Actions

```text
merge PR
close issue
delete branch
delete release
add/remove member
change member role
create/update/delete webhook
raw API call
token/config operation
```

## Recommended Bitable Tables from GitLink Data

The current `feishu +bitable-records` command emits summary records. For project management views, GitLink CLI should eventually produce row-level records.

Recommended row-level tables:

```text
repositories
pull_requests
issues
ci_runs
pipeline_runs
milestones
releases
contributors
members
webhook_deliveries
action_audit
```

View mapping:

```text
Kanban:
  pull_requests by stage
  issues by status

Gantt:
  milestones by start_date / due_date
  releases by release window

Calendar:
  issue due dates
  review due dates
  release dates

Gallery:
  contributors
  features / merged PRs

Dashboard:
  repository health
  PR stage counts
  Issue risk counts
  CI pass rate
  stale work

Personal task panel:
  PRs authored by mapped user
  issues assigned to mapped user
  PRs waiting for mapped reviewer
```

## Final Boundary

The clean GitLink boundary for Feishu is:

```text
Stage 1:
  Feishu displays GitLink state.
  GitLink is read-only from Feishu.

Stage 2:
  Feishu stores GitLink-derived artifacts.
  GitLink remains read-only from Feishu.

Stage 3:
  Feishu can request low-risk GitLink collaboration actions.
  gitlink-cli executes only after validation, dry-run, confirmation, and audit.

Stage 4:
  High-risk GitLink actions are designed but disabled by default.
```

This boundary keeps `gitlink-cli` credible as the safe execution layer while still allowing Feishu to become the collaboration entry point.
