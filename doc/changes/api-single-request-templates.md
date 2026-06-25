# Raw API 单次调用模板变量与请求头支持

`gitlink-cli api` 的单次调用模式现在和批处理模式对齐了模板渲染能力，不再只能手写完整路径。现在可以直接在路径里使用 `/:owner/:repo`，也可以在 path、query、JSON body、header 中使用 `{{owner}}`、`{{repo}}` 和自定义 `{{var}}` 变量；其中 `owner` / `repo` 会优先读取 `--var`，否则自动复用全局 `--owner`、`--repo` 或当前仓库上下文。

这次改动同时把原来声明但未实际生效的 `--header` 接上了。单次请求现在支持通过 `--header key:value` 传递一个或多个自定义请求头，header 名和值都可以参与模板渲染，适合调试网关、透传审计字段、补充实验性接口所需头信息。

本次提交补充了路径占位符回归测试、query/body/header 联动渲染测试、非法 header 校验测试，以及自定义 header 真正发到服务端的行为验证。README 和 README.zh-CN 也同步加入了单次请求模板变量示例，方便维护者、脚本和 Agent 直接复用。
