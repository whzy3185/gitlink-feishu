# PR 本地检出命令

新增 `gitlink-cli pr +checkout`，用于把指定 PR 的源分支拉取到本地并切换到 review 分支。维护者和贡献者在审查 PR 时，经常需要从“查看 PR 元信息”进入“本地构建、测试、复现”的流程，之前只能手动查看 fork 信息、拼接远端地址、fetch 分支，再 checkout；新命令把这套流程收敛为一个 Shortcut。

命令会读取 GitLink PR 详情，自动识别同仓库 PR 与 fork PR 的源仓库和源分支，不依赖本地 remote 必须叫 `origin`。默认创建 `pr-<id>` 本地分支，也支持 `--branch` 指定分支名、`--force` 重置已存在分支、`--dry-run` 预览将执行的 git 命令。

本次变更补充了单元测试、README 示例、PR Skill 说明和独立参考文档，覆盖 fork PR 解析、同仓库 head 解析、dry-run、force checkout 以及非法分支名校验。
