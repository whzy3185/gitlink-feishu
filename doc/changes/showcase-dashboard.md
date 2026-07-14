# 变更说明：Showcase Dashboard 交互式展示页

## 概述

新增 GitLink CLI Showcase Dashboard —— 一个基于 Go + HTML 的 Web 展示面板，展示 gitlink-cli 的模块功能，支持在线执行命令并查看结果。

## 功能

- **12 个模块卡片**：milestone、webhook、label、commit、wiki、file、member、watch、star、issue 批量操作、repo 批量操作、org 批量操作
- **在线执行**：每个卡片可输入参数，点击执行按钮直接调用 gitlink-cli 命令
- **实时输出**：命令结果以 JSON/表格形式展示在页面上
- **Docker 部署**：提供 Dockerfile，一键容器化部署

## 文件说明

| 文件 | 说明 |
|------|------|
| `showcase/main.go` | Go HTTP 服务器，提供 `/api/run` 接口执行 CLI 命令 |
| `showcase/index.html` | 前端页面，暗色主题，12 个模块卡片 |
| `showcase/Dockerfile` | 多阶段构建，基于 golang:1.26 镜像 |
| `showcase/deploy.sh` | 部署脚本 |
| `showcase/pipeline.yml` | GitLink DevOps 流水线配置 |

## 访问方式

```bash
# 本地运行
cd showcase && go run main.go
# 访问 http://localhost:9090

# Docker 部署
docker build -t gitlink-cli-showcase ./showcase
docker run -p 9090:9090 -e GITLINK_TOKEN=xxx gitlink-cli-showcase
```

## 注意事项

- 需要预编译 `gitlink-cli` 二进制文件放在同目录
- 服务器环境需配置 `GITLINK_TOKEN` 环境变量
