# 执行样例：jiangtx/gitlink-cli 贡献者活跃度分析

> 执行日期：2026-06-03
> 执行版本：gitlink-cli (当前版本)
> 说明：本文件记录了一次完整的贡献者分析执行过程，包含实际命令输出和最终报告。可作为后续分析的参考模板。

---

## 实际执行的命令与输出

### 1. `gitlink-cli repo +info`

```json
{
  "ok": true,
  "data": {
    "author": { "id": 148911, "login": "jiangtx", "name": "jiangtx" },
    "clone_url": "https://gitlink.org.cn/jiangtx/gitlink-cli.git",
    "contributor_users_count": 2,
    "default_branch": "master",
    "fork_info": {
      "fork_form_name": "gitlink-cli",
      "fork_project_identifier": "gitlink-cli",
      "fork_project_user_login": "Gitlink",
      "fork_project_user_name": "GitLink"
    },
    "forked_from_project_id": 1513956,
    "full_name": "jiangtx/gitlink-cli",
    "identifier": "gitlink-cli",
    "issues_count": 0,
    "permission": "Manager",
    "private": false,
    "project_id": 1547588,
    "pull_requests_count": 9,
    "size": "13.4 MB",
    "watchers_count": 0
  }
}
```

### 2. `gitlink-cli pr +list` (关键数据源)

9 个 PR，全部已合并。按作者汇总：

| author_login | PR 数 | PR 编号 | 时间范围 |
|-------------|--------|---------|----------|
| lindiwen23 | 5 | #1, #6, #7, #8, #9 | 2026-06-01～06-03 |
| jiangtx | 4 | #2, #3, #4, #5 | 2026-06-02～06-03 |

PR 详细列表：

```json
// lindiwen23 的 PR
{ "pull_request_number": 1, "author_login": "lindiwen23",
  "name": "feat: 新增 3 个 Skill（onboarding / issue-triage / research-tracker）",
  "pr_full_time": "2026-06-01T11:33:00.000+08:00", "pull_request_status": 1,
  "journals_count": 3 }

{ "pull_request_number": 6, "author_login": "lindiwen23",
  "name": "fix: detectHTMLResponse 跳过 XML 声明，添加 HTML 响应检测",
  "pr_full_time": "2026-06-03T08:55:10.000+08:00", "pull_request_status": 1,
  "journals_count": 2 }

{ "pull_request_number": 7, "author_login": "lindiwen23",
  "name": "feat: org +teams/+create-team/+remove-user, search +code/+issues",
  "pr_full_time": "2026-06-03T09:13:15.000+08:00", "pull_request_status": 1,
  "journals_count": 2 }

{ "pull_request_number": 8, "author_login": "lindiwen23",
  "name": "feat: 新建 notification 模块并注册",
  "pr_full_time": "2026-06-03T09:17:32.000+08:00", "pull_request_status": 1,
  "journals_count": 2 }

{ "pull_request_number": 9, "author_login": "lindiwen23",
  "name": "fix: Skills 文件 api 命令替换为 Shortcut 命令 (~107 处)",
  "pr_full_time": "2026-06-03T09:38:05.000+08:00", "pull_request_status": 1,
  "journals_count": 2 }

// jiangtx 的 PR
{ "pull_request_number": 2, "author_login": "jiangtx",
  "name": "基础设施修复",
  "pr_full_time": "2026-06-02T16:54:43.000+08:00", "pull_request_status": 1,
  "journals_count": 2 }

{ "pull_request_number": 3, "author_login": "jiangtx",
  "name": "repo 域补全",
  "pr_full_time": "2026-06-02T23:27:46.000+08:00", "pull_request_status": 1,
  "journals_count": 2 }

{ "pull_request_number": 4, "author_login": "jiangtx",
  "name": "pr 域补全",
  "pr_full_time": "2026-06-03T00:09:46.000+08:00", "pull_request_status": 1,
  "journals_count": 2 }

{ "pull_request_number": 5, "author_login": "jiangtx",
  "name": "label 模块新建",
  "pr_full_time": "2026-06-03T00:25:32.000+08:00", "pull_request_status": 1,
  "journals_count": 2 }
```

