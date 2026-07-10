---
name: gitlink-research-progress-tracker
version: 0.1.0
description: "科研项目进度跟踪与预警：基于 GitLink Issue、PR、Release、贡献者和仓库更新时间生成课题组周报、停滞风险、开放问题队列和下一步建议。用于科研项目管理、中期检查和例会汇报。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli issue --help"
---

# gitlink-research-progress-tracker

开始前必须先阅读 `../gitlink-shared/SKILL.md`，确认认证、权限、安全规则和 GitLink API 注意事项。

## 安全规则

- 默认只读执行。
- 所有 GitLink 操作必须使用 `gitlink-cli`。
- 所有命令使用 `--format json`。
- 不输出 Token、Cookie 或认证 Header。
- 不自动关闭 Issue、不合并 PR、不发布周报评论。
- 如果用户要求发布周报到 Issue，先生成 dry-run 草稿并等待确认。

## 工作流

1. 获取仓库概览：

```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
```

2. 获取协作队列：

```bash
gitlink-cli issue +list --owner <owner> --repo <repo> --state open --format json
gitlink-cli issue +list --owner <owner> --repo <repo> --state closed --format json
gitlink-cli pr +list --owner <owner> --repo <repo> --state open --format json
gitlink-cli pr +list --owner <owner> --repo <repo> --state merged --format json
```

3. 获取版本和贡献者信息：

```bash
gitlink-cli release +list --owner <owner> --repo <repo> --format json
gitlink-cli repo +contributors --owner <owner> --repo <repo> --format json
```

如果当前 CLI 版本没有 `repo +contributors`，回退：

```bash
gitlink-cli api GET /<owner>/<repo>/contributors --format json
```

4. 按 `references/risk-model.md` 判断风险，并生成周报。

## 输出格式

```markdown
# 科研项目周报：<owner>/<repo>

## 本期概览

| 指标 | 数值 |
|---|---:|
| 开放 Issue | ... |
| 已关闭 Issue | ... |
| 开放 PR | ... |
| 已合并 PR | ... |
| Release 数 | ... |
| 贡献者 | ... |

## 风险预警

| 风险 | 等级 | 证据 | 建议 |
|---|---|---|---|

## 下一步建议

1. ...
```

## 分析原则

- 先报告事实，再给建议。
- 把“没有数据”和“数据为 0”区分开。
- 如果 PR/Issue 的 state 过滤不精确，根据返回字段二次归类，并在报告中说明。
- 重点识别阻塞项：长期开放 Issue、开放 PR 堆积、无 Release、近期无维护活动、单人维护风险。
