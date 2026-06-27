# Feishu Environment Variables

Date: 2026-06-26

Do not commit real values. Use a local shell profile, CI secret store, or test terminal session.

Recommended local workflow:

```powershell
.\scripts\feishu-gitlink-setup.ps1
.\scripts\feishu-gitlink-env-check.ps1 -Layer stable
.\scripts\feishu-gitlink-smoke.ps1 -Mode preview
```

`feishu-gitlink-setup.ps1` opens the relevant Feishu / GitLink pages, lets the user paste values locally, and writes only to:

```text
.local/feishu-gitlink.env.ps1
```

The real local env file is ignored. The tracked example is:

```text
.local/feishu-gitlink.env.example.ps1
```

## Stable Custom Bot Variables

| Name | Purpose | Required | Used by | Sensitive | How to obtain |
| --- | --- | --- | --- | --- | --- |
| `FEISHU_WEBHOOK_URL` | Feishu custom bot webhook URL | Required for `--send` bot delivery | `+bot-test`, `+notify`, `+weekly-report`, `+owner-digest`, `+contributor-digest` | Yes | Feishu group custom bot settings |
| `FEISHU_WEBHOOK_SECRET` | Optional custom bot signing secret | Optional | same as above | Yes | Feishu group custom bot security settings |

Example:

```powershell
$env:FEISHU_WEBHOOK_URL="https://open.feishu.cn/open-apis/bot/v2/hook/REDACTED"
$env:FEISHU_WEBHOOK_SECRET="REDACTED"
```

## Open Platform App Variables

| Name | Purpose | Required | Used by | Sensitive | How to obtain |
| --- | --- | --- | --- | --- | --- |
| `FEISHU_APP_ID` | Self-built app ID | Required for Open Platform `--send` or diagnostic `--remote` | `+app-check`, `+doc-check`, `+bitable-check`, `+task-check`, `+doc-export`, `+bitable-sync`, `+task-create` | Yes | Feishu Open Platform app page |
| `FEISHU_APP_SECRET` | Self-built app secret | Required for Open Platform `--send` or diagnostic `--remote` | same as above | Yes | Feishu Open Platform app credentials |

Example:

```powershell
$env:FEISHU_APP_ID="cli_REDACTED"
$env:FEISHU_APP_SECRET="REDACTED"
```

## DocX / Wiki Variables

| Name | Purpose | Required | Used by | Sensitive | How to obtain |
| --- | --- | --- | --- | --- | --- |
| `FEISHU_WIKI_URL` | Existing Wiki page URL | Optional target | `+doc-check`, `+doc-export` | Can expose workspace/resource ID | Copy from Feishu Wiki |
| `FEISHU_WIKI_NODE_TOKEN` | Existing Wiki node token | Optional target | `+doc-check`, `+doc-export` | Yes | Parsed from Wiki URL or API |
| `FEISHU_FOLDER_TOKEN` | Folder token for creating a new DocX | Optional target | `+doc-check`, `+doc-export` | Yes | Feishu Drive folder URL / Open Platform docs |
| `FEISHU_DOCUMENT_ID` | Existing DocX document ID for append | Optional target | `+doc-check`, `+doc-export` | Yes | Existing Feishu DocX URL or Open Platform docs |

Legacy compatibility:

```text
FEISHU_DOC_FOLDER_TOKEN is still accepted after FEISHU_FOLDER_TOKEN.
```

Example:

```powershell
$env:FEISHU_WIKI_URL="https://example.feishu.cn/wiki/REDACTED"
$env:FEISHU_FOLDER_TOKEN="REDACTED"
$env:FEISHU_DOCUMENT_ID="REDACTED"
```

For localized output, generate the source workflow report and the Feishu output
with the same language flag:

```powershell
go run . workflow +repo-report --owner "$env:GITLINK_OWNER" --repo "$env:GITLINK_REPO" --lang zh-CN --format json > .local\report.zh-CN.json
go run . feishu +notify --from-workflow-json .local\report.zh-CN.json --lang zh-CN --format table
```

