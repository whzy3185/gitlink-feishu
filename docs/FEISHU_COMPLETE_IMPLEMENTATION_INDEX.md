# GitLink 飞书完整实现与资料索引

日期：2026-07-31

汇总分支：

```text
feat/feishu-complete-review-integration
```

仓库：

```text
https://github.com/whzy3185/gitlink-feishu
```

## 1. 汇总目标

本分支把当前机器上与 GitLink 飞书、复赛 PR Review 协作、企业微信对照设计相关的
代码和文档集中在一个可浏览分支中，方便：

- 飞书 Agent 查看完整实现；
- ChatGPT 网页端读取真实代码；
- 比赛复赛评审查看阶段演进；
- 开发者继续排障和实现 P2.2；
- 拥有者核对边界、安全与测试证据。

这是 **内容汇总分支**。它以当前 P2.1 顶端为基线，不通过高冲突 octopus merge
强行拼接所有历史分支；旧分支提交仍保留在 GitHub，关键提交在本文列出。

## 2. 安全排除

本分支明确不包含：

```text
.local/
App Secret
tenant_access_token
user_access_token
refresh_token
GitLink Token
Webhook Secret
真实 chat_id
真实 open_id
SQLite 运行数据库
WAL / SHM
运行日志
未脱敏的消息历史
```

测试 fixture 中只使用示例 ID。

## 3. 实现阶段

| 阶段 | 能力 | 关键提交 |
|---|---|---|
| 飞书导出基础 | 消息、卡片、Base、Doc、Task preview 与导出 | 主线 Feishu 合入及历史 `feat/feishu-export-clean` |
| P0 | 恢复根命令与只读 Review 协作输入 | `5a3d4b07` |
| P1 | Review Context / WorkItem 稳定模型 | `9c22c180` |
| P1.1 | Reviewer 聚合、线程正文、partial/error、真实 GET smoke | `2ae67e56` |
| P2.0 | Channel SDK、Gateway、SQLite、GET-only Executor | `4e86154a` |
| P2 预备 | 错误脱敏和 Reviewer 保守排序 | `1995eafa` |
| P2.1 | 可靠 Job、真实回复、handler 门禁、结果持久化 | `fa3c1fb5` |
| 能力说明 | 机器人完整能力和飞书平台映射 | `61c44fdb` |
| 排障交接 | 入站无响应证据与飞书 Agent 问题 | `63017d24` |

原本地 P0 提交：

```text
9303afb62c32a18420e61aa0745c61b105a772a4
```

在 GitHub 迁移/rebase 后对应内容提交为：

```text
5a3d4b07
```

## 4. 飞书代码

### 4.1 原有飞书协作导出

目录：

```text
shortcuts/feishu/
```

包括：

- `bitable.go`：多维表格能力；
- `bitable_sync.go`：记录同步；
- `card.go`：卡片结构；
- `client.go`：OpenAPI 客户端；
- `diagnostics.go`：环境和应用检查；
- `digest.go`：摘要；
- `doc_export.go`：文档导出；
- `feishu.go`：命令注册；
- `openapi.go`：OpenAPI 路由；
- `render.go`：输出渲染；
- `task.go`：任务 preview；
- `workflow_input.go`：工作流输入；
- `sign.go`：自定义机器人签名；
- `l10n.go`：本地化；
- 对应单元测试。

### 4.2 P2 Review Gateway

核心文件：

```text
shortcuts/feishu/review_gateway_command.go
shortcuts/feishu/review_gateway_models.go
shortcuts/feishu/review_gateway_store.go
shortcuts/feishu/review_gateway_executor.go
shortcuts/feishu/review_gateway_reply.go
shortcuts/feishu/review_gateway_test.go
```

fixture：

```text
shortcuts/feishu/testdata/review_gateway_bindings.json
shortcuts/feishu/testdata/review_gateway_event.json
```

实现链路：

```text
Feishu WebSocket
-> P2MessageReceiveV1
-> normalize
-> bot mention
-> PolicyGate
-> chat/user binding
-> SQLite atomic enqueue
-> durable worker
-> GitLink GET-only
-> result_json
-> reply original message
```

## 5. GitLink Review Core

P1/P1.1 相关代码：

```text
shortcuts/workflow/review_context.go
shortcuts/workflow/review_context_models.go
shortcuts/workflow/review_context_redaction.go
shortcuts/workflow/review_queue.go
shortcuts/workflow/triage_fetch.go
shortcuts/workflow/api_types.go
shortcuts/workflow/workflow.go
```

