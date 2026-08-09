# 新增命令别名（alias）

新增 `gitlink-cli alias` 根命令组，用于把常用命令行保存为用户自定义的快捷方式，行为对齐 `gh alias`。别名统一存放在配置文件（`config.yaml`）的 `aliases` 字段，通过 yaml.v3 与其他配置项一起读写，无需额外文件。

## 子命令

- `alias set <name> <expansion>...`：创建或更新别名。`name` 之后的所有参数会拼成展开内容，因此多词展开在 shell 上加不加引号都可以，例如 `alias set bugs issue +list --label bug`。
- `alias list`：按名称排序列出全部别名；没有别名时给出提示。
- `alias delete <name>`：删除别名，别名不存在时返回明确错误。
- `alias import`：从 YAML 的 `name: expansion` 映射批量导入，来源可用 `--file` 指定文件，或从标准输入读取（便于管道注入）；导入采用合并语义，同名覆盖。

## 别名展开

别名展开已接入根命令分发：在 `cmd.Execute()` 里，若 `os.Args` 的第一个位置参数命中已保存的别名，就把该 token 替换为别名展开（按 shell 风格拆分，识别单双引号）后再交给 cobra 分发。展开发生在翻译器解析之后，因此 `--lang` 等全局标志不受影响。

内置命令始终优先：展开前会先检查该 token 是否为已注册的根命令（或其 cobra 别名），命中则不展开。这样别名无法遮蔽真实命令——与内置命令同名的别名只是永远不会被触发，因此 `set` 不做额外的命名冲突校验。

## 本次变更

`internal/config` 的 `Config` 增加 `Aliases map[string]string` 字段并验证读写往返；新增 `cmd/alias` 命令组与 `cmd/expand.go` 展开逻辑，并在 `cmd/root.go` 注册与接线；中英文帮助与提示文案补齐到 `en-US.json` 与 `zh-CN.json`。测试覆盖配置往返、set/list/delete、文件与标准输入导入、非法 YAML，以及展开的别名命中、内置优先、首个为标志、未知 token 等分支和引号拆分。
