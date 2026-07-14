# gitlink-research-profile — API 字段参考

本文档列出 `gitlink-cli profile` 各子命令对应的 GitLink 平台接口、参数与返回字段，
字段结构基于真实接口响应整理。所有命令均为只读。

> 统一说明：`gitlink-cli` 返回结构为 `{"ok": true, "data": {...}}`，下文"字段"均指 `data` 内的路径。
> 平台原始接口路径为 `/api/users/{owner}/statistics/*` 与 `/api/users/{owner}/headmaps`，
> `gitlink-cli` 自动处理认证与 `.json` 后缀。

---

## 1. `profile +ability` — 开发能力

- 命令：`gitlink-cli profile +ability --user <login> [--start-time <unix>] [--end-time <unix>]`
- 接口：`GET /api/users/{login}/statistics/develop`

| 字段 | 类型 | 说明 |
|------|------|------|
| `platform.influence` | int(0–100) | 平台影响力基线（头部参照） |
| `platform.contribution` | int | 平台贡献度基线 |
| `platform.activity` | int | 平台活跃度基线 |
| `platform.experience` | int | 平台项目经验基线 |
| `platform.language` | int | 平台语言能力基线 |
| `user.influence` | int(0–100) | 用户影响力 |
| `user.contribution` | int | 用户贡献度 |
| `user.activity` | int | 用户活跃度 |
| `user.experience` | int | 用户项目经验 |
| `user.language` | int | 用户语言能力 |
| `user.languages_percent` | object{string:float} | 各语言代码量占比（0–1），键为语言名 |
| `user.each_language_score` | object{string:int} | 各语言能力分（0–100） |

示例（节选）：

```json
{
  "platform": {"activity":99,"contribution":99,"experience":99,"influence":99,"language":98},
  "user": {
    "activity":94,"contribution":97,"experience":97,"influence":79,"language":90,
    "languages_percent": {"Go":0.2,"Java":0.13,"HTML":0.11,"JavaScript":0.09},
    "each_language_score": {"Go":87,"Java":83,"HTML":81,"JavaScript":80}
  }
}
```

---

## 2. `profile +major` — 学科 / 研究方向

- 命令：`gitlink-cli profile +major --user <login> [--start-time] [--end-time]`
- 接口：`GET /api/users/{login}/statistics/major`

| 字段 | 类型 | 说明 |
|------|------|------|
| `categories` | [string] | 平台推断的研究/技术方向标签，可能为空数组 |

示例：

```json
{ "categories": ["人工智能","大数据","生物医药健康","天文地球物理","操作系统"] }
```

> 标签数量极多（20+）通常意味着账号参与/镜像了大量异质项目（平台维护者或镜像聚合），
> 需结合 `user +info.mirror_projects_count` 判断方向是否"虚泛"。

---

## 3. `profile +role` — 角色定位

- 命令：`gitlink-cli profile +role --user <login> [--start-time] [--end-time]`
- 接口：`GET /api/users/{login}/statistics/role`

| 字段 | 类型 | 说明 |
|------|------|------|
| `role.owner.count` / `role.owner.percent` | int / float | 作为 owner 的项目数 / 占比 |
| `role.manager.count` / `.percent` | int / float | 作为 manager 的项目数 / 占比 |
| `role.developer.count` / `.percent` | int / float | 作为 developer 的项目数 / 占比 |
| `role.reporter.count` / `.percent` | int / float | 作为 reporter 的项目数 / 占比 |
| `total_projects_count` | int | 参与项目总数 |

示例：

```json
{
  "role": {
    "owner": {"count":20,"percent":0.99},
    "manager": {"count":2,"percent":0},
    "developer": {"count":5,"percent":0},
    "reporter": {"count":1,"percent":0}
  },
  "total_projects_count": 28
}
```

> `percent` 为四舍五入小数，小占比可能显示为 0；解读时以 `count` 为准。

---

## 4. `profile +activity` — 近期活动

- 命令：`gitlink-cli profile +activity --user <login>`
- 接口：`GET /api/users/{login}/statistics/activity`

| 字段 | 类型 | 说明 |
|------|------|------|
| `dates` | [string] | 日期序列（如 `2026.06.14`），通常近一周 |
| `commits_count` | [int] | 与 `dates` 等长，逐日提交数 |
| `issues_count` | [int] | 与 `dates` 等长，逐日疑修数 |
| `pull_requests_count` | [int] | 与 `dates` 等长，逐日合并请求数 |

> 四个数组**等长且按下标对齐**：第 i 天 = `dates[i]`，当日提交 = `commits_count[i]`。

---

## 5. `profile +contribution` — 贡献热力图

- 命令：`gitlink-cli profile +contribution --user <login> [--year <yyyy>]`
- 接口：`GET /api/users/{login}/headmaps`

| 字段 | 类型 | 说明 |
|------|------|------|
| `total_contributions` | int | 时间范围内贡献总量 |
| `headmaps` | [object] | 每日贡献数组 |
| `headmaps[].date` | string | 日期（`YYYY-MM-DD`） |
| `headmaps[].contributions` | int | 当日贡献数 |

示例（节选）：

```json
{ "total_contributions": 1280, "headmaps": [ {"date":"2025-06-16","contributions":80}, {"date":"2025-06-18","contributions":28} ] }
```

---

## 6. 辅助命令

| 命令 | 用途 | 关键字段 |
|------|------|----------|
| `gitlink-cli user +info --login <login>` | 基础资料 | `name`/`real_name`、`custom_department`（机构）、`city`/`province`、`common_projects_count`、`mirror_projects_count`、`created_time` |
| `gitlink-cli search +users -k <kw>` | 按关键词找用户、确认 login | 结果含 `login`、`name` |
| `gitlink-cli dataset +list --ids <projectIds>` | 查询科研数据集（论文/许可证） | `project_datasets[].{title,description,paper_content,license}` |

---

## 等价 Raw API（当 `profile` 命令不可用时）

```bash
gitlink-cli api GET /users/<login>/statistics/develop
gitlink-cli api GET /users/<login>/statistics/major
gitlink-cli api GET /users/<login>/statistics/role
gitlink-cli api GET /users/<login>/statistics/activity
gitlink-cli api GET /users/<login>/headmaps
```
