---
name: gitlink-changelog
version: 1.0.0
description: "版本变更对比：对比提交历史，按 conventional commits 归类生成结构化变更日志，标注不兼容变更与贡献者。当用户提到「变更日志」「changelog」「版本对比」「两个版本之间改了什么」「发版变更」「what changed」时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
    optional_bins: ["python"]
  cliHelp: "gitlink-cli release --help"
---

# gitlink-changelog（版本变更对比）

**CRITICAL — 开始前先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**
**CRITICAL — 本技能全程只读，不修改任何远程数据。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。

## 何时使用本技能

- 发版前想生成一份变更日志（changelog）
- 用户问「这个版本相比上个版本改了什么」「最近有哪些变更」
- 需要把提交历史整理成结构化的版本说明

## 何时不使用

- 自动创建 Release 并发布 → 用 `gitlink-release` / `gitlink-release-auto`
- 仅比对代码 diff → 用 `gitlink-compare`

## 能力概览

| 能力 | 说明 |
|------|------|
| 提交归类 | 按 conventional commits（feat/fix/docs…）分组 |
| 不兼容变更标注 | 识别 `!` 标记与 BREAKING CHANGE |
| scope 与作者 | 提取每条变更的 scope 与提交者 |
| 贡献者汇总 | 列出本次范围内的全部贡献者 |

## 工作流：生成变更日志

### 方式 A：配套脚本（推荐）

```bash
# 生成最近提交的变更日志
python scripts/changelog.py --owner Gitlink --repo gitlink-cli

# 标注版本范围（用于报告标题）
python scripts/changelog.py --owner Gitlink --repo gitlink-cli --since v0.1.17 --until v0.1.18

# JSON 输出
python scripts/changelog.py --owner Gitlink --repo gitlink-cli --format json
```

参数说明：

| 参数 | 类型 | 必填 | 说明 |
|------|------|:----:|------|
| `--owner` / `--repo` | string | 是* | 仓库（*或用 `--slug`） |
| `--slug` | string | 否 | `owner/repo` 或完整 URL |
| `--since` / `--until` | string | 否 | 版本/标签，用于报告标注 |
| `--max-pages` | int | 否 | 提交采集页数（每页 50），默认 6 |
| `--format` | string | 否 | `markdown`（默认）或 `json` |
| `--output` | string | 否 | 输出文件 |

### 方式 B：用 gitlink-cli 命令

```bash
# 获取版本列表（定位版本时间）
gitlink-cli release +list --owner Gitlink --repo gitlink-cli --format json

# 获取提交历史
gitlink-cli api GET /:owner/:repo/commits --query 'page=1&limit=50' --format json
```

## API 注意事项

- GitLink 的 compare 接口需要鉴权，本技能改用提交列表分析，无需登录即可处理公开仓库。
- conventional commits 规范化率低的仓库，归类精度会下降；报告会标注规范化提交占比。

## References

- [api-reference.md](references/api-reference.md) — 采集接口与字段
- [conventions.md](references/conventions.md) — conventional commits 归类规则
- [gitlink-shared](../gitlink-shared/SKILL.md) — 认证、全局参数、安全规则
