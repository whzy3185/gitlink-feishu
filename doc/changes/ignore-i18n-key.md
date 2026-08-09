# Ignore group i18n key

`register.go` 用 `tr.T("cmd.ignore.short")` 设置 `ignore` 组描述，但该键在两个语言包中都缺失，导致 `--help` 里直接显示字面量 `cmd.ignore.short`：

- 补齐 `en-US.json` / `zh-CN.json` 的 `cmd.ignore.short`
- `TestRegisterAllGroupDescriptions` 增加断言：组描述不得是未解析的 i18n 键
