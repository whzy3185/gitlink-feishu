---
name: gitlink-stale
version: 1.0.0
description: "陈旧 Issue/PR 清理：检测久未更新的开放 Issue 与 PR，按陈旧度分级（活跃/留意/陈旧/僵尸），生成清理建议清单。当用户提到「陈旧 Issue」「积压」「久未更新」「清理 Issue」「stale」「僵尸 PR」「待清理」时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
    optional_bins: ["python"]
  cliHelp: "gitlink-cli issue --help"
---

# gitlink-stale（陈旧 Issue/PR 清理）

**CRITICAL — 开始前先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**
**CRITICAL — 关闭 Issue/PR 属于写操作，执行前必须征得用户确认。本技能只生成建议清单，不自动关闭。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。

## 何时使用本技能

- 维护者想清理长期无人跟进的 Issue/PR，控制积压
- 用户问「有哪些 Issue 很久没动了」「该清理哪些 PR」
- 定期做 backlog 健康检查

## 何时不使用

- 项目综合健康度 → 用 `gitlink-health` / `gitlink-insight`
- 普通列出 Issue/PR → 用 `gitlink-issue` / `gitlink-pr`

## 陈旧度分级

| 等级 | 停滞时长 | 含义 |
|:----:|:--------:|------|
| 🟢 活跃 | ≤ 14 天 | 近期有更新 |
| 🟡 留意 | ≤ 45 天 | 开始放缓 |
| 🟠 陈旧 | ≤ 120 天 | 需要跟进 |
| 🔴 僵尸 | > 120 天 | 建议评估关闭 |

## 工作流：扫描陈旧项

### 方式 A：配套脚本（推荐）

```bash
python scripts/stale.py --owner Gitlink --repo gitlink-cli
python scripts/stale.py --owner Gitlink --repo gitlink-cli --format json
```

参数说明：

| 参数 | 类型 | 必填 | 说明 |
|------|------|:----:|------|
| `--owner` / `--repo` | string | 是* | 仓库（*或用 `--slug`） |
| `--slug` | string | 否 | `owner/repo` 或完整 URL |
| `--format` | string | 否 | `markdown`（默认）或 `json` |
| `--output` | string | 否 | 输出文件 |

### 方式 B：用 gitlink-cli 命令

```bash
gitlink-cli issue +list --owner Gitlink --repo gitlink-cli --state open --format json
gitlink-cli pr +list --owner Gitlink --repo gitlink-cli --state open --format json
```

### 处理建议（写操作，需确认）

```bash
# 确认后关闭陈旧 Issue（web 序号）
gitlink-cli issue +close --owner Gitlink --repo gitlink-cli --number <web序号>
```

## API 注意事项

- 停滞天数依据 `updated_at` / `created_at` 的相对时间（如"2个月前"）估算，为粗略值。
- 只扫描开放项（Issue 状态非「关闭」、PR `pull_request_status=0`）。
- 关闭操作是写操作，本技能只给建议，需用户确认后执行。

## References

- [api-reference.md](references/api-reference.md) — 采集接口与字段
- [grading.md](references/grading.md) — 陈旧度分级与时间估算规则
- [gitlink-shared](../gitlink-shared/SKILL.md) — 认证、全局参数、安全规则
