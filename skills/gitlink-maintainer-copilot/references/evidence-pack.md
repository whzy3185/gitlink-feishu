# Maintainer Evidence Pack

Evidence Pack 是维护者驾驶舱的证据层。Agent 必须先采集证据，再给诊断和建议。

## 采集原则

- 优先只读命令。
- 每条诊断至少引用一个证据编号。
- 采集失败时记录缺失，不猜测。
- 同一个指标来自多个命令时，以更具体的数据为准，并说明来源。

## 证据清单

| 编号 | 名称 | 命令 | 主要字段 | 用途 |
|---|---|---|---|---|
| E1 | repo_profile | `gitlink-cli repo +info --format json` | `full_name`, `description`, `default_branch`, `license_name`, `watchers_count`, `forked_count`, `issues_count`, `pull_requests_count`, `empty` | 判断基础治理、项目吸引力、默认分支和公开信息 |
| E2 | open_issues | `gitlink-cli issue +list --state open --limit 100 --format json` | `subject`, `project_issues_index`, `created_at`, `updated_at`, `status_name`, `priority_name`, `assigners`, `tags`, `comment_journals_count` | 判断 Issue 积压、无人响应、分类质量 |
| E3 | closed_issues | `gitlink-cli issue +list --state closed --limit 100 --format json` | 同 E2 | 判断处理节奏、关闭质量、近期维护痕迹 |
| E4 | open_prs | `gitlink-cli pr +list --state open --limit 100 --format json` | `title`, `pull_request_number`, `created_at`, `updated_at`, `pull_request_status`, `user`, `head`, `base` | 判断 PR 堵塞、评审延迟、分支目标 |
| E5 | merged_prs | `gitlink-cli pr +list --state merged --limit 100 --format json` | 同 E4 | 判断合并效率、近期协作活跃度 |
| E6 | closed_prs | `gitlink-cli pr +list --state closed --limit 100 --format json` | 同 E4 | 判断拒绝/关闭模式 |
| E7 | releases | `gitlink-cli release +list --format json` | `name`, `tag_name`, `created_at`, `description`, `version_id` | 判断发布成熟度 |
| E8 | ci_builds | `gitlink-cli ci +builds --format json` | `id`, `status`, `created_at`, `duration`, `branch`, `commit` | 判断自动化质量 |
| E9 | commits | `gitlink-cli api GET /v1/<owner>/<repo>/commits --query 'page=1&limit=100' --format json` | `sha`, `commit_message`, `commit_time`, `author`, `files` | 判断近期活跃、提交分布和变更主题 |
| E10 | contributors | `gitlink-cli api GET /v1/<owner>/<repo>/contributors/stat --format json` | `total_count`, `contributors[].login`, `contributions`, `additions`, `deletions` | 判断贡献者集中度和协作风险 |
| E11 | languages | `gitlink-cli api GET /<owner>/<repo>/languages --format json` | language map | 判断技术栈和 README/CI 建议 |
| E12 | readme | `gitlink-cli api GET /<owner>/<repo>/readme --format json` | `content`, `encoding`, `sha` | 判断新手上手信息 |

## 缺失数据处理

记录格式：

```markdown
| 缺失项 | 命令 | 影响 | 后续建议 |
|---|---|---|---|
| E8 ci_builds | `gitlink-cli ci +builds ...` | 无法判断 CI 稳定性 | 在报告中把 CI 结论标为“未验证” |
```

常见处理：

- `401`：提示用户运行 `gitlink-cli auth login`，继续公开数据诊断。
- `403`：说明权限不足，避免输出内部治理判断。
- `404`：确认 owner/repo 是否正确。
- 空数组：这不是失败。应解读为“当前样本为空”，例如暂无 Release 或暂无 open PR。

## 证据引用格式

在报告中使用紧凑引用：

```markdown
- PR 堵塞风险高：[E4] open PR 8 个，其中 3 个超过 14 天未更新。
- 发布成熟度不足：[E7] 未发现 Release；[E9] 最近 100 个提交中已有 12 个 feat/fix 变更。
```

## 可信度等级

| 等级 | 条件 |
|---|---|
| High | E1-E7 至少 6 项可用，且 E9 或 E10 至少 1 项可用 |
| Medium | E1-E7 至少 4 项可用 |
| Low | 少于 4 项可用，或关键数据仅来自单一来源 |

可信度低时，输出应以“建议先补齐数据采集/权限”为主。
