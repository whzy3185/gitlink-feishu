# Repo Tree Shortcut

## Summary

Adds `gitlink-cli repo +tree` so users and AI Agents can list repository files and directories without manually calling the Raw API.

## Command

| Command | Purpose |
|---------|---------|
| `gitlink-cli repo +tree` | List repository files and directories at the repository root or a specified path |

## Usage

```bash
# List repository root entries
gitlink-cli repo +tree --owner Gitlink --repo forgeplus --ref master

# List entries under a directory
gitlink-cli repo +tree --owner Gitlink --repo forgeplus --path src --ref main

# AI Agent usage with structured output
gitlink-cli repo +tree --owner Gitlink --repo forgeplus --format json
```

## Behavior

- Calls `GET /{owner}/{repo}/sub_entries`.
- Maps `--path, -p` to the `filepath` query parameter.
- Maps `--ref, -r` to the `ref` query parameter.
- Defaults `--ref` to `master` when omitted.
- Reuses existing repository context resolution and output formatting.

## Tests

Unit tests cover:

- root directory listing with the default `master` ref;
- directory listing with explicit `--path` and `--ref`;
- endpoint path and query parameter mapping.

Validation command:

```bash
make test
```

## 中文说明

### 变更内容

- 新增 `repo +tree` 仓库文件树查询命令。
- 支持查看仓库根目录或指定目录下的文件和子目录。
- 支持通过 `--ref` 指定分支、标签或提交引用。
- 更新 README、README.zh-CN、设计文档和 `gitlink-repo` Skill 文档。
- 新增 `gitlink-repo` Skill 参考文档，便于 Agent 在项目结构检查、复现性检查和合规检查中复用。

### 验证

- `make test`
