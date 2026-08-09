# gitlink-branch-hygiene 使用示例

> 触发方式：自然语言自动匹配
> 日期：2026-06-11
> Agent 平台：Codex
> 验证状态：✅ 通过（命令全部真实执行，完成了分支分类和可删除候选识别）

---

## 用户输入

```text
帮我看看 Gitlink/gitlink-cli 现在有哪些分支可以安全清理，先不要真的删除，给我一个治理计划。
```

## Skill 触发

Agent 根据“分支”“清理”“治理计划”“先不要真的删除”等关键词，自动匹配 `gitlink-branch-hygiene`，并按只读优先策略执行。

---

## 执行过程摘要

### Step 1：读取分支列表

```bash
gitlink-cli branch +list --owner Gitlink --repo gitlink-cli --format json
```

得到 6 条分支：

| 分支 | 是否保护 | 是否有关联 PR | 最近提交 |
|------|----------|---------------|----------|
| master | 否 | 是 | 2天前 |
| fix/pr-review-journal-sync | 否 | 否 | 9天前 |
| test/slash-1778992192 | 是 | 否 | 25天前 |
| main | 否 | 否 | 25天前 |
| test/branch-skill-1778991125 | 是 | 否 | 25天前 |
| test/unprotect-1778991372 | 是 | 否 | 25天前 |

### Step 2：读取开放 PR，避免误删协作分支

```bash
gitlink-cli pr +list --owner Gitlink --repo gitlink-cli --state open --limit 50 --format json
```

结果：当前仓库存在开放 PR，默认分支 `master` 仍在被协作流程使用，不能纳入删除候选。

### Step 3：检查无 PR 分支与默认分支的差异

先检查 `fix/pr-review-journal-sync`：

```bash
gitlink-cli compare +view --owner Gitlink --repo gitlink-cli --head fix/pr-review-journal-sync --base master --format json
```

返回：

```text
[-2] 分支内容相同，无需创建合并请求
```

结论：该分支与 `master` 已无内容差异，可视为零差异分支。

再检查 `main`：

```bash
gitlink-cli compare +view --owner Gitlink --repo gitlink-cli --head main --base master --format json
```

同样返回零差异，但由于 `main` 属于主分支别名，不直接列入删除候选。

---

## 生成的治理计划

```markdown
# Gitlink/gitlink-cli 分支治理计划

> 生成时间：2026-06-11
> 默认分支：master

## 一、分支概览

- 总分支数：6
- 受保护分支：3
- 可直接保留：5
- 可删除候选：1

## 二、保留分支

| 分支 | 原因 |
|------|------|
| master | 默认分支，持续承载协作流程 |
| main | 与 master 零差异，但属于主分支别名，默认保留 |
| test/slash-1778992192 | 受保护分支，不做自动清理 |
| test/branch-skill-1778991125 | 受保护分支，不做自动清理 |
| test/unprotect-1778991372 | 受保护分支，不做自动清理 |

## 三、可删除候选

| 分支 | 原因 | 删除前最后确认 |
|------|------|----------------|
| fix/pr-review-journal-sync | 无开放 PR、未受保护、距今 9 天、与 master 零差异 | 确认没有外部流程仍引用该分支后，可执行 `branch +delete` |
```

---

## 验证结论

| 检查项 | 结果 |
|--------|------|
| Skill 自动触发 | ✅ |
| 能识别默认分支和保护分支 | ✅ |
| 能把 `main` 识别为主分支别名而非误删对象 | ✅ |
| 能识别无 PR 分支 | ✅ |
| 能正确把 compare 的 `-2` 解释为“零差异” | ✅ |
| 能输出只读的治理计划而不直接执行删除 | ✅ |
