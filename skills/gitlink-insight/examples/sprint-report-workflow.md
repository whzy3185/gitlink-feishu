# Sprint 周报生成工作流示例

**场景**：每周五下午，项目 Maintainer 需要生成团队周报。

## Step 1：获取本周完成的工作

```bash
# 获取本周合并的 PR（按状态过滤）
gitlink-cli pr +list --state merged --format json

# 获取本周关闭的 Issue
gitlink-cli issue +list --state closed --format json

# 获取本周创建的 Release
gitlink-cli release +list --format json
```

## Step 2：获取进行中的工作

```bash
# 获取所有开放 Issue
gitlink-cli issue +list --state open --format json

# 获取开放 PR
gitlink-cli pr +list --state open --format json
```

## Step 3：获取项目动态

```bash
gitlink-cli api GET /:owner/:repo/activity --format json
```

## Step 4：生成周报

将采集的数据整理为以下格式：

```markdown
## 📅 Sprint 周报 — Gitlink/forgeplus

📆 周期：2026-05-14 ~ 2026-05-20

### ✅ 本周完成
| 类型 | 数量 | 关键事项 |
|------|:----:|----------|
| Issue 关闭 | 12 | #234 用户权限修复、#235 性能优化 |
| PR 合并 | 8 | feat: 添加登录模块、fix: 修复内存泄漏 |
| Release | 1 | v2.3.0 发布 |

### 🚧 进行中
| Issue | 负责人 | 状态 |
|-------|:------:|:----:|
| #240 API 文档重构 | zhangsan | Review 中 |
| #241 数据导出功能 | lisi | 进行中 |

### 📊 本周统计
| 指标 | 本周 | 上周 | 环比 |
|------|:----:|:----:|:----:|
| 关闭 Issue | 12 | 8 | +50% |
| 合并 PR | 8 | 6 | +33% |
| 新增 Issue | 5 | 7 | -29% |
| 新增贡献者 | 2 | 1 | +100% |

### ⚠️ 关注事项
- #238 数据库迁移 PR 依赖基础设施组支持，已阻塞 3 天
```

---

## 自动化建议

每周固定时间运行时，可以：
1. 保存上一期报告的输出
2. 本期与上期数据做环比
3. 在项目 Wiki 中归档历史周报
4. 在团队 Issue 或 Discussion 中发布
