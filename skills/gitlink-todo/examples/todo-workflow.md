# 我的待办完整工作流示例

**场景**：开发者上班第一件事，想知道「今天有哪些事在等我」。

## 前置条件

- `gitlink-cli` 已登录

## 工作流步骤

### Step 1：识别身份

```bash
gitlink-cli api GET "users/me" --format json
# 取 login = zhangsan
```

### Step 2：多维查询

```bash
# 分配给我的开放 Issue
gitlink-cli search +issues --assignee zhangsan --category opened

# @我的消息
gitlink-cli api GET "users/zhangsan/messages.json"

# 我的 PR 状态
gitlink-cli pr +list --format json
```

### Step 3：AI 生成待办清单

输出示例：

```markdown
# ✅ 我的待办 — zhangsan（2026-06-23）

## 🔴 紧急（2）
1. 🔔 Issue #188 @你：「认证模块的重试逻辑需要你确认」（停留 2 天）
2. 🔀 PR #90 你被指派 review，已等待 3 天

## 🟡 本周内（4）
- 分配给你的 Issue：
  - #180 修复搜索分页（优先级 high）
  - #175 补充 API 文档
- 待 Review 的 PR：#90、#87

## 🔵 你的 PR（2）
- #92 feat: 批量导入 — ✅ 已 review，待合并
- #88 fix: 登录超时 — ⏳ 等待 review（2 天）

---
*由 gitlink-todo Skill 生成，共 8 项待办*
```

### Step 4（可选）：批量处理

```bash
# 回复 @我 的消息
gitlink-cli issue +comment --number 188 --body "已确认，今天处理"

# 批量给待办加优先级标签（⚠️ 先 dry-run）
gitlink-cli issue +batch-update --ids 180,175 --tag-ids 3 --dry-run
```

---

## 完整命令速览

```bash
gitlink-cli api GET "users/me" --format json
gitlink-cli search +issues --assignee <me> --category opened
gitlink-cli api GET "users/<me>/messages.json"
gitlink-cli pr +list --format json
gitlink-cli issue +comment --number <n> --body "..."
gitlink-cli issue +batch-update --ids <ids> --tag-ids <ids> --dry-run
```

## 注意事项

- 核心只读汇总；批量写操作必须先确认或 `--dry-run`
- `--assignee` 需传 login/id，先用 `/users/me` 获取
