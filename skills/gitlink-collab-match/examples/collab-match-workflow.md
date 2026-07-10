# gitlink-collab-match · 端到端示例

## 场景
分析科研仓库技术缺口 + 候选人画像，智能匹配协作伙伴（S4）。

## 前置条件

- 已安装 gitlink-cli（`npm install -g @gitlink-ai/cli` 或 `go build`）
- 已登录：`gitlink-cli auth login`（平台命令需认证）
- 目标仓库：`--owner <owner> --repo <repo>`（git 仓库内可自动解析）

## 分步操作

```bash
gitlink-cli research +match --help"
```

## 输出示例

命令返回统一 envelope：
```json
{"ok":true,"data":{ ... }}
```

## 命令速览

`gitlink-cli research +match --help"`

