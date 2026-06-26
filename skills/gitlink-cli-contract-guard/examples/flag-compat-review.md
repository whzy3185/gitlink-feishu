# 示例：参数兼容性审查

用户请求：

```text
Use $gitlink-cli-contract-guard 检查这个 PR 有没有破坏现有 flag、默认值或帮助文案。
```

期望动作：

1. 定位变更是否触及 `cmd/`、`shortcuts/`、`README`。
2. 检查 flag 的 `Name`、`Short`、`Default`、`Required` 是否变化。
3. 检查 `--help` 和示例命令是否同步。
4. 输出按严重性排序的问题和缺失验证。

输出要点：

- 明确指出是“新增能力”还是“破坏旧用法”。
- 如果默认值改变，要说明对已有用户的影响。
