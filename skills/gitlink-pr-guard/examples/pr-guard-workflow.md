# gitlink-pr-guard · 端到端示例

## 场景
PR 提交后跑完 5 步质量门禁并给出通过/拒绝判定（任务三）。

## 前置条件

- 已安装 gitlink-cli（`npm install -g @gitlink-ai/cli` 或 `go build`）
- 已登录：`gitlink-cli auth login`（平台命令需认证）
- 目标仓库：`--owner <owner> --repo <repo>`（git 仓库内可自动解析）

## 分步操作

```bash
gitlink-cli pr +view --id <pr_id> --format json
gitlink-cli pr +files --id <pr_id> --format json
gitlink-cli pr +diff --id <pr_id> --format json # 完整 diff
gitlink-cli pr +diff --id <pr_id> --stat # 仅统计摘要
gitlink-cli ci +builds --owner <owner> --repo <repo> --format json
gitlink-cli ci +log --build <build_number>
```

## 输出示例

命令返回统一 envelope：
```json
{"ok":true,"data":{ ... }}
```

工作流型 Skill 的输出符合其 SKILL.md「输出模板」（含评分/判定/报告关键字段）。

## 命令速览

`gitlink-cli pr +view --id <pr_id> --format json` | `gitlink-cli pr +files --id <pr_id> --format json` | `gitlink-cli pr +diff --id <pr_id> --format json # 完整 diff` | `gitlink-cli pr +diff --id <pr_id> --stat # 仅统计摘要`

