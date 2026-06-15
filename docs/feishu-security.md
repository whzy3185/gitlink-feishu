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
FEISHU_APP_ID experimental doc-export only
FEISHU_APP_SECRET experimental doc-export only
```

Do not commit real webhook URLs, app secrets, access tokens, Base app tokens, table IDs, or document tokens.

Command output redacts webhook URLs. Tests use fake webhook IDs.

## Stable Surface

The stable surface uses Feishu custom bot webhooks:

```text
feishu +bot-test
feishu +notify
feishu +weekly-report
```

These commands can send Feishu cards, but they do not read or write Feishu documents, tables, users, or groups.

## Dry-Run Surface

The Bitable commands are local only:

```text
feishu +bitable-schema
feishu +bitable-records
```

They do not call Feishu OpenAPI and cannot create, update, or upsert Bitable records.

## Experimental Surface

`feishu +doc-export` is experimental. It uses:

```text
app_id
app_secret
tenant_access_token
Wiki OpenAPI
DocX OpenAPI
```

It should not be treated as part of the stable clean export path. If used, grant the self-built app only the minimum required document permissions.

## Non-Goals

```text
BotBuilder integration
Feishu Robot Assistant workflows
automatic Feishu permission changes
GitLink remote writes
GitLink comments
Issue closure
merge actions
real Bitable writes
```

