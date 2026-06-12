# 真实平台验证记录（2026-06-12）

> 在 GitLink 线上真实平台、用户自有 fork `recorder/gitlink-cli` 上跑通完整 IssueOps 闭环。
> 执行环境：Claude Code（AI Agent）驱动 `gitlink-cli`（npm `@gitlink-ai/cli` 0.2.0 官方发布版）。
> 下表所有对象 id 均真实存在，可在平台上逐一复核。

## 闭环五步与产物

| 步骤 | 命令 | 真实结果 |
|------|------|---------|
| 1. 挂 webhook | `webhook +create --owner recorder --repo gitlink-cli -u https://httpbin.org/post -e issues_only,issue_comment` | webhook **id 51579**，`events: [issues_only, issue_comment]`，active |
| 2. 建任务 Issue | `issue +create -t "[agent] IssueOps 验证：请为本仓库生成一份贡献者快速上手清单" -b "…"` | Issue **#3**（全局 id **144169**） |
| 3. 事件回放 | `webhook +tasks -i 51579 --format json` | hooktask **id 4836246**：`event_type=issues`、`action=opened`、`is_delivered/is_succeed=true`、`payload_content.issue` 含完整标题/正文/作者 |
| 4. Agent 执行并回帖 | 解析 `[agent]` 前缀 → 生成贡献者快速上手清单 → `issue +comment --number 3 --body "…"` | comment **id 475692**（含处理链说明） |
| 5. 闭环标记 | `label +create -n "agent:done" -c "#22C55E"` → `label +list` 查得 tag id **373052** → v1 `PATCH /api/v1/recorder/gitlink-cli/issues/3.json`，body `{"tag_ids":[373052]}` | HTTP 200；复核 `issue +view --number 3` → `issue_tags: ["agent:done"]` ✅ |

## 验证中的真实发现（已沉淀进 REFERENCE.md）

1. `--events issues` 非法，白名单名是 `issues_only`；
2. 投递异步：建 Issue 后任务记录约几秒至几十秒出现，轮询要带等待；
3. `+tasks` 数据在 `data.hooktasks[]`，**端点是否收到都会记录完整负载**——Replay 模式的根基；
4. 普通 Issue 打标签：老 API `POST /issues/<全局id>` 404，必须 v1 `PATCH /issues/<编号>` + `tag_ids`；
5. CLI 0.2.0 的 `api` 命令单次调用不替换 `:owner/:repo` 占位符，需写字面路径。

## 复现

把上表 owner/repo 换成你自己的仓库即可逐步复现；全程只对自有仓库写入，对外零打扰。
