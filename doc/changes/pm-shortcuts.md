# PM shortcuts

## Background

The `gitlink-pm` Skill documented GitLink project-management workflows but had
to call Raw API paths under `/pm/...` directly. That made Sprint reports, weekly
issue summaries, board inspection, and PM pipeline checks less discoverable for
users and agents.

This change adds a first-class `pm` shortcut group for read-only PM data.

## New shortcuts

- `pm +dashboards` lists PM dashboards for a project.
- `pm +sprint-issues` lists Sprint issues.
- `pm +weekly-issues` lists weekly issues.
- `pm +issue-tags` lists PM issue tags.
- `pm +pipelines` lists PM pipelines.
- `pm +action-runs` lists PM action run records.

All commands require `--project-id` and accept `--page`/`--limit`.

## Examples

```bash
gitlink-cli pm +dashboards --project-id 123 --limit 20
gitlink-cli pm +sprint-issues --project-id 123 --page 1 --limit 20
gitlink-cli pm +weekly-issues --project-id 123
gitlink-cli pm +issue-tags --project-id 123
gitlink-cli pm +pipelines --project-id 123
gitlink-cli pm +action-runs --project-id 123
```

## Documentation updates

- README and README.zh-CN include PM usage examples.
- `skills/gitlink-pm` now prefers `pm +...` shortcuts instead of Raw API calls.
- The Skills overview lists the new PM commands.

## Tests

Unit tests cover:

- all PM shortcut endpoint mappings;
- `project_id`, `page`, and `limit` query construction;
- default pagination;
- required `--project-id` validation;
- HTTP error propagation.

Suggested verification:

```bash
go test ./shortcuts/pm ./shortcuts
```

Full project verification:

```bash
go test ./...
```
