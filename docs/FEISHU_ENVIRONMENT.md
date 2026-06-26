# Feishu Environment Variables

Date: 2026-06-26

Do not commit real values. Use a local shell profile, CI secret store, or test terminal session.

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
| `FEISHU_APP_ID` | Self-built app ID | Required for Open Platform `--send` | `+doc-export`, `+bitable-sync`, `+task-create` | Yes | Feishu Open Platform app page |
| `FEISHU_APP_SECRET` | Self-built app secret | Required for Open Platform `--send` | same as above | Yes | Feishu Open Platform app credentials |

Example:

```powershell
$env:FEISHU_APP_ID="cli_REDACTED"
$env:FEISHU_APP_SECRET="REDACTED"
```

## DocX / Wiki Variables

| Name | Purpose | Required | Used by | Sensitive | How to obtain |
| --- | --- | --- | --- | --- | --- |
| `FEISHU_WIKI_URL` | Existing Wiki page URL | Optional target | `+doc-export` | Can expose workspace/resource ID | Copy from Feishu Wiki |
| `FEISHU_WIKI_NODE_TOKEN` | Existing Wiki node token | Optional target | `+doc-export` | Yes | Parsed from Wiki URL or API |
| `FEISHU_FOLDER_TOKEN` | Folder token for creating a new DocX | Optional target | `+doc-export` | Yes | Feishu Drive folder URL / Open Platform docs |

Legacy compatibility:

```text
FEISHU_DOC_FOLDER_TOKEN is still accepted after FEISHU_FOLDER_TOKEN.
```

Example:

```powershell
$env:FEISHU_WIKI_URL="https://example.feishu.cn/wiki/REDACTED"
$env:FEISHU_FOLDER_TOKEN="REDACTED"
```

## Base / Bitable Variables

| Name | Purpose | Required | Used by | Sensitive | How to obtain |
| --- | --- | --- | --- | --- | --- |
| `FEISHU_BASE_APP_TOKEN` | Base app token | Required for `+bitable-sync --send` | `+bitable-sync` | Yes | Feishu Base URL / Open Platform docs |
| `FEISHU_REPORT_TABLE_ID` | Reports table ID | Required when syncing `reports` | `+bitable-sync` | Yes | Base table settings / API |
| `FEISHU_ISSUE_TABLE_ID` | Issues table ID | Required when syncing `issues` | `+bitable-sync` | Yes | Base table settings / API |
| `FEISHU_PR_TABLE_ID` | Pull request table ID | Required when syncing `prs` | `+bitable-sync` | Yes | Base table settings / API |
| `FEISHU_CONTRIBUTOR_TABLE_ID` | Contributors table ID | Optional | `+bitable-sync` | Yes | Base table settings / API |
| `FEISHU_TASK_TABLE_ID` | Task-candidate table ID | Optional | `+bitable-sync` | Yes | Base table settings / API |

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
| `FEISHU_TASK_PROJECT_ID` | Optional task project target | Optional | `+task-create` | Yes | Feishu Task project settings / API |
| `FEISHU_TASK_SECTION_ID` | Optional task section target | Optional | `+task-create` | Yes | Feishu Task section settings / API |

Current limitation:

```text
The experimental task create command creates task candidates through the Task API.
Project/section placement may require additional Feishu Task identifiers and scopes.
If placement fails, record the Open Platform error in the smoke report.
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
Never print raw webhook URLs or app secrets in smoke reports.
Do not commit tenant_access_token or user_access_token.
Do not enable --send in shared scripts unless the target test enterprise is intentional.
```
