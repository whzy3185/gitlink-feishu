# Feishu Review Gateway 部署与使用指南

Feishu Review Gateway 将企业自建 Feishu App 中的 Bot 作为 GitLink Pull Request 协作入口。Gateway 通过 Long Connection 接收群聊消息，读取 GitLink PR 与 Review 数据，并在飞书中维护负责人、审查截止时间和受控操作计划。

```text
飞书用户
  -> Feishu Bot
  -> Review Gateway
  -> GitLink Pull Request / Review API
```

GitLink 仍是仓库、PR、Review 和仓库权限的事实来源。领取 PR 或绑定飞书身份不会授予额外 GitLink 权限。

## 1. 能力边界

核心链路支持：

- 群聊中通过 `@Bot` 查询公开 GitLink PR；
- 展示 PR、版本、变更、Review、审查者和当前结论；
- 领取、取消领取和维护审查截止时间；
- 生成普通审查意见、批准、需要修改、拒绝并关闭、合并的 ActionPlan（受控操作计划）；
- 在持有对应 GitLink Credential 的本地 CLI 中最终确认 ActionPlan；
- 通过 GitLink GET 回读确认受控操作结果；
- 使用 SQLite 持久化事件、任务、回复、协作状态和操作状态；
- 提供健康检查、就绪检查、Metrics、备份和恢复能力。

Base、DocX、Wiki 和 Task 是可选协作资产投影，不是核心 Bot -> Gateway -> GitLink 链路的前置条件。

## 2. 部署前准备

### 2.1 GitLink 侧

由 Gateway 维护者准备：

1. 一个可正常访问目标仓库的 GitLink 账号；
2. 已安装或已从源码构建的 `gitlink-cli`；
3. 交互登录保存的 Credential，或环境变量 `GITLINK_TOKEN`；
4. 需要执行协作或受控操作时，每位操作人的飞书身份与 GitLink Login 映射；
5. 需要领取 PR 时，映射的 GitLink Login 必须是目标仓库的 Collaborator。

