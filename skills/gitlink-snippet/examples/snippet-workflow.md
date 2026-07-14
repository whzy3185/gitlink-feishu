# 本地代码片段管理完整工作流示例

**场景**：开发者想在本地沉淀常用代码片段（SQL、模板、脚本），并按需检索复用。

## 前置条件

- `gitlink-cli` 已安装
- **无需登录认证**（本功能为本地存储，不调用 GitLink 平台 API）

## 工作流步骤

### Step 1：创建代码片段

```bash
# 方式 1：直接传内容
gitlink-cli snippet +create \
  --title "Postgres 慢查询检查" \
  --language sql \
  --tags db,performance \
  --content "SELECT * FROM pg_stat_statements ORDER BY total_time DESC LIMIT 10;"
```

**输出示例：**
```json
{
  "ok": true,
  "data": {
    "id": "a1b2c3d4",
    "title": "Postgres 慢查询检查",
    "language": "sql",
    "tags": ["db", "performance"]
  }
}
```

```bash
# 方式 2：从 stdin 读取大段代码（推荐用于多行）
cat ./init_script.py | gitlink-cli snippet +create \
  --title "项目初始化脚本" \
  --language python \
  --tags init,setup \
  --content -
```

### Step 2：浏览与过滤片段

```bash
# 列出所有片段
gitlink-cli snippet +list --format json

# 按标签过滤
gitlink-cli snippet +list --tag db

# 按语言过滤
gitlink-cli snippet +list --language python

# 按标题关键词过滤
gitlink-cli snippet +list --keyword "Postgres"
```

### Step 3：检索与查看

```bash
# 全文搜索（匹配 title 和 content）
gitlink-cli snippet +search --query "慢查询"

# 查看片段详情
gitlink-cli snippet +view --id a1b2c3d4 --format json
```

### Step 4：更新片段

```bash
# 更新标题和标签
gitlink-cli snippet +update --id a1b2c3d4 --title "PG 慢查询 Top 10" --tags db,perf

# 更新内容（stdin）
echo "SELECT * FROM pg_stat_statements ORDER BY mean_exec_time DESC LIMIT 20;" | \
  gitlink-cli snippet +update --id a1b2c3d4 --content -
```

### Step 5：导出与复用

```bash
# 导出到文件
gitlink-cli snippet +export --id a1b2c3d4 --output ./slow_query.sql

# 导出后直接执行
psql -d mydb -f ./slow_query.sql
```

### Step 6：清理不需要的片段（⚠️ 需确认）

```bash
# 先确认要删除的内容
gitlink-cli snippet +view --id a1b2c3d4

# 确认无误后删除（不可恢复）
gitlink-cli snippet +delete --id a1b2c3d4
```

---

## 完整命令速览

```bash
# 增
gitlink-cli snippet +create --title "..." --language <lang> --tags <t1,t2> --content "<code>"
gitlink-cli snippet +create --title "..." --content -                                    # stdin

# 查
gitlink-cli snippet +list [--tag <t>] [--language <lang>] [--keyword <kw>] --format json
gitlink-cli snippet +view --id <id> --format json
gitlink-cli snippet +search --query "<keyword>"

# 改
gitlink-cli snippet +update --id <id> [--title "..." --tags "..." --content "..."]

# 删 / 导出
gitlink-cli snippet +delete --id <id>
gitlink-cli snippet +export --id <id> --output <file>
```

---

## 注意事项

- 片段存储于 `~/.config/gitlink-cli/snippets.json`（可用 `GITLINK_CONFIG_DIR` 改路径）
- 不需要 `auth login`，纯本地功能
- `+delete` 不可恢复，删除前用 `+view` 确认
- `--content -` 表示从 stdin 读取，适合保存剪贴板或多行代码
