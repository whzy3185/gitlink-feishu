# 维护者工作流 Skills

本次补充了两条面向仓库维护者的新 Skill，重点解决“交接信息散落”和“旧分支不敢清理”这两类高频但容易被忽略的问题。

## 新增内容

### 1. `gitlink-maintainer-handoff`

- 汇总站内消息、仓库健康度、开放 PR、开放 Issue 和最近发布
- 输出可直接交给下一位维护者的 Markdown 交接摘要
- 默认只读，适合值班交接、周报、比赛期间的维护汇总

### 2. `gitlink-branch-hygiene`

- 面向分支治理场景，识别默认分支、主分支别名、保护分支、无 PR 分支和可删除候选
- 把 compare 的零差异返回 `[-2] 分支内容相同，无需创建合并请求` 解释为正常治理信号，而不是失败
- 先输出计划，再要求用户确认删除，避免误删分支

## 文档与示例

- 新增 `skills/gitlink-maintainer-handoff/SKILL.md`
- 新增 `skills/gitlink-maintainer-handoff/examples/gitlink-cli-maintainer-handoff.md`
- 新增 `skills/gitlink-branch-hygiene/SKILL.md`
- 新增 `skills/gitlink-branch-hygiene/examples/gitlink-cli-branch-hygiene.md`
- 更新 `skills/README.md`

## 平台验证

本次示例在 Codex 中完成验证，触发方式为自然语言请求 + `gitlink-cli` 实际命令执行。Skill 本身仅依赖 Markdown 规则和标准 CLI 调用，也兼容 Claude Code、Cursor 等可读取 `SKILL.md` 的主流 Agent 平台。

## 本地验证命令

```bash
go run . user +me --format json
go run . api GET "users/Mengz/messages.json" --query "status=1&limit=20" --format json
go run . workflow +repo-report --owner Gitlink --repo gitlink-cli --lang zh-CN --format markdown
go run . workflow +pr-summary --owner Gitlink --repo gitlink-cli --number 209 --lang zh-CN --format markdown
go run . issue +list --owner Gitlink --repo gitlink-cli --state open --limit 10 --format json
go run . release +list --owner Gitlink --repo gitlink-cli --limit 3 --format json
go run . branch +list --owner Gitlink --repo gitlink-cli --format json
go run . compare +view --owner Gitlink --repo gitlink-cli --head fix/pr-review-journal-sync --base master --format json
go run . compare +view --owner Gitlink --repo gitlink-cli --head main --base master --format json
```
