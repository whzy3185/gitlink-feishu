# GitLink CLI x Feishu Redesign Research

Date: 2026-06-26

## 1. Executive Summary

The current `gitlink-cli feishu` implementation is a safe export module. It can preview and send Feishu custom bot cards, render weekly reports, generate Bitable schemas and Bitable-ready records, and experiment with DocX / Wiki export.

The redesigned product direction should be broader but staged:

```text
GitLink project collaboration gateway for maintainers, contributors, and teams inside Feishu.
```

The stable first path should remain conservative:

```text
GitLink data -> Feishu cards / reports / Bitable-ready records / Doc-ready content
```

The later path can become a permissioned collaboration gateway:

```text
Feishu app / card callback -> identity mapping -> GitLink action preview -> confirmation -> audited action
```

The key design correction is that Feishu should not be treated only as a message sink. Feishu's public product materials, Bilibili tutorials, and official Base documentation consistently position Bitable as:

```text
one data source
multiple role-specific views
lightweight business system
dashboard / cockpit
workflow and AI-enabled collaboration base
```

For GitLink projects, this means the most useful Feishu integration is not "send one message per PR". It is:

```text
1. owner cockpit
2. contributor task panel
3. PR / Issue / CI history base
4. milestone Gantt
5. weekly report archive
6. Doc / Wiki project knowledge space
7. optional low-risk GitLink action gateway
```

## 2. Research Sources

### Current Repository

Inspected local repository paths:

```text
README.md
README.zh-CN.md
shortcuts/feishu
shortcuts/workflow
shortcuts/issue
shortcuts/pr
shortcuts/webhook
shortcuts/member
shortcuts/ci
shortcuts/pipeline
skills/
docs/feishu-integration.md
docs/feishu-security.md
reports/FEISHU_TASK_COMPLETION.md
reports/FEISHU_TEST_ENTERPRISE_SMOKE_20260615.md
feishu-export-design/ROLE_BASED_COLLABORATION.md
```

Local command checks:

```text
gitlink-cli feishu --help
gitlink-cli issue --help
gitlink-cli pr --help
gitlink-cli webhook --help
gitlink-cli workflow --help
gitlink-cli member --help
gitlink-cli ci --help
gitlink-cli pipeline --help
```

### Feishu / Lark Official Sources

Primary references:

```text
https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot
https://open.feishu.cn/document/feishu-cards/quick-start/send-message-cards-with-custom-bot?lang=zh-CN
https://open.feishu.cn/document/server-docs/authentication-management/access-token/tenant_access_token_internal?lang=zh-CN
https://open.feishu.cn/document/server-docs/im-v1/message/create?lang=zh-CN
https://open.feishu.cn/document/server-docs/docs/docs/docx-v1/document/create
https://open.feishu.cn/document/server-docs/docs/docs/docx-v1/document-block/create?lang=zh-CN
https://open.feishu.cn/document/server-docs/docs/bitable-v1/app-table-record/create?lang=zh-CN
https://open.feishu.cn/document/task-v2/task/create?lang=zh-CN
https://www.feishu.cn/product/base
https://www.feishu.cn/hc/zh-CN/articles/360049067931-%E4%BD%BF%E7%94%A8%E5%A4%9A%E7%BB%B4%E8%A1%A8%E6%A0%BC%E8%A7%86%E5%9B%BE
https://www.feishu.cn/hc/zh-CN/articles/558830919244-%E4%BD%BF%E7%94%A8%E5%A4%9A%E7%BB%B4%E8%A1%A8%E6%A0%BC%E7%9A%84%E7%94%98%E7%89%B9%E8%A7%86%E5%9B%BE
https://www.feishu.cn/hc/zh-CN/articles/161059314076-%E4%BD%BF%E7%94%A8%E5%A4%9A%E7%BB%B4%E8%A1%A8%E6%A0%BC%E4%BB%AA%E8%A1%A8%E7%9B%98
```

### Bilibili / Social Content Sources

Observed Bilibili content:

```text
https://www.bilibili.com/video/BV1rd4y167KZ/
```

This is a Feishu Help Center Bilibili video in the Feishu Base practical course collection. The collection includes:

```text
views: one Base, multiple presentation modes
forms: information collection and aggregation
dashboard: data visualization
relations and lookup: data relationship modeling
automation and advanced permissions
Base-building methodology: build Base like a product
sales management system
HR recruiting and probation management system
product and engineering agile development management
Base plus Feishu collaboration combinations
```

