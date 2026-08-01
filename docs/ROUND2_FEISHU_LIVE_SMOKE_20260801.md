# GitLink 飞书 Review Gateway 真实验收记录

日期：2026-08-01

分支：`feat/round2-feishu-platform-v2`

场景：飞书测试群 → Review Gateway → GitLink 公共 PR GET-only → 飞书原消息回复

## 1. 验收范围

本次只验证飞书消息收发、任务持久化、GitLink 只读查询和日志隐私边界：

- 仓库：`Gitlink/gitlink-cli`
- PR：`#431`
- 命令：`@gitlink 查看 PR #431`
- GitLink Review、评论、Reviewer 和合并写入：全部关闭
- Base、Doc、Task 资源同步：关闭
- GitLink 用户 Token：未配置

没有在本文保存飞书消息正文、原始用户 ID、原始群 ID、原始消息 ID、App Secret、Token、Cookie 或 Webhook Secret。

## 2. 真实链路结果

最终安全回归任务：`job-54c2cf80f3027985`

```text
飞书测试群真实用户 @机器人
→ WebSocket 收到新的消息事件
→ mention、群绑定和策略检查通过
→ SQLite 在 1 ms 内持久化 Job
→ 尽力回执发送成功
→ GitLink GET-only 读取 PR #431
→ collection_status=complete
→ 后台任务 completed
→ 最终交互卡片回复原消息成功
→ reply_status=sent
```

飞书端可见结果：

- 已接收只读 Review 请求；
- 阶段 `triaged`；
- 决策 `pending`；
- 协作状态 `unassigned`；
- 负责人未认领；
- 截止时间未设置；
- 两次回复均显示 `GitLink 写入：0`。

## 3. SQLite 证据

只读查询任务表得到：

```json
{
  "job_id": "job-54c2cf80f3027985",
  "status": "completed",
  "action": "read_review_context",
  "repository": "Gitlink/gitlink-cli",
  "pr_number": 431,
  "handler_latency_ms": 1,
  "attempt_count": 1,
  "reply_status": "sent",
  "reply_message_id": "present",
  "result_status": "completed",
  "collection_status": "complete",
  "read_only_gitlink": true,
  "mutates_gitlink": false,
  "gitlink_writes": 0,
  "head_sha": "5b40a9088726134829e68ca656b85cfcaac1a8a2",
  "source_fingerprint": "sha256:79505723ecdcf855a7a92a39fac881685b37ddeb55323aa86fc164551986414a"
}
```

远端回复 ID 仅记录为 `present`，不写入证据文档。

## 4. 日志隐私回归

第一轮真实 smoke 暴露出两个日志问题：

1. 实时 Gateway 曾输出包含原始事件、消息、群和用户 ID 的完整 Receipt/Result；
2. Channel SDK 的发送函数曾通过标准 logger 输出原始群、消息和回复 ID。

本轮修复后：

- 实时运行只输出哈希化 `feishu.review-observation/v1` 和非敏感生命周期事件；
- 离线 `--from-event` JSON 结果仍保留，便于本地合同调试；
- 飞书回复改用项目内 OpenAPI 发送器，不再经过会打印原始 ID 的 SDK 发送函数；
- bot identity 获取也不再经过会打印响应体的 SDK 启动路径。

对最终安全回归的 stdout 和 stderr 扫描结果：

```text
raw open_id matches:   0
raw chat_id matches:   0
raw message_id matches: 0
raw event_id matches:  0
```

日志仍保留 12 位不可逆摘要，用于串联 raw、normalized、identity、policy、gateway、result 和 reply 阶段。

## 5. 工程门禁

修复后执行：

```text
go test ./shortcuts/feishu ./shortcuts/wecom ./shortcuts/workflow
scripts/verify-round2-p5.ps1
```

结果：

- Feishu Review 协作测试通过；
- 企业微信适配器测试通过；
- Review Core 与 Agent 编排测试通过；
- 企业微信 Sidecar 合同测试 8/8 通过；
- 全仓构建通过；
- P2–P5 `go vet` 通过。

## 6. 当前结论与剩余边界

P2.1 的真实飞书只读主链路和日志隐私门禁已经通过。当前可以确认：

> 飞书端能够触发 GitLink PR 只读查询，团队能够在原消息下收到持久化结果，失败恢复数据保存在 SQLite，且 GitLink 写入为 0。

本次没有验证：

- 同一群绑定两个真实仓库后的歧义选择；
- Base、Doc、Task 的真实首次写入和幂等更新；
- 固定卡片在第二个 patchset 后 PATCH 原卡片；
- GitLink Webhook 的公网真实投递；
- 受控 common Review 写回；
- 企业微信真实群端到端；
- 外部 Agent Provider 真实调用。

这些项目继续按 `ROUND2_PLATFORM_EVIDENCE_CHECKLIST.md` 分项验收，不得用 CI 绿灯代替外部平台证据。
