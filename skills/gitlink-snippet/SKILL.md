---
name: gitlink-snippet
version: 1.0.0
description: "本地代码片段管理：创建、查看、搜索、更新、删除、导出个人代码片段库。当用户需要保存/查找常用代码片段、代码片段管理、代码片段检索时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli snippet --help"
---

# gitlink-snippet（本地代码片段管理）

**CRITICAL — 所有删除操作（`snippet +delete`）前必须确认用户意图，删除不可恢复。**
**CRITICAL — 本技能为本地功能，不调用 GitLink 平台 API、不需要登录认证。**
**CRITICAL — `--content -` 表示从 stdin 读取内容，适合粘贴大段代码。**

> **与其它 Skill 的区别**：本 Skill 操作的是本地代码片段库（存储在 `~/.config/gitlink-cli/snippets.json`），不涉及 GitLink 平台的 Issue/PR/Repo 等资源。若需操作平台资源，请使用对应域的 Skill（如 `gitlink-issue`、`gitlink-pr`）。

## 功能定位

gitlink-snippet 是开发者的「个人代码片段收藏夹」，帮助 AI Agent 和用户：
- **沉淀常用代码**：把反复查阅的片段（SQL 模板、正则、初始化脚本）存起来
- **快速检索**：按标签、语言、关键词或全文检索定位片段
- **复用与导出**：把片段导出为文件，或直接粘贴到当前任务中

## Shortcuts

| Shortcut | 说明 | 操作类型 |
|----------|------|----------|
| `snippet +create` | 创建代码片段 | ⚠️ Write |
| `snippet +list` | 列出所有片段（支持过滤） | Read |
| `snippet +view` | 查看片段详情 | Read |
| `snippet +search` | 全文搜索片段 | Read |
| `snippet +update` | 更新片段字段 | ⚠️ Write |
| `snippet +delete` | 删除片段 | 🔴 Destructive |
| `snippet +export` | 导出片段到文件 | Read |

## 参数参考

### snippet +create

| 参数 | 简写 | 必填 | 说明 |
|------|------|:----:|------|
| `--title` | `-t` | ✅ | 片段标题 |
| `--language` | `-l` | 否 | 编程语言（如 `go`、`python`） |
| `--tags` | `-g` | 否 | 标签（逗号分隔，如 `sql,template`） |
| `--content` | `-c` | 否 | 片段内容；传 `-` 表示从 stdin 读取 |

### snippet +list

| 参数 | 简写 | 必填 | 说明 |
|------|------|:----:|------|
| `--tag` | `-t` | 否 | 按标签过滤 |
| `--language` | `-l` | 否 | 按语言过滤 |
| `--keyword` | `-k` | 否 | 按标题关键词过滤 |

### snippet +view / +update / +delete / +export

| 参数 | 简写 | 必填 | 说明 |
|------|------|:----:|------|
| `--id` | `-i` | ✅ | 片段 ID（8 位十六进制，从 `+list` 获取） |

`+update` 额外支持 `--title / -t`、`--language / -l`、`--tags / -g`、`--content / -c`（至少传一个字段）。

`+export` 额外支持 `--output / -o`（输出文件路径，默认打印到 stdout）。

### snippet +search

| 参数 | 简写 | 必填 | 说明 |
|------|------|:----:|------|
| `--query` | `-q` | ✅ | 搜索关键词（全文匹配） |

## 使用示例

```bash
# 创建片段（直接传内容）
gitlink-cli snippet +create \
  --title "Postgres 慢查询检查" \
  --language sql \
  --tags db,performance \
  --content "SELECT * FROM pg_stat_statements ORDER BY total_time DESC LIMIT 10;"

# 创建片段（从 stdin 读取大段代码）
cat ./init_script.py | gitlink-cli snippet +create \
  --title "项目初始化脚本" \
  --language python \
  --tags init \
  --content -

# 列出所有片段
gitlink-cli snippet +list --format json

# 按标签 + 语言过滤
gitlink-cli snippet +list --tag sql --language sql

# 按标题关键词过滤
gitlink-cli snippet +list --keyword "Postgres"

# 查看片段详情
gitlink-cli snippet +view --id a1b2c3d4 --format json

# 全文搜索
gitlink-cli snippet +search --query "慢查询"

# 更新片段标题和标签
gitlink-cli snippet +update --id a1b2c3d4 --title "PG 慢查询 Top 10" --tags db,perf

# 导出片段到文件
gitlink-cli snippet +export --id a1b2c3d4 --output ./slow_query.sql

# 删除片段（⚠️ 不可恢复）
gitlink-cli snippet +delete --id a1b2c3d4
```

## 输出示例（`+list --format json`）

```json
{
  "ok": true,
  "data": [
    {
      "id": "a1b2c3d4",
      "title": "Postgres 慢查询检查",
      "language": "sql",
      "tags": ["db", "performance"],
      "content": "SELECT * FROM pg_stat_statements ORDER BY total_time DESC LIMIT 10;",
      "created_at": "2026-06-15T10:30:00+08:00",
      "updated_at": "2026-06-15T10:30:00+08:00"
    }
  ]
}
```

## 工作流

### 工作流 1：沉淀反复使用的代码

**场景**：在排查/开发中遇到一段值得复用的代码，保存下来供日后使用。

```bash
# 1. 把代码通过 stdin 存入（适合多行）
cat ./docker-compose-template.yml | gitlink-cli snippet +create \
  --title "通用 docker-compose 模板" \
  --language yaml \
  --tags docker,template \
  --content -

# 2. 验证已保存
gitlink-cli snippet +list --keyword "docker"
```

### 工作流 2：按需检索并复用

**场景**：写新代码时想调用之前存的片段。

```bash
# 1. 按关键词搜索
gitlink-cli snippet +search --query "慢查询"

# 2. 查看命中的片段内容
gitlink-cli snippet +view --id a1b2c3d4

# 3. 导出为文件后直接使用，或由 AI Agent 直接读取 content 字段复用
gitlink-cli snippet +export --id a1b2c3d4 --output ./tmp_query.sql
```

## 决策规则

| 场景 | 推荐命令 |
|------|---------|
| 已知片段 ID，想看内容 | `snippet +view --id <id>` |
| 只记得片段大概是关于什么 | `snippet +search --query "<关键词>"` |
| 想按标签浏览一类片段 | `snippet +list --tag <tag>` |
| 要批量查看某种语言的片段 | `snippet +list --language <lang>` |
| 大段代码要保存 | `snippet +create --content -`（stdin） |
| 要把片段给到脚本/工具用 | `snippet +export --id <id> --output <file>` |

## 注意事项

- **纯本地存储**：片段存于 `~/.config/gitlink-cli/snippets.json`，可通过 `GITLINK_CONFIG_DIR` 环境变量改变存储目录
- **不需要登录**：本 Skill 不调用 GitLink API，不依赖 `auth login`
- **ID 格式**：8 位十六进制随机串（如 `a1b2c3d4`），通过 `+list` / `+search` 获取
- **`+delete` 不可恢复**：删除前用 `+view` 确认内容，避免误删
- **stdin 模式**：`--content -` 表示从标准输入读取，适合保存剪贴板或多行代码
- **标签是字符串数组**：`--tags` 用逗号分隔（`sql,template`），不支持空格
- **全文搜索**：`+search` 匹配 title 和 content；若只想按标题找，用 `+list --keyword`

## References

- 全局参数与输出格式：[gitlink-shared/SKILL.md](../gitlink-shared/SKILL.md)
