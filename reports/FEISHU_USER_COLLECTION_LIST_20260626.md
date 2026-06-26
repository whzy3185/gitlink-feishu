# 飞书 / GitLink 本地验证信息收集清单

Date: 2026-06-26

用途：这份清单只说明需要从飞书和 GitLink 页面收集哪些值。真实值不要写进本文件，也不要提交到仓库。真实值只放到本地忽略文件：

```text
.local/feishu-gitlink.env.ps1
```

## 当前状态

```text
自定义机器人 webhook：已配置并真实发送通过。
自建应用 app_id/app_secret：已配置并获取 tenant_access_token 通过。
DocX 目标：已配置并真实追加报告通过。
多维表格 Base：已配置；当前测试链接是同一个 Base 的同一张表的多个视图。
多维表格字段：已通过 OpenAPI 为测试表补齐。
Bitable search/create/update：已真实通过。
飞书任务创建：已真实通过；项目/分组归属尚未接入请求体。
GitLink 仓库：已使用 Gitlink/gitlink-cli 生成真实 workflow report。
i18n：feishu 命令 zh-CN 输出可用；仓库全局 i18n check 仍有既有 en-US.json 格式化问题。
截图：仍需从飞书 UI 手工截取。
```

## 1. 稳定层：飞书自定义机器人

这些值用于真实发送飞书群卡片。

| 需要收集 | 填入变量 | 是否敏感 | 获取位置 | 当前用途 |
| --- | --- | --- | --- | --- |
| 自定义机器人 Webhook URL | `FEISHU_WEBHOOK_URL` | 是 | 飞书群聊 -> 群设置 -> 机器人 -> 自定义机器人 | `+bot-test`, `+notify`, `+weekly-report`, `+owner-digest`, `+contributor-digest --send` |
| 自定义机器人签名密钥 | `FEISHU_WEBHOOK_SECRET` | 是 | 自定义机器人安全设置，若开启签名 | 同上 |

最小可验证：

```text
只要有 FEISHU_WEBHOOK_URL，就可以先测试稳定消息卡片。
如果机器人开启了签名，还必须填 FEISHU_WEBHOOK_SECRET。
```

## 2. 飞书开放平台自建应用

这些值用于 DocX、Wiki、多维表格、任务等实验性 OpenAPI 写入。

| 需要收集 | 填入变量 | 是否敏感 | 获取位置 | 当前用途 |
| --- | --- | --- | --- | --- |
| App ID | `FEISHU_APP_ID` | 是 | 飞书开放平台 -> 自建应用 -> 凭证与基础信息 | `+doc-export`, `+bitable-sync`, `+task-create --send` |
| App Secret | `FEISHU_APP_SECRET` | 是 | 同上 | 获取 `tenant_access_token` |

需要确认：

```text
1. 应用已经创建。
2. 应用在测试企业内可用。
3. 需要的 API 权限已经申请或开通。
4. 目标文档、知识库、多维表格或任务空间已经给应用必要权限。
```

## 3. DocX / Wiki 验证目标

这些值用于把 GitLink workflow report 写入飞书云文档或知识库。

| 需要收集 | 填入变量 | 是否敏感 | 获取位置 | 当前用途 |
| --- | --- | --- | --- | --- |
| Wiki 页面 URL | `FEISHU_WIKI_URL` | 可能敏感 | 目标飞书知识库页面地址栏 | `+doc-export --wiki-url ... --send` |
| Wiki node token | `FEISHU_WIKI_NODE_TOKEN` | 是 | 可从 Wiki URL 解析，或 OpenAPI 返回 | `+doc-export` |
| 文件夹 token | `FEISHU_FOLDER_TOKEN` | 是 | 飞书云空间文件夹 URL | 创建新 DocX |
| 已有 DocX document ID | `FEISHU_DOCUMENT_ID` | 是 | DocX URL 或 OpenAPI 返回 | 追加已有 DocX |

三选一即可开始：

```text
方案 A：提供 FEISHU_WIKI_URL，让命令解析 Wiki node。
方案 B：提供 FEISHU_FOLDER_TOKEN，让命令新建 DocX。
方案 C：提供 FEISHU_DOCUMENT_ID，追加已有 DocX。
```

必须人工处理：

```text
gitlink-cli 不会替你修改飞书文档权限。
你需要在飞书里给自建应用目标文档、知识库或文件夹的编辑权限。
```

