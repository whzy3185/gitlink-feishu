# repo +rename 仓库重命名命令

## 背景

`gitlink-cli repo` 已提供创建、Fork、删除等仓库管理能力，但缺少对标 `gh repo rename` 的重命名命令。用户或 AI Agent 想改名当前仓库时，过去只能手动调用 PATCH 更新项目接口。

本次变更把重命名封装为 `repo +rename`，与 GitHub CLI 的 `gh repo rename` 语义对齐。

## 变更内容

- 新增 `gitlink-cli repo +rename` Shortcut。
- 调用 `PATCH /{owner}/{repo}` 更新项目，请求体同时设置 `name`（项目名称）与 `identifier`（项目标识）。
- 支持 `--name, -n` 指定新名称，必填；空值或纯空白会在调用接口前被拒绝。
- 复用现有仓库上下文解析、API 调用和统一输出格式。
- 补充中英文 i18n 文案，避免命令帮助信息硬编码。

## 命令示例

```bash
# 重命名当前仓库（owner/repo 从 git remote 自动解析）
gitlink-cli repo +rename --name new-name

# 显式指定仓库
gitlink-cli repo +rename --owner Gitlink --repo forgeplus --name new-name
```

## 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `--name, -n` | 是 | 新的仓库名称与标识 |
| `--owner` | 否 | 全局参数，仓库所有者，可从 git remote 自动解析 |
| `--repo` | 否 | 全局参数，仓库名称，可从 git remote 自动解析 |
| `--format` | 否 | 全局参数，输出格式：`json`、`table` 或 `yaml` |

## API 映射

| Shortcut | Method | API path | 请求体字段 |
|----------|--------|----------|-----------|
| `repo +rename` | PATCH | `/api/{owner}/{repo}.json` | `name`、`identifier` |

参照 API 文档「PATCH 更新项目」与「PATCH 更新项目（完整）」，`name` 与 `identifier` 均为必填。`name` 是展示用项目名称，`identifier` 是 URL 标识（等价于 `gh repo rename` 改动的仓库 URL 标识），两者都从 `--name` 取值。

## 兼容性提示

重命名会同时修改 `identifier`，也就是仓库的 URL 标识，因此**克隆地址会随之变化**（与 `gh repo rename` 行为一致）。执行后本地已有的 remote 需要相应更新。

## 测试覆盖

- 表驱动单元测试断言请求方法为 `PATCH`、路径为 `/owner/repo.json`，且请求体 `name` 与 `identifier` 均为新名称。
- 覆盖前后空白被裁剪的场景。
- 缺失或纯空白的 `--name` 在调用接口前即返回校验错误，不触发任何 HTTP 请求。

验证命令：

```bash
go build ./...
go test ./shortcuts/repo/
```

## 交付要求核对

- 功能代码：`shortcuts/repo/repo.go`
- 单元测试：`shortcuts/repo/repo_test.go`
- i18n 文案：`internal/i18n/locales/en-US.json`、`internal/i18n/locales/zh-CN.json`
- 变更说明文档：`doc/changes/repo-rename.md`
