# AI 工作流完整工作流示例

**场景**：利用 AI 自动化工作流完成 Issue 分类、PR Review 和报告生成。

## 工作流步骤

### 工作流 A：Issue 自动分类

```bash
# 1. 获取开放的 Issue
gitlink-cli issue +list --state open --format json

# 2. 分析每个 Issue 并分类
# AI 根据关键词判断：bug → bug标签，feature → enhancement标签
gitlink-cli issue +view --number 200 --format json

# 3. 应用标签
gitlink-cli api POST /:owner/:repo/issues/:id --body '{"issue_tag_ids":[1]}'
```

### 工作流 B：生成 PR 摘要报告

```bash
# 生成 PR 审查摘要（只读模式）
gitlink-cli workflow +pr-summary --owner Gitlink --repo gitlink-cli --number 1 --format markdown
```

### 工作流 C：生成仓库报告

```bash
# 生成综合仓库报告
gitlink-cli workflow +repo-report --owner Gitlink --repo gitlink-cli --format markdown
```

**输出示例（Markdown）：**
```markdown
# 仓库报告 — Gitlink/gitlink-cli

## 基本信息
- 语言：Go
- 开放 Issue：12
- 开放 PR：3
- 最近贡献者：@dev1, @dev2, @dev3

## Issue 分布
- Bug：5
- Enhancement：4
- Question：3

## 建议
- 3 个 Issue 超过 30 天未响应，建议处理
```

### 工作流 D：Sprint 报告

```bash
# 获取 Issue 和 PR 统计
gitlink-cli issue +list --state open --format json
gitlink-cli issue +list --state closed --format json
gitlink-cli pr +list --state merged --format json

# 获取项目动态
gitlink-cli api GET /:owner/:repo/activity --format json
```

---

## 完整命令速览

```bash
gitlink-cli workflow +pr-summary --owner <owner> --repo <repo> --number <n> --format markdown
gitlink-cli workflow +repo-report --owner <owner> --repo <repo> --format markdown
gitlink-cli workflow +repo-report --owner <owner> --repo <repo> --format json
```
