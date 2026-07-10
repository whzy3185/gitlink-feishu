# gitlink-issueops · 端到端示例

## 场景
创建 Issue 即经 webhook 触发 Agent 自动处理（事件驱动）。

## 前置条件

- 已安装 gitlink-cli（`npm install -g @gitlink-ai/cli` 或 `go build`）
- 已登录：`gitlink-cli auth login`（平台命令需认证）
- 目标仓库：`--owner <owner> --repo <repo>`（git 仓库内可自动解析）

## 分步操作

```bash
gitlink-cli webhook +create --owner <you> --repo <repo>
gitlink-cli webhook +tasks --owner <you> --repo <repo> -i <webhook_id> --format json
gitlink-cli issue +comment --owner <you> --repo <repo> --number <编号> --body "<结果 + 处理链说明>"
gitlink-cli label +create --owner <you> --repo <repo> -n "agent:done" -c "#22C55E"
```

## 输出示例

命令返回统一 envelope：
```json
{"ok":true,"data":{ ... }}
```

## 命令速览

`gitlink-cli webhook +create` | `gitlink-cli webhook +tasks` | `gitlink-cli issue +comment` | `gitlink-cli label +create`

