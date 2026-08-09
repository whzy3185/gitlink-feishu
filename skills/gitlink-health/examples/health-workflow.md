# 项目健康度分析完整工作流示例

**场景**：项目维护者需要全面了解项目健康状况，生成健康度报告。

## 前置条件

- `gitlink-cli` 已安装并登录
- 仓库有一定数量的 Issue 和 PR 历史

## 工作流步骤

### Step 1：采集仓库数据

```bash
# 采集 PR 和 Issue 数据到 SQLite
gitlink-cli health +fetch --owner myorg --repo myproject
```

**输出示例：**
```
正在采集仓库数据...
✓ 获取仓库信息: myorg/myproject
✓ 采集 Issue 数据: 156 条
✓ 采集 PR 数据: 89 条
✓ 数据已保存到 ~/.agents/skills/gitlink-health/data/gitlink_health.db
```

### Step 2：查询关键指标

参考 `references/queries.md` 执行 SQL 查询：

```bash
# 查询 Issue 平均解决时长
sqlite3 ~/.agents/skills/gitlink-health/data/gitlink_health.db \
  "SELECT AVG(julianday(closed_at) - julianday(created_at)) as avg_days FROM issues WHERE closed_at IS NOT NULL"

# 查询 PR 合并率
sqlite3 ~/.agents/skills/gitlink-health/data/gitlink_health.db \
  "SELECT COUNT(CASE WHEN status='merged' THEN 1 END)*100.0/COUNT(*) as merge_rate FROM pulls"

# 查询贡献者活跃度
sqlite3 ~/.agents/skills/gitlink-health/data/gitlink_health.db \
  "SELECT author, COUNT(*) as pr_count FROM pulls GROUP BY author ORDER BY pr_count DESC LIMIT 10"
```

### Step 3：生成健康度报告

按照 `asset/health_report_template.md` 模板组装报告：

```markdown
## 🏥 项目健康度报告 — myorg/myproject

### 总体评分：⭐⭐⭐⭐ (4/5)

| 维度 | 状态 | 评分 | 建议 |
|------|:----:|:----:|------|
| 📖 文档 | ✅ | ☆☆☆☆☆ | README 完整，有 API 文档 |
| 📜 许可证 | ✅ | ☆☆☆☆☆ | Apache-2.0 |
| 🔧 CI/CD | ✅ | ☆☆☆☆☆ | Gitea Actions 配置完善 |
| 🐛 Issue 管理 | ⚠️ | ☆☆☆☆☆ | 平均解决时长 8.5 天，偏长 |
| 🔀 PR 活跃度 | ✅ | ☆☆☆☆☆ | 合并率 78%，活跃度良好 |
| 👥 贡献者 | ⚠️ | ☆☆☆☆☆ | 核心贡献者 3 人，较集中 |

### 关键发现
1. Issue 解决时长偏长，建议引入自动分拣
2. 贡献者集中度高，需吸引更多外部贡献者
3. CI 配置完善，构建成功率高
```

---

## 完整命令速览

```bash
gitlink-cli health +fetch --owner <owner> --repo <repo>
gitlink-cli health +fetch --owner <owner> --repo <repo> --max-pages 5
```
