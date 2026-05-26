# 贡献排行 & 徽章 SQL 查询参考

所有查询使用 SQLite。将 `<repo_id>`、`<start_date>`、`<end_date>` 替换为实际值。

> **已知数据质量问题：** `health +fetch` 在采集标签时可能将跨仓库的同名标签复用为同一个 `tag_id`，导致 `pull_tags`/`issue_tags` 引用了不属于目标仓库的标签。为此，所有 JOIN `tags` 的查询都加了 `t.repo_id = <repo_id>` 防护。如果发现标签数据异常，请先执行查询 8 确认目标仓库的实际标签列表。

> **注意：** 时间范围条件需用 `date()` 包裹列名，避免字典序比较导致当日数据被排除。下文所有查询已按此规则编写。

## 表结构

gitlink-growth 与 gitlink-health 共用同一 SQLite 数据库，表结构由 `health +fetch` 创建。核心表如下：

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

### pulls

```sql
CREATE TABLE pulls (
    id INTEGER PRIMARY KEY,
    repo_id INTEGER NOT NULL,
    number INTEGER NOT NULL,
    creater_id INTEGER,
    processor_id INTEGER,
    status TEXT CHECK(status IN ('merged', 'closed', 'open')),
    create_time TIMESTAMP,
    close_time TIMESTAMP,
    FOREIGN KEY (repo_id) REFERENCES repos(id),
    FOREIGN KEY (creater_id) REFERENCES users(id),
    FOREIGN KEY (processor_id) REFERENCES users(id)
);
```

`status` 取值：`open`（开放）、`merged`（已合并）、`closed`（已关闭未合并）。`close_time` 始终为 NULL（API 不提供）。
`processor_id` 表示 PR 的**指派人**（assignee）——因为 GitLink API 不提供 `merged_by`（合并者）信息，所以用 `processor_id` 代表"处理者"角色，即被指派来 review/merge 该 PR 的人。

### issues

```sql
CREATE TABLE issues (
    id INTEGER PRIMARY KEY,
    repo_id INTEGER NOT NULL,
    number INTEGER,
    creater_id INTEGER,
    processor_id INTEGER,
    create_time TIMESTAMP,
    close_time TIMESTAMP,
    status TEXT CHECK(status IN ('close', 'open')),
    FOREIGN KEY (repo_id) REFERENCES repos(id),
    FOREIGN KEY (creater_id) REFERENCES users(id),
    FOREIGN KEY (processor_id) REFERENCES users(id)
);
```

`status` 取值：`open`（开放）、`close`（已关闭）。
`processor_id` 表示 Issue 的**指派人**（assignee）——与 PR 同理，因 API 无 `merged_by`，`processor_id` 代表负责处理该 Issue 的人。

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

### pull_tags / issue_tags

```sql
CREATE TABLE pull_tags (
    pull_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    FOREIGN KEY (pull_id) REFERENCES pulls(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (pull_id, tag_id)
);

CREATE TABLE issue_tags (
    issue_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (issue_id, tag_id)
);
```