Social-platform-facing Feishu content also emphasizes:

```text
one table for company operations
enterprise cockpit
multi-view switching
project-management Gantt charts
Kanban management
calendar scheduling
form collection
dashboard analysis
Xiaohongshu / Douyin / e-commerce content data management
AI fields / AI summaries / content creation workflows
```

Direct Xiaohongshu pages are not reliably accessible through normal web indexing in this environment. The useful signal comes from Feishu official template and content pages that explicitly mention Xiaohongshu data management and content workflows:

```text
https://www.feishu.cn/template/group/base
https://www.feishu.cn/content/article/7584012383060757728
https://www.feishu.cn/content/article/7582904528077917408
```

### lark-cli Sources

Primary references:

```text
https://github.com/larksuite/cli
https://github.com/larksuite/cli/blob/main/README.zh.md
https://open.larksuite.com/document/mcp_open_tools/feishu-cli-let-ai-actually-do-your-work-in-feishu
https://www.feishu.cn/feishu-cli
```

Key lark-cli claims observed:

```text
official Lark / Feishu CLI
built for humans and AI Agents
covers Messenger, Docs, Base, Sheets, Calendar, Mail, Tasks, Meetings, Markdown, etc.
200+ commands
20+ AI Agent Skills
structured output
schema introspection
explicit safety warnings for AI Agent use
```

## 3. Current Repository Capability Assessment

### Implemented Stable Commands

```bash
gitlink-cli feishu +bot-test
gitlink-cli feishu +notify
gitlink-cli feishu +weekly-report
gitlink-cli feishu +bitable-schema
gitlink-cli feishu +bitable-records
```

Stable behavior:

```text
local preview by default
real Feishu custom bot delivery only with --send
--send and --dry-run conflict validation
webhook URL redaction
custom bot signing
interactive card payload building
workflow JSON input support
weekly report rendering
--doc-url button support
Bitable schema dry-run
Bitable-ready record dry-run
UTF-8 / UTF-16 workflow JSON input support
mockable HTTP client
```

### Experimental Command

```bash
gitlink-cli feishu +doc-export
```

Experimental behavior:

```text
app_id / app_secret
tenant_access_token
Wiki node resolution
DocX block write attempt
document export preview
explicit --send for write attempt
```

Observed real Feishu test:

```text
custom bot send: passed
notify send: passed
weekly-report send: passed
Bitable schema / records dry-run: passed
tenant_access_token: acquired
Wiki node: resolved
DocX write: blocked by Feishu 403 / 1770032 / forBidden
```

Interpretation:

```text
The app credentials and Wiki read path can work, but document writes still require correct DocX / Drive scopes and target document or folder permissions.
```

### Current Bitable Usage

Current Bitable implementation is local-only.

Implemented:

```text
feishu +bitable-schema
feishu +bitable-records
```

Current tables:

```text
reports
issues
prs
contributors
```

Current records:

```text
reports: one row per workflow report
issues: summary bucket rows
prs: summary bucket rows
contributors: reserved, empty unless workflow JSON contains contributor details
```

Not implemented:

```text
real Bitable API write
create Base
create table
create field
create view
batch create records
update records
upsert records
search before update
Gantt / Kanban / Calendar / Gallery / Dashboard creation
field permission setup
```

Conclusion:

```text
Current Bitable support is useful as schema and data-shape proof, but not yet useful as an actual project management cockpit.
```

To support project management, the next Bitable model must become row-level instead of summary-only:

```text
one row per PR
one row per Issue
one row per CI run
one row per contributor-period
one row per milestone
one row per release
one row per action/audit event
```

## 4. Feishu Capability Boundary

### Custom Bot

Fit:

```text
one-way group notifications
message cards
weekly report cards
owner digest cards
links to GitLink / Feishu Docs / Bitable views
low-friction smoke test
```

Not fit:

```text
document writes
Bitable writes
task creation
user identity mapping
personal direct messages
interactive GitLink write actions
permission management
callback server by itself
```

Design rule:

```text
Custom bot = notification and link surface.
```

### Self-Built App

Fit:

