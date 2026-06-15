# 修复 `api` 命令单次调用不替换 `:owner/:repo` 占位符

## 背景

`gitlink-cli api <METHOD> <PATH>` 单次调用此前直接把 `<PATH>` 原样发送给服务端，**不会替换 REST 风格的 `:owner` / `:repo` 占位符**。这导致：

- 该命令自身的帮助 `Example`（如 `api POST /:owner/:repo/issues`）跑不通；
- 依赖 `:owner/:repo` 写法的 Skill / 文档（如 `api GET /:owner/:repo/commits`）报错；
- 占位符替换能力此前只存在于 `--batch-file` 批处理模式的 `{{var}}` 模板中，单次调用无法复用。

对应 Issue：`bug: api 命令单次调用不替换 :owner/:repo 占位符（0.2.0）`。

## 变更

`api` 单次调用现在按以下顺序处理路径：

1. **`{{var}}` 模板渲染**：若提供了 `--var key=value`，复用与批处理模式相同的模板引擎渲染 `<PATH>` 中的 `{{key}}`，缺失变量时报错。
2. **`:owner` / `:repo` 占位符替换**：当路径包含 `:owner` / `:repo` 时，使用与所有 shortcut 一致的解析逻辑 `context.ResolveOwnerRepo(--owner, --repo → git remote origin)` 解析仓库归属并替换；无法解析时给出明确错误提示。
3. 保持原有的「缺失前导 `/` 自动补全」行为。

不含占位符、且未传 `--var` 的调用（如 `api GET /users/me`）行为完全不变。

## 示例

```bash
# 自动从当前 git 仓库或 --owner/--repo 解析
gitlink-cli api GET /:owner/:repo/commits --owner Gitlink --repo gitlink-cli

# 单次调用也支持 {{var}} 模板
gitlink-cli api GET /v1/{{owner}}/gitlink-cli/issues --var owner=Gitlink
```

## 实现与测试

- 改动集中在 `cmd/api/api.go`：新增 `resolveAPIPath` 辅助函数与 `:owner` / `:repo` 占位符正则（`\b` 边界避免误伤 `:owner_id` 等更长 token；`ReplaceAllLiteralString` 避免 `$` 被当作正则替换引用）。
- 复用既有 `parseBatchVars` / `renderTemplate`（`cmd/api/batch.go`）与 `internal/context.ResolveOwnerRepo`，无新增依赖。
- 更新命令 `Example` 帮助文案。
- 新增单元测试：`:owner/:repo` 解析替换、单次调用 `{{var}}` 渲染、缺失模板变量报错；既有测试全部通过。
