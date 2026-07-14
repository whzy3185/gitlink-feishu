# gitlink-scholar-profile · 端到端示例

## 场景
跨仓库聚合学者/团队科研产出，生成影响力雷达 + 代表性成果。

## 前置条件

- 已安装 gitlink-cli（`npm install -g @gitlink-ai/cli` 或 `go build`）
- 已登录：`gitlink-cli auth login`（平台命令需认证）
- 目标仓库：`--owner <owner> --repo <repo>`（git 仓库内可自动解析）

## 分步操作

```bash
gitlink-cli user +info --login <user_input> --format json
gitlink-cli search +users -k <user_input> --format json
gitlink-cli user +info --login <resolved_login> --format json
gitlink-cli org +info --login <user_input> --format json
gitlink-cli search +repos -k <user_input> --format json
gitlink-cli user +info --login <login> --format json
```

## 输出示例

命令返回统一 envelope：
```json
{"ok":true,"data":{ ... }}
```

## 命令速览

`gitlink-cli user +info --login <user_input> --format json` | `gitlink-cli search +users -k <user_input> --format json` | `gitlink-cli user +info --login <resolved_login> --format json` | `gitlink-cli org +info --login <user_input> --format json`

