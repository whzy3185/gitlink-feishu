# Code history and batch file shortcuts

## Background

Agent workflows such as PR review, commit quality checks, repository health
reports, and research reproducibility audits need commit timelines, changed
files, commit diffs, and sometimes controlled multi-file updates. Before this
change, several of these operations required Raw API calls.

This change adds a larger Subtask 1 feature set around repository code history
and file operations.

## New shortcuts

Repository shortcuts:

- `repo +files` searches repository files with optional `--search` and `--ref`.
- `repo +commits` lists commits for a branch, tag, or commit ref with pagination.
- `repo +commit-files` lists files changed by a commit, with optional file-path filtering.
- `repo +commit-diff` returns a commit diff.
- `repo +tags` lists repository tags with pagination and optional name filtering.
- `repo +tag` returns one tag's metadata and target commit.
- `repo +delete-tag` deletes a repository tag after explicit confirmation.
- `repo +batch-commit` creates, updates, or deletes multiple files in one commit.

Pull request shortcuts:

- `pr +commits` lists commits included in a pull request.

## Safety model

All history and file inspection commands are read-only.

`repo +delete-tag` and `repo +batch-commit` can modify repository content, so
they require an explicit confirmation flag for remote writes:

```bash
gitlink-cli repo +delete-tag --owner me --repo proj --name v0.1.0 --dry-run
gitlink-cli repo +delete-tag --owner me --repo proj --name v0.1.0 --yes
```

```bash
gitlink-cli repo +batch-commit --owner me --repo proj \
  --branch master --message "docs: update" \
  --files 'update:README.md:# Updated' \
  --dry-run

gitlink-cli repo +batch-commit --owner me --repo proj \
  --branch master --message "docs: update" \
  --files 'update:README.md:# Updated' \
  --yes
```

The `--files` format is:

```text
action:path[:content][;action:path[:content]...]
```

Supported actions are `create`, `update`, and `delete`. `create` and `update`
require content; `delete` does not.

## Tests

Unit tests cover:

- repository file search query mapping;
- commit list pagination and ref mapping;
- commit changed-file and diff endpoints;
- repository tag list/detail/delete endpoint mapping;
- PR commit list endpoint;
- `repo +batch-commit` dry-run behavior;
- `repo +batch-commit` remote write protection without `--yes`;
- batch file operation payload construction and validation.

Suggested verification:

```bash
go test ./shortcuts/repo ./shortcuts/pr
```

Full project verification:

```bash
make test
```
