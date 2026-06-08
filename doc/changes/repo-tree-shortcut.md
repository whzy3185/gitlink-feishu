# repo +tree 仓库文件树查询命令

## 背景

`gitlink-cli repo` 已经提供仓库详情、README、语言统计和贡献者查询能力，但缺少直接查看仓库目录结构的 Shortcut。用户或 AI Agent 如果要判断仓库中是否存在 README、LICENSE、依赖清单、测试目录、文档目录等文件，过去需要手动调用 Raw API `/sub_entries`。

本次变更把仓库文件树查询封装为 `repo +tree`，降低普通用户和自动化工作流的使用门槛。

## 变更内容

- 新增 `gitlink-cli repo +tree` Shortcut。
- 调用 `GET /{owner}/{repo}/sub_entries` 获取仓库根目录或指定目录下的文件和子目录。
- 支持 `--path, -p` 指定目录路径；不传时查询仓库根目录。
- 支持 `--ref, -r` 指定分支、标签或提交引用；默认值为 `master`。
- 复用现有仓库上下文解析、API 调用和统一输出格式。
- 补充中英文 i18n 文案，避免新增命令帮助信息硬编码。

## 命令示例

```bash
# 查看仓库根目录
gitlink-cli repo +tree --owner Gitlink --repo forgeplus --ref master

# 查看指定目录
gitlink-cli repo +tree --owner Gitlink --repo forgeplus --path src --ref main

# Agent 场景建议使用 JSON 输出
gitlink-cli repo +tree --owner Gitlink --repo forgeplus --format json
```

## 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `--path, -p` | 否 | 要查看的目录路径，不传时查询仓库根目录 |
| `--ref, -r` | 否 | 分支、标签或提交引用，默认 `master` |
| `--owner` | 否 | 全局参数，仓库所有者，可从 git remote 自动解析 |
| `--repo` | 否 | 全局参数，仓库名称，可从 git remote 自动解析 |
| `--format` | 否 | 全局参数，输出格式：`json`、`table` 或 `yaml` |

## 测试覆盖

单元测试覆盖以下内容：

- 根目录查询默认使用 `master`。
- 根目录查询不发送空 `filepath` 参数。
- 指定 `--path` 和 `--ref` 时正确映射到 `filepath` 与 `ref` 查询参数。
- `repo +tree` 的命令说明和 `--path/-p`、`--ref/-r` 参数注册完整。

验证命令：

```bash
make test
```

## 交付要求核对

- 功能代码：`shortcuts/repo/repo.go`
- 单元测试：`shortcuts/repo/repo_test.go`
- 命令帮助文档：`README.md`、`README.zh-CN.md`、`skills/gitlink-repo/SKILL.md`、`skills/gitlink-repo/references/gitlink-repo-tree.md`
- 变更说明文档：`doc/changes/repo-tree-shortcut.md`

## 兼容性

该变更只新增 Shortcut、单元测试和文档，不修改已有命令参数或输出结构。根目录查询时不再发送空 `filepath` 查询参数，语义更清晰，对现有功能无破坏性影响。