测试与 golden fixture：

```text
shortcuts/workflow/review_context_models_test.go
shortcuts/workflow/review_context_test.go
shortcuts/workflow/review_queue_test.go
shortcuts/workflow/testdata/review_context_p1_fixture.json
shortcuts/workflow/testdata/review_context_p1.golden.md
shortcuts/workflow/testdata/review_context_p11_observed_fixture.json
shortcuts/workflow/testdata/review_context_p11_observed.golden.md
```

主要输出：

- `review.context/v1`；
- `review.work-item/v1`；
- Reviewer summaries；
- Review freshness；
- 线程正文；
- patchset；
- collection status；
- structured fetch errors；
- partial snapshot protection；
- merged / closed / re-review 路由。

## 6. Agent Skill

```text
skills/gitlink-pr-review-warroom/SKILL.md
skills/gitlink-feishu/SKILL.md
skills/gitlink-pr-review-quality/SKILL.md
skills/gitlink-code-review/SKILL.md
```

Skill 负责指导 Agent 使用稳定的 CLI/Review Context，不把开放式 Agent 推理放进
飞书事件 Gateway。

## 7. 当前实施文档

### 总览

- [完整设计](./ROUND2_PR_REVIEW_COLLABORATION_DESIGN_V2.md)
- [机器人能力总结](./FEISHU_REVIEW_BOT_CAPABILITY_SUMMARY.md)
- [本完整索引](./FEISHU_COMPLETE_IMPLEMENTATION_INDEX.md)

### 阶段记录

- [P0 实施](./ROUND2_PR_REVIEW_COLLABORATION_P0_IMPLEMENTATION.md)
- [P1 实施](./ROUND2_PR_REVIEW_COLLABORATION_P1_IMPLEMENTATION.md)
- [P1.1 门禁](./ROUND2_PR_REVIEW_COLLABORATION_P11_GATE.md)
- [P2.0 实施](./ROUND2_PR_REVIEW_COLLABORATION_P2_IMPLEMENTATION.md)
- [P2.1 实施](./ROUND2_PR_REVIEW_COLLABORATION_P21_IMPLEMENTATION.md)

### 飞书研究和运行

- [能力分层](./FEISHU_CAPABILITY_LAYERS.md)
- [环境说明](./FEISHU_ENVIRONMENT.md)
- [OpenAPI 清单](./FEISHU_OPENAPI_INVENTORY.md)
- [GitLink/飞书重设计研究](./FEISHU_GITLINK_REDESIGN_RESEARCH.md)
- [PR 活动策略](./FEISHU_PR_ACTIVITY_STRATEGY.md)
- [Base schema](./feishu-bitable-schema.md)
- [飞书集成说明](./feishu-integration.md)
- [安全边界](./feishu-security.md)

### 当前排障

- [飞书 Agent 排障交接](./FEISHU_AGENT_DEBUG_HANDOFF.md)
- [飞书 Agent 回复与行动修订](./FEISHU_AGENT_RESPONSE_20260731.md)

## 8. 迁入的独立复赛资料

原本地目录：

```text
E:\GitLinkCLI-Competition\round2-preparation
```

本分支位置：

```text
docs/round2-preparation/
```

17 个文件已逐文件哈希校验一致，包括：

1. 当前项目盘点与边界；
2. 飞书与企业微信接入知识库；
3. GitLink 定位与智能体协同原则；
4. 功能策划与一轮拓展计划；
5. 复赛演示与验收清单；
6. 集成尝试结果与未来展望；
7. GitLink 接入企业微信功能评估；
8. GitLink 企业微信机器人接入方案；
9. GitLink 飞书 PR Review Warroom 完整规划；
10. 飞书阶段成果与企业微信接入计划；
11. 飞书、企业微信与最新主线信息收集；
12. 复赛 PR Review 协同集成完整设计 v2；
13. 飞书卡片、企微模板卡片和协作事件实验 fixture。

入口：

- [复赛准备目录](./round2-preparation/README.md)

## 9. 迁入的早期飞书设计

原本地目录：

```text
E:\GitLinkCLI-Competition\feishu-integration-design
```

本分支位置：

```text
docs/feishu-integration-design/
```

2 个文件已逐文件哈希校验一致：

