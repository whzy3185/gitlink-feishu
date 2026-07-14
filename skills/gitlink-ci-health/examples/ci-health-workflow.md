# gitlink-ci-health · 端到端示例

## 场景
巡检仓库 CI/CD 授权与构建成功率，生成 CI 健康度报告。

## 前置条件

- 已安装 gitlink-cli（`npm install -g @gitlink-ai/cli` 或 `go build`）
- 已登录：`gitlink-cli auth login`（平台命令需认证）
- 目标仓库：`--owner <owner> --repo <repo>`（git 仓库内可自动解析）

## 分步操作

```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
gitlink-cli ci +builds --owner <owner> --repo <repo> --format json
gitlink-cli ci +logs --owner <owner> --repo <repo> --build <build_id> --format json
```

## 输出示例

命令返回统一 envelope：
```json
{"ok":true,"data":{ ... }}
```

## 命令速览

`gitlink-cli repo +info` | `gitlink-cli ci +builds` | `gitlink-cli ci +logs`