```text
tenant_access_token
app-level authentication
IM message send
DocX / Wiki write
Bitable app / table / record operations
Task API
card callback / event subscription
user identity resolution
chat_id / repo binding
permission diagnostics
```

Design rule:

```text
Self-built app = authorization, Feishu resource write, callbacks, and future interaction surface.
```

### lark-cli

Fit:

```text
optional external bridge
AI Agent operations on Feishu resources
Docs / Base / Tasks / Calendar / Mail / IM operations
manual experiments before native implementation
developer productivity
cross-platform workflows where Codex controls both CLIs
```

Not fit as a hard dependency:

```text
gitlink-cli mainline runtime
stable Go tests
minimal install path
security-critical callback gateway
deterministic server-side behavior
```

Design rule:

```text
gitlink-cli should not depend on lark-cli.
lark-cli can be an optional bridge for agents and operators.
```

### gitlink-cli

Fit:

```text
authoritative GitLink data reader
authoritative GitLink action executor
workflow report generation
Issue / PR / Review / CI / Webhook / Member operations
dry-run and confirmation behavior
audit log source
```

Design rule:

```text
GitLink actions should be executed through gitlink-cli native commands, not through Feishu-side ad hoc HTTP calls hidden in callbacks.
```

Detailed GitLink capability boundary:

```text
docs/GITLINK_CLI_CAPABILITY_BOUNDARY.md
```

## 5. Bilibili / Xiaohongshu / Social Content Findings

### What Feishu Emphasizes Publicly

The Bilibili tutorial collection and Feishu public materials repeatedly emphasize:

```text
Base is positioned as a lightweight business system, not just a spreadsheet.
The same data can be presented through different views.
Managers and executors should see role-specific views.
Dashboards serve as business cockpits.
Forms collect structured input.
Kanban views show workflow state.
Gantt views show time plans.
Calendar views show schedules.
Gallery views show card-style records.
Automation reduces manual data movement.
Advanced permissions protect sensitive data.
AI fields and summaries support content production, classification, and analysis.
```

This matters for GitLink because GitLink data naturally has multiple roles:

```text
owner / maintainer
reviewer
contributor
PM / project manager
organization manager
AI agent
```

The Feishu product narrative suggests that we should not build a single flat report. We should design a data model that supports multiple views.

### Mapping Social Product Narrative to GitLink

| Feishu narrative | GitLink equivalent | Product implication |
| --- | --- | --- |
| one table, many views | one project data model, many project views | Store PR / Issue / CI / milestone rows in Bitable |
| management cockpit | owner dashboard | owner digest + dashboard records |
| personal task panel | contributor pending action view | contributor digest and assigned PR/Issue records |
| Kanban | PR / Issue stage board | group by review stage, issue status, CI status |
| Gantt | milestone and release planning | start/end dates, dependencies, owners |
| Calendar | review deadlines, release dates, meetings | due_date and scheduled review fields |
| Gallery | project showcase and PR cards | card-style PR / contributor / feature display |
| Form | collection and intake | issue intake, contributor weekly update, review request |
| Dashboard | high-level metrics | open PRs, stale PRs, merged PRs, risk count, CI pass rate |
| AI field / summary | summarization and classification | AI can summarize PRs, classify risks, draft weekly report |

### Implication for Current Bitable Commands

Current `+bitable-records` produces summary buckets. That is not enough for:

```text
Kanban
Gantt
Calendar
Gallery
Form-driven workflow
individual task panel
enterprise cockpit
```

Required next data shape:

```text
Pull Requests table:
  pr_key
  repository
  number
  title
  author
  assignee
  reviewer
  stage
  stage_color
  risk_level
  review_rounds
  patchset_count
  changed_files
  additions
  deletions
  mergeable
  needs_rebase
  ci_status
  opened_at
  updated_at
  due_date
  url

Issues table:
  issue_key
  repository
  number
  title
  author
  assignee
  status
  priority
  labels
  risk_level
  due_date
  stale_days
  url

CI Runs table:
  ci_key
  repository
  ref
  commit
  status
  started_at
  finished_at
  duration
  url

Milestones table:
  milestone_key
  repository
  title
  owner
  start_date
  due_date
  status
  progress_percent
  linked_prs
  linked_issues
  risk_level

Contributors table:
  contributor_key
  gitlink_username
  feishu_user_id
  role
  active_prs
  merged_prs
  pending_actions
  last_activity_at

Audit table:
  action_id
  source
  actor_feishu_id
  actor_gitlink_user
  repository
  action_type
  risk_level
  dry_run
  confirmed
  result
  created_at
```

