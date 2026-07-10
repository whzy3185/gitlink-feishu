# gitlink-stale-issue-manager · 端到端示例

## 场景
识别长期无活动 Issue，按过期等级标记/提醒/批量关闭。

## 前置条件

- 已安装 gitlink-cli（`npm install -g @gitlink-ai/cli` 或 `go build`）
- 已登录：`gitlink-cli auth login`（平台命令需认证）
- 目标仓库：`--owner <owner> --repo <repo>`（git 仓库内可自动解析）

## 分步操作

```bash
gitlink-cli issue +list --state open --owner <owner> --repo <repo> --page 1 --limit 20 --format json
gitlink-cli issue +comment
gitlink-cli issue +close
gitlink-cli issue +comment --number <issue_id> --owner <owner> --repo <repo> --body "<stale 提醒>"
gitlink-cli issue +close --number <issue_id> --owner <owner> --repo <repo>
```

## 输出示例

命令返回统一 envelope：
```json
{"ok":true,"data":{ ... }}
```

## 命令速览

`gitlink-cli issue +list --state open` | `gitlink-cli issue +comment` | `gitlink-cli issue +close` | `gitlink-cli issue +comment --number <issue_id>`

