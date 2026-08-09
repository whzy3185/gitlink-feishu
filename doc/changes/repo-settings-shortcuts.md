# Repository Settings Shortcuts

Submitter: Wang Yue

This change expands repository shortcut coverage for repository metadata, settings, project topics, navigation units, and transfer OpenAPI endpoints.

## Commands

| Shortcut | 说明 | 需要认证 |
|----------|------|----------|
| `repo +detail` | 获取完整项目详情 | 否（公开项目） |
| `repo +simple` | 获取简版项目详情 | 否（公开项目） |
| `repo +settings` | 获取项目设置 | 是 |
| `repo +units` | 获取导航单元配置 | 是 |
| `repo +units-update` | 更新导航单元配置 | 是 |
| `repo +topics` | 搜索项目标签 | 否（公开项目） |
| `repo +topic-add` | 添加项目标签 | 是 |
| `repo +topic-delete` | 删除项目标签 | 是 |
| `repo +transfer-orgs` | 获取可转移组织列表 | 是 |
| `repo +transfer` | 发起仓库转移 | 是 |
| `repo +transfer-cancel` | 取消仓库转移 | 是 |

## API Mapping

| Shortcut | Method | API path |
|----------|--------|----------|
| `repo +detail` | GET | `/api/{owner}/{repo}/detail.json` |
| `repo +simple` | GET | `/api/{owner}/{repo}/simple.json` |
| `repo +settings` | GET | `/api/{owner}/{repo}/edit.json` |
| `repo +units` | GET | `/api/{owner}/{repo}/project_units.json` |
| `repo +units-update` | POST | `/api/{owner}/{repo}/project_units.json` |
| `repo +topics` | GET | `/api/v1/project_topics.json` |
| `repo +topic-add` | POST | `/api/v1/project_topics.json` |
| `repo +topic-delete` | DELETE | `/api/v1/project_topics/{id}.json` |
| `repo +transfer-orgs` | GET | `/api/{owner}/{repo}/applied_transfer_projects/organizations.json` |
| `repo +transfer` | POST | `/api/{owner}/{repo}/applied_transfer_projects.json` |
| `repo +transfer-cancel` | POST | `/api/{owner}/{repo}/applied_transfer_projects/cancel.json` |

## 参数说明

### repo +detail / +simple / +settings / +units
- `--owner`: 仓库所有者（可从 git remote 自动推断）
- `--repo`: 仓库名称（可从 git remote 自动推断）

### repo +units-update
- `--owner`: 仓库所有者
- `--repo`: 仓库名称
- `--unit-types`: 导航单元类型列表，逗号分隔（如 `code,issues,pulls`）
- 有效值：`code`, `issues`, `pulls`, `wiki`, `devops`, `versions`, `services`

### repo +topics
- `--keyword`: 搜索关键词（可选）

### repo +topic-add
- `--name`: 标签名称
- `--project-id`: 项目 ID（必需）

### repo +topic-delete
- `--id`: 标签 ID
- `--project-id`: 项目 ID（必需）

### repo +transfer-orgs
- `--owner`: 仓库所有者
- `--repo`: 仓库名称

### repo +transfer
- `--owner`: 仓库所有者
- `--repo`: 仓库名称
- `--org-name`: 目标组织名称

### repo +transfer-cancel
- `--owner`: 仓库所有者
- `--repo`: 仓库名称

## 使用示例

```bash
# 查看项目详情
gitlink-cli repo +detail --owner Gitlink --repo forgeplus
gitlink-cli repo +simple --owner Gitlink --repo forgeplus

# 查看和更新项目设置
gitlink-cli repo +settings --owner myuser --repo myproject
gitlink-cli repo +units --owner myuser --repo myproject
gitlink-cli repo +units-update --owner myuser --repo myproject --unit-types code,issues,pulls,wiki

# 管理项目标签
gitlink-cli repo +topics --keyword "前端"
gitlink-cli repo +topic-add --name "Vue" --project-id 123
gitlink-cli repo +topic-delete --id 456 --project-id 123

# 仓库转移
gitlink-cli repo +transfer-orgs --owner myuser --repo myproject
gitlink-cli repo +transfer --owner myuser --repo myproject --org-name myorg --dry-run
gitlink-cli repo +transfer --owner myuser --repo myproject --org-name myorg
gitlink-cli repo +transfer-cancel --owner myuser --repo myproject
```

## 实现细节

- 所有命令遵循现有 `shortcuts/repo/repo.go` 的代码模式
- 使用 `ctx.RequireArg()` 获取必需参数
- 使用 `ctx.CallAPI()` 调用 API
- 写入操作支持 `--dry-run` 参数预览
- `repo +units-update` 使用 `parseCommaSeparatedList()` 解析逗号分隔的单元类型列表，自动去重

## Verification

- 单元测试覆盖所有 11 个新命令
- 测试用例包括：请求方法、路径、查询参数、JSON 请求体、dry-run 行为、CSV 去重、无效 project-id 验证
- 所有写入和状态变更命令支持 `--dry-run`
- 测试文件：`shortcuts/repo/repo_test.go`
- 测试命令：`go test ./shortcuts/repo/...`