登录方式参见项目 [README](../README.md#configure--use)：

```bash
gitlink-cli auth login
```

非交互环境可以使用：

```bash
export GITLINK_TOKEN="<GITLINK_TOKEN>"
```

PowerShell：

```powershell
$env:GITLINK_TOKEN = "<GITLINK_TOKEN>"
```

`GITLINK_TOKEN` 是敏感信息。不要写入 `bindings.json`、命令历史、截图或 Git 历史。

### 2.2 Feishu 侧

由飞书管理员或应用管理员准备：

- 企业自建 App；
- 已启用的 Feishu Bot；
- App ID 与 App Secret；
- Long Connection 接收方式；
- 消息事件 `im.message.receive_v1`；
- 使用交互卡片时的回调 `card.action.trigger`；
- 应用可用范围和已发布的应用版本；
- 已加入目标群聊的 Bot。

### 2.3 Gateway 主机

从源码构建需要 Go 1.26 或更高版本。运行主机还需要可访问 GitLink 与 Feishu Open Platform 的网络、可写的 SQLite 数据目录，以及长期运行时使用的受限服务账号。

Gateway 主动建立 Feishu WebSocket 长连接。基础模式不需要公网 IP、公网域名或 Feishu HTTP Callback URL，但运行主机必须能够访问公网。

## 3. 配置 Feishu Bot

以下步骤以飞书开放平台当前的企业自建 App 界面为准。界面名称调整时，应选择语义相同的配置项。

### 步骤 1：创建或进入企业自建 App

- 操作者：飞书管理员或应用管理员；
- 位置：飞书开放平台开发者后台；
- 操作：创建企业自建 App，或进入现有 App；
- 成功状态：可以看到 App ID，并可管理凭证、能力、权限和事件。

### 步骤 2：启用 Bot

- 操作者：应用管理员；
- 位置：应用能力 -> Bot；
- 操作：启用 Bot，设置企业成员可识别的名称和头像；
- 成功状态：Bot 能力显示为已启用。

文档示例使用 `@gitlink`，实际使用时应替换为部署者设置的 Bot 名称。

### 步骤 3：开通核心权限

| 分类 | Permission | 用途 | 缺失影响 |
| --- | --- | --- | --- |
| Required | `im:message.group_at_msg:readonly` | 接收群聊中用户 @Bot 的消息 | Gateway 收不到核心命令 |
| Required | `im:message:send_as_bot` | 发送、回复文本和 Review Card | Bot 无法回复查询或操作结果 |
| Recommended | `contact:contact.base:readonly` | 配合读取企业成员基础信息 | 可能无法显示企业成员名称 |
| Recommended | `contact:user.base:readonly` | 读取成员名称 | 失败时回退显示 GitLink Login，核心流程仍可运行 |
| Optional | 群列表读取 API 对应权限 | 仅供 `--discover-chats` 查找 Bot 可见群 | 不影响已配置群的运行 |
| Optional | Base / DocX / Wiki / Task 对应权限 | 仅供协作资产投影 | 不影响核心 Review 链路 |

企业成员名称读取使用：

```text
GET /open-apis/contact/v3/users/<OPEN_ID>?user_id_type=open_id
```

SDK 还通过 `GET /open-apis/bot/v3/info` 获取 Bot 自身身份，避免处理 Bot 自己发送的消息。该 endpoint 的飞书文档标记为无权限要求，因此本指南不把 `application:bot.basic_info:read` 列入 Required。若后台或运行日志明确返回权限错误，再按当前后台提示补充权限。

### 步骤 4：配置事件与卡片回调

进入“事件与回调”：

1. 订阅 `im.message.receive_v1`，用于接收群内 @Bot 消息；
2. 使用 Review Card 或 Controlled Actions 时，配置 `card.action.trigger`；
3. 不要订阅本项目没有 Handler 的 `drive.file.bitable_record_changed_v1`。

`im.message.receive_v1` 是事件；`card.action.trigger` 是交互卡片回调。它们不是 Permission 名称。

### 步骤 5：选择 Long Connection

- 位置：事件与回调 -> 订阅方式；
- 操作：选择“使用长连接接收事件 / 回调”；
- 成功状态：后台显示长连接为接收方式。

Long Connection 由 Gateway 中的 `larkws.NewClient` 主动建立，不需要配置 HTTP Callback URL。

### 步骤 6：设置范围并发布

1. 将目标用户或部门加入应用可用范围；
2. 按飞书当前发布流程创建并发布应用版本；
3. 确认新权限、事件与回调配置已经生效；
4. 将 Bot 加入目标群聊。

未发布的新配置不会自动应用到正在使用的 Bot。

## 4. Credential 与运行参数

| 参数 | 用途 | 必须 | 来源 | 敏感 |
| --- | --- | --- | --- | --- |
| `<FEISHU_APP_ID>` | 标识企业自建 App | 是 | 飞书开放平台“凭证与基础信息” | 是 |
| `<FEISHU_APP_SECRET>` | 换取 tenant access token、建立 SDK 连接 | 是 | 飞书开放平台“凭证与基础信息” | 是 |
| `<GITLINK_TOKEN>` | 调用 GitLink API | 二选一 | GitLink 私人 Token，或交互登录存储 | 是 |
| `<GITLINK_LOGIN>` | 绑定飞书用户与 GitLink 身份 | 协作/受控操作需要 | GitLink 用户名 | 否 |
| `<GITLINK_OWNER>` | 仓库拥有者 | 是 | GitLink 仓库路径 | 否 |
| `<GITLINK_REPO>` | 仓库名 | 是 | GitLink 仓库路径 | 否 |
| `<DATA_DIR>` | SQLite、锁和运行数据目录 | 是 | 部署者选择 | 可能包含敏感协作数据 |

Gateway 优先读取 `--app-id` / `--app-secret`，未提供时读取 `FEISHU_APP_ID` / `FEISHU_APP_SECRET`。推荐使用受限的环境文件或服务环境，不要把 Secret 放进命令行。

```bash
export FEISHU_APP_ID="<FEISHU_APP_ID>"
export FEISHU_APP_SECRET="<FEISHU_APP_SECRET>"
export GITLINK_TOKEN="<GITLINK_TOKEN>"
```

PowerShell：

```powershell
$env:FEISHU_APP_ID = "<FEISHU_APP_ID>"
$env:FEISHU_APP_SECRET = "<FEISHU_APP_SECRET>"
$env:GITLINK_TOKEN = "<GITLINK_TOKEN>"
```

GitLink Credential 的读取优先级和存储方式参见 [README](../README.md#configure--use)。

## 5. Repository Binding 与身份绑定

Gateway 使用一个 JSON 文件配置群聊与仓库，并维护飞书用户到 GitLink Login 的映射。

```json
{
  "schema_version": "feishu.review-bindings/v1",
  "bindings": [
    {
      "chat_id": "<FEISHU_CHAT_ID>",
      "repository": "<GITLINK_OWNER>/<GITLINK_REPO>",
      "enabled": true,
      "admin_user_ids": ["<FEISHU_ADMIN_OPEN_ID>"],
      "allowed_user_ids": [],
      "deadline_hours": 24,
      "binding_revision": "v1"
    }
  ],
  "identity_bindings": [
    {
      "feishu_user_id": "<FEISHU_USER_OPEN_ID>",
      "gitlink_login": "<GITLINK_LOGIN>",
      "verification_method": "manual",
      "enabled": true
    }
  ]
}
```

`chat_id` 决定允许 Gateway 处理消息的群，`repository` 是该群的默认仓库，格式必须为 `owner/repo`。`allowed_user_ids` 为空时不额外限制普通读取；`identity_bindings` 是飞书用户到 GitLink Login 的显式映射。

Long Connection 模式至少需要一个启用的群绑定。群进入 SDK 白名单后，用户在命令中显式写出 `owner/repo`，可以查询不同于默认仓库的公开 PR：

```text
@<BOT_NAME> 查看 Gitlink/forgeplus PR #356
```

省略仓库名的 `查看 PR #<编号>` 使用群的默认仓库。推荐始终写完整 `owner/repo`，避免歧义。

管理员可以用只读发现命令列出企业自建 App 当前可见的群：

```bash
gitlink-cli feishu +review-gateway --discover-chats
```

该命令读取 `FEISHU_APP_ID` 和 `FEISHU_APP_SECRET`，调用飞书群列表 API，不修改群配置。若飞书返回权限错误，请按开放平台对“获取群列表”API 的当前提示开通对应 Optional 权限；本指南不猜测未由平台确认的正式 scope 名称。

身份和权限必须分开理解：

- ReviewIdentityBinding 只表示飞书用户对应哪个 GitLink Login；
- Identity Binding 不等于 GitLink Repository Permission；
- Claim 不会授予 GitLink 权限；
- Gateway Admin 不等于 GitLink Repository Admin；
- 领取 PR 时，Gateway 还会读取 GitLink Collaborator 列表；
- 最终受控操作由 GitLink 服务端基于实际 Credential 再次检查权限。

没有身份绑定时，公开 PR 查询仍可使用；领取、截止时间和受控操作会被拒绝。

## 6. 构建 Review Gateway

从源码构建：

```bash
git clone https://www.gitlink.org.cn/Gitlink/gitlink-cli.git
cd gitlink-cli
go build -o bin/gitlink-cli .
```

Windows PowerShell：

```powershell
git clone https://www.gitlink.org.cn/Gitlink/gitlink-cli.git
Set-Location gitlink-cli
go build -o bin\gitlink-cli.exe .
```

也可以按 [README 安装说明](../README.md#installation--quick-start) 安装已经包含 Review Gateway 的发行版本。

确认命令已注册：

```bash
gitlink-cli feishu +review-gateway --help
```

## 7. 第一次启动

### 开发与快速验证

Linux / macOS：

```bash
gitlink-cli feishu +review-gateway \
  --listen \
  --bindings "<DATA_DIR>/bindings.json" \
  --state-db "<DATA_DIR>/review.db"
```

Windows PowerShell：

```powershell
gitlink-cli.exe feishu +review-gateway `
  --listen `
  --bindings "<DATA_DIR>\bindings.json" `
  --state-db "<DATA_DIR>\review.db"
```

`--state-db` 决定数据目录。Gateway 会自动创建父目录、SQLite 文件、表和索引，并启用 WAL；不需要手工执行数据库初始化脚本。

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `--state-db` | `.local/review-gateway.db` | SQLite 状态库 |
| `--queue-size` | `128` | 内存任务队列上限 |
| `--stale-minutes` | `30` | 拒绝过旧消息的时间窗口 |
| `--job-timeout-seconds` | `60` | 单个 GitLink 任务超时 |
| `--handler-timeout-ms` | `2000` | 飞书入站处理总预算 |
| `--sqlite-timeout-ms` | `500` | 入站 SQLite 写入预算 |
| `--shutdown-timeout` | `30s` | 优雅关闭超时 |
| `--admin-listen` | `127.0.0.1:8787` | 健康、就绪、Metrics 和只读管理接口 |

`--admin-users` 可增加逗号分隔的全局 Gateway 管理员飞书用户 ID；普通部署优先在 Binding 中使用最小范围的 `admin_user_ids`。

正常启动后重点观察生命周期输出：

```text
type=ready
Feishu Channel SDK connected
```

以及：

```text
type=service_ready
Review gateway ready
```

Long Connection 建立后，SDK 日志会显示 WebSocket 已连接。若只有 `service_ready` 而 `/readyz` 仍返回 503，应继续检查 Feishu Channel、SQLite、配置和实例锁状态。

### 只读离线模式

用于本地 Fixture 或排障，不连接 Long Connection：

```bash
gitlink-cli feishu +review-gateway \
  --from-event <NORMALIZED_EVENT_JSON> \
  --bindings <BINDINGS_JSON>
```

增加 `--execute-read-only` 后执行被接受的 GitLink GET 任务：

```bash
gitlink-cli feishu +review-gateway \
  --from-event <NORMALIZED_EVENT_JSON> \
  --bindings <BINDINGS_JSON> \
  --execute-read-only
```

## 8. 第一次连接飞书

1. 确认应用版本已发布；
2. 确认 Bot 已加入 `bindings.json` 中的目标群；
3. 保持 Gateway 进程运行；
4. 在群里发送：

```text
@<BOT_NAME> 帮助
```

5. 收到“GitLink PR Review 助手”命令列表后，再发送：

```text
@<BOT_NAME> 查看 <GITLINK_OWNER>/<GITLINK_REPO> PR #<PR_NUMBER>
```

收到 PR 标题、状态、分支、当前版本、变更、Review、负责人和 GitLink 链接，表示消息接收、命令解析、GitLink GET 和飞书回复链路均已工作。

## 9. 常用命令

命令中的 `@<BOT_NAME>` 应替换为实际 Bot。

### 查询与协作

```text
@<BOT_NAME> 帮助
@<BOT_NAME> 查看 <拥有者>/<仓库> PR #<编号>
@<BOT_NAME> 领取 <拥有者>/<仓库> PR #<编号>
@<BOT_NAME> 取消领取 <拥有者>/<仓库> PR #<编号>
@<BOT_NAME> 设置 <拥有者>/<仓库> PR #<编号> 审查截止 <YYYY-MM-DD>
@<BOT_NAME> 清除 <拥有者>/<仓库> PR #<编号> 审查截止
```

领取要求当前飞书用户存在 Identity Binding，且映射的 GitLink Login 是仓库 Collaborator。同一 PR 同一时间只有一个负责人；重复领取自己的 PR 是幂等操作。

Reviewer 来自 GitLink 的真实 Review 记录，负责人来自飞书协作状态，两者是不同概念。

### 受控操作

```text
@<BOT_NAME> 提交审查意见 <拥有者>/<仓库> PR #<编号> <意见>
@<BOT_NAME> 批准 <拥有者>/<仓库> PR #<编号> <说明>
@<BOT_NAME> 需要修改 <拥有者>/<仓库> PR #<编号> <原因>
@<BOT_NAME> 拒绝并关闭 <拥有者>/<仓库> PR #<编号> <原因>
@<BOT_NAME> 合并 <拥有者>/<仓库> PR #<编号>
```

`需要修改` 只提交 Review 结论，PR 保持开放；`拒绝并关闭` 会关闭 PR。以上命令在飞书中只生成 ActionPlan，不会由普通卡片点击直接完成 GitLink 写入。

兼容命令 `review`、`approve`、`reject`、`refuse`、`merge` 仍可解析，但新文档和日常使用推荐中文正式命令。

## 10. Controlled Actions

ActionPlan 把飞书中的操作意图冻结为可追踪的执行计划：

```text
飞书确定“做什么”
  -> ActionPlan 冻结仓库、PR、操作人、GitLink Login、Head 和内容指纹
  -> 本地 CLI 确认“由谁执行”
  -> GitLink 执行并通过 GET 回读确认结果
```

在能够访问 Gateway 同一 SQLite 状态库的主机上，以绑定的 GitLink 身份登录，然后执行：

```bash
gitlink-cli feishu +review-confirm-local \
  --plan-id <PLAN_ID> \
  --state-db <DATA_DIR>/review.db
```

先做零写入预览：

```bash
gitlink-cli feishu +review-confirm-local \
  --plan-id <PLAN_ID> \
  --state-db <DATA_DIR>/review.db \
  --dry-run
```

CLI 会调用 `/users/me` 核对当前 Credential 的 Login，并重新检查 Expected Head 与数据指纹。交互确认需要输入 `yes`；受控自动化环境可以显式增加 `--yes`，但不应绕过 Credential 和状态检查。

| 状态 | 用户应执行的操作 |
| --- | --- |
| 待本地确认 | 检查操作内容，在正确 GitLink 身份下执行本地确认 |
| stale | PR Head 或数据指纹已经变化；刷新 PR，重新生成计划 |
| Unknown / 结果待核对 | 远程结果无法确定；先到 GitLink 核对，系统不会盲目重试 |
| completed | GitLink 写入完成，且 GET 回读已经确认 |
| cancelled | 计划已取消，没有执行 GitLink 写入 |

同一计划具备租约、幂等和终态保护，不能重复执行。GitLink 仍负责最终权限判断。

## 11. 运行状态检查

Gateway 默认只在 loopback 地址 `127.0.0.1:8787` 提供运维接口。

### Health 与 Readiness

```bash
curl -fsS http://127.0.0.1:8787/healthz
curl -fsS http://127.0.0.1:8787/readyz
```

`/healthz` 返回 HTTP 200 和 `status: ok` 表示进程存活。`/readyz` 返回 HTTP 200 和 `status: ready` 表示配置、SQLite、Schema Migration、实例锁、管理 HTTP 和必要组件均已就绪；HTTP 503 时根据响应中的 `components` 定位问题。

### Metrics

先配置独立管理 Token，并在启动参数中引用：

```bash
export FEISHU_REVIEW_ADMIN_TOKEN="<RANDOM_ADMIN_TOKEN>"
```

```text
--admin-token-ref env:FEISHU_REVIEW_ADMIN_TOKEN
```

```bash
curl -fsS \
  -H "Authorization: Bearer <RANDOM_ADMIN_TOKEN>" \
  http://127.0.0.1:8787/metrics
```

`/metrics` 输出 Prometheus 文本；未配置或未提供正确 Bearer Token 时返回 401。`/admin/v1/*` 同样需要 Token，且只允许 GET。当前实现只接受 `--admin-token-ref env:VARIABLE`，不要使用 `file:` 引用。

## 12. 长期运行

### 12.1 单实例、关闭与维护

同一 App ID 和状态库只允许一个 Gateway 实例持有锁。收到中断信号后，服务停止接收新任务，等待组件退出，再关闭 HTTP 和 SQLite。前台运行时使用 `Ctrl+C`；systemd 示例使用 `SIGINT`。

一次只执行一种 SQLite 维护操作：

```bash
gitlink-cli feishu +review-maintenance --state-db <DATA_DIR>/review.db --backup-to <BACKUP_DIR>/review.db
gitlink-cli feishu +review-maintenance --state-db <DATA_DIR>/restored.db --restore-from <BACKUP_DIR>/review.db
gitlink-cli feishu +review-maintenance --state-db <DATA_DIR>/review.db --retain-days 30
gitlink-cli feishu +review-maintenance --state-db <DATA_DIR>/review.db --wal-checkpoint
```

备份使用 SQLite 一致性快照且不覆盖已有目标。恢复要求目标数据库不存在，并执行完整性检查。Retention 只清理过期去重数据和终态 Event/Job，不删除 active 或 unknown Operation。

### 12.2 Linux systemd

仓库提供：

```text
deploy/systemd/gitlink-feishu-review.service
deploy/systemd/review-gateway.env.example
```

推荐布局：

```text
/usr/local/bin/gitlink-cli
/etc/gitlink-feishu-review/bindings.json
/etc/gitlink-feishu-review/review-gateway.env
/var/lib/gitlink-feishu-review/review.db
/var/backups/gitlink-feishu-review/
```

复制示例后，使用受限服务账号、`0600` 环境文件权限和真实路径进行调整：

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now gitlink-feishu-review
sudo systemctl status gitlink-feishu-review
```

示例 unit 使用 `EnvironmentFile` 注入 Secret，并限制可写目录。

### 12.3 Windows Service

仓库提供 WinSW 包装配置，不是原生 PowerShell 安装脚本：

```text
deploy/windows/gitlink-feishu-review.xml.example
deploy/windows/review-gateway.env.example
```

下载 WinSW 后，将其可执行文件、XML、`gitlink-cli.exe` 和 `bindings.json` 放入服务目录；WinSW 可执行文件与 XML 使用相同基础文件名。为服务账号配置 `FEISHU_APP_ID`、`FEISHU_APP_SECRET`、`GITLINK_TOKEN` 和 `FEISHU_REVIEW_ADMIN_TOKEN`，再按当前 WinSW 文档使用 `install`、`start`、`status` 和 `stop`。

`review-gateway.env.example` 是变量清单模板，当前 XML 不会自动读取该文件。应使用 WinSW 或 Windows 服务账号支持的环境注入方式，不要把 Secret 直接写入 XML。

### 12.4 Nginx / Caddy

`deploy/nginx/` 与 `deploy/caddy/` 提供可选反向代理示例，只代理 `/healthz` 和 `/readyz`。Long Connection 本身不需要反向代理。不要把 `/metrics` 或 `/admin/v1/*` 暴露到公网。

## 13. 可选协作资产投影

不配置以下参数时，核心查询、协作和受控操作不会调用 Base、DocX、Wiki 或 Task API。

| 能力 | 启用参数 | 当前行为 |
| --- | --- | --- |
| Base / Bitable | `--review-base-app-token` 与 `--review-base-table-id` | 按稳定 `unique_key` 创建或更新 PR 最新事实 |
| DocX | `--review-document-id` 或 `--review-document-folder-token` | 向现有文档追加快照，或按 PR 创建文档 |
| Wiki | `--review-wiki-node-token` | 解析 Wiki Node 并向对应 DocX 对象追加快照 |
| Task | `--review-task-enabled` | 为活跃且已领取的 Review 创建/更新任务，PR 关闭或合并后完成已有任务 |

```bash
gitlink-cli feishu +review-gateway \
  --listen \
  --bindings <DATA_DIR>/bindings.json \
  --state-db <DATA_DIR>/review.db \
  --review-base-app-token <FEISHU_BASE_APP_TOKEN> \
  --review-base-table-id <FEISHU_BASE_TABLE_ID> \
  --review-document-id <FEISHU_DOCUMENT_ID> \
  --review-task-enabled
```

这些资源需要对应 Feishu OpenAPI 权限和目标资源级访问权限。正式 Permission 名称应按部署时飞书开放平台对 Base 记录、DocX Block、Wiki Node 和 Task API 的提示选择，不应把它们加入核心最小权限。

Gateway 使用 SQLite 保存远程资源 ID、内容指纹和同步状态。相同指纹不会重复写入；数据不完整的 GitLink 快照不会覆盖已有完整资源；远程结果不确定时会停止自动重试，等待人工核对。

## 14. Troubleshooting

| 现象 | 常见原因 | 处理方式 |
| --- | --- | --- |
| Bot 没有回复 | App 版本未发布、Bot 未入群、可用范围不含用户、群未绑定 | 发布应用版本，将 Bot 加入目标群，把群 ID 加入启用的 `bindings` |
| Long Connection 未建立 | App ID/Secret 错误、未选择 Long Connection、主机无法访问公网 | 核对 Credential、订阅方式和出站网络，查看 `channel_error` |
| 能收到消息但不能回复 | 缺少 `im:message:send_as_bot` | 开通权限并发布新版本 |
| @Bot 消息未进入 Gateway | 缺少事件或消息只读权限 | 核对 `im.message.receive_v1`、核心权限和 Long Connection |
| 卡片按钮没有反应 | 未配置 `card.action.trigger` | 在事件与回调中配置卡片回调并发布 |
| 提示群未绑定 | `chat_id` 不在启用的 Binding 中 | 更新 `bindings.json` 后重启 Gateway |
| 提示飞书用户未绑定 GitLink 身份 | 缺少 `identity_bindings` | 添加 `feishu_user_id` 到 `gitlink_login` 的映射 |
| 领取被拒绝 | 映射 Login 不是仓库 Collaborator，或 PR 已由他人负责 | 核对 GitLink Collaborator 和当前负责人 |
| 最终确认身份不匹配 | 当前 Credential 的 `/users/me` 与计划中的 Login 不同 | 切换到正确 GitLink 账号，不要复用他人 Token |
| PR 找不到 | `owner/repo` 或 PR 编号错误，仓库不可访问，API 暂时异常 | 打开 GitLink 核对，并检查 Gateway 日志中的脱敏错误 |
| 企业成员名称显示为 GitLink Login | Contact 权限或成员读取失败 | 开通 Recommended Contact 权限；该降级不影响核心 Review |
| Bot identity 不可用 | `GET /open-apis/bot/v3/info` 失败 | 检查 App Credential 和飞书返回；有缓存时 SDK 使用旧值，无缓存时当前消息不会处理 |
| SQLite locked 或写入超时 | 多实例、目录权限错误、磁盘忙 | 保证单实例，检查目录权限和 `/readyz`，不要直接删除 WAL/锁文件 |
| `/readyz` 返回 503 | 配置、SQLite、Migration、实例锁或必要组件未就绪 | 根据响应中的 `components` 定位问题 |
| ActionPlan stale | PR Head 或数据指纹已变化 | 刷新 PR 后重新生成计划，本次不会写入 GitLink |
| ActionPlan 进入 Unknown | 写入结果无法可靠确认 | 先在 GitLink 核对状态，再进行 Reconciliation；不要重复确认 |

## 15. 安全说明

- 不要提交 Feishu App Secret、GitLink Token、tenant token、Cookie 或 Authorization Header；
- 不要在截图中展示 Credential、完整个人标识或无关账号信息；
- `bindings.json` 中的群 ID 和用户 Open ID 应按内部配置管理；
- Identity Binding、Claim 和 Gateway Admin 都不会授予 GitLink 权限；
- Controlled Action 最终仍受当前 GitLink Credential 和 GitLink 服务端权限限制；
- `/metrics` 与 `/admin/v1/*` 只应在 loopback 或受信任管理网络使用；
- SQLite 备份可能包含协作状态、用户映射和操作记录，应加密保存并限制访问；
- 不要在多个实例之间共享同一个可写 SQLite 状态库。

## 16. 相关文档

- [安装、认证和通用命令](../README.md)
- [Feishu 通用集成](./feishu-integration.md)
- [Feishu 安全边界](./feishu-security.md)
- [Feishu OpenAPI 清单](./FEISHU_OPENAPI_INVENTORY.md)
