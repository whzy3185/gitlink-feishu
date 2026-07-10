# gitlink-competition-manager · 端到端示例

## 场景
批量创建编程竞赛队伍仓库、初始化题目与权限。

## 前置条件

- 已安装 gitlink-cli（`npm install -g @gitlink-ai/cli` 或 `go build`）
- 已登录：`gitlink-cli auth login`（平台命令需认证）
- 目标仓库：`--owner <owner> --repo <repo>`（git 仓库内可自动解析）

## 分步操作

```bash
gitlink-cli user +info --login <member_login_name> --format json
gitlink-cli issue +create
gitlink-cli issue +list --state open --owner <owner> --repo <main_repo> --format json
gitlink-cli pr +list --state open --owner <org> --repo <team_repo> --format json
gitlink-cli pr +list --state merged --owner <org> --repo <team_repo> --format json
gitlink-cli pr +view --id <pr_id> --owner <org> --repo <team_repo> --format json
```

## 输出示例

命令返回统一 envelope：
```json
{"ok":true,"data":{ ... }}
```

## 命令速览

`gitlink-cli user +info --login <member_login_name> --format json` | `gitlink-cli issue +create` | `gitlink-cli issue +list --state open` | `gitlink-cli pr +list --state open`