### Recommended Feishu Views

| View | Target user | Backing table | Purpose |
| --- | --- | --- | --- |
| Table view | Admin / maintainer | all tables | raw data inspection |
| Kanban view | Owner / reviewer | PRs, Issues | stage tracking |
| Gantt view | PM / owner | Milestones, PRs | release and milestone planning |
| Calendar view | Reviewer / contributor | PRs, Issues, Milestones | due dates and review windows |
| Gallery view | Community manager | PRs, Contributors | showcase features and contributor highlights |
| Form view | Contributors | Issues, Updates | issue intake and weekly updates |
| Dashboard | Owner / organization | PRs, Issues, CI, Milestones | enterprise cockpit |
| Personal task panel | Contributor | PRs, Issues | my pending actions |

### Enterprise Cockpit

For organization managers, the cockpit should answer:

```text
How many active repositories are healthy?
Which repos have stale PRs?
Which maintainers are overloaded?
Which milestones are at risk?
How many PRs are near merge?
How many PRs need rebase?
What is the CI pass rate trend?
Which contributors need response?
Which issues are high risk?
```

This should be implemented through Bitable records and dashboard views, not through long chat messages.

### Personal Task Panel

For contributors, the personal panel should answer:

```text
Which of my PRs need rebase?
Which PRs received review comments?
Which issues are assigned to me?
Which CI runs failed on my branch?
Which maintainer action am I waiting for?
What was merged this week?
```

This requires identity mapping:

```text
GitLink username -> Feishu open_id / union_id / email
```

Until identity mapping exists, the CLI should only generate dry-run contributor records.

## 6. Current gitlink-cli Capability Matrix

This section summarizes the command surface. The detailed action-risk boundary is maintained in:

```text
docs/GITLINK_CLI_CAPABILITY_BOUNDARY.md
```

### Issue

Available commands:

```text
issue +list
issue +view
issue +create
issue +update
issue +comment
issue +close
issue +batch-close
issue +authors
issue +assigners
issue +priorities
issue +statuses
issue +tags
```

Gateway classification:

```text
read: list, view, metadata list
low-risk write: comment, create
medium-risk write: update metadata
high-risk write: close, batch-close
```

### Pull Request

Available commands:

```text
pr +list
pr +view
pr +files
pr +diff
pr +versions
pr +version-diff
pr +reviews
pr +review
pr +comment
pr +create
pr +merge
pr +refuse
pr +reopen
```

Gateway classification:

```text
read: list, view, files, diff, versions, version-diff, reviews
low-risk write: comment, review comment, approve, request changes
medium-risk write: create, reopen
high-risk write: merge, refuse / close
```

### Webhook

Available commands:

```text
webhook +list
webhook +view
webhook +create
webhook +update
webhook +delete
webhook +test
webhook +tasks
```

Gateway classification:

```text
read: list, view, tasks
medium-risk write: test
high-risk write: create, update, delete
```

### Workflow

Available commands:

```text
workflow +triage
workflow +health
workflow +pr-summary
workflow +repo-report
```

Gateway classification:

```text
read / local analysis: all workflow commands
```

These are ideal first-class inputs for Feishu reports.

### CI / Pipeline

Available commands:

```text
ci +builds
ci +logs
ci +restart
ci +stop

pipeline +list
pipeline +view
pipeline +runs
pipeline +run
pipeline +logs
pipeline +results
pipeline +save-yaml
pipeline +enable
pipeline +disable
pipeline +delete
```

Gateway classification:

```text
read: builds, logs, list, view, runs, results, save-yaml
medium-risk write: restart, run, stop, enable, disable
high-risk write: delete
```

### Member

Available commands:

```text
member +list
member +add
member +batch-add
member +remove
member +role
member +invite-link
member +invite-info
member +accept-invite
```

Gateway classification:

```text
read: list, invite-info
medium-risk write: accept-invite
high-risk write: add, batch-add, remove, role, invite-link
```

## 7. Functional Matrix

