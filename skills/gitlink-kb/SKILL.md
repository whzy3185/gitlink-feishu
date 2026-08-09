---
name: gitlink-kb
version: 1.0.0
description: "仓库知识库问答：索引 README、docs 目录与各类 Markdown 文档，支持关键词检索、文档地图生成、FAQ 提取，让仓库沉淀的知识可被快速查询。当用户提到「文档里怎么说」「如何使用/安装/配置」「这个项目的文档」「FAQ」「常见问题」「知识库」「文档地图」「搜索文档」时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
    optional_bins: ["python"]
  cliHelp: "gitlink-cli repo --help"
---

# gitlink-kb（仓库知识库问答助手）

**CRITICAL — 开始前先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**
**CRITICAL — 本技能全程只读，不修改任何远程数据。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。

## 何时使用本技能

- 用户问「这个项目怎么安装/配置/使用」，希望从仓库文档里找答案
- 想快速了解一个仓库都有哪些文档、讲了什么（文档地图）
- 想从文档中提取 FAQ / 常见问题
- 在不克隆仓库的情况下检索文档内容

## 何时不使用

- 检索代码实现而非文档 → 用代码搜索类工具
- 仅读取单个文件 → 用 `gitlink-repo` 的 readme/文件接口

## 能力概览

| 能力 | 说明 |
|------|------|
| 关键词检索 | 在 README + docs 等文档中检索与问题最相关的段落（支持中英文） |
| 文档地图 | 按文档归类所有标题，呈现仓库文档结构 |
| FAQ 提取 | 自动识别文档中形似问题的标题，提取问答对 |

## 工作流：从仓库文档中查找答案

### 方式 A：用配套脚本（推荐）

```bash
# 关键词/问题检索
python scripts/kb.py --owner Gitlink --repo gitlink-cli --query "如何安装"

# 生成文档地图
python scripts/kb.py --owner Gitlink --repo gitlink-cli --map

# 提取 FAQ
python scripts/kb.py --owner Gitlink --repo gitlink-cli --faq

# JSON 输出
python scripts/kb.py --owner Gitlink --repo gitlink-cli --query "登录" --format json
```

参数说明：

| 参数 | 类型 | 必填 | 说明 |
|------|------|:----:|------|
| `--owner` | string | 是* | 仓库所有者（*或用 `--slug`） |
| `--repo` | string | 是* | 仓库名称 |
| `--slug` | string | 否 | `owner/repo` 或完整 URL |
| `--ref` | string | 否 | 分支或标签，默认 master |
| `--query` | string | 否 | 检索关键词/问题 |
| `--map` | flag | 否 | 生成文档地图 |
| `--faq` | flag | 否 | 提取 FAQ |
| `--max-files` | int | 否 | 最多索引的文档数，默认 20 |
| `--format` | string | 否 | `markdown`（默认）或 `json` |
| `--output` | string | 否 | 输出文件 |

### 方式 B：用 gitlink-cli 读取文档

```bash
# 读取 README
gitlink-cli repo +readme --owner Gitlink --repo gitlink-cli --ref master --format json

# 列出 docs 目录
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=docs&ref=master' --format json

# 读取某个文档文件
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=docs/guide.md&ref=master' --format json
```

## 检索说明

- 检索基于关键词命中计分，标题命中加权；中文查询会做 2-gram 切分，兼顾中英文文档。
- 索引范围：README + `docs/`、`doc/`、`.gitlink/`、`wiki/` 等目录下的 Markdown/文本文件。
- 这是基于规则的检索，不依赖大模型，结果可解释。

## API 注意事项

- README 通过 `readme` 接口读取，其余文档通过 `sub_entries` 接口读取内容。
- 数据采集全程只读。

## 输出示例

参见 [`examples/`](examples/) 的真实检索结果与 FAQ。

## References

- [api-reference.md](references/api-reference.md) — 采集接口、字段与输出结构
- [search.md](references/search.md) — 索引范围、检索算法与 FAQ 提取规则
- [gitlink-shared](../gitlink-shared/SKILL.md) — 认证、全局参数、安全规则
