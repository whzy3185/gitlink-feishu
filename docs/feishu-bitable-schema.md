# Feishu Bitable Dry-Run Schema

`gitlink-cli feishu` generates Bitable schema and records locally.

`+bitable-schema` and `+bitable-records` do not call Feishu Bitable OpenAPI.

`+bitable-sync` is an experimental Open Platform command. It requires explicit `--send` before it writes.

## Commands

```bash
gitlink-cli feishu +bitable-schema --format markdown
gitlink-cli feishu +bitable-records --from-workflow-json report.json --format json
gitlink-cli feishu +bitable-sync --from-workflow-json report.json --tables reports,issues,prs,tasks --format table
```

## Tables

Default tables:

```text
issues
prs
contributors
reports
tasks
```

## Record Semantics

Records are summary records derived from `workflow +repo-report` JSON.

They are not one row per GitLink issue or one row per GitLink pull request.

Current behavior:

```text
reports: one summary row per repo report
issues: summary buckets by issue type and priority
prs: summary buckets by change type and risk
contributors: role-oriented contributor digest summary
tasks: task candidates derived from recommendations, high-risk issues, high-risk PRs, and missing information
```

## Real Write Boundary

Implemented experimentally:

```text
Bitable record search by unique_key
Bitable record create
Bitable record update
create-only fallback when search fails
no-delete behavior
```

Not implemented:

```text
pagination
batch create
Base creation
table creation
view creation
field creation
person/open_id mapping
rate limits
```

Real Bitable writes require existing Base app and table IDs. The target tables should include a text field named `unique_key`.

