# 科研项目合规与复现性检查报告 — caoweiqiong/zwf

> 场景 S3 · 子赛题四「应用 GitLink 辅助科研」

- **默认分支**: `master`
- **识别许可证**: `None`
- **复现性评分**: **7.0/10**（及格）
- **合规性评分**: **6.2/10**（及格）

## 一、复现性检查清单

| 检查项 | 通过 | 得分 | 证据 |
|--------|:----:|:----:|------|
| CI 配置 | FAIL | 0/2 | 未找到 .gitea/.github/.gitlab 等 CI 配置 |
| 依赖锁文件 | PASS | 2/2 | 存在 lockfile: package-lock.json |
| README 复现说明 | PASS | 2/2 | README 含复现关键词 11 个: install, 环境, 依赖, build, 构建 |
| 版本 tag | FAIL | 1/2 | repo_info 无显式 tag 字段（默认分支: master），建议打 tag 固定可复现版本 |
| 容器化环境 | PASS | 2/2 | 存在容器配置: docker-compose.yml |

## 二、合规性检查清单

| 检查项 | 通过 | 得分 | 证据 |
|--------|:----:|:----:|------|
| LICENSE 文件 | FAIL | 0/2 | 缺少 LICENSE 文件 |
| 安全策略 SECURITY.md | FAIL | 0/2 | 缺少 SECURITY.md，无安全披露流程 |
| 版权声明 | FAIL | 1/2 | 未在 LICENSE/README 中发现版权声明（建议源文件头补 Copyright 注释） |
| 依赖清单声明 | PASS | 2/2 | 存在依赖管理文件（建议核对各依赖许可证兼容性） |
| 贡献指南 | FAIL | 1/2 | 缺少 CONTRIBUTING.md |

## 三、数据隐私检查

| 检查项 | 通过 | 得分 | 证据 |
|--------|:----:|:----:|------|
| 数据目录入库 | PASS | 2/2 | 未发现 data/ 目录入库 |
| .env 入库 | PASS | 2/2 | .env 未入库 |
| .gitignore 忽略 .env | PASS | 2/2 | .gitignore 已配置忽略 .env |

## 四、风险项（按严重程度排序）

| 级别 | 类别 | 名称 | 文件:行 | 证据 |
|:----:|------|------|---------|------|
| medium | repro/compliance | CI 配置 |  | 未找到 .gitea/.github/.gitlab 等 CI 配置 |
| medium | repro/compliance | 版本 tag |  | repo_info 无显式 tag 字段（默认分支: master），建议打 tag 固定可复现版本 |
| medium | repro/compliance | LICENSE 文件 |  | 缺少 LICENSE 文件 |
| medium | repro/compliance | 安全策略 SECURITY.md |  | 缺少 SECURITY.md，无安全披露流程 |
| medium | repro/compliance | 版权声明 |  | 未在 LICENSE/README 中发现版权声明（建议源文件头补 Copyright 注释） |
| medium | repro/compliance | 贡献指南 |  | 缺少 CONTRIBUTING.md |

_复现分 7.0/10 · 合规分 6.2/10 · 树节点 22_