## Base / Bitable Variables

| Name | Purpose | Required | Used by | Sensitive | How to obtain |
| --- | --- | --- | --- | --- | --- |
| `FEISHU_BASE_APP_TOKEN` | Base app token | Required for `+bitable-check` and `+bitable-sync --send` | `+bitable-check`, `+bitable-sync` | Yes | Feishu Base URL / Open Platform docs |
| `FEISHU_REPORT_TABLE_ID` | Reports table ID | Required when checking or syncing `reports` | `+bitable-check`, `+bitable-sync` | Yes | Base table settings / API |
| `FEISHU_ISSUE_TABLE_ID` | Issues table ID | Required when checking or syncing `issues` | `+bitable-check`, `+bitable-sync` | Yes | Base table settings / API |
| `FEISHU_PR_TABLE_ID` | Pull request table ID | Required when checking or syncing `prs` | `+bitable-check`, `+bitable-sync` | Yes | Base table settings / API |
| `FEISHU_CONTRIBUTOR_TABLE_ID` | Contributors table ID | Optional unless selected | `+bitable-check`, `+bitable-sync` | Yes | Base table settings / API |
| `FEISHU_TASK_TABLE_ID` | Task-candidate table ID | Optional unless selected | `+bitable-check`, `+bitable-sync` | Yes | Base table settings / API |

Example:

```powershell
$env:FEISHU_BASE_APP_TOKEN="REDACTED"
$env:FEISHU_REPORT_TABLE_ID="REDACTED"
$env:FEISHU_ISSUE_TABLE_ID="REDACTED"
$env:FEISHU_PR_TABLE_ID="REDACTED"
$env:FEISHU_CONTRIBUTOR_TABLE_ID="REDACTED"
$env:FEISHU_TASK_TABLE_ID="REDACTED"
```

## Feishu Task Variables

| Name | Purpose | Required | Used by | Sensitive | How to obtain |
| --- | --- | --- | --- | --- | --- |
| `FEISHU_TASK_PROJECT_ID` | Optional task project target | Optional | `+task-check`, `+task-create` | Yes | Feishu Task project settings / API |
| `FEISHU_TASK_SECTION_ID` | Optional task section target | Optional | `+task-check`, `+task-create` | Yes | Feishu Task section settings / API |

Current limitation:

```text
The experimental task create command creates task candidates through the Task API.
Task project and section IDs are currently collected and redacted in output,
but they are not yet mapped into the create-task request body.
Project/section placement should be wired only after the official request fields
and test-enterprise behavior are confirmed.
```

## GitLink Test Variables

| Name | Purpose | Required | Used by | Sensitive | How to obtain |
| --- | --- | --- | --- | --- | --- |
| `GITLINK_OWNER` | Test repository owner | Optional for local smoke | workflow report generation | No | GitLink repository URL |
| `GITLINK_REPO` | Test repository name | Optional for local smoke | workflow report generation | No | GitLink repository URL |
| `GITLINK_TEST_PR_IDS` | Comma-separated PR IDs for smoke reference | Optional | smoke report only unless workflow supports filtering | No | GitLink PR URLs |
| `GITLINK_TOKEN` | GitLink API token | Optional if already logged in | workflow read operations | Yes | GitLink account settings |

Example:

```powershell
$env:GITLINK_OWNER="OWNER"
$env:GITLINK_REPO="REPO"
$env:GITLINK_TEST_PR_IDS="1,2,3"
$env:GITLINK_TOKEN="REDACTED"
```

## Safety Warnings

```text
Never paste real secrets into committed docs.
Never paste real secrets into ChatGPT.
Never print raw webhook URLs or app secrets in smoke reports.
Do not commit tenant_access_token or user_access_token.
Do not enable --send in shared scripts unless the target test enterprise is intentional.
```
