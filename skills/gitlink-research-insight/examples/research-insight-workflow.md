# gitlink-research-insight · 端到端示例

## 场景
四维科研评分（可复现性/活跃度/引用价值/协作健康）+ 协作图谱（S1）。

## 前置条件

- 已安装 gitlink-cli（`npm install -g @gitlink-ai/cli` 或 `go build`）
- 已登录：`gitlink-cli auth login`（平台命令需认证）
- 目标仓库：`--owner <owner> --repo <repo>`（git 仓库内可自动解析）

## 分步操作

```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
gitlink-cli repo +languages --owner <owner> --repo <repo> --format json
gitlink-cli repo +contributors --owner <owner> --repo <repo> --format json
gitlink-cli repo +readme --owner <owner> --repo <repo>
gitlink-cli file +get --owner <owner> --repo <repo> --path LICENSE
gitlink-cli issue +list --owner <owner> --repo <repo> --format json # Issue 列表可用
```

## 输出示例

命令返回统一 envelope：
```json
{"ok":true,"data":{ ... }}
```

工作流型 Skill 的输出符合其 SKILL.md「输出模板」（含评分/判定/报告关键字段）。

## 命令速览

`gitlink-cli repo +info` | `gitlink-cli repo +languages` | `gitlink-cli repo +contributors` | `gitlink-cli repo +readme`

