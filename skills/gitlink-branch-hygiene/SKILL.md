---
name: gitlink-branch-hygiene
version: 1.0.0
description: "分支治理与清理建议：识别默认分支、保护分支、无差异旧分支、无 PR 分支和潜在可删除分支，生成安全的分支治理计划。当用户需要整理仓库分支、降低分支噪音、准备清理旧分支时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli branch --help"
---

# gitlink-branch-hygiene（分支治理与清理建议）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — `branch +delete` 是破坏性操作，必须先输出清理计划，再在用户明确确认后逐条删除。**
**CRITICAL — `compare +view` / `compare +files` 返回 `[-2] 分支内容相同，无需创建合并请求` 时，应将其解释为“分支与基线无差异”，而不是分析失败。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)。
> **执行样例：** 参见 [`examples/gitlink-cli-branch-hygiene.md`](examples/gitlink-cli-branch-hygiene.md)

---

## 功能概述

本 Skill 解决维护者日常很常见、但很容易因为怕误删而长期拖着不做的事情：分支治理。

它的目标不是“看到旧分支就删”，而是先把分支分成三类：

1. **保留**：默认分支、主分支别名、受保护分支、有开放 PR 的分支
2. **人工确认**：无 PR 但可能还有内容差异、名称特殊、仍有保留价值的分支
3. **可删除候选**：无 PR、无保护、与默认分支无差异、明显完成历史使命的分支

---

## 工作流：生成分支治理计划

### Step 1：获取全部分支

```bash
gitlink-cli branch +list --owner <owner> --repo <repo> --format json
```

重点字段：

- `name`
- `default_branch`
- `protected`
- `has_pull_request`
- `commit_time`
- `commit_time_from_now`

### Step 2：获取开放 PR，建立“正在协作中的分支”集合

```bash
gitlink-cli pr +list --owner <owner> --repo <repo> --state open --limit 50 --format json
```

规则：

- PR 列表可能混入非 open 状态，必要时按返回字段再次过滤
- 只要某个分支仍被开放 PR 使用，就不能直接列入删除候选

### Step 3：对可疑分支做差异检查

对满足以下条件的分支，进一步执行 compare：

- 不是默认分支
- 不是 `main` / `master` 主分支别名
- 没有关联开放 PR
- 当前未受保护

```bash
gitlink-cli compare +view --owner <owner> --repo <repo> --head <branch> --base <default-branch> --format json
```

必要时补充文件级差异：

```bash
gitlink-cli compare +files --owner <owner> --repo <repo> --head <branch> --base <default-branch> --format json
```

判断规则：

- 返回 `[-2] 分支内容相同，无需创建合并请求`：记为“零差异”
- 返回正常 compare 数据：说明分支仍有内容差异，放入“人工确认”

### Step 4：分类输出

#### 一定保留

- 默认分支
- `main` / `master` 主分支别名
- `protected=true`
- `has_pull_request=true` 或仍被开放 PR 引用
- `release/`、`hotfix/`、`support/` 等维护分支

#### 人工确认

- 无 PR 但仍有内容差异
- 名称带业务语义且最近仍有提交
- 用户明确说明要保留的实验分支

#### 可删除候选

- 非默认分支
- 非 `main` / `master`
- 无开放 PR
- 未受保护
- 与默认分支零差异
- 最近提交时间已明显落后，且名称不属于长期维护约定

### Step 5：用户确认后再执行删除

```bash
gitlink-cli branch +delete --owner <owner> --repo <repo> --name <branch>
```

执行策略：

- 一次删除一条，不要批量盲删
- 每删除一条都重新确认剩余候选集合
- 删除后重新跑 `branch +list`，确认结果已生效

---

## 输出模板

```markdown
# 分支治理计划

> 仓库：{{owner}}/{{repo}}
> 默认分支：{{default_branch}}
> 生成时间：{{now}}

## 一、分支概览

- 总分支数：{{total}}
- 受保护分支：{{protected_count}}
- 开放 PR 使用中的分支：{{active_pr_count}}
- 可删除候选：{{delete_candidate_count}}

## 二、保留分支

| 分支 | 原因 |
|------|------|
| {{branch}} | {{reason}} |

## 三、人工确认

| 分支 | 原因 | 下一步 |
|------|------|--------|
| {{branch}} | {{reason}} | {{next_action}} |

## 四、可删除候选

| 分支 | 原因 | 删除前最后确认 |
|------|------|----------------|
| {{branch}} | {{reason}} | {{check}} |
```

---

## 关键注意事项

### 1. `main` 不等于“可删的非默认分支”

很多仓库虽然默认分支是 `master`，但仍会保留 `main` 作为兼容分支或迁移别名。遇到这种情况，应默认保留，除非维护者明确要求清理。

### 2. 保护分支优先于差异判断

即使 compare 显示和默认分支零差异，只要分支受保护，也应先归入“保留”或“人工确认”。

### 3. compare 的 `[-2]` 不是错误

GitLink compare 在零差异场景会返回错误码 `-2`，语义其实是“无差异”。Skill 必须把这个结果转成正常判断，避免误报。

### 4. 删除动作必须逐条确认

分支治理的价值来自“减少噪音”，不是“激进清空”。凡是有疑问的分支，都先留在“人工确认”而不是强删。

---

## 最佳实践

- 先做只读报告，再做删除
- 对每个候选分支写出“为什么删”和“删前还要看什么”
- 对有 PR 但长期未推进的分支，优先建议“催进度 / 关闭 PR”，不是直接删分支
- 报告建议用 Markdown 输出，便于直接贴到团队群或维护者交接文档中
