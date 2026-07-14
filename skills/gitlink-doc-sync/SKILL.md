---
name: gitlink-doc-sync
version: 1.0.0
description: "中英文档一致性守护：检测 README/文档中英文版本之间的内容漂移（章节缺失、更新滞后、示例不一致），生成漂移报告并自动提交同步修复 PR。当用户需要检查文档翻译是否同步、维护双语文档、或在发版前审计文档一致性时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli file --help"
---

# gitlink-doc-sync（中英文档一致性守护）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有写操作（`file +update` / `pr +create`）必须先向用户展示漂移报告并获得确认，且一律通过 `--new-branch` 提交到新分支走 PR 流程，禁止直接推主分支。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)；文件读写命令详见 [`../gitlink-file/SKILL.md`](../gitlink-file/SKILL.md)。

---

## 功能概述

开源项目普遍维护中英双语文档（如 `README.md` + `README.zh-CN.md`），但两个版本极易漂移：英文更新了新功能，中文没跟上；中文修了错误，英文仍是旧说法。本技能提供完整的文档一致性守护流程：

1. **文档对发现** — 自动识别仓库中的双语文档对
2. **漂移检测** — 逐节比对两个版本的结构与内容新鲜度
3. **漂移报告** — 生成分级报告（缺失章节 / 滞后章节 / 示例不一致）
4. **同步修复** — 翻译补齐漂移内容，经用户确认后提交同步 PR
5. **发版前审计** — 与 `release` 流程配合，在发版前做文档一致性门禁

---

## 一、文档对发现

```bash
# 列出仓库根目录，寻找双语文档对
gitlink-cli repo +tree --owner <owner> --repo <repo> --ref master --format json

# 常见文档对命名约定（按文件名搜索确认）
gitlink-cli file +search --owner <owner> --repo <repo> --keyword README
gitlink-cli file +search --owner <owner> --repo <repo> --keyword zh-CN
```

**AI 判断逻辑**：以下命名模式视为文档对（`<base>` 为主文档）：

| 主文档 | 翻译文档 |
|--------|----------|
| `README.md` | `README.zh-CN.md` / `README_zh.md` / `README.zh.md` |
| `docs/<name>.md` | `docs/<name>.zh-CN.md` / `docs/zh/<name>.md` |
| `CONTRIBUTING.md` | `CONTRIBUTING.zh-CN.md` |

若用户明确指定了文档对，跳过本步骤。

## 二、漂移检测

### 2.1 拉取两个版本的内容

```bash
gitlink-cli file +view --owner <owner> --repo <repo> --path README.md --raw > /tmp/doc-en.md
gitlink-cli file +view --owner <owner> --repo <repo> --path README.zh-CN.md --raw > /tmp/doc-zh.md
```

### 2.2 结构比对（AI 逐节分析）

对两个文件分别提取标题大纲（`#`/`##`/`###`），按语义对齐章节后检查：

- **缺失章节**：一侧存在、另一侧完全没有对应章节（严重）
- **滞后章节**：章节都在，但一侧的命令示例、表格行数、功能列表明显多于另一侧（中等）
- **示例不一致**：代码块中的命令、参数、输出与另一侧不一致（中等）
- **链接/版本号不一致**：安装版本、徽章、链接指向不同目标（轻微）

### 2.3 新鲜度佐证（可选，用提交历史判断谁滞后）

```bash
# 查看两个文件最近的提交时间，更新较早的一侧通常是滞后方
gitlink-cli api GET "/:owner/:repo/commits?filepath=README.md&limit=1" --format json
gitlink-cli api GET "/:owner/:repo/commits?filepath=README.zh-CN.md&limit=1" --format json
```

## 三、漂移报告

向用户输出报告（先报告，后动手）：

```markdown
# 文档一致性报告：README.md ⇄ README.zh-CN.md

| 等级 | 类型 | 位置 | 说明 |
|------|------|------|------|
| 🔴 严重 | 缺失章节 | ## File Operations | 中文版缺少「文件操作」整节 |
| 🟡 中等 | 滞后章节 | ## Features 表格 | 英文 19 行 / 中文 18 行，缺 File 一行 |
| 🟢 轻微 | 版本号 | 安装说明 | 英文 Go 1.26+ / 中文 Go 1.25+ |

建议：同步 2 处到 README.zh-CN.md，1 处到 README.md。是否创建同步 PR？
```

## 四、同步修复（需用户确认）

```bash
# 1. 在本地补齐滞后内容（AI 完成翻译/同步，保持术语与既有风格一致）
#    修改后的完整文件保存为 /tmp/doc-zh-fixed.md

# 2. 提交到新分支
gitlink-cli file +update --owner <owner> --repo <repo> --path README.zh-CN.md \
  --content-file /tmp/doc-zh-fixed.md -b master --new-branch docs/sync-readme-zh \
  -m "docs: 同步 README.zh-CN 与英文版"

# 3. 创建 PR
gitlink-cli pr +create --owner <owner> --repo <repo> \
  --title "docs: 同步中英文 README" \
  --head docs/sync-readme-zh --base master \
  --body "由 gitlink-doc-sync 检测到文档漂移并同步，详见漂移报告。"
```

**翻译原则**：

- 术语不翻译（命令名、参数、API 路径、专有名词保持原样）
- 保持代码块逐字一致，仅翻译注释
- 保持表格结构、章节顺序与主文档对齐
- 不确定的表述在 PR 描述中标注，交由维护者复核

## 五、发版前文档审计（与 release 流程配合）

```bash
# 发版前对所有文档对跑一遍漂移检测，存在 🔴 严重漂移时提醒用户暂缓发版
gitlink-cli release +list --owner <owner> --repo <repo> --format json
```

审计输出示例：`3 个文档对，1 个存在严重漂移（README ⇄ README.zh-CN），建议同步后再发版。`

---

## 安全与边界

- 只读命令（`+view` / `+search` / `+tree`）可直接执行；任何写操作必须先出报告并确认。
- 同步修复一律 `--new-branch` + PR，由维护者 review 合并。
- 漂移判断由 AI 语义比对完成，报告中必须给出具体位置与证据，不允许笼统结论。
