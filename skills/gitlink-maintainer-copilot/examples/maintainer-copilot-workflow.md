# Maintainer Copilot Workflow

本示例展示 Agent 如何使用 `gitlink-maintainer-copilot` 生成维护者驾驶舱，并在用户确认后创建治理 Issue。

## 1. 用户请求

```text
帮我给 Gitlink/gitlink-cli 生成一个维护者驾驶舱，并准备一个可落地的治理 Issue。
```

## 2. Agent 声明只读采集

```text
我会先使用 gitlink-cli 只读采集仓库、Issue、PR、Release、CI、Commit 和贡献者数据。不会写入 GitLink；如果需要创建治理 Issue，我会先展示完整草稿并等待你确认。
```

## 3. 采集证据

```bash
gitlink-cli repo +info --owner Gitlink --repo gitlink-cli --format json
gitlink-cli issue +list --owner Gitlink --repo gitlink-cli --state open --limit 100 --format json
gitlink-cli issue +list --owner Gitlink --repo gitlink-cli --state closed --limit 100 --format json
gitlink-cli pr +list --owner Gitlink --repo gitlink-cli --state open --limit 100 --format json
gitlink-cli pr +list --owner Gitlink --repo gitlink-cli --state merged --limit 100 --format json
gitlink-cli pr +list --owner Gitlink --repo gitlink-cli --state closed --limit 100 --format json
gitlink-cli release +list --owner Gitlink --repo gitlink-cli --format json
gitlink-cli ci +builds --owner Gitlink --repo gitlink-cli --format json
gitlink-cli api GET /v1/Gitlink/gitlink-cli/commits --query 'page=1&limit=100' --format json
gitlink-cli api GET /v1/Gitlink/gitlink-cli/contributors/stat --format json
gitlink-cli api GET /Gitlink/gitlink-cli/languages --format json
gitlink-cli api GET /Gitlink/gitlink-cli/readme --format json
```

## 4. 输出驾驶舱报告

报告结构：

```markdown
# GitLink Maintainer Copilot Report

## 维护状态总览
...

## Evidence Pack
...

## 关键风险
...

## 匹配 Playbooks
...

## 30/60/90 天治理计划
...

## Governance Issue 草稿
...

## 数据缺失与可信度
...
```

完整样例见 [`sample-dashboard-report.md`](sample-dashboard-report.md)。

## 5. 请求写入确认

Agent 展示完整治理 Issue 草稿后询问：

```text
以上是将创建的治理 Issue 内容。请明确回复“确认创建治理 Issue”，我才会执行写入。
```

## 6. 用户确认后创建 Issue

```bash
gitlink-cli issue +create --owner Gitlink --repo gitlink-cli \
  --title "chore: 建立 gitlink-cli 项目维护治理计划" \
  --body "<完整治理 Issue 正文>"
```

## 7. 完成后回报

```text
治理 Issue 已创建。后续维护者可以按 30/60/90 天计划逐项勾选，并在下次运行 Maintainer Copilot 时对比进度。
```
