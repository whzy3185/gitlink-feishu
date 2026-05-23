# gitlink-insight API 参考

## 数据聚合 API

以下 API 用于采集项目健康度、协作洞察所需的基础数据。

### 仓库信息

```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `data.project_id` | int | 项目 ID |
| `data.identifier` | string | 仓库标识 |
| `data.issues_count` | int | Issue 总数 |
| `data.pull_requests_count` | int | PR 总数 |
| `data.forked_count` | int | Fork 数 |
| `data.praises_count` | int | Star 数 |
| `data.watchers_count` | int | 关注数 |
| `data.contributor_users_count` | int | 贡献者数 |
| `data.version_releases_count` | int | 版本发布数 |
| `data.default_branch` | string | 默认分支 |
| `data.author.login` | string | 所有者 |
| `data.author.name` | string | 所有者名称 |
| `data.private` | boolean | 是否私有 |
| `data.clone_url` | string | 克隆地址 |

### Issue 列表

```bash
gitlink-cli issue +list --owner <owner> --repo <repo> --format json
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `data.closed_count` | int | 已关闭总数 |
| `data.issues[].id` | int | Issue ID |
| `data.issues[].name` | string | 标题 |
| `data.issues[].author.login` | string | 作者 |
| `data.issues[].assigners` | array | 指派人 |
| `data.issues[].created_at` | string | 创建时间 |
| `data.issues[].priority.name` | string | 优先级 |
| `data.issues[].milestone_name` | string | 里程碑 |
| `data.issues[].comment_journals_count` | int | 评论数 |

### PR 列表

```bash
gitlink-cli pr +list --state <open|merged|closed> --owner <owner> --repo <repo> --format json
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `data.open_count` | int | 开放 PR 数 |
| `data.close_count` | int | 关闭 PR 数 |
| `data.merged_issues_size` | int | 已合并 PR 数 |
| `data.search_count` | int | 搜索总数 |
| `data.issues[].pull_request_number` | int | PR 编号 |
| `data.issues[].name` | string | PR 标题 |
| `data.issues[].author_login` | string | 作者登录名 |
| `data.issues[].author_name` | string | 作者显示名 |
| `data.issues[].pull_request_status` | int | **0=open, 1=merged, 2=closed** |
| `data.issues[].pull_request_base` | string | 目标分支 |
| `data.issues[].pull_request_head` | string | 源分支 |
| `data.issues[].pr_created_unix` | int | 创建时间戳 |
| `data.issues[].pr_full_time` | string | 创建完整时间 |
| `data.issues[].journals_count` | int | 评论数 |
| `data.issues[].is_original` | boolean | 是否原始 PR |
| `data.issues[].fork_project_user` | string | Fork 来源用户 |

### Release 列表

```bash
gitlink-cli release +list --format json
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `data[].version_id` | int | 版本 ID |
| `data[].title` | string | 版本标题 |
| `data[].tag_name` | string | 标签名 |
| `data[].created_at` | string | 创建时间 |
| `data[].body` | string | 发布说明 |

### 仓库文件列表

```bash
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=&ref=<branch>' --format json
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `data.entries[].name` | string | 文件名 |
| `data.entries[].type` | string | "file" 或 "dir" |
| `data.entries[].size` | int | 文件大小 |
| `data.entries[].sha` | string | 文件 SHA |

### 仓库语言统计

```bash
gitlink-cli api GET /:owner/:repo/languages --format json
```

**返回示例：** `{ "Ruby": "90.2%", "JavaScript": "6.1%", "CSS": "3.7%" }`

### 贡献者列表

```bash
gitlink-cli api GET /:owner/:repo/contributors --format json
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `data[].login` | string | 登录名 |
| `data[].name` | string | 显示名 |
| `data[].image_url` | string | 头像 |
| `data[].commits_count` | int | 提交数（如返回） |

### 仓库动态

```bash
gitlink-cli api GET /:owner/:repo/activity --format json
```

### 获取用户详情

```bash
gitlink-cli api GET /users/:user_id --format json
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `data.login` | string | 登录名 |
| `data.name` | string | 名称 |
| `data.user_id` | int | 用户 ID |
| `data.user_identity` | string | 身份 |
| `data.user_projects_count` | int | 项目数 |
| `data.created_time` | string | 注册时间 |

---

## 健康度评分标准

| 维度 | 权重 | 数据来源 | 评分方法 |
|------|:----:|----------|----------|
| 文档完整性 | 15% | `sub_entries` 检查 README、CONTRIBUTING、CHANGELOG | 存在=5分/个，内容质量酌情加减 |
| 许可证合规 | 10% | `sub_entries` 检查 LICENSE | 存在=10分，有内容=+5分 |
| 工程化程度 | 15% | CI 配置、linter、.gitignore | 每项配置=5分 |
| 代码质量 | 20% | 测试文件数 / 源码文件数 | >30%=满分，逐步递减 |
| 协作指标 | 20% | Issue/PR 响应和关闭数据 | 响应<24h=10分，关闭率>80%=10分 |
| 社区活跃 | 10% | Star/Fork/贡献者数 | 相对评分，取仓库间的百分位 |
| 依赖管理 | 10% | 依赖配置文件完整性 | 有配置文件=10分 |

---

## 时间范围过滤

对于周报/月报报告，通过 API 返回的 `created_at` / `pr_full_time` 字段在 Agent 端过滤：

```javascript
// 示例：获取本周的数据
const now = new Date();
const weekAgo = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
// 客户端过滤 pr_full_time >= weekAgo
```

---

## 分页处理

对于大量 Issue/PR，API 使用服务端分页：

```bash
# 目前 API 返回所有数据，无显式分页参数
# Agent 端可根据数据量自行分批处理
```

## 注意事项

- PR 列表返回的 `--state` 参数仅影响统计计数，实际过滤需客户端判断 `pull_request_status`
- 仓库活动（activity）API 返回的数据格式未完全验证，Agent 需做容错处理
- 贡献者详情需循环调用 `/users/:user_id` 分别获取
- 所有数据尽可能使用 `--format json` 以确保解析稳定性
