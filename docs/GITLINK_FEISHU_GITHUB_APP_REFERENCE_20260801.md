# GitLink 飞书接入的 GitHub App 参考模型

日期：2026-08-01

适用分支：`feat/round2-feishu-platform-v2`

## 1. 结论

GitLink 飞书机器人应参考 GitHub App，而不是参考“一个机器人保存一个全局个人 Token”的旧式 OAuth Bot：

```text
GitLink App 定义
  -> 声明能力和最小权限
Installation
  -> 仓库拥有者批准 App 可访问的仓库集合
User authorization / Identity binding
  -> 用户批准机器人代表自己执行动作
Feishu chat binding
  -> 决定某个群默认操作哪个 Installation 和仓库
ActionPlan confirmation
  -> 在写入前绑定 PR patchset、用户、动作和审计记录
```

公共仓库读取与 Installation 授权必须是两条不同的数据通道。公共发现只能执行无凭据 GET；私有读取和任何写操作必须使用 Installation 范围内的凭据。

## 2. GitHub 官方依据

GitHub App 安装与用户授权是两个不同动作：

- Installation 授予组织和仓库资源权限，并选择 App 能访问的仓库；
- User authorization 允许 App 验证用户身份并代表该用户操作；
- 二者可以分别存在，不能互相替代。

官方文档：

- [Authorizing GitHub Apps](https://docs.github.com/en/apps/using-github-apps/authorizing-github-apps)
- [Installing a GitHub App from a third party](https://docs.github.com/en/apps/using-github-apps/installing-a-github-app-from-a-third-party)

GitHub App 默认没有权限，应按功能申请最小权限。用户访问令牌的实际能力同时受用户权限和 App 权限约束；用户和 App 任一方没有权限，动作就不能执行。

- [Choosing permissions for a GitHub App](https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/choosing-permissions-for-a-github-app)
- [Generating a user access token](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app)

Installation access token 可以进一步限定仓库和权限，不能超过 Installation 已获授权范围，并且会过期。GitLink 当前未提供等价的 GitHub App token 生命周期，所以本项目只能先用 `credential_ref` 模拟安全边界，不能宣称已经具备完整 GitHub App 基础设施。

- [Generating an installation access token](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-an-installation-access-token-for-a-github-app)

## 3. 映射到当前代码

| GitHub App 概念 | 当前 GitLink 飞书实现 | 边界 |
|---|---|---|
| App permissions | 固定 intent allowlist | 未声明的动作不可从聊天触发 |
| Installation | `GitLinkInstallation` | host、owner、仓库范围、运行模式、凭据引用 |
| Selected repositories | `allowed_repositories` | 私有读取与全部写入的硬边界 |
| User authorization | `ReviewIdentityBinding` | 飞书 `open_id` 映射 GitLink login |
| Installation credential | `credential_ref=env:...` | 只保存引用，不保存 Token 值 |
| Chat installation | `ReviewChatBinding` | 群到 Installation、默认仓库与仓库列表 |
| Webhook delivery | SQLite dedupe + Job | 至少一次投递、异步执行、可恢复 |
| User-to-server write | P3 ActionPlan | 用户、patchset、fingerprint、二次确认、对账 |

## 4. 公共仓库发现策略

新增策略字段：

```json
{
  "installation": {
    "allow_public_read": true
  },
  "chat_binding": {
    "allow_public_read": true
  }
}
```

只有 Installation 和群绑定同时开启时，才允许查询未在绑定列表中的显式仓库：

```text
@gitlink 查看 owner/repository PR #123
```

执行合同：

1. 只放行 `read_review_context`；
2. 强制剥离 Installation credential 和全局 credential；
3. GitLink 请求仅使用 GET；
4. 返回后必须确认仓库信息中 `is_public=true`；
5. 无法确认公开状态时 fail closed；
6. 只回复消息，不创建 Base、Doc、Task 或本地协作 WorkItem；
7. 不能生成 Agent 审查、认领任务、设置截止时间或准备 Review 写回；
8. 需要高级协作时，管理员先把仓库加入 Installation 和群绑定。

因此公共发现并不是仓库通配符，也不会扩大 `allowed_repositories`。

## 5. 私有读取和 Review 写回

私有读取至少需要：

```text
repository ∈ Installation.allowed_repositories
repository ∈ ChatBinding.repositories
Installation enabled
credential_ref 可解析
```

Review 写回还需要：

```text
Installation.operation_mode = write
启动参数显式允许 Review write
飞书用户存在启用的 IdentityBinding
GitLink 用户本身拥有目标仓库 Review 权限
ActionPlan 未过期且仍为 pre_write
PR head SHA 与 source fingerprint 未变化
用户二次确认同一 plan_id
写后保存 Review ID 或进入 reconciliation
```

contributor 身份不自动等价于 Review 权限。必须通过真实测试 PR 和 GitLink API 返回验证。

## 6. 孔明职配验收路径

目标仓库：

```text
puygob236/KongMing-Job-Matching-Agent
```

当前开放 PR 数为 0。推荐由协作者创建一条只修改测试文档的 PR，然后依次执行：

1. 公共无凭据 `查看 owner/repo PR #编号`；
2. 将仓库加入 Installation 和群绑定；
3. 只读生成 Review Context 与 ActionPlan preview；
4. 完成飞书账号和 GitLink login 的身份绑定；
5. 单独配置测试凭据并验证当前用户；
6. 开启一次性 Review 写入验收；
7. 提交 `common` Review，不测试 merge、approve、reject；
8. 保存 before、after、head SHA、fingerprint、Review ID 和对账记录；
9. 立即关闭写入开关并移除测试凭据。

在 GitLink 平台提供 App Installation 和短期 token 之前，生产环境不应长期保存个人 Token。