### 3. `gitlink-cli user +info --login jiangtx`

```json
{
  "ok": true,
  "data": {
    "login": "jiangtx", "name": "jiangtx",
    "user_id": 148911,
    "created_time": "2026-04-28 11:46",
    "user_projects_count": 3,
    "user_org_count": 0,
    "user_identity": "专业人士"
  }
}
```

### 4. `gitlink-cli user +info --login lindiwen23`

```json
{
  "ok": true,
  "data": {
    "login": "lindiwen23", "name": "lindiwen23",
    "user_id": 141609,
    "created_time": "2025-05-26 22:40",
    "user_projects_count": 6,
    "user_org_count": 1,
    "user_identity": "专业人士"
  }
}
```

### 5. 不可用的命令

| 命令 | 结果 |
|------|------|
| `gitlink-cli repo +contributors` | 命令不存在，返回 repo 帮助文本 |
| `gitlink-cli user +heatmap` | 命令不存在（user 仅 `+info` / `+me`） |
| `gitlink-cli user +stats` | 命令不存在 |
| `gitlink-cli user +trends` | 命令不存在 |
| `gitlink-cli api GET "/api/v1/repos/.../contributors"` | 返回 HTML 页面，非 JSON |
| `gitlink-cli api GET "/api/v1/users/.../heatmap"` | 返回 HTML 页面，非 JSON |

---

## 数据处理过程

### 贡献者发现

由于 `repo +contributors` 不可用：
1. 从 `repo +info` 获取 `contributor_users_count = 2`
2. 从 `pr +list` 提取唯一 `author_login`：`["lindiwen23", "jiangtx"]`（2 人，一致）

### 活跃天数计算

从 PR 的 `pr_full_time` 字段提取日期：

| 贡献者 | 活跃日期 | 活跃天数 |
|--------|----------|----------|
| lindiwen23 | 2026-06-01, 2026-06-03 | 2 天 |
| jiangtx | 2026-06-02, 2026-06-03 | 2 天 |

### 趋势判断

按日聚合 PR 数：
- 06-01: 1 PR
- 06-02: 2 PRs
- 06-03: 6 PRs

趋势：↑ 上升（日产出加速：1→2→6）

### 贡献类型分类

| 贡献者 | PR 数 | Issue 数 | 类型 |
|--------|-------|----------|------|
| lindiwen23 | 5 | 0 | 代码贡献者 |
| jiangtx | 4 | 0 | 代码贡献者 |

> Issue 来源：`repo +info` 中 `issues_count = 0`，无 Issue 需要获取。

### 分级调整

由于项目仅 3 天历史（< 30 天），适用年轻项目特殊处理：
- jiangtx（PR=4，2 活跃天）→ 🔥 核心贡献者
- lindiwen23（PR=5，2 活跃天）→ 🔥 核心贡献者

---

## 完整输出报告

（见当天执行输出，此处省略以保持文件精简。核心结构：团队概览 → 排行榜 → 个人分析 → 健康度评估 → 建议。）

---

## 经验总结

1. **PR 数据可作为贡献者分析的主要数据源**：`pr +list` 提供了作者、时间、状态、标题等丰富信息
2. **`pr_full_time` 字段足够做时间分布分析**：可计算活跃天数、贡献频率、趋势
3. **`user +info` 补充贡献者画像**：注册时间、项目数、组织数可用于背景分析
4. **极端年轻项目的分级需放宽**：标准分级（PR > 10）对 3 天项目不适用
5. **Raw API 不可靠**：GitLink 的 API 结构与标准 Gitea 不同，建议仅使用 Shortcut 命令