> 完整字段映射说明见 [`../gitlink-health/references/queries.md`](../gitlink-health/references/queries.md#表结构)。

## 数据库路径

默认：`~/.agents/skills/gitlink-health/data/gitlink_health.db`

```bash
DB=~/.agents/skills/gitlink-health/data/gitlink_health.db
```

## 时间范围计算

| 周期 | SQL 计算 |
|------|---------|
| 本周 | `date('now', 'weekday 0', '-6 days')` ~ `date('now')` |
| 本月 | `date('now', 'start of month')` ~ `date('now')` |
| 本年 | `date('now', 'start of year')` ~ `date('now')` |
| 自定义 | `'<user-start>'` ~ `'<user-end>'` |
| 全量 | `'1970-01-01'` ~ `date('now')` |

## 0. 确定目标仓库

```bash
# 列出所有仓库
sqlite3 "$DB" "SELECT r.id, u.user_name AS owner, r.repo_name FROM repos r JOIN users u ON r.owner_id = u.id;"

# 精确查找 repo_id
sqlite3 "$DB" "SELECT r.id FROM repos r JOIN users u ON r.owner_id = u.id WHERE u.user_name = '<owner>' AND r.repo_name = '<repo>';"
```

---

## 1. 综合贡献排名（PR + Issue）

```sql
SELECT
    u.user_name,
    COALESCE(pr.cnt, 0) AS pr_count,
    COALESCE(iss.cnt, 0) AS issue_count,
    COALESCE(pr.cnt, 0) + COALESCE(iss.cnt, 0) AS total
FROM users u
LEFT JOIN (
    SELECT creater_id, COUNT(*) AS cnt
    FROM pulls
    WHERE repo_id = <repo_id>
      AND date(create_time) BETWEEN '<start_date>' AND '<end_date>'
    GROUP BY creater_id
) pr ON u.id = pr.creater_id
LEFT JOIN (
    SELECT creater_id, COUNT(*) AS cnt
    FROM issues
    WHERE repo_id = <repo_id>
      AND date(create_time) BETWEEN '<start_date>' AND '<end_date>'
    GROUP BY creater_id
) iss ON u.id = iss.creater_id
WHERE pr.creater_id IS NOT NULL OR iss.creater_id IS NOT NULL
ORDER BY total DESC;
```

---

## 2. PR 创建数排名

```sql
SELECT
    u.user_name,
    COUNT(*) AS pr_created,
    SUM(CASE WHEN p.status = 'merged' THEN 1 ELSE 0 END) AS merged_count,
    ROUND(CAST(SUM(CASE WHEN p.status = 'merged' THEN 1 ELSE 0 END) AS REAL) / NULLIF(COUNT(*), 0) * 100, 1) AS merge_rate_pct
FROM pulls p
JOIN users u ON p.creater_id = u.id
WHERE p.repo_id = <repo_id>
  AND date(p.create_time) BETWEEN '<start_date>' AND '<end_date>'
GROUP BY p.creater_id
ORDER BY pr_created DESC;
```

---

## 3. Issue 创建数排名

```sql
SELECT
    u.user_name,
    COUNT(*) AS issue_created,
    SUM(CASE WHEN i.status = 'close' THEN 1 ELSE 0 END) AS closed_count
FROM issues i
JOIN users u ON i.creater_id = u.id
WHERE i.repo_id = <repo_id>
  AND date(i.create_time) BETWEEN '<start_date>' AND '<end_date>'
GROUP BY i.creater_id
ORDER BY issue_created DESC;
```

---

## 4. PR 处理数排名

```sql
SELECT
    u.user_name,
    COUNT(*) AS processed_count
FROM pulls p
JOIN users u ON p.processor_id = u.id
WHERE p.repo_id = <repo_id>
  AND date(p.create_time) BETWEEN '<start_date>' AND '<end_date>'
GROUP BY p.processor_id
ORDER BY processed_count DESC;
```

---

## 5. Issue 处理数排名

```sql
SELECT
    u.user_name,
    COUNT(*) AS processed_count
FROM issues i
JOIN users u ON i.processor_id = u.id
WHERE i.repo_id = <repo_id>
  AND date(i.create_time) BETWEEN '<start_date>' AND '<end_date>'
GROUP BY i.processor_id
ORDER BY processed_count DESC;
```

---

## 6. 按标签统计 PR 排名

```sql
SELECT
    t.name AS tag,
    u.user_name,
    COUNT(*) AS pr_count
FROM pulls p
JOIN users u ON p.creater_id = u.id
JOIN pull_tags pt ON p.id = pt.pull_id
JOIN tags t ON pt.tag_id = t.id
WHERE p.repo_id = <repo_id>
  AND t.repo_id = <repo_id>
  AND date(p.create_time) BETWEEN '<start_date>' AND '<end_date>'
GROUP BY t.name, p.creater_id
ORDER BY t.name, pr_count DESC;
```

---

## 7. 按标签统计 Issue 排名

```sql
SELECT
    t.name AS tag,
    u.user_name,
    COUNT(*) AS issue_count
FROM issues i
JOIN users u ON i.creater_id = u.id
JOIN issue_tags it ON i.id = it.issue_id
JOIN tags t ON it.tag_id = t.id
WHERE i.repo_id = <repo_id>
  AND t.repo_id = <repo_id>
  AND date(i.create_time) BETWEEN '<start_date>' AND '<end_date>'
GROUP BY t.name, i.creater_id
ORDER BY t.name, issue_count DESC;
```

---

## 8. 标签列表（确认仓库有哪些标签）

```sql
SELECT t.name, COUNT(pt.pull_id) + COUNT(it.issue_id) AS usage_count
FROM tags t
LEFT JOIN pull_tags pt ON t.id = pt.tag_id
LEFT JOIN issue_tags it ON t.id = it.tag_id
WHERE t.repo_id = <repo_id>
GROUP BY t.id
ORDER BY usage_count DESC;
```

---

## 9. 徽章生成

Agent 应根据仓库实际数据自行设计徽章，而非套用预定义列表。

### 流程

1. 执行查询 1-8 获取排名和标签数据
2. 从数据中识别值得表彰的模式（活跃度、质量、持续性、领域专精等）
3. 为每种模式设计徽章：命名、设定条件、确定获奖者

### 勘探查询

以下附加查询用于提取维度数据，帮助发现模式。

#### 用户综合画像

```sql
SELECT
    u.user_name,
    COUNT(DISTINCT p.id) AS pr_created,
    SUM(CASE WHEN p.status = 'merged' THEN 1 ELSE 0 END) AS pr_merged,
    COUNT(DISTINCT i.id) AS issue_created,
    COUNT(DISTINCT proc_pr.id) AS pr_processed,
    COUNT(DISTINCT proc_is.id) AS issue_processed
FROM users u
LEFT JOIN pulls p ON u.id = p.creater_id AND p.repo_id = <repo_id>
  AND date(p.create_time) BETWEEN '<start_date>' AND '<end_date>'
LEFT JOIN issues i ON u.id = i.creater_id AND i.repo_id = <repo_id>
  AND date(i.create_time) BETWEEN '<start_date>' AND '<end_date>'
LEFT JOIN pulls proc_pr ON u.id = proc_pr.processor_id AND proc_pr.repo_id = <repo_id>
  AND date(proc_pr.create_time) BETWEEN '<start_date>' AND '<end_date>'
LEFT JOIN issues proc_is ON u.id = proc_is.processor_id AND proc_is.repo_id = <repo_id>
  AND date(proc_is.create_time) BETWEEN '<start_date>' AND '<end_date>'
GROUP BY u.id
HAVING pr_created > 0 OR issue_created > 0 OR pr_processed > 0 OR issue_processed > 0
ORDER BY (pr_created + pr_merged + issue_created + pr_processed + issue_processed) DESC;
```

#### 新人发现

```sql
SELECT
    u.user_name,
    MIN(MIN(p.create_time), MIN(i.create_time)) AS first_activity,
    COALESCE(pr.cnt, 0) + COALESCE(iss.cnt, 0) AS period_activity
FROM users u
LEFT JOIN pulls p ON u.id = p.creater_id AND p.repo_id = <repo_id>
LEFT JOIN issues i ON u.id = i.creater_id AND i.repo_id = <repo_id>
LEFT JOIN (
    SELECT creater_id, COUNT(*) AS cnt FROM pulls
    WHERE repo_id = <repo_id> AND date(create_time) BETWEEN '<start_date>' AND '<end_date>'
    GROUP BY creater_id
) pr ON u.id = pr.creater_id
LEFT JOIN (
    SELECT creater_id, COUNT(*) AS cnt FROM issues
    WHERE repo_id = <repo_id> AND date(create_time) BETWEEN '<start_date>' AND '<end_date>'
    GROUP BY creater_id
) iss ON u.id = iss.creater_id
WHERE first_activity IS NOT NULL
GROUP BY u.id
HAVING date(first_activity) BETWEEN '<start_date>' AND '<end_date>'
ORDER BY period_activity DESC;
```

#### 活跃持续性（按周）

```sql
SELECT
    u.user_name,
    COUNT(DISTINCT week) AS active_weeks,
    MIN(week) AS first_week,
    MAX(week) AS last_week
FROM (
    SELECT creater_id, strftime('%Y-%W', create_time) AS week FROM pulls
    WHERE repo_id = <repo_id> AND date(create_time) BETWEEN '<start_date>' AND '<end_date>'
    UNION
    SELECT creater_id, strftime('%Y-%W', create_time) FROM issues
    WHERE repo_id = <repo_id> AND date(create_time) BETWEEN '<start_date>' AND '<end_date>'
) activity
JOIN users u ON activity.creater_id = u.id
GROUP BY u.id
HAVING active_weeks >= 2
ORDER BY active_weeks DESC;
```

#### 标签贡献模板

先执行查询 8 获取标签列表，再逐个代入：

```sql
-- 将 <tag_name> 替换为查询 8 返回的实际标签名
SELECT u.user_name, COUNT(*) AS tag_contribution
FROM users u
LEFT JOIN (
    SELECT creater_id FROM pulls p
    JOIN pull_tags pt ON p.id = pt.pull_id
    JOIN tags t ON pt.tag_id = t.id
    WHERE p.repo_id = <repo_id> AND t.repo_id = <repo_id>
      AND t.name = '<tag_name>'
      AND date(p.create_time) BETWEEN '<start_date>' AND '<end_date>'
) tag_pr ON u.id = tag_pr.creater_id
LEFT JOIN (
    SELECT creater_id FROM issues i
    JOIN issue_tags it ON i.id = it.issue_id
    JOIN tags t ON it.tag_id = t.id
    WHERE i.repo_id = <repo_id> AND t.repo_id = <repo_id>
      AND t.name = '<tag_name>'
      AND date(i.create_time) BETWEEN '<start_date>' AND '<end_date>'
) tag_iss ON u.id = tag_iss.creater_id
WHERE tag_pr.creater_id IS NOT NULL OR tag_iss.creater_id IS NOT NULL
GROUP BY u.id
ORDER BY tag_contribution DESC;
```

### 设计原则

- 徽章名称从数据特征派生（如标签名、行为类型），不套用固定模板
- 阈值参考数据分布（取前 N 名、高于均值、或连续 N 周活跃等）
- 每个徽章至少 1 人获得；无人达标则不发
- 在报告中说明每个徽章的数据依据

---

## 报告组装清单

**按以下顺序逐项执行查询、填入模板。每完成一项打勾，最终输出前核对所有 ✅ 是否齐全。**

### 排行榜数据

- [ ] **综合贡献排名** → 执行查询 1，取全部结果
- [ ] **PR 创建数排名** → 执行查询 2，取前 10 名
- [ ] **Issue 创建数排名** → 执行查询 3，取前 10 名
- [ ] **PR 处理数排名** → 执行查询 4，取前 10 名（结果为空则跳过此项）
- [ ] **Issue 处理数排名** → 执行查询 5，取前 10 名（结果为空则跳过此项）

### 细分领域排行（仅当有标签数据时）

- [ ] **标签列表** → 执行查询 8，确认仓库有哪些标签
- [ ] **按标签 PR 排名** → 执行查询 6，有标签数据时执行
- [ ] **按标签 Issue 排名** → 执行查询 7，有标签数据时执行
- [ ] **按标签汇总填充** → 填入模板的「细分领域排行」章节

### 徽章数据

- [ ] 执行查询 1-8 和勘探查询获取完整数据画像
- [ ] 分析数据，识别值得表彰的模式
- [ ] 为每种模式设计徽章（名称、条件、获奖者），附数据依据
- [ ] 若全维度均无突出数据，输出"本期无人获得徽章"

### 最终输出核对

- [ ] 报告包含「徽章展示」「贡献排行榜」「细分领域排行」三个主要章节
- [ ] 徽章表格不为空（即使无徽章，也显示"本期无人获得徽章"）
- [ ] 排行榜表格有列名
- [ ] 报告头部包含 owner/repo、分析期间、生成时间
