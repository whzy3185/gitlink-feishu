---
name: gitlink-onboard
version: 1.0.0
description: "新贡献者上手指南：一站式生成项目简介、技术栈、核心目录导航、社区文件检查、good-first-issue、核心贡献者联系与上手步骤，帮助新人快速参与项目。当用户提到「上手指南」「怎么参与这个项目」「新人指南」「onboarding」「接手项目」「从哪开始贡献」时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
    optional_bins: ["python"]
  cliHelp: "gitlink-cli repo --help"
---

# gitlink-onboard（新贡献者上手指南生成器）

**CRITICAL — 开始前先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**
**CRITICAL — 本技能全程只读，不修改任何远程数据。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。

## 何时使用本技能

- 新人想参与一个项目，需要一份完整的上手指南
- 用户问「这个项目怎么参与 / 从哪开始 / 代码在哪 / 找谁问」
- 接手一个陌生仓库前先了解全貌

## 与 gitlink-newcomer 的区别

`gitlink-newcomer` 聚焦「识别 good-first-issue 并生成引导评论」（面向维护者整理任务）；
本技能输出的是**面向新人的完整上手指南**：项目是什么、技术栈、代码在哪、社区文件、
从哪个 Issue 开始、找谁问、怎么提 PR——一站式覆盖。

## 能力概览

| 能力 | 说明 |
|------|------|
| 项目概览 | 简介、社区指标、默认分支 |
| 技术栈识别 | 从依赖文件推断（go.mod/package.json…） |
| 核心目录导航 | 列出目录并标注语义（cmd/src/internal…） |
| 社区文件检查 | README/CONTRIBUTING/行为准则是否齐全 |
| good-first-issue | 适合新手上手的 Issue |
| 核心贡献者 | 遇到问题找谁 |
| 上手步骤 | 标准 Fork → 改 → PR 流程 |

## 工作流：生成上手指南

### 方式 A：配套脚本（推荐）

```bash
python scripts/onboard.py --owner Gitlink --repo gitlink-cli
python scripts/onboard.py --owner Gitlink --repo gitlink-cli --format json
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
gitlink-cli repo +info --owner Gitlink --repo gitlink-cli --format json
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=&ref=master' --format json
gitlink-cli issue +list --owner Gitlink --repo gitlink-cli --state open --format json
gitlink-cli api GET /:owner/:repo/contributors --format json
```

## API 注意事项

- 技术栈依据根目录依赖文件推断；核心目录导航依据 `sub_entries` 返回的目录。
- good-first-issue 用关键词启发式识别（typo/docs/test/翻译 等）。
- 数据采集全程只读。

## References

- [api-reference.md](references/api-reference.md) — 采集接口与字段
- [guide-structure.md](references/guide-structure.md) — 上手指南结构与识别规则
- [gitlink-shared](../gitlink-shared/SKILL.md) — 认证、全局参数、安全规则
