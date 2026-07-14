---
name: gitlink-doctor
version: 1.0.0
description: "环境自诊断与故障排除助手：运行 doctor 体检并按检查结果逐项修复配置、认证、仓库上下文与 API 连通性问题。当用户报告 gitlink-cli 不工作、认证失败、401/404 报错、配置异常，或需要环境体检、排障、troubleshooting 时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli doctor --help"
---

# gitlink-doctor（环境自诊断与故障排除）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 本 Skill 的诊断步骤为只读；仅「修复动作」一节涉及写配置文件，执行前须告知用户将写入的路径与内容。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

---

## 功能概述

1. **一键体检** — 运行 `doctor` 得到 5 类检查的结构化结果（config_file / config_values / auth / repo_context / api_connectivity）
2. **逐项修复** — 按每项检查自带的 `suggestion` 字段执行对应修复动作并复检
3. **升级排障** — doctor 全绿但具体命令仍失败时，按「已知平台缺口对照表」定位是否为平台侧问题

## 使用场景

- "gitlink-cli 用不了了 / 一直 401"
- "帮我检查一下 CLI 环境"
- "为什么这个命令返回 404？"

## 执行步骤

### 第 1 步：运行体检

```bash
gitlink-cli doctor --format json
```

读取 `data.summary`（ok/warning/error 计数）与 `data.checks[]`。离线环境加 `--skip-network`。

### 第 2 步：按检查项逐项修复

| 检查项 | 常见状态 | 修复动作 |
|--------|----------|----------|
| `config_file` | warning：配置文件缺失 | `gitlink-cli config init`（写入 `~/.config/gitlink-cli/config.yaml`，属写操作，先告知用户） |
| `config_values` | error：非法取值 | 按 message 中的字段名修正配置，再 `config view` 复核 |
| `auth` | error：未登录/PAT 失效 | `gitlink-cli auth login` 重新认证；`auth status` 复核 |
| `repo_context` | warning：无法解析 owner/repo | 在 git 仓库内执行，或显式传 `--owner/--repo` |
| `api_connectivity` | error：不可达/超时 | 检查网络与 base_url 配置；`gitlink-cli api GET /users/me --debug` 看原始请求 |

每次修复后重跑 `doctor --format json`，直到 error=0；warning 属可运行状态，向用户说明含义即可。

### 第 3 步：doctor 全绿但命令仍失败 → 平台缺口对照

| 症状 | 根因（平台侧，非本地环境） |
|------|------|
| `wiki` 写操作 401 | wiki 微服务不认 PAT，仅认网页会话态 |
| `ci` 全部子命令失败 | 旧版 Trustie CI 已下线 |
| `dataset` 多数子命令 404 | 后端仅部署了 5 个端点中的 1 个 |
| v1 `contents` 返回 HTML | v1 文件内容端点缺失，改用 `sub_entries` 或 `repo +readme` |
| 跨 fork compare 失败 | 跨仓库 compare 端点不可用，改用 branch +list + git merge-base |

命中对照表时直接告知用户属平台缺口并给出表中替代方案，不要继续折腾本地环境。

## 输出格式

```markdown
# 环境诊断报告
- 体检：ok 4 / warning 1 / error 0（共 5 项）
- 修复动作：auth 重新登录 ✔（复检通过）
- 遗留 warning：config_file 缺失（使用内置默认值，可 `config init` 消除）
- 结论：环境可用 / 问题属平台缺口（附替代方案）
```

## 注意事项

- `doctor` 的 `data.ok` 表示无 error，允许存在 warning——不要把 warning 当故障处理
- `api_connectivity` 检查走认证接口：PAT 失效时会与 `auth` 同时报错，先修 auth 再看连通性
- 修复配置前先 `gitlink-cli config view` 备份当前值，改坏可回退
