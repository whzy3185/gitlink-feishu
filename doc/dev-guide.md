# 开发环境指南（dev-guide）

> 本指南覆盖 gitlink-cli 的本地开发、调试、测试全流程，对应起步资源包中的「开发环境指南」。
> 面向首次贡献者：从零环境到提交第一个 PR。

## 1. 环境准备

| 依赖 | 版本 | 说明 |
|------|------|------|
| Go | ≥ 1.26 | 见 `go.mod`（`go 1.26.1`） |
| git | 任意较新版本 | 克隆与提交 |
| make | 可选 | 使用 Makefile 快捷目标 |
| golangci-lint | 可选 | `make lint` 需要 |

```bash
# 克隆（建议先 Fork 到自己账号）
git clone https://www.gitlink.org.cn/<你的账号>/gitlink-cli.git
cd gitlink-cli

# 构建（自动注入版本号）
make build          # 产出 ./gitlink-cli
# 或不依赖 make：
go build -o gitlink-cli .
```

## 2. 本地运行与认证

```bash
./gitlink-cli --help          # 全部命令
./gitlink-cli auth login      # 交互式登录（或配置 PAT）
./gitlink-cli auth status     # 验证登录态
./gitlink-cli user +me        # 冒烟验证
```

配置文件位置：`~/.config/gitlink-cli/`（含凭据，请勿提交到仓库）。

## 3. 代码结构速览

```
main.go            # 入口
cmd/               # 内置命令（api / auth / config / doctor）与根命令
shortcuts/         # 各资源命令组（issue、pr、repo、branch……），每组一个包
shortcuts/common/  # 命令组共享的运行时上下文、flag、API 调用助手
internal/client/   # HTTP 客户端（认证注入、错误处理）
internal/output/   # 输出格式化（json / table / yaml）
internal/i18n/     # 国际化（locales/en-US.json、zh-CN.json）
skills/            # AI Agent Skills（Markdown）
examples/          # 端到端工作流示例
doc/changes/       # 每个功能/修复的变更说明（PR 需附带）
```

## 4. 新增一个 shortcut 的标准步骤

1. 在对应命令组包（如 `shortcuts/issue/`）的 `Shortcuts` 列表中添加条目：`Name` / `Description`（用 `tr.T("key")`）/ `Flags` / `Run`
2. 在 `internal/i18n/locales/en-US.json` 与 `zh-CN.json` 中添加文案键（按字母序插入）
3. 在同包 `*_test.go` 中添加基于 `httptest` 的单测（断言请求方法、路径与参数）
4. 在 `README.md` 与 `README.zh-CN.md` 中补使用示例
5. 在 `doc/changes/` 下新增一篇变更说明

## 5. 调试技巧

```bash
# 全局 --debug 打印请求/响应细节
./gitlink-cli issue +list --owner Gitlink --repo forgeplus --debug

# 用 api 命令直接访问任意端点，验证平台真实语义（ground truth）
./gitlink-cli api GET /:owner/:repo/issues --owner Gitlink --repo forgeplus

# 自诊断：配置、认证、仓库上下文、连通性
./gitlink-cli doctor
```

提示：平台同时存在 `/api`（遗留）与 `/api/v1`（现行）两代端点，同名资源的参数与 ID 语义可能不同；
提交涉及 API 行为的 PR 前，建议在生产平台用 `api` 命令实测取证并在 PR 描述中附复现步骤。

## 6. 测试与质量门禁

```bash
make test    # go test -race ./...
make vet     # go vet ./...
make fmt     # gofmt -s 检查（有未格式化文件会失败）
make lint    # golangci-lint（需自行安装）
make check   # fmt + vet + lint + test 一次跑完
make cover   # 覆盖率报告
```

单测约定：不访问真实网络，用 `net/http/httptest` 起本地假服务断言请求；写操作命令必须覆盖「确认保护」路径（如 `--yes`）。

## 7. 提交 PR

1. 从 `master` 切出特性分支：`git checkout -b feat/<名字>`
2. 确保 `make check` 全绿
3. push 到自己的 Fork，然后向 `gitlink/gitlink-cli:master` 发起 PR
4. PR 描述建议包含：背景、变更点、生产验证步骤与输出、关联 issue

## 8. 常见问题

- **构建报 Go 版本过低**：升级到 `go.mod` 声明的版本及以上
- **命令返回 401**：先 `auth login`；注意部分微服务（如 wiki 写接口）只认会话态
- **列表数据不全**：多数 list 命令有分页，检查 `page`/`limit` 参数
- **Windows**：构建用 `go install .`；全局安装可用 `npm install -g @gitlink-ai/cli`
