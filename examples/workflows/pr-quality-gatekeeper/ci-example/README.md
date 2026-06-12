# CI 集成示例 —— 门禁接 CI

本目录演示如何把 **gitlink-gatekeeper** 的 PR 看门人门禁接到 CI 上，让裁决直接挡住不达标的 PR。

> 这是**示例**，不是开箱即用的生产配置；`gitlink-cli` 的安装方式、PR 编号字段名需按你的 runner 实际情况调整。

## 文件

- [`gatekeeper.gitea.yml`](gatekeeper.gitea.yml)：Gitea Actions 工作流（GitLink 基于 Gitea，语法与 GitHub Actions 兼容）。

## 用法

1. 把 `gatekeeper.gitea.yml` 复制到目标仓库的 `.gitea/workflows/` 目录。
2. 在仓库 **Settings → Actions → Secrets** 新增 `GITLINK_TOKEN`，值为有权读取该仓库 PR 的访问令牌（供 `gitlink-cli` 认证）。**Token 切勿写进仓库或日志。**
3. 提一个 PR 触发工作流即可。

## 工作原理

- 触发：PR 的 `opened` / `synchronize` / `reopened` 事件。
- 步骤：检出 → 准备 Python 3.9（脚本纯标准库，无需装依赖）→ 装 `gitlink-cli` → 跑 `scripts/gatekeeper_workflow.py` 采集本次 PR 上下文并评分裁决。
- **退出码即门禁**：
  - `0` = PASS / COMMENT → job 通过，放行。
  - `2` = REQUEST_CHANGES → 工作流把它转成 job 失败，挡住该 PR。
  - `1` = 可预期错误（缺参数 / 未装 `gitlink-cli` 等）→ 同样失败。
- 产物：评分卡与 `summary.json` 落在 `outputs/`，工作流用 `upload-artifact` 上传，便于在 CI 页面查看裁决依据。

调门禁松紧只需改 `--policy` 指向的 `gatekeeper.yaml`（策略字段说明见 [`gitlink-gatekeeper` Skill REFERENCE](../../../../skills/gitlink-gatekeeper/REFERENCE.md)）。
