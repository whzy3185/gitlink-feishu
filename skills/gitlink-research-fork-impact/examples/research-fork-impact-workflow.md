# gitlink-research-fork-impact · 端到端示例

## 场景
分析科研 fork 的改进方向与影响力，生成想法传播图谱。

## 前置条件

- 已安装 gitlink-cli（`npm install -g @gitlink-ai/cli` 或 `go build`）
- 已登录：`gitlink-cli auth login`（平台命令需认证）
- 目标仓库：`--owner <owner> --repo <repo>`（git 仓库内可自动解析）

## 分步操作

```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
gitlink-cli search +repos -k "<repo_name>" --format json
gitlink-cli pr +list --owner <original_owner> --repo <repo> --state merged format json
gitlink-cli pr +list --state merged --owner <original_owner> --repo <repo> --format json
gitlink-cli pr +list --state open --owner <original_owner> --repo <repo> --format json
```

## 输出示例

命令返回统一 envelope：
```json
{"ok":true,"data":{ ... }}
```

工作流型 Skill 的输出符合其 SKILL.md「输出模板」（含评分/判定/报告关键字段）。

## 命令速览

`gitlink-cli repo +info` | `gitlink-cli search +repos -k "<repo_name>" --format json` | `gitlink-cli pr +list` | `gitlink-cli pr +list --state merged`

