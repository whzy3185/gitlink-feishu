# pr +status 与我相关的合并请求概览命令

## 背景

`gitlink-cli pr` 已经提供列表、创建、查看、合并、评审等能力，但缺少一个类似 `gh pr status` 的入口，用于快速回答“当前登录用户在本仓库里有哪些待处理的合并请求”。用户或 AI Agent 过去需要先查自己的身份，再手动拼 `pr +list` 的过滤参数，还要区分“我提的”和“等我评审的”。

本次变更把这一常见诉求封装为 `pr +status`，一次调用给出两组结果：你创建的、以及请求你评审的开启中合并请求。

## 变更内容

- 新增 `gitlink-cli pr +status` Shortcut（只读）。
- 先调用 `GET /users/me` 解析当前用户的 `login` 与数值 `id`（复用 `user +me` 的接口）。
- 复用合并请求列表接口 `GET /v1/{owner}/{repo}/pulls`（api_ref「获取合并请求列表」）拉取数据：
  - **你创建的**：以 `status=0` 拉取开启中的合并请求，再按 `issue.author.login` 与当前用户在**客户端**匹配。该列表接口没有 author 过滤参数，故只能客户端过滤。
  - **请求你评审的**：以 `status=0` 加 `reviewer_id={当前用户 id}` 由**服务端**过滤（`reviewer_id` 是列表接口文档化的审查人员过滤参数）。
- 输出统一封装为结构化数据：`login`、`created`、`review_requested` 两组合并请求数组，沿用现有输出格式（json/table/yaml）。
- 补充中英文 i18n 文案（`cmd.pr.status.short` / `cmd.pr.status.long`），避免命令帮助信息硬编码。

## 命令示例

```bash
# 查看与你相关的合并请求（owner/repo 可从 git remote 自动解析）
gitlink-cli pr +status --owner Gitlink --repo forgeplus

# Agent 场景建议 JSON 输出
gitlink-cli pr +status --owner Gitlink --repo forgeplus --format json
```

## 输出结构

```json
{
  "ok": true,
  "data": {
    "login": "currentuser",
    "created": [ /* 你创建的开启中合并请求 */ ],
    "review_requested": [ /* 请求你评审的开启中合并请求 */ ]
  }
}
```

## 设计说明

- 该命令刻意只使用列表接口文档化的查询参数（`status`、`reviewer_id`），不引入未在 api_ref 中出现的参数。
- author 侧过滤放在客户端，是因为列表接口只支持 `reviewer_id` / `assign_user_id` 等数值过滤，没有 author 过滤参数；这一点在上文与代码注释中都做了说明。
- 全流程只读，不修改任何合并请求状态。

## 测试覆盖

`shortcuts/pr/pr_test.go` 中新增表驱动单元测试（mock `/users/me` 与合并请求列表接口）：

- 分组正确：混合作者的开启中合并请求被正确拆分为“你创建的”与“请求你评审的”。
- author 客户端过滤：他人创建的合并请求不进入“你创建的”分组。
- 空仓库：两组均为空。
- `reviewer_id` 断言：确认按当前用户数值 id 向服务端发起评审过滤查询。
- 错误路径：`/users/me` 返回 500、或响应缺少 `login` 时命令报错。
- 端到端：`pr +status` 走完整 Run 路径（含输出）不报错。

验证命令：

```bash
go build ./...
go test ./shortcuts/pr/
```

## 交付要求核对

- 功能代码：`shortcuts/pr/pr.go`
- 单元测试：`shortcuts/pr/pr_test.go`
- i18n 文案：`internal/i18n/locales/en-US.json`、`internal/i18n/locales/zh-CN.json`
- 变更说明文档：`doc/changes/pr-status.md`

## 兼容性

该变更只新增一个只读 Shortcut、对应单元测试、i18n 文案与文档，不修改已有命令的参数或输出结构，对现有功能无破坏性影响。
