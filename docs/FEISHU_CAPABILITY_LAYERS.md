# Feishu Capability Layers

Date: 2026-06-26

This document defines the implemented and planned Feishu integration layers for `gitlink-cli`.

## Layer 1: Stable Webhook Export

Status: stable surface.

Purpose:

```text
Export GitLink workflow report summaries to Feishu without modifying GitLink or Feishu resources.
```

Required Feishu permission:

```text
Feishu custom bot webhook in a target chat.
```

Required environment variables:

```text
FEISHU_WEBHOOK_URL
FEISHU_WEBHOOK_SECRET optional
```

Implemented commands:

```text
gitlink-cli feishu +bot-test
gitlink-cli feishu +notify
gitlink-cli feishu +weekly-report
gitlink-cli feishu +owner-digest
gitlink-cli feishu +contributor-digest
gitlink-cli feishu +bitable-schema
gitlink-cli feishu +bitable-records
gitlink-cli feishu +task-preview
```

What it can do:

```text
Send Feishu custom bot cards when --send is explicit.
Render weekly reports as markdown.
Generate owner-oriented digests.
Generate contributor-oriented digests.
Generate Bitable-ready local records.
Generate task candidates locally.
Add navigation-only buttons to GitLink or Feishu URLs.
```

What it cannot do:

```text
Write Feishu Docs.
Write Feishu Wiki.
Write Feishu Base / Bitable.
Create Feishu Tasks.
Receive card callbacks.
Modify GitLink issues.
Review GitLink pull requests.
Merge pull requests.
Close issues.
Modify members.
Modify GitLink webhooks.
```

Testing:

```text
Unit and mock tests are implemented.
Real custom bot sending can be tested when FEISHU_WEBHOOK_URL exists.
```

## Layer 1.5: Configuration Diagnostics

Status: stable read/check surface.

Purpose:

```text
Help users discover missing env vars, invalid tokens, missing table IDs, missing
unique_key fields, and next-stage Task limitations before running --send writes.
```

Implemented commands:

```text
gitlink-cli feishu +app-check
gitlink-cli feishu +doc-check
gitlink-cli feishu +bitable-check
gitlink-cli feishu +task-check
```

Default behavior:

```text
Local checks only. No Feishu OpenAPI call is made unless --remote is explicit.
```

Remote behavior:

```text
--remote acquires tenant_access_token.
+doc-check --remote may resolve a Wiki node.
+bitable-check --remote searches a sentinel unique_key to verify table access
and the unique_key field.
+task-check --remote verifies app credentials only; it does not create tasks.
```

What it cannot do:

```text
Create Feishu resources.
Modify Feishu resources.
Modify GitLink resources.
Guarantee DocX edit permission without an actual append.
Guarantee Feishu Task project/section placement.
```

## Layer 2: Experimental Open Platform Validation

Status: experimental validation surface.

Purpose:

```text
Validate Feishu self-built app integration for Docs, Wiki, Base, and Tasks.
```

Required Feishu permission:

```text
Self-built app with approved scopes and resource-level access.
```

Test-enterprise note:

```text
The local validation enterprise used a self-built app with broad permissions so
that DocX, Base, and Task APIs could be tested end to end. This is only a
validation setup. Production deployments should use least-privilege scopes and
resource-level access selected by maintainers or administrators.
```

Required environment variables:

```text
FEISHU_APP_ID
FEISHU_APP_SECRET
FEISHU_WIKI_URL or FEISHU_WIKI_NODE_TOKEN optional for doc-export
FEISHU_FOLDER_TOKEN optional for doc-export
FEISHU_BASE_APP_TOKEN for bitable-sync
FEISHU_REPORT_TABLE_ID for reports table
FEISHU_ISSUE_TABLE_ID for issues table
FEISHU_PR_TABLE_ID for pull request table
FEISHU_CONTRIBUTOR_TABLE_ID optional
FEISHU_TASK_TABLE_ID optional
FEISHU_TASK_PROJECT_ID optional
FEISHU_TASK_SECTION_ID optional
```

Implemented commands:

```text
gitlink-cli feishu +app-check
gitlink-cli feishu +doc-check
gitlink-cli feishu +bitable-check
gitlink-cli feishu +task-check
gitlink-cli feishu +doc-export
gitlink-cli feishu +bitable-sync
gitlink-cli feishu +task-create
```

What it can do:

```text
Acquire tenant_access_token.
Resolve Wiki node tokens.
Attempt DocX / Wiki append or document creation with --send.
Preview or write Bitable records with --send.
Search Bitable records by unique_key before update.
Fall back to create-only if Bitable search fails.
Preview or create Feishu Tasks with --send.
Print diagnostic errors for permission, scope, ID, and resource-access failures.
```

What it cannot do:

```text
Create Base apps, tables, fields, or views.
Modify Feishu document permissions.
Guarantee Task deduplication against existing Feishu tasks.
Place created Feishu tasks into a specific Task project or section.
Assign task executors or followers.
Guarantee Bitable upsert if unique_key is missing from the target table.
Treat Open Platform writes as stable zero-config behavior.
```

Next-stage Open Platform boundary:

```text
Task project placement, section placement, executors, followers, and Feishu-side
dedupe/search should be implemented in a later stage after the exact Task API
request fields and tenant behavior are confirmed. They are not part of the
current stable or experimental surface.
```

Testing:

```text
Mock HTTP tests cover DocX/Wiki, Bitable sync, and Task create paths.
Real Open Platform calls require a configured test enterprise.
Failures should be preserved in smoke reports rather than converted into fake passes.
```

## Layer 3: GitLink Management Planning

Status: future work only.

Purpose:

```text
Plan a permissioned path where Feishu can become an entry point for selected GitLink actions.
```

Implemented commands:

```text
none
```

Planned requirements:

```text
Feishu callback verification.
Repo binding.
Feishu open_id / union_id to GitLink identity mapping.
GitLink permission checks.
Dry-run action preview.
Explicit confirmation.
Audit logs.
Maintainer-controlled policy.
```

Not implemented in this code path:

```text
issue comment
PR comment
PR review
PR approve
PR request changes
issue close
PR merge
member add/remove
webhook create/update
branch delete
release delete
callback server
action apply
```

Authorization policy:

```text
GitLink write permissions must be defined by GitLink official maintainers, project owners, and deployers.
This module must not hard-code a write-action authorization policy.
```
