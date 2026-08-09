# Shell 自动补全命令

新增 `gitlink-cli completion [bash|zsh|fish|powershell]`，用于为 Bash、Zsh、Fish 和 PowerShell 生成原生自动补全脚本。用户安装 CLI 后可以直接把脚本写入对应 shell 的补全目录，减少记忆 Shortcut 命令、全局参数和子命令名称的成本，也让跨平台安装体验更完整。

命令支持 `--no-descriptions`，在不需要补全文案的终端配置中可以输出更精简的脚本。实现复用 Cobra 官方补全生成能力，不引入远端 API 依赖，并通过单元测试覆盖四类 shell 输出、描述开关和非法 shell 参数校验。