| Function | Custom bot | Self-built app | lark-cli | gitlink-cli | Recommendation |
| --- | --- | --- | --- | --- | --- |
| Project progress card | yes | yes | possible | data source | Stage 1 custom bot |
| Weekly report | yes | yes | possible | workflow | Stage 1 custom bot |
| Owner digest | yes | yes | possible | workflow / PR / Issue | Stage 1 custom bot |
| Contributor digest | group only | direct message possible | possible | data source | Stage 1 dry-run, Stage 2 app |
| PR stage cards | yes | yes | possible | PR reader | Stage 1 |
| Issue risk summary | yes | yes | possible | Issue/workflow | Stage 1 |
| CI summary | yes | yes | possible | CI/pipeline | Stage 1 read |
| Link to GitLink | yes | yes | yes | URL source | Stage 1 |
| DocX / Wiki write | no | yes | yes | content source | Stage 2 |
| Bitable real sync | no | yes | yes | record source | Stage 2 |
| Bitable views | no | partial through API/manual | possible | schema source | Stage 2/manual first |
| Task creation | no | yes | yes | task source | Stage 2 |
| Card callback | no as standalone | yes | possible | action executor | Stage 3 |
| Issue comment | no | entry only | bridge possible | issue +comment | Stage 3 low-risk |
| PR review | no | entry only | bridge possible | pr +review | Stage 3 low-risk |
| Merge PR | no | entry only | bridge possible | pr +merge | High-risk, disabled by default |
| Close issue | no | entry only | bridge possible | issue +close | High-risk, disabled by default |
| Member management | no | entry only | bridge possible | member commands | High-risk, disabled by default |

## 8. Revised Product Stages

### Stage 1: Safe Export and Notification

Goal:

```text
Make GitLink project state visible in Feishu without Feishu-triggered GitLink writes.
```

Capabilities:

```text
custom bot notification
weekly report
owner digest
contributor digest dry-run
PR stage summary
Issue risk summary
CI summary
Bitable-ready row-level records
Doc/Wiki-ready markdown preview
explicit --send for notification
```

Implementation priority:

```text
1. owner digest card
2. contributor digest preview
3. PR row records
4. Issue row records
5. CI row records
6. milestone records
7. README / Doc-ready markdown
```

No:

```text
no callback server
no GitLink write actions
no real Bitable writes
no task creation
no permission management
```

### Stage 2: Feishu Open Platform App Integration

Goal:

```text
Turn Feishu into the project collaboration space while still avoiding GitLink writes from Feishu.
```

Capabilities:

```text
app_id / app_secret config
tenant_access_token cache
app-check
scope diagnostics
chat_id / repo binding
DocX / Wiki write
Bitable record sync
task creation
IM send through app bot
permission diagnostics
lark-cli-check optional
```

No:

```text
no PR merge
no issue close
no member management
no branch protection changes
```

### Stage 3: Callback-Based Low-Risk GitLink Actions

Goal:

```text
Allow permissioned, audited low-risk GitLink actions from Feishu cards.
```

Capabilities:

```text
callback server
callback signature verification
action payload parser
repo binding check
identity mapping
GitLink permission check
dry-run preview
confirmation
audit log
issue comment
PR review comment
PR approve
PR request changes
create issue
result writeback to Feishu
```

No:

```text
no merge by default
no close by default
no delete by default
no member role change by default
```

### Stage 4: High-Risk Action Design

Goal:

```text
Design high-risk actions but keep them disabled unless explicitly enabled.
```

Actions:

```text
merge PR
close issue
delete branch
delete release
add/remove member
change member role
protect/unprotect branch
disable/delete pipeline
```

Required safeguards:

```text
--enable-dangerous-actions
dry-run first
second confirmation
maintainer role check
repo allowlist
audit log
rate limit
rollback guidance where possible
```

## 9. lark-cli and gitlink-cli Interaction

### Decision

```text
gitlink-cli should not depend on lark-cli.
lark-cli should be an optional agent bridge.
```

Reasons:

```text
hard dependency increases installation complexity
hard dependency complicates Go tests
hard dependency mixes Feishu and GitLink ownership
security boundary becomes unclear
lark-cli is broad and fast-moving
gitlink-cli needs a minimal stable Feishu path
```

### Recommended Interaction Modes

Mode A: gitlink-cli to Feishu

```text
gitlink-cli reads GitLink data
gitlink-cli renders card/report/records
gitlink-cli sends custom bot card or exports dry-run data
```

