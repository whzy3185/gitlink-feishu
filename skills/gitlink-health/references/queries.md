# SQL 查询参考

所有查询使用 SQLite，通过命令行逐条执行。将 `<repo_id>` 等占位符替换为实际值。

**每个查询都标注了它对应报告模板中的哪些字段。按照[底部的组装清单](#报告组装清单)逐项执行，确保不遗漏。**

## 表结构

### users

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_name TEXT NOT NULL UNIQUE
);
```

### repos

```sql
CREATE TABLE repos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_name TEXT NOT NULL,
    owner_id INTEGER NOT NULL,
    FOREIGN KEY (owner_id) REFERENCES users(id),
    UNIQUE(repo_name, owner_id)
);
```

### issues

```sql
CREATE TABLE issues (
    id INTEGER PRIMARY KEY,
    repo_id INTEGER NOT NULL,
    number INTEGER,          -- 仓库内 Issue 编号 (project_issues_index)
    creater_id INTEGER,      -- 创建者 → users.id
    processor_id INTEGER,    -- 处理人 → users.id（nullable）
    create_time TIMESTAMP,   -- 创建时间，格式 "YYYY-MM-DD HH:MM"
    close_time TIMESTAMP,    -- 关闭时间（nullable）
    status TEXT CHECK(status IN ('close', 'open')),
    FOREIGN KEY (repo_id) REFERENCES repos(id),
    FOREIGN KEY (creater_id) REFERENCES users(id),
    FOREIGN KEY (processor_id) REFERENCES users(id)
);
```

API 字段映射：`creater_id` ← `author.login`（detail）或 `author_login`（list），`status` ← `status.name`（detail）等于"关闭"→close。

### pulls

```sql
CREATE TABLE pulls (
    id INTEGER PRIMARY KEY,
    repo_id INTEGER NOT NULL,
    number INTEGER NOT NULL,    -- PR 编号
    creater_id INTEGER,         -- 创建者 → users.id
    status TEXT CHECK(status IN ('merged', 'closed', 'open')),
    processor_id INTEGER,       -- 指派人 → users.id（nullable，来自 list 的 assign_user_login）
    create_time TIMESTAMP,      -- 创建时间，ISO 格式
    merged_at TIMESTAMP,        -- 合并时间（从 PR 详情 API 获取）
    FOREIGN KEY (repo_id) REFERENCES repos(id),
    FOREIGN KEY (creater_id) REFERENCES users(id),
    FOREIGN KEY (processor_id) REFERENCES users(id)
);
```

API 字段映射：`status` ← `pull_request_status`（0=open, 1=merged, 2=closed），`creater_id` ← `author_login`；`merged_at` 需要调用 PR 详情 API（`GET /:owner/:repo/pulls/:number`）获取。

### tags

```sql
CREATE TABLE tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    FOREIGN KEY (repo_id) REFERENCES repos(id),
    UNIQUE(repo_id, name)
);
```

API 字段映射：`name` ← `name`，先调用 `GET /:owner/:repo/tags` 批量写入，后续 PR/Issue 保存时只做关联查询。

### issue_tags

```sql
CREATE TABLE issue_tags (
    issue_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (issue_id, tag_id)
);
```

关系表：一个 Issue 可关联多个 tag。写入来源：Issue detail 中的 `tags` 数组。

### pull_tags

```sql
CREATE TABLE pull_tags (
    pull_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    FOREIGN KEY (pull_id) REFERENCES pulls(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (pull_id, tag_id)
);
```

关系表：一个 PR 可关联多个 tag。写入来源：PR list 中的 `issue_tags` 数组。

## 1. 确定目标仓库

数据库可能包含多个仓库的数据，查询前需先获取目标仓库的 `repo_id`：

```bash
# 列出所有仓库
sqlite3 data/gitlink_health.db "SELECT r.id, u.user_name AS owner, r.repo_name FROM repos r JOIN users u ON r.owner_id = u.id;"

# 根据 owner/repo 精确查找
sqlite3 data/gitlink_health.db "SELECT r.id FROM repos r JOIN users u ON r.owner_id = u.id WHERE u.user_name = '<owner>' AND r.repo_name = '<repo>';"
```

## 2. Issue 解决时长

### 2.1 单个 Issue 解决时长（天）

*（补充查询，报告模板不使用）*

```sql
SELECT
    i.id,
    i.number,
    (julianday(i.close_time) - julianday(i.create_time)) as resolution_days
FROM issues i
WHERE i.id = <issue_id>
  AND i.status = 'close';
```

### 2.2 已关闭 Issue 数 + 平均/中位数/最短/最长解决天数

**对应模板字段：`已关闭 Issue 数`、`平均天数`、`中位数`、`最短`、`最长`**

```sql
SELECT
    COUNT(*) as closed_issues_count,
    AVG(resolution_days) as avg_days,
    -- 中位数：排序后取中间值
    (SELECT resolution_days FROM (
        SELECT (julianday(close_time) - julianday(create_time)) as resolution_days,
               ROW_NUMBER() OVER (ORDER BY julianday(close_time) - julianday(create_time)) as rn,
               COUNT(*) OVER () as cnt
        FROM issues
        WHERE repo_id = <repo_id>
          AND status = 'close'
          AND close_time IS NOT NULL
          AND create_time IS NOT NULL
    ) WHERE rn IN ((cnt + 1) / 2, (cnt + 2) / 2)
    ORDER BY rn LIMIT 1) as median_days,
    MIN(resolution_days) as min_days,
    MAX(resolution_days) as max_days
FROM (
    SELECT (julianday(close_time) - julianday(create_time)) as resolution_days
    FROM issues
    WHERE repo_id = <repo_id>
      AND status = 'close'
      AND close_time IS NOT NULL
      AND create_time IS NOT NULL
);
```

### 2.3 解决时长分布

**对应模板字段：`解决时长分布：1天内 x，1周内 x，1个月内 x`**

```sql
SELECT
    CASE
        WHEN resolution_days < 1 THEN '< 1天'
        WHEN resolution_days < 3 THEN '1-3天'
        WHEN resolution_days < 7 THEN '3-7天'
        WHEN resolution_days < 30 THEN '7-30天'
        ELSE '> 30天'
    END as duration_range,
    COUNT(*) as issue_count
FROM (
    SELECT (julianday(close_time) - julianday(create_time)) as resolution_days
    FROM issues
    WHERE repo_id = <repo_id>
      AND status = 'close'
      AND close_time IS NOT NULL
      AND create_time IS NOT NULL
)
GROUP BY duration_range
ORDER BY MIN(resolution_days);
```

> 填入模板时：`1天内` = `'< 1天'` 的数量，`1周内` = `'1-3天' + '3-7天'` 的数量之和，`1个月内` = `'7-30天'` 的数量。

### 2.4 按时间段统计 Issue 解决时长

*（补充查询，报告模板不使用）*

```sql
SELECT
    strftime('%Y-%m', create_time) as month,
    AVG(julianday(close_time) - julianday(create_time)) as avg_resolution_days,
    COUNT(*) as closed_count
FROM issues
WHERE repo_id = <repo_id>
  AND status = 'close'
  AND close_time IS NOT NULL
  AND create_time IS NOT NULL
GROUP BY month
ORDER BY month;
```

## 3. PR 合并效率

### 3.1 PR 合并率（整体）

**对应模板字段：`合并率`、`总数`、`已合并`、`开放`、`已关闭`**

```sql
SELECT
    CAST(SUM(CASE WHEN status = 'merged' THEN 1 ELSE 0 END) AS REAL) /
    NULLIF(COUNT(*), 0) as merge_rate,
    COUNT(*) as total_prs,
    SUM(CASE WHEN status = 'merged' THEN 1 ELSE 0 END) as merged_count,
    SUM(CASE WHEN status = 'closed' THEN 1 ELSE 0 END) as closed_count,
    SUM(CASE WHEN status = 'open' THEN 1 ELSE 0 END) as open_count
FROM pulls
WHERE repo_id = <repo_id>;
```

> 填入模板时：`合并率` = `merge_rate * 100`（保留整数百分比）。

### 3.2 按时间段统计 PR 合并率

*（补充查询，报告模板不使用）*

```sql
SELECT
    strftime('%Y-%m', create_time) as month,
    CAST(SUM(CASE WHEN status = 'merged' THEN 1 ELSE 0 END) AS REAL) /
    NULLIF(COUNT(*), 0) as merge_rate,
    COUNT(*) as total_prs
FROM pulls
WHERE repo_id = <repo_id>
GROUP BY month
ORDER BY month;
```

### 3.3 PR 关闭率

*（补充查询，报告模板不使用）*

```sql
SELECT
    CAST(COUNT(CASE WHEN status IN ('closed', 'merged') THEN 1 END) AS REAL) /
    NULLIF(COUNT(*), 0) as closure_rate
FROM pulls
WHERE repo_id = <repo_id>;
```

## 4. 贡献者活跃度

### 4.1 活跃贡献者数量

**对应模板字段：`活跃贡献者（最近一个月）`**

```sql
SELECT
    COUNT(DISTINCT creater_id) as active_contributors
FROM pulls
WHERE repo_id = <repo_id>
  AND create_time BETWEEN '<start_date>' AND '<end_date>';
```

> `<start_date>` 设为 30 天前的日期，`<end_date>` 设为当天。

### 4.2 新贡献者（首次 PR 在最近一个月内）

**对应模板字段：`新增贡献者（最近一个月）`**

```sql
SELECT
    u.user_name,
    MIN(p.create_time) as first_pr_time
FROM pulls p
JOIN users u ON p.creater_id = u.id
WHERE p.repo_id = <repo_id>
GROUP BY p.creater_id
HAVING first_pr_time BETWEEN '<start_date>' AND '<end_date>';
```

> `<start_date>` 同上设为 30 天前。查询结果的行数 = 新增贡献者数，user_name 列表可附在报告中。

### 4.3 TOP 贡献者（按 PR 数排名）

**对应模板字段：`TOP 贡献者（按 PR 数排名）`**

```sql
SELECT
    u.user_name,
    COUNT(*) as pr_count
FROM pulls p
JOIN users u ON p.creater_id = u.id
WHERE p.repo_id = <repo_id>
GROUP BY p.creater_id
ORDER BY pr_count DESC;
```

> 取前 5~10 名填入模板，格式：`user1 x, user2 x, ...`

### 4.4 指定时间段内贡献者活跃度

*（补充查询，报告模板不使用）*

```sql
SELECT
    u.user_name,
    COUNT(*) as pr_count
FROM pulls p
JOIN users u ON p.creater_id = u.id
WHERE p.repo_id = <repo_id>
  AND p.create_time BETWEEN '<start_date>' AND '<end_date>'
GROUP BY p.creater_id
ORDER BY pr_count DESC;
```

### 4.5 贡献者月度活跃度趋势

*（补充查询，报告模板不使用）*

```sql
SELECT
    strftime('%Y-%m', p.create_time) as month,
    u.user_name,
    COUNT(*) as pr_count
FROM pulls p
JOIN users u ON p.creater_id = u.id
WHERE p.repo_id = <repo_id>
GROUP BY month, p.creater_id
ORDER BY month, pr_count DESC;
```

## 5. 补充查询

*（以下查询不对应报告模板字段，用于数据验证或扩展分析）*

### Issue 状态汇总

```sql
SELECT status, COUNT(*) as count
FROM issues WHERE repo_id = <repo_id>
GROUP BY status;
```

### PR 状态汇总

```sql
SELECT status, COUNT(*) as count
FROM pulls WHERE repo_id = <repo_id>
GROUP BY status;
```

### 贡献者总数

```sql
SELECT COUNT(DISTINCT creater_id) as total_contributors
FROM (
    SELECT creater_id FROM pulls WHERE repo_id = <repo_id> AND creater_id IS NOT NULL
    UNION
    SELECT creater_id FROM issues WHERE repo_id = <repo_id> AND creater_id IS NOT NULL
);
```

### 标签列表

```sql
SELECT name FROM tags WHERE repo_id = <repo_id> ORDER BY name;
```

### 按标签统计 Issue 数量

```sql
SELECT t.name, COUNT(*) as issue_count
FROM tags t
JOIN issue_tags it ON t.id = it.tag_id
JOIN issues i ON it.issue_id = i.id
WHERE t.repo_id = <repo_id>
GROUP BY t.id
ORDER BY issue_count DESC;
```

### 按标签统计 PR 数量

```sql
SELECT t.name, COUNT(*) as pr_count
FROM tags t
JOIN pull_tags pt ON t.id = pt.tag_id
JOIN pulls p ON pt.pull_id = p.id
WHERE t.repo_id = <repo_id>
GROUP BY t.id
ORDER BY pr_count DESC;
```

### 查询某 Issue 的所有标签

```sql
SELECT t.name
FROM tags t
JOIN issue_tags it ON t.id = it.tag_id
WHERE it.issue_id = <issue_id>;
```

### 查询某 PR 的所有标签

```sql
SELECT t.name
FROM tags t
JOIN pull_tags pt ON t.id = pt.tag_id
WHERE pt.pull_id = <pull_id>;
```

---

## 报告组装清单

**按以下顺序逐项执行查询、填入模板。每完成一项打勾，最终输出前核对所有 ✅ 是否齐全。**

- [ ] **已关闭 Issue 数** → 执行 2.2，取 `closed_issues_count`
- [ ] **平均天数** → 执行 2.2，取 `avg_days`（保留 1 位小数）
- [ ] **中位数** → 执行 2.2，取 `median_days`（保留 1 位小数）
- [ ] **最短 x 天** → 执行 2.2，取 `min_days`（保留 1 位小数）
- [ ] **最长 x 天** → 执行 2.2，取 `max_days`（保留 1 位小数）
- [ ] **解决时长分布** → 执行 2.3，汇总：1天内 = `'< 1天'` 数，1周内 = `1-3天 + 3-7天`，1个月内 = `7-30天`
- [ ] **合并率 x%** → 执行 3.1，取 `merge_rate * 100`（整数）
- [ ] **总数** → 执行 3.1，取 `total_prs`
- [ ] **已合并** → 执行 3.1，取 `merged_count`
- [ ] **开放** → 执行 3.1，取 `open_count`
- [ ] **已关闭** → 执行 3.1，取 `closed_count`
- [ ] **活跃贡献者（最近一个月）** → 执行 4.1，设 start_date = 30天前
- [ ] **新增贡献者（最近一个月）** → 执行 4.2，设 start_date = 30天前
- [ ] **TOP 贡献者** → 执行 4.3，取前 5~10 名

**输出前核对：报告必须包含上述全部 14 个字段，不允许省略任何一个。**
