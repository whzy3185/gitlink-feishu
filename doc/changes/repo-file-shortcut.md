# repo +file 仓库文件读取快捷命令

`gitlink-cli repo` 已经支持查看仓库信息、README 和目录树，但当用户想直接读取 `go.mod`、`.gitignore`、配置文件、脚本或许可证内容时，仍然需要回退到 Raw API。对于脚本、Agent 和日常排查来说，这是一个很常见的能力缺口。

这次新增 `gitlink-cli repo +file`，把仓库任意文件读取封装成高层 Shortcut。命令基于 `GET /{owner}/{repo}/sub_entries` 的文件模式实现，支持 `--path` 指定仓库内文件路径，支持 `--ref` 读取指定分支、标签或提交，也支持 `--content-only` 只输出文件正文，方便直接做管道消费或作为 Agent 上下文输入。

为了让这个命令在真实使用里更顺手，这次还补了两个常见边界处理：

- `--path` 设为必填，并对空路径或仅 `/` 这类无效输入给出明确报错。
- 如果用户传入的是目录路径，而不是文件路径，命令会直接提示改用 `repo +tree`，避免得到难以理解的 API 结果。

测试覆盖了默认分支、显式 `--ref`、路径归一化、帮助参数注册、目录误传报错、`--content-only` 缺少内容报错以及结果扁平化输出等关键分支。

本次交付包含：

- 功能代码：`shortcuts/repo/repo.go`
- 单元测试：`shortcuts/repo/repo_test.go`
- 帮助文档更新：`README.md`、`README.zh-CN.md`、`skills/gitlink-repo/SKILL.md`、`skills/gitlink-repo/references/gitlink-repo-file.md`
- 变更说明：本文档
