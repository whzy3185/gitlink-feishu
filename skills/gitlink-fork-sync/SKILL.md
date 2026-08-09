---
name: gitlink-fork-sync
version: 1.0.0
description: "Fork 与上游仓库同步助手：检测 fork 与上游的分支落后/分叉状态，给出安全的同步方案并复核结果。当用户需要同步 fork、更新 fork、检查 fork 是否落后上游、rebase 分支到最新 master、贡献前对齐上游时触发。"
metadata:
  requires:
    bins: ["gitlink-cli", "git"]
  cliHelp: "gitlink-cli branch --help"
---

# gitlink-fork-sync（Fork 同步助手）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 本 Skill 涉及对用户 fork 的 git push 写操作，执行任何推送前必须向用户展示同步计划并获得确认；禁止在未确认时对 fork 的 master 执行 force push。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

---

## 功能概述

1. **落后检测** — 用平台 API 比对 fork 与上游各分支的 HEAD commit，判断 fork 是否落后/分叉，不依赖本地 clone
2. **同步方案** — 按分支状态给出确定性的同步命令（fast-forward / rebase / 保留分叉），标注每条命令的风险
3. **执行与复核** — 用户确认后执行同步，并再次通过 API 比对 HEAD 验证同步生效

## 使用场景

- "帮我把 fork 同步到上游最新"
- "我的 fork 落后官方仓库多少？"
- "提 PR 前把我的分支 rebase 到最新 master"

## 执行步骤

### 第 1 步：定位 fork 与上游

若用户未指定，从当前目录 git remote 解析：`origin` 为 fork、`upstream` 为上游；缺 `upstream` 时用平台 fork 关系确认上游后补加 remote。

```bash
git remote -v
git remote add upstream https://www.gitlink.org.cn/<上游owner>/<repo>.git   # 缺失时
```

### 第 2 步：API 比对分支 HEAD（无需本地拉取）

```bash
gitlink-cli branch +list --owner <上游owner> --repo <repo> --format json
gitlink-cli branch +list --owner <fork-owner> --repo <repo> --format json
```

取两侧同名分支的 `commit_id` 字段比对，逐分支判定（确定性规则）：

| 状态 | 判定 | 建议动作 |
|------|------|----------|
| 一致 | 两侧 commit_id 相同 | 无需操作 |
| 落后 | fork 的 HEAD 是上游 HEAD 的祖先 | fast-forward |
| 分叉 | 互不为祖先 | rebase（feature 分支）或保留（fork 的 master 有独立提交时**必须**先问用户） |

祖先关系用本地 git 验证：`git merge-base --is-ancestor <fork-sha> <upstream-sha>`（需先 `git fetch upstream origin`）。

### 第 3 步：生成同步计划并请求确认

向用户列出将执行的每条命令后再执行。典型 fast-forward 同步：

```bash
git fetch upstream
git checkout master
git merge --ff-only upstream/master
git push origin master
```

feature 分支 rebase（用于刷新已提 PR 的分支）：

```bash
git checkout <feature-branch>
git rebase upstream/master
git push --force-with-lease origin <feature-branch>   # 仅限自己的 feature 分支
```

### 第 4 步：复核

重新执行第 2 步的两条 `branch +list`，确认目标分支两侧 `commit_id` 一致（或 feature 分支已基于上游最新 HEAD），并把前后对比写入报告。

## 输出格式

```markdown
# Fork 同步报告：<fork-owner>/<repo> ⇆ <上游owner>/<repo>
| 分支 | 上游 HEAD | fork HEAD | 状态 | 动作 |
|------|-----------|-----------|------|------|
| master | 9749a4c | 604303e | 落后 | fast-forward ✔ |
- 复核：同步后 master 两侧 commit_id 一致
```

## 注意事项

- 平台的跨 fork compare 端点（`from=master&to=<fork>:master`）当前返回「获取对比信息失败」，跨仓库比对必须走两次 `branch +list` + 本地 `git merge-base`，不要依赖 compare 命令
- `branch +list` 的 `commit_id` 为完整 SHA，比对时不要截断后再比较
- fork 的 master 含独立提交时 fast-forward 会失败——此时降级为报告分叉状态并询问用户，禁止自作主张 force push
- `--force-with-lease` 只用于用户自己的 feature 分支；受保护分支（`protected: true`）直接跳过并在报告中标注
