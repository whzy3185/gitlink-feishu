# Round 2 M1：PR 结果卡片实施记录

日期：2026-08-02
分支：`feat/round2-platform-v2-m1-pr-result-card`
基线：`5c4c1b01a0ebb66c55c55c6bbee88fcbca11bac9`

## 1. 阶段定位

M1 只完善 PR 查询结果的飞书展示层，不扩大 GitLink 写入范围。Owner 在群内查看 PR 后，可以直接看到当前 patchset、Review 状态、风险和下一步，不需要先打开 Doc。

本阶段仍保持：

```text
approve / reject / merge：不支持
行级评论：不支持
GitLink 默认写入：0
公开未绑定仓库：只读、无协作资源入口
```

## 2. 完成的结构

### 2.1 有界展示合同

`feishu.review-result/v2` 新增 `ReviewGatewayPullRequestView`，只保存卡片需要的确定性字段：

- 标题、作者、base/head 分支；
- GitLink PR URL、短 head、patchset；
- 文件、提交、增删行；
- GitLink 状态、Review 阶段和决定；
- Reviewer 摘要、线程数量；
- collection status、partial、风险；
- unknowns 和建议下一步。

不把原始 API payload、飞书消息 ID、用户 OpenID、Token、Cookie 或未截断错误放入展示合同。

### 2.2 固定卡片 schema

complete、partial、failed 共用同一构造入口：

```text
GitLink Review Context
-> ReviewGatewayPullRequestView
-> bounded result card
-> interactive reply
-> oversized 时 text fallback
```

complete 卡片显示完整事实；partial 卡片以黄色告警强调不能覆盖已有完整快照；failed 卡片以红色显示脱敏错误和重试建议。

### 2.3 公开仓库边界

公开、未绑定仓库也可以收到 PR 结果卡片，但卡片只包含 GitLink PR 跳转，不包含：

- Base、Doc、Task 协作资源；
- 负责人和截止时间；
- 领取、释放、设置截止日期；
- GitLink 写入动作。

### 2.4 安全降级

- 标题、分支、Reviewer、unknowns 和下一步均有 rune 级上限；
- Reviewer 最多展示 8 人，unknowns 最多展示 5 项；
- 卡片只显示 12 位 head；
- 飞书 OpenID 在交互卡片中使用 `<at id=OPEN_ID></at>` 由飞书解析为真实成员名称；纯文本、Doc 和日志只使用已解析名称或不可逆短哈希；
- 卡片 JSON 安全预算为 28,000 字节；超限或编码失败时退回 3,000 字符以内的文本；
- 公共卡片的 URL 固定从标准 GitLink owner/repo/PR 地址构造。

## 3. 测试证据

新增或更新的代码门禁包括：

- complete、partial、failed 三种卡片 golden；
- open、merged、closed 模板；
- Reviewer 和 unknowns 数量/长度上限；
- 完整 SHA、额外 Reviewer 和额外 unknown 不进入卡片；
- 公开仓库卡片不包含绑定协作字段；
- 超长卡片自动降级为文本；
- 原结果回复测试改为验证 interactive 卡片中的 `GitLink 写入：0`；
- 原有单卡更新、资源幂等、partial 快照保护和受控写回测试继续执行。

本地门禁命令：

```powershell
go clean -testcache
./scripts/verify-round2-p5.ps1
```

## 4. 尚未宣称完成的内容

- 尚未在真实飞书群对 #356 保存 M1 卡片截图；
- 尚未取得本分支精确 SHA 的 GitHub Actions 结果；
- Doc、Base、Task 的真实可点击链接需要 M3 在资源创建和回读后补入；
- 领取、释放、刷新和截止日期按钮属于 M2；
- 本阶段没有执行真实 GitLink POST。

因此当前状态应写为：

> M1 代码实现完成并通过本地门禁后，可推送获取精确 SHA CI；真实飞书卡片截图通过后才关闭 M1 平台验收。

## 5. 2026-08-03 真实群验证修正

真实查询 `Gitlink/forgeplus PR #356` 时，入站、异步查询和固定卡片更新均已成功，但新消息线程只显示“已接收”回执。原因是“一条 PR 一张固定卡片”的幂等策略更新了群内已有卡片，没有在本次消息下产生可见的最终回复。

本轮修正后：

- 首次查询仍创建一张固定结果卡片；
- 后续查询仍更新同一张固定卡片，避免卡片泛滥；
- 每次更新或命中未变化结果后，都会在当前源消息下重新输出完整交互卡片；
- 固定卡片资源映射继续保存真实卡片消息 ID，不会被通知消息覆盖；
- GitLink API 缺少 PR 标题时，以 head 分支名作为展示标题，再降级为 `PR #number`；
- 全链路仍为 GitLink GET-only，GitLink 写入保持 0。

对应回归测试覆盖固定卡片“创建一次、更新一次、当前消息可见通知一次”和标题降级行为。

仓库作用域规则同步收紧：所有 PR 级命令必须显式指定 `owner/repo`，绑定列表只定义授权范围，不再选择所谓默认仓库。未限定仓库的命令不会创建 Job，并会返回带正确示例的提示。

## 6. 2026-08-04 数据作用域修正

PR 快照现按 `installation + repository + PR` 保存；认领、截止日期和卡片/Base/Doc/Task 资源映射按 `installation + chat + repository + PR` 保存。同一 PR 在不同群中可以独立协作，不会互相继承负责人。旧映射只在 installation 可唯一确定时幂等迁移。

此前“已认领（飞书成员）”是写死的占位文案，现已删除。交互卡片使用飞书成员 mention 展示真实负责人；未解析身份不会伪造姓名。详细实现和后续任务见 [作用域隔离实施记录](./ROUND2_GITHUB_LARK_SCOPE_ISOLATION_IMPLEMENTATION_20260804.md)。
