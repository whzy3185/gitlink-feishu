# gitlink-code-review API 参考

## PR 相关 API

### 获取 PR 详情

```bash
gitlink-cli pr +view --id <pull_request_id> --format json
```

**返回字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `data.issue.id` | int | Issue/PR 内部 ID |
| `data.issue.subject` | string | PR 标题 |
| `data.issue.description` | string | PR 描述 |
| `data.issue.created_at` | string | 创建时间 |
| `data.issue.issue_status` | string | 状态 |
| `data.issue.author_login` | string | 作者登录名 |
| `data.issue.author_name` | string | 作者名称 |
| `data.pull_request.base` | string | 目标分支 |
| `data.pull_request.head` | string | 源分支 |
| `data.pull_request.status` | int | 0=open, 1=merged, 2=closed |
| `data.pull_request.state` | string | "open" / "closed" |
| `data.pull_request.mergeable` | boolean | 是否可合并 |
| `data.pull_request.reviewers` | array | 审查者列表 |
| `data.commits_count` | int | 提交数 |
| `data.files_count` | int | 变更文件数 |
| `data.conflict_files` | array | 冲突文件列表 |

### 获取 PR 变更文件列表

```bash
gitlink-cli pr +files --id <pull_request_id> --format json
```

**返回字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `files_count` | int | 文件总数 |
| `total_addition` | int | 新增行数 |
| `total_deletion` | int | 删除行数 |
| `files[].name` | string | 文件名 |
| `files[].addition` | int | 该文件新增行数 |
| `files[].deletion` | int | 该文件删除行数 |
| `files[].type` | int | 1=新增, 2=修改, 3=删除 |
| `files[].isCreated` | boolean | 是否新建文件 |
| `files[].isDeleted` | boolean | 是否删除文件 |
| `files[].isRenamed` | boolean | 是否重命名 |

### 获取 PR Diff

```bash
gitlink-cli pr +diff --id <pull_request_id> --format json
```

**返回字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `files_count` | int | 文件总数 |
| `total_addition` | int | 总新增行数 |
| `total_deletion` | int | 总删除行数 |
| `files[].sections[].lines[].leftIdx` | int | 原文件行号 |
| `files[].sections[].lines[].rightIdx` | int | 新文件行号 |
| `files[].sections[].lines[].type` | int | 1=未变, 2=新增, 3=删除, 4=统计信息 |
| `files[].sections[].lines[].content` | string | 行内容 |

---

## PR 列表 API

```bash
gitlink-cli pr +list --state <open|merged|closed> --format json
```

**返回字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `data.open_count` | int | 开放 PR 数 |
| `data.close_count` | int | 关闭 PR 数 |
| `data.merged_issues_size` | int | 已合并 PR 数 |
| `data.issues[].id` | int | PR 内部 ID |
| `data.issues[].pull_request_number` | int | PR 编号（对外展示用） |
| `data.issues[].name` | string | PR 标题 |
| `data.issues[].author_login` | string | 作者登录名 |
| `data.issues[].author_name` | string | 作者名称 |
| `data.issues[].pull_request_status` | int | **关键字段：** 0=open, 1=merged, 2=closed |
| `data.issues[].pull_request_base` | string | 目标分支 |
| `data.issues[].pull_request_head` | string | 源分支 |
| `data.issues[].pr_created_unix` | int | 创建时间戳 |
| `data.issues[].pr_full_time` | string | 创建完整时间 |
| `data.issues[].journals_count` | int | 评论数 |
| `data.issues[].reviewers` | array | Reviewers 列表 |
| `data.issues[].fork_project_id` | int | Fork 项目 ID |
| `data.issues[].is_original` | boolean | 是否来自 Fork |

---

## Issue 相关 API

### 获取 Issue 列表

```bash
gitlink-cli issue +list --owner <owner> --repo <repo> --format json
```

**返回字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `data.closed_count` | int | 已关闭 Issue 数 |
| `data.issues[].id` | int | Issue ID |
| `data.issues[].name` | string | Issue 标题 |
| `data.issues[].author.login` | string | 作者登录名 |
| `data.issues[].author.name` | string | 作者名称 |
| `data.issues[].assigners` | array | 指派人列表 |
| `data.issues[].priority.name` | string | 优先级名称 |
| `data.issues[].priority.id` | int | 优先级 ID |
| `data.issues[].created_at` | string | 创建时间 |
| `data.issues[].milestone_name` | string | 里程碑名称 |
| `data.issues[].due_date` | string | 截止日期 |

---

## 仓库 API

### 获取仓库信息

```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
```

**返回字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `data.project_id` | int | 项目 ID |
| `data.identifier` | string | 仓库标识 |
| `data.name` | string | 项目名称 |
| `data.full_name` | string | 全名（owner/repo） |
| `data.default_branch` | string | 默认分支 |
| `data.private` | boolean | 是否私有 |
| `data.empty` | boolean | 是否空仓库 |
| `data.issues_count` | int | Issue 总数 |
| `data.pull_requests_count` | int | PR 总数 |
| `data.forked_count` | int | Fork 数 |
| `data.praises_count` | int | Star 数 |
| `data.watchers_count` | int | 关注数 |
| `data.contributor_users_count` | int | 贡献者数 |
| `data.version_releases_count` | int | Release 数 |
| `data.size` | string | 仓库大小 |
| `data.clone_url` | string | HTTPS 克隆地址 |
| `data.ssh_url` | string | SSH 克隆地址 |
| `data.author.login` | string | 仓库所有者 |

### 获取仓库文件列表

```bash
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=<path>&ref=<branch>' --format json
```

**返回字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `data.entries[].name` | string | 文件/目录名 |
| `data.entries[].type` | string | "dir" 或 "file" |
| `data.entries[].path` | string | 相对路径 |
| `data.entries[].size` | int | 文件大小(bytes) |
| `data.entries[].sha` | string | 文件 SHA |
| `data.entries[].content` | string | 文件内容（部分文件可能直接返回） |

### 获取仓库语言统计

```bash
gitlink-cli api GET /:owner/:repo/languages --format json
```

**返回示例：**
```json
{ "Ruby": "90.2%", "JavaScript": "6.1%", "CSS": "3.7%" }
```

### 获取仓库原始文件

```bash
gitlink-cli api GET /:owner/:repo/raw/<branch>/<filepath>
```

---

## 用户 API

### 查看当前用户

```bash
gitlink-cli user +me --format json
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `data.login` | string | 登录名 |
| `data.user_id` | int | 用户 ID |
| `data.username` | string | 用户名 |
| `data.email` | string | 邮箱 |
| `data.phone` | string | 手机号 |
| `data.admin` | boolean | 是否管理员 |

### 获取用户信息

```bash
gitlink-cli api GET /users/:user_id --format json
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `data.login` | string | 登录名 |
| `data.name` | string | 显示名称 |
| `data.user_id` | int | 用户 ID |
| `data.user_identity` | string | 身份（专业人士/学生等） |
| `data.user_projects_count` | int | 项目数 |
| `data.common_projects_count` | int | 参与项目数 |
| `data.created_time` | string | 注册时间 |
| `data.gender` | int | 性别 |

---

## 数据获取最佳实践

1. **始终使用 `--format json`** 确保可解析输出
2. PR ID 使用 `pull_request_id`（内部 ID），PR 编号展示用 `pull_request_number`
3. PR 状态通过 `pull_request_status` 判断：0=open, 1=merged, 2=closed
4. 大型 PR 的 diff 可能非常大，建议按文件分组逐文件分析
5. Issue 的 `closed_count` 和 `open_count` 在列表的数据顶层