- [早期评估](./feishu-integration-design/ASSESSMENT.md)
- [早期设计计划](./feishu-integration-design/DESIGN_PLAN.md)

这些文档用于保留设计演进，不代表当前实现状态；当前事实以 P2.1 文档和源码为准。

## 10. 企业微信资料

企业微信当前是 P4 候选适配层，没有写入当前 Go Gateway。

相关资料集中在：

```text
docs/round2-preparation/02-飞书与企业微信接入知识库.md
docs/round2-preparation/07-GitLink接入企业微信功能评估-参考GitHub聊天室插件.md
docs/round2-preparation/08-GitLink企业微信机器人接入方案-v1.md
docs/round2-preparation/10-飞书阶段成果与企业微信接入计划汇报.md
docs/round2-preparation/11-信息收集文档-飞书企业微信与最新主线-20260729.md
```

实验 fixture：

```text
docs/round2-preparation/experiments/wecom-template-card.sample.json
```

## 11. 运行和核对脚本

```text
scripts/feishu-gitlink-env-check.ps1
scripts/feishu-gitlink-setup.ps1
scripts/feishu-gitlink-smoke.ps1
scripts/feishu-gitlink-screenshot-check.ps1
```

最后一个脚本从 GitHub `main` 补回，是当前分支此前唯一缺失的飞书相关历史文件。
它是手工截图清单；清单中的图片在本地和 GitHub `main` 均不存在，因此本分支没有
伪造截图。

## 12. 本地来源覆盖检查

已检查工作树：

```text
E:\GitLinkCLI-Competition\gitlink-cli-feishu-clean
E:\GitLinkCLI-Competition\gitlink-cli-round2-p0
E:\GitLinkCLI-Competition\gitlink-cli-round2-p1
E:\GitLinkCLI-Competition\gitlink-cli
```

结果：

- 当前树包含 `gitlink-cli-feishu-clean` 的全部飞书相关文件；
- P1、P1.1、P2.0 都是当前分支祖先；
- P0 在 GitHub 迁移时 rebase，当前树包含等价内容；
- GitHub `main` 仅有一个当前树缺失的飞书截图核对脚本，已补回；
- 未复制其他工作树的 `.local` 或无关脏改动；
- 主 `gitlink-cli` 工作树的 `.gitattributes` 删除属于其他工作，不进入本分支。

## 13. 当前真实状态

### 已成立

```text
GitLink GET-only Review Queue / Context
飞书 App 鉴权
bot identity
WebSocket 连接
应用主动发群消息
群历史读取
SQLite 可靠 Job 与回复状态
P2.1 单元测试
```

### 未成立

```text
真实群消息稳定进入业务 handler
完整两次回复链路
真实用户 allowlist
开发者后台事件版本/发布核对
常驻部署
Base / Doc / Task 真实同步
GitLink 写回
```

## 14. 验证命令

目标测试：

```powershell
go test ./shortcuts/feishu
go test ./shortcuts/workflow
go test . ./shortcuts ./shortcuts/pr
go vet ./shortcuts/feishu/...
git diff --check
```

全仓：

```powershell
go test ./...
```

全仓仍有当前分支未修改的历史基线失败，不能把目标包通过写成全仓通过。

## 15. 推荐阅读顺序

### 飞书 Agent / 技术支持

1. `FEISHU_AGENT_DEBUG_HANDOFF.md`
2. `FEISHU_AGENT_RESPONSE_20260731.md`
3. `ROUND2_PR_REVIEW_COLLABORATION_P21_IMPLEMENTATION.md`
4. `shortcuts/feishu/review_gateway_command.go`
5. `shortcuts/feishu/review_gateway_models.go`

### 比赛评审

1. `FEISHU_REVIEW_BOT_CAPABILITY_SUMMARY.md`
2. `ROUND2_PR_REVIEW_COLLABORATION_DESIGN_V2.md`
3. `ROUND2_PR_REVIEW_COLLABORATION_P21_IMPLEMENTATION.md`
4. `round2-preparation/05-复赛演示与验收清单.md`

### 后续开发

1. `FEISHU_COMPLETE_IMPLEMENTATION_INDEX.md`
2. `ROUND2_PR_REVIEW_COLLABORATION_P11_GATE.md`
3. `FEISHU_AGENT_RESPONSE_20260731.md`
4. Review Gateway 源码和测试。
