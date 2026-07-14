# 每日简报完整工作流示例

**场景**：项目负责人早上想快速了解「昨天项目发生了什么」，AI 采集多源数据生成一份简报。

## 前置条件

- `gitlink-cli` 已登录
- 在目标仓库目录下（自动解析 owner/repo），或手动传 `--owner/--repo`

## 工作流步骤

### Step 1：并行采集多源数据

```bash
# 近期 Issue
gitlink-cli issue +list --state open --format json

# 近期 PR
gitlink-cli pr +list --state open --format json

# CI 构建状态
gitlink-cli ci +builds --owner myorg --repo myproject --format json

# 通知消息（Raw API）
gitlink-cli api GET "users/zhangsan/messages.json"

# 活跃度统计（Raw API）
gitlink-cli api GET "users/zhangsan/statistics/activity.json"
```

### Step 2：AI 分类聚合

输出示例：

```markdown
# 📰 项目简报 — myorg/myproject（2026-06-23）

## 🔴 需立即关注
1. ❌ CI 构建 #156 失败（分支 master）— test_phase 报错：连接超时
2. 🔔 @你 在 PR #88：「请帮忙看下认证模块的重试逻辑」

## 🟢 今日新增
- **新 Issue**：4 个（bug 2 / enhancement 1 / question 1）
  - #201 登录页面 500 错误
  - #202 支持导出 CSV
- **新 PR**：2 个
  - #90 feat: 增加批量导入

## 🔵 进行中
- 待 Review 的 PR：#88、#87
- 有更新的 Issue：#198、#195

## 📊 健康指标
- 开放 Issue：23（较昨日 +3）
- 开放 PR：5
- 近期活跃度：94

---
*由 gitlink-digest Skill 于 2026-06-23 09:15 生成*
```

### Step 3（可选）：发布简报

```bash
# 发布为 Wiki 页面（⚠️ 写操作，需确认）
gitlink-cli wiki +create --name "Daily-2026-06-23" --content "<简报内容>"
```

---

## 完整命令速览

```bash
gitlink-cli issue +list --state open --format json
gitlink-cli pr +list --state open --format json
gitlink-cli ci +builds --owner <owner> --repo <repo> --format json
gitlink-cli api GET "users/<me>/messages.json"
gitlink-cli api GET "users/<me>/statistics/activity.json"
```

## 注意事项

- 纯只读聚合，安全可随时运行
- 通知/活跃度为 Raw API，首次使用带 `--debug` 确认路径
