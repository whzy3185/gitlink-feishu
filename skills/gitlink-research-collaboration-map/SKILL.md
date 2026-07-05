---
name: gitlink-research-collaboration-map
description: "科研协作画像与合作建议：分析 GitLink 科研仓库的贡献者、Issue、PR、语言和用户信息，识别核心维护者、潜在协作者、未响应研究问题和协作分工建议。用于课题组协作复盘、跨团队合作匹配和学生贡献画像。"
---

# gitlink-research-collaboration-map

开始前必须先阅读 `../gitlink-shared/SKILL.md`，确认认证、权限、安全规则和 GitLink API 注意事项。

## 安全规则

- 只读执行，不自动指派 Issue、不邀请成员。
- 所有 GitLink 操作必须使用 `gitlink-cli`。
- 所有命令使用 `--format json`。
- 不输出 Token、Cookie 或认证 Header。
- 不输出私人联系方式或敏感个人信息。
- 协作建议必须基于公开仓库数据，避免做人身评价。

## 工作流

1. 获取仓库、语言、贡献者：

```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
gitlink-cli repo +languages --owner <owner> --repo <repo> --format json
gitlink-cli repo +contributors --owner <owner> --repo <repo> --format json
```

如 shortcut 不可用，回退：

```bash
gitlink-cli api GET /<owner>/<repo>/languages --format json
gitlink-cli api GET /<owner>/<repo>/contributors --format json
```

2. 获取 Issue 和 PR：

```bash
gitlink-cli issue +list --owner <owner> --repo <repo> --state open --format json
gitlink-cli issue +list --owner <owner> --repo <repo> --state closed --format json
gitlink-cli pr +list --owner <owner> --repo <repo> --state open --format json
gitlink-cli pr +list --owner <owner> --repo <repo> --state merged --format json
```

3. 对高频作者或贡献者，按需获取公开用户信息：

```bash
gitlink-cli user +info --login <login> --format json
```

4. 按 `references/profile-fields.md` 生成协作画像。

## 输出格式

```markdown
# 科研协作画像：<owner>/<repo>

## 协作结构

| 角色 | 候选人/账号 | 证据 | 建议 |
|---|---|---|---|
| 核心维护者 | ... | commits/issues/pr | ... |

## 未响应问题

| Issue/PR | 主题 | 等待动作 | 建议协作者 |
|---|---|---|---|

## 合作建议

1. ...
```

## 分析原则

- 贡献者画像使用“贡献事实”和“适合跟进的任务”，不要给人格标签。
- Issue/PR 作者、assignee、reviewer 字段可能缺失，缺失时不要强推断。
- 镜像仓库的 GitLink 协作数据可能很少，需要标注局限。
- 合作建议应给出证据：语言栈、历史贡献、Issue/PR 主题。
