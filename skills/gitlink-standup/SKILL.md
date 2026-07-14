---
name: gitlink-standup
version: 1.0.0
description: "个人/团队日报周报：汇总成员的提交、Issue、PR、版本发布活动，按类型统计生成团队同步日报/周报。当用户提到「日报」「周报」「standup」「团队同步」「某人最近做了什么」「成员活动」「team report」时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
    optional_bins: ["python"]
  cliHelp: "gitlink-cli user --help"
---

# gitlink-standup（个人/团队日报周报）

**CRITICAL — 开始前先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**
**CRITICAL — 本技能全程只读，仅统计公开活动。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。

## 何时使用本技能

- 团队每日/每周同步，需要汇总成员近期做了什么
- 用户问「@某人最近在忙什么」「这周团队有哪些活动」
- 生成个人工作日报/周报

## 何时不使用

- 项目级健康度/协作分析 → 用 `gitlink-insight` / `gitlink-health`
- 单个仓库的 Issue/PR 列表 → 用 `gitlink-issue` / `gitlink-pr`

## 能力概览

| 能力 | 说明 |
|------|------|
| 活动汇总 | 按 commit / Issue / PR / 版本发布分类统计 |
| 多成员 | 一次汇总整个团队 |
| 样例展示 | 每类活动展示代表性标题 |

## 工作流：生成日报/周报

### 方式 A：配套脚本（推荐）

```bash
# 单个成员
python scripts/standup.py --user wbtiger

# 整个团队
python scripts/standup.py --users wbtiger,wangyue789 --period 周

# JSON 输出
python scripts/standup.py --user wbtiger --format json
```

参数说明：

| 参数 | 类型 | 必填 | 说明 |
|------|------|:----:|------|
| `--user` | string | 是* | 单个成员 login（*或 `--users`） |
| `--users` | string | 否 | 多个成员 login，逗号分隔 |
| `--limit` | int | 否 | 每人采集的活动条数上限，默认 50 |
| `--period` | string | 否 | 周期标注（如 日 / 周） |
| `--format` | string | 否 | `markdown`（默认）或 `json` |
| `--output` | string | 否 | 输出文件 |

### 方式 B：用 gitlink-cli 命令

```bash
# 查看用户信息
gitlink-cli user +info --login wbtiger --format json

# 用户动态（Raw API）
gitlink-cli api GET /users/wbtiger/project_trends --query 'page=1&limit=50' --format json
```

## API 注意事项

- 活动数据来自 `users/:login/project_trends`，`action_time` 为相对时间（如"3天前"），
  因此本技能按活动条数与类型汇总，而非精确时间区间过滤。
- 仅统计公开活动；私有仓库活动不在其中。

## References

- [api-reference.md](references/api-reference.md) — 采集接口与字段
- [activity-types.md](references/activity-types.md) — 活动类型说明
- [gitlink-shared](../gitlink-shared/SKILL.md) — 认证、全局参数、安全规则
