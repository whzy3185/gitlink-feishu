# Shortcut 开发模板（shortcuts/_template）

对应起步资源包「Shortcut 开发模板」：新增一个 CLI 命令组的标准模板与脚手架。
目录以 `_` 开头，Go 工具链自动忽略，**不影响构建**；复制改名后即成为真实命令组。

## 使用方法（5 步，详见 [doc/dev-guide.md](../../doc/dev-guide.md)）

```bash
# 1. 复制模板为你的命令组（例如 gadget）
cp -r shortcuts/_template shortcuts/gadget
mv shortcuts/gadget/template.go shortcuts/gadget/gadget.go
mv shortcuts/gadget/template_test.go shortcuts/gadget/gadget_test.go
# 把包名 template 全部改为 gadget，按需实现子命令
```

2. 在 `shortcuts/register.go` 注册：import 你的包，并在 `groups` 列表加一行
   `{Name: "gadget", Short: tr.T("cmd.gadget.short"), Shortcuts: gadget.Shortcuts()}`
3. 在 `internal/i18n/locales/en-US.json` 与 `zh-CN.json` 中按字母序添加 `cmd.gadget.short` 等文案键
4. 补 `httptest` 单测（模板已含可直接改写的样例）并在两份 README 补使用示例
5. 在 `doc/changes/` 新增一篇变更说明，`make check` 全绿后提 PR

## 模板内容

- `template.go`：一个最小命令组——`+list`（GET 带查询参数与分页）、`+create`（POST 必填 flag）、`+delete`（DELETE，`--yes` 确认保护），覆盖读/写/删三种典型形态与本仓库全部惯例（ResolveOwnerRepo、CallAPIWithQuery、ctx.Output 信封）
- `template_test.go`：基于 `net/http/httptest` 的单测样例（断言请求方法、路径、查询参数、请求体；不访问真实网络）

## 惯例清单（评审常看的点）

- 子命令一律 `+verb` 形式；写操作必须有 `--yes` 或等价确认保护
- 所有网络调用经 `shortcuts/common` 助手，禁止在命令组内直接 `http.Get`
- 输出统一走 `ctx.Output(env)` 信封（`{ok, data}`），自动兼容 `--format json/table/yaml` 与 `--jq`
- 文案不硬编码，走 i18n 键（中英两份 locale 同步添加）
- 涉及平台 API 行为的改动，PR 描述中附生产实测复现步骤
