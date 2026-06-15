# Feishu Bitable Dry-Run Schema

`gitlink-cli feishu` currently generates Bitable schema and records locally.

It does not call Feishu Bitable OpenAPI.

## Commands

```bash
gitlink-cli feishu +bitable-schema --format markdown
gitlink-cli feishu +bitable-records --from-workflow-json report.json --format json
```

## Tables

Default tables:

```text
issues
prs
contributors
reports
```

## Record Semantics

Records are summary records derived from `workflow +repo-report` JSON.

They are not one row per GitLink issue or one row per GitLink pull request.

Current behavior:

```text
reports: one summary row per repo report
issues: summary buckets by issue type and priority
prs: summary buckets by change type and risk
contributors: reserved schema; records are empty unless workflow JSON later includes contributor details
```

## Real Write Boundary

Not implemented:

```text
Bitable OpenAPI create
Bitable OpenAPI batch create
Bitable update
Bitable upsert
Base creation
table creation
view creation
field creation
person/open_id mapping
```

Real Bitable writes require a separate design for:

```text
app authentication
table IDs
record unique keys
search-before-update
pagination
partial failure handling
rate limits
permission diagnostics
```