Mode B: Feishu to gitlink-cli

```text
Feishu callback arrives at action gateway
gateway validates Feishu identity and repo binding
gateway maps action to gitlink-cli dry-run
user confirms
gateway executes allowed GitLink action
gateway writes audit log and Feishu result message
```

Mode C: Agent bridge with both CLIs

```text
Codex / Claude / OpenClaw loads gitlink-cli skills and lark-cli skills
gitlink-cli handles GitLink
lark-cli handles Feishu Docs / Base / Tasks / Calendar / IM
Agent orchestrates both for one-off or operator-assisted workflows
```

### Effect of lark-cli Skills on GitLink Organizations

For maintainers:

```text
less context switching between GitLink, Feishu, and terminal
Feishu Docs can become project memory
Bitable can become project cockpit
meetings can generate GitLink issues or review checklists
weekly reports can be generated from GitLink data and written to Feishu
```

For contributors:

```text
personal pending-action panel
review feedback pushed into Feishu
rebase / CI failure reminders
quick jump back to GitLink PR / Issue
self-summary of weekly contribution
```

For AI agents:

```text
gitlink-cli skills provide GitLink understanding and action execution
lark-cli skills provide Feishu resource operations
structured outputs improve cross-platform automation
dry-run / confirmation policies reduce accidental writes
```

Main risk:

```text
An AI Agent with Feishu user authorization and GitLink token can accidentally bridge two permission domains.
```

Required control:

```text
least privilege
explicit --send / --apply
dry-run first
identity mapping
repo binding
confirmation
audit log
high-risk actions disabled
secret redaction
```

## 10. Recommended Documents to Add Next

This research report should be followed by focused boundary documents:

```text
docs/FEISHU_CAPABILITY_BOUNDARY.md
docs/FEISHU_OPEN_PLATFORM_PLAN.md
docs/FEISHU_ACTION_GATEWAY_SECURITY.md
docs/FEISHU_LARK_CLI_INTEROP.md
```

Recommended command planning documents:

```text
gitlink-cli feishu +app-check
gitlink-cli feishu +binding-list
gitlink-cli feishu +binding-add
gitlink-cli feishu +binding-remove
gitlink-cli feishu +action-preview
gitlink-cli feishu +serve
gitlink-cli feishu +audit-log
gitlink-cli feishu +lark-cli-check
```

Do not implement all commands immediately. Document the gateway and permission model first.

## 11. Implementation Implications for Current Code

### Keep

```text
custom bot signer
custom bot client
card builders
workflow JSON reader
weekly report renderer
Bitable schema builder
Bitable records builder
DocX / Wiki experimental client
redaction and send-mode validation
```

### Refactor Later

```text
current Bitable records are summary-oriented
new Bitable records should support row-level PR / Issue / CI / milestone records
card builders need owner digest and contributor digest variants
Doc export needs clearer permission diagnostics and app-check
OpenAPI token client needs cache and scope diagnostics
```

### Add Later

```text
role-aware data models
PR stage classifier
owner digest
contributor digest
row-level Bitable records
README / Doc-ready markdown exporter
app-check
binding model
action gateway preview
audit log
```

### Avoid

```text
claiming real Bitable sync before API writes exist
claiming Feishu task creation before Task API exists in code
claiming card callback support before server and validation exist
claiming Feishu-triggered GitLink write actions before gateway exists
putting real secrets, IDs, table IDs, chat IDs, open IDs, or document tokens in repo
```

## 12. Final Verdict

The project should be repositioned from:

```text
GitLink workflow export to Feishu
```

to:

```text
GitLink project collaboration gateway for Feishu, implemented in staged safety layers.
```

The immediate engineering plan should still stay conservative:

```text
Stage 1 = safe visibility.
Stage 2 = Feishu workspace integration.
Stage 3 = low-risk action gateway.
Stage 4 = high-risk action design only.
```

The most important design insight from Feishu's Bilibili and public content is:

```text
Feishu's value is not just notification. Its value is turning structured data into role-specific work surfaces.
```

For GitLink, that means:

```text
owner cockpit
contributor task panel
PR / Issue / CI / milestone Base
Doc / Wiki project memory
weekly report archive
optional permissioned action gateway
```

This direction is practical, extensible, and matches Feishu's own product narrative while keeping the first implementation safe enough for a mainline PR.
