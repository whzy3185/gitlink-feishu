# Repository raw file shortcuts

## Background

Repository inspection workflows often need to read files such as `LICENSE`,
`README.md`, dependency manifests, and source files. Those workflows previously
had to use Raw API calls for `/raw/<ref>/<path>` and `sub_entries`.

This change adds first-class read-only shortcuts for raw file content and common
manifest discovery.

## New shortcuts

- `repo +raw` reads raw file content at a branch, tag, or commit ref.
- `repo +file-exists` checks whether a repository file exists without returning
  the full content.
- `repo +manifest` reads common dependency manifests for `go`, `node`,
  `python`, `rust`, and `java`.

## Examples

```bash
gitlink-cli repo +raw --owner Gitlink --repo forgeplus --path LICENSE --ref master
gitlink-cli repo +file-exists --owner Gitlink --repo forgeplus --path package.json --ref master
gitlink-cli repo +manifest --owner Gitlink --repo forgeplus --kind go --ref master
```

## Manifest mapping

| Kind | Candidate files |
|------|-----------------|
| `go` | `go.mod` |
| `node` | `package.json` |
| `python` | `requirements.txt`, `pyproject.toml`, `Pipfile`, `setup.py` |
| `rust` | `Cargo.toml` |
| `java` | `pom.xml`, `build.gradle`, `build.gradle.kts` |

## Safety and path handling

- These shortcuts are read-only.
- `--ref` defaults to `master`.
- File paths are cleaned of leading/trailing slashes.
- Parent path segments (`..`) are rejected before making API requests.
- Raw paths are escaped segment by segment for paths containing spaces or other
  special characters.

## Documentation updates

- README and README.zh-CN include raw file, file existence, and manifest examples.
- `skills/gitlink-repo` documents the new shortcuts.
- `skills/gitlink-compliance` and `skills/gitlink-code-review` now prefer
  shortcuts over Raw API calls for file tree and content reads.

## Tests

Unit tests cover:

- raw file path construction and default refs;
- file existence checks through `sub_entries`;
- manifest kind mapping and fallback behavior;
- missing manifests;
- invalid paths and unknown manifest kinds;
- HTTP error propagation.

Suggested verification:

```bash
go test ./shortcuts/repo ./shortcuts
```

Full project verification:

```bash
go test ./...
```
