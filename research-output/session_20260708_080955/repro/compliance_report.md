# 科研项目合规与复现性检查报告 — whale_hihihi/gitlink-cli

> 场景 S3 · 子赛题四「应用 GitLink 辅助科研」

- **默认分支**: `master`
- **识别许可证**: `MulanPSL-2.0`
- **复现性评分**: **9.0/10**（良好）
- **合规性评分**: **7.5/10**（及格）

## 一、复现性检查清单

| 检查项 | 通过 | 得分 | 证据 |
|--------|:----:|:----:|------|
| CI 配置 | PASS | 2/2 | 检测到 CI 配置: .devops, .gitea, .github |
| 依赖锁文件 | PASS | 2/2 | 存在 lockfile: go.sum |
| README 复现说明 | PASS | 2/2 | README 含复现关键词 10 个: install, setup, build, 运行, run |
| 版本 tag | FAIL | 1/2 | repo_info 无显式 tag 字段（默认分支: master），建议打 tag 固定可复现版本 |
| 容器化环境 | PASS | 2/2 | 存在容器配置: Dockerfile |

## 二、合规性检查清单

| 检查项 | 通过 | 得分 | 证据 |
|--------|:----:|:----:|------|
| LICENSE 文件 | PASS | 2/2 | LICENSE 声明为 MulanPSL-2.0 |
| 安全策略 SECURITY.md | FAIL | 0/2 | 缺少 SECURITY.md，无安全披露流程 |
| 版权声明 | PASS | 2/2 | LICENSE/README 中含 copyright/版权 声明 |
| 依赖清单声明 | PASS | 2/2 | 存在依赖管理文件（建议核对各依赖许可证兼容性） |
| 贡献指南 | FAIL | 1/2 | 缺少 CONTRIBUTING.md |

## 三、数据隐私检查

| 检查项 | 通过 | 得分 | 证据 |
|--------|:----:|:----:|------|
| 数据目录入库 | PASS | 2/2 | 未发现 data/ 目录入库 |
| .env 入库 | PASS | 2/2 | .env 未入库 |
| .gitignore 忽略 .env | FAIL | 1/2 | .gitignore 未忽略 .env（建议添加 .env） |

## 四、风险项（按严重程度排序）

| 级别 | 类别 | 名称 | 文件:行 | 证据 |
|:----:|------|------|---------|------|
| high | privacy | 数据隐私 |  | .gitignore 未忽略 .env（预防性建议） |
| medium | repro/compliance | 版本 tag |  | repo_info 无显式 tag 字段（默认分支: master），建议打 tag 固定可复现版本 |
| medium | repro/compliance | 安全策略 SECURITY.md |  | 缺少 SECURITY.md，无安全披露流程 |
| medium | repro/compliance | 贡献指南 |  | 缺少 CONTRIBUTING.md |
| medium | repro/compliance | .gitignore 忽略 .env |  | .gitignore 未忽略 .env（建议添加 .env） |

_复现分 9.0/10 · 合规分 7.5/10 · 树节点 31_
