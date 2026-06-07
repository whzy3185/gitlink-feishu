# Repo Insight Shortcuts

## Summary

Adds read-only repository insight shortcuts so maintainers and agents can inspect project health without falling back to Raw API calls.

## Commands

| Command | Purpose |
|---------|---------|
| `gitlink-cli repo +languages` | Show repository language statistics |
| `gitlink-cli repo +contributors` | List repository contributors |
| `gitlink-cli repo +contributor-stats` | List contributor statistics with additions and deletions |
| `gitlink-cli repo +code-stats` | Show repository code statistics |
| `gitlink-cli repo +watchers` | List repository watchers |
| `gitlink-cli repo +stargazers` | List repository stargazers |
| `gitlink-cli repo +follow` | Follow a repository |
| `gitlink-cli repo +unfollow` | Unfollow a repository |
| `gitlink-cli repo +like` | Like a repository |
| `gitlink-cli repo +unlike` | Unlike a repository |

## Validation

- `repo +contributor-stats --pass-year` must be a positive integer.
- `repo +watchers` and `repo +stargazers` accept optional `--start-at` and `--end-at` Unix timestamps.
- Time range timestamps must be non-negative, and `--start-at` cannot be greater than `--end-at`.
- `repo +follow`, `repo +unfollow`, `repo +like`, and `repo +unlike` accept optional `--project-id`; if omitted, the project ID is resolved from `--owner/--repo`.
- Repository interaction actions support `--dry-run` so callers can preview the resolved project ID and endpoint before changing remote state.

## Tests

Unit tests cover endpoint paths, query parameter mapping, optional ref and time-range filters, project ID auto-resolution, dry-run previews, and invalid argument handling before any API request is sent.

## 中文说明

### 变更内容

- 新增 `repo +languages`、`repo +contributors`、`repo +contributor-stats`、`repo +code-stats`、`repo +watchers`、`repo +stargazers` 等仓库洞察命令。
- 新增 `repo +follow`、`repo +unfollow`、`repo +like`、`repo +unlike` 仓库互动命令，并支持 `--project-id` 和 `--dry-run`。
- `repo +contributor-stats` 和 `repo +code-stats` 使用 v1 API，支持 `--ref` 和 `--pass-year` 参数。
- `repo +watchers` 和 `repo +stargazers` 支持 `--start-at` / `--end-at` 时间范围，并在请求前校验时间戳。
- 更新 README、README.zh-CN、`gitlink-repo` Skill 和变更说明，减少仓库分析场景对 Raw API 的依赖。
- 提交者：王越

### 验证

- `GOPROXY=https://goproxy.cn,direct go test ./...`
- `go run . repo --help`
- `go run . repo +contributor-stats --help`
- `go run . repo +watchers --help`
- `git diff --check`
