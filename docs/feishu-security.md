# Feishu Security Notes

## Default Behavior

All Feishu shortcut commands default to local preview.

Real network writes require `--send`.

`--send` and `--dry-run` cannot be used together.

## Secrets

Supported environment variables:

```text
FEISHU_WEBHOOK_URL
FEISHU_WEBHOOK_SECRET
FEISHU_APP_ID experimental Open Platform commands only
FEISHU_APP_SECRET experimental Open Platform commands only
FEISHU_BASE_APP_TOKEN experimental bitable-sync only
FEISHU_REPORT_TABLE_ID experimental bitable-sync only
FEISHU_ISSUE_TABLE_ID experimental bitable-sync only
FEISHU_PR_TABLE_ID experimental bitable-sync only
FEISHU_WIKI_URL experimental doc-export only
FEISHU_WIKI_NODE_TOKEN experimental doc-export only
FEISHU_FOLDER_TOKEN experimental doc-export only
```

Do not commit real webhook URLs, app secrets, access tokens, Base app tokens, table IDs, or document tokens.

Command output redacts webhook URLs. Tests use fake webhook IDs.

## Stable Surface

The stable surface uses Feishu custom bot webhooks:

```text
feishu +bot-test
feishu +notify
feishu +weekly-report
feishu +owner-digest
feishu +contributor-digest
```

These commands can send Feishu cards, but they do not read or write Feishu documents, tables, users, or groups.

## Dry-Run Surface

The Bitable commands are local only:

```text
feishu +bitable-schema
feishu +bitable-records
feishu +task-preview
```

They do not call Feishu OpenAPI and cannot create, update, or upsert remote Feishu resources.

## Experimental Surface

These commands are experimental:

```text
feishu +doc-export
feishu +bitable-sync
feishu +task-create
```

They use:

```text
app_id
app_secret
tenant_access_token
Wiki OpenAPI
DocX OpenAPI
Bitable OpenAPI
Task OpenAPI
```

They should not be treated as part of the stable clean export path. If used, grant the self-built app only the minimum required resource permissions.

`+bitable-sync` never deletes records. It searches by `unique_key`, updates when found, creates when missing, and records diagnostics when Feishu rejects the call.

`+task-create` does not deduplicate against existing Feishu tasks unless Feishu-side identifiers and scopes later support that search path.

## Non-Goals

```text
BotBuilder integration
Feishu Robot Assistant workflows
automatic Feishu permission changes
GitLink remote writes
GitLink comments
Issue closure
merge actions
Feishu card callback execution
GitLink write actions from Feishu
```