## 4. 多维表格 Base / Bitable

这些值用于实验性真实同步记录。

| 需要收集 | 填入变量 | 是否敏感 | 获取位置 | 当前用途 |
| --- | --- | --- | --- | --- |
| Base app token | `FEISHU_BASE_APP_TOKEN` | 是 | 多维表格 URL 或开发者工具 API | `+bitable-sync --send` |
| reports 表 ID | `FEISHU_REPORT_TABLE_ID` | 是 | 多维表格表设置/API | 报告汇总行 |
| issues 表 ID | `FEISHU_ISSUE_TABLE_ID` | 是 | 同上 | Issue 汇总行 |
| prs 表 ID | `FEISHU_PR_TABLE_ID` | 是 | 同上 | PR 汇总行 |
| contributors 表 ID | `FEISHU_CONTRIBUTOR_TABLE_ID` | 是 | 同上 | 贡献者汇总行，可选 |
| tasks 表 ID | `FEISHU_TASK_TABLE_ID` | 是 | 同上 | 任务候选行，可选 |

当前测试说明：

```text
你提供的多维表格链接当前是同一个 Base 的同一张表，只是不同视图。
为了验证 OpenAPI 写入，我把 reports/issues/prs/contributors/tasks 都指向了同一张测试表，并补齐了需要字段。
这适合验证 search/create/update，但不是最终项目驾驶舱模型。
```

正式模型建议：

```text
1. 要么拆成 reports / issues / prs / contributors / tasks 多张表。
2. 要么改成更强的行级统一模型，支持看板、甘特图、日历、画册、表单和仪表盘。
3. 当前 CLI 不自动创建 Base、表、字段或视图。
4. Kanban / Gantt / Calendar / Gallery / Dashboard 视图先建议人工配置。
```

## 5. 飞书任务

这些值用于实验性创建飞书任务。

| 需要收集 | 填入变量 | 是否敏感 | 获取位置 | 当前用途 |
| --- | --- | --- | --- | --- |
| 任务项目 ID | `FEISHU_TASK_PROJECT_ID` | 是 | 飞书任务项目设置/API | 当前仅收集和脱敏输出 |
| 任务分组/section ID | `FEISHU_TASK_SECTION_ID` | 是 | 飞书任务项目设置/API | 当前仅收集和脱敏输出 |

当前限制：

```text
+task-create 真实请求目前只发送任务 summary 和 description。
project / section 设置字段还没有接入请求体。
已验证普通任务创建；后续再确认项目/分组字段。
```

## 6. GitLink 真实仓库数据

这些值用于生成真实 workflow report。

| 需要收集 | 填入变量 | 是否敏感 | 获取位置 | 当前用途 |
| --- | --- | --- | --- | --- |
| 仓库 owner | `GITLINK_OWNER` | 否 | GitLink 仓库 URL | `workflow +repo-report` |
| 仓库名 | `GITLINK_REPO` | 否 | GitLink 仓库 URL | `workflow +repo-report` |
| 测试 PR IDs | `GITLINK_TEST_PR_IDS` | 否 | 之前 3 个 PR URL/编号 | 烟测报告记录 |
| GitLink Token | `GITLINK_TOKEN` | 是 | GitLink 账号设置/API token | 若本地未登录且需要远程读取 |

示例，不要照抄：

```powershell
$env:GITLINK_OWNER="OWNER"
$env:GITLINK_REPO="REPO"
$env:GITLINK_TEST_PR_IDS="1,2,3"
$env:GITLINK_TOKEN="REDACTED"
```

## 7. 仍需人工完成

```text
1. 从飞书群里截取 bot card、weekly report、owner digest、contributor digest。
2. 从飞书多维表格里截取同步后的记录或视图。
3. 从飞书 DocX 里截取追加后的报告内容。
4. 从飞书任务里截取创建后的任务列表。
5. 截图前确认没有暴露 app secret、webhook、token、table id、open_id 或 union_id。
```

截图目标路径见：

```text
docs/PR_VISUAL_GUIDE.md
```

## 8. 安全提醒

```text
不要把 app secret、webhook、token、table id、wiki token、folder token 发到公开聊天或提交到仓库。
真实值只放在 .local/feishu-gitlink.env.ps1。
如果需要继续真实验证，优先复用本地 env 文件，不要把值写进 docs、reports、README。
```
