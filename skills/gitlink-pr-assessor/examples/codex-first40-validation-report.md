# Gitlink/gitlink-cli PR 待审队列报告

> 说明：这是 `gitlink-pr-assessor` 在 Codex 中执行“扫描当前前 40 条 open PR”得到的实际报告快照。原始运行目录位于本地临时路径，未纳入仓库。

- 扫描时间：2026-06-24T16:17:20.250220+08:00
- 范围：当前列表顺序下前 40 条 `pull_request_status == 0` 的 open PR
- 纳入评估：39 条
- 跳过：1 条
- CI 信息：不可用（[-1] 接口数据异常）
- 原始证据目录：本地临时产物，未纳入仓库
- 基线验证：`origin/master` 上 `go build ./...` 与 `go test ./...` 均通过
- 执行验证记录：本地临时产物，未纳入仓库

> 说明：本轮没有向任何 PR 写回内容。Go 执行验证在临时 clone 的 PR head 上执行；由于本地 Git merge 多数 PR head 与 upstream master 表现为 unrelated histories，报告单独标注集成层状态，避免把 head 测试通过误读为可直接合并。

## 队列总览

| PR | 标题 | 结论 | 风险 | 优先级 | 执行验证 | 集成状态 | 说明 |
|----|------|------|------|--------|----------|----------|------|
| #276 | 注册重编激活 25 域、新增 6 个 Skill、补全索引与示例、产出任务三/四的端到端工作流 | 建议拒绝当前形态或要求拆分重做 | 高 | P1 | blocked | blocked | 包含部署/CI 自动化中的硬编码主机、root 用户或破坏性 shell 命令，且改动面过大或存在冲突。 |
| #275 | feat(shortcut): add repo upload | 建议修复测试失败后再进入人工评审 | 中 | P1 | failed | unverified_unrelated_histories | PR head 可以构建，但 `go test ./...` 失败，当前不应推进合并。 |
| #274 | feat(shortcuts): add template shortcuts | 建议修复测试失败后再进入人工评审 | 中 | P1 | failed | unverified_unrelated_histories | PR head 可以构建，但 `go test ./...` 失败，当前不应推进合并。 |
| #262 | feat(shortcuts): add user key shortcuts | 建议进入合并前人工确认 | 低 | P2 | passed | unverified_unrelated_histories | PR head 的 `go build ./...` 与 `go test ./...` 均通过，改动范围可控。 |
| #273 | feat(repo): 补齐仓库治理中的转移与模块配置能力 | 建议进入合并前人工确认 | 低 | P2 | passed | unverified_unrelated_histories | PR head 的 `go build ./...` 与 `go test ./...` 均通过，改动范围可控。 |
| #272 | feat(message): 补齐通知中心与消息设置管理工作流 | 建议聚焦代码路径人工确认后推进 | 中 | P2 | passed | unverified_unrelated_histories | PR head 的构建和测试通过，但改动跨多个文件，且本地无法验证与 master 的真实 merge commit。 |
| #271 | feat(issue): 补齐 Issue 批量导出与评论管理工作流 | 建议聚焦代码路径人工确认后推进 | 中 | P2 | passed | unverified_unrelated_histories | PR head 的构建和测试通过，但改动跨多个文件，且本地无法验证与 master 的真实 merge commit。 |
| #270 | feat(message-settings): 增加消息通知设置快捷命令 | 建议进入合并前人工确认 | 低 | P2 | passed | unverified_unrelated_histories | PR head 的 `go build ./...` 与 `go test ./...` 均通过，改动范围可控。 |
| #269 | 新增gitlink-dev-full-cycle技能 | 建议维护者确认安全表述后再合入 | 中 | P3 | static_only | not_required | 主要是文档/skill 变更，未触发 Go 构建需求，但包含凭据/Token 相关表述，需人工确认边界。 |
| #268 | feat(branch): add lifecycle shortcuts | 建议进入合并前人工确认 | 低 | P2 | passed | unverified_unrelated_histories | PR head 的 `go build ./...` 与 `go test ./...` 均通过，改动范围可控。 |
| #267 | feat(member): add application shortcuts | 建议进入合并前人工确认 | 低 | P2 | passed | unverified_unrelated_histories | PR head 的 `go build ./...` 与 `go test ./...` 均通过，改动范围可控。 |
| #265 | feat(skills): 新增 PR审查效率看板 的skill : gitlink-pr-review-quality | 建议进入维护者内容确认 | 低 | P3 | static_only | not_required | 主要为文档/skill 变更，改动范围较小，静态检查未见阻塞。 |
| #263 | 新增 template 命令，支持 GitLink 平台项目模板的增删改查操作 | 建议作者先 rebase/解决冲突后再审 | 中 | P2 | blocked | blocked | GitLink 元数据显示不可合并或存在冲突，集成验证被阻塞。 |
| #261 | 添加 gitlink-cli auth checkin 命令，用于定时刷新认证会话，防止用户登录状态过期 | 建议作者先 rebase/解决冲突后再审 | 高 | P2 | blocked | blocked | GitLink 元数据显示不可合并或存在冲突，集成验证被阻塞。 |
| #238 | feat(user): add statistics shortcuts | 建议进入合并前人工确认 | 低 | P2 | passed | unverified_unrelated_histories | PR head 的 `go build ./...` 与 `go test ./...` 均通过，改动范围可控。 |
| #259 | 修改skill | 建议拒绝当前形态或要求拆分重做 | 高 | P1 | blocked | blocked | 包含部署/CI 自动化中的硬编码主机、root 用户或破坏性 shell 命令，且改动面过大或存在冲突。 |
| #258 | fix(output): deterministic --format table ordering and show  | 建议进入合并前人工确认 | 低 | P2 | passed | unverified_unrelated_histories | PR head 的 `go build ./...` 与 `go test ./...` 均通过，改动范围可控。 |
| #257 | feat(skills): 新增 科研复现性评估的skill : gitlink-research-reproducib | 建议进入维护者内容确认 | 低 | P3 | static_only | not_required | 主要为文档/skill 变更，改动范围较小，静态检查未见阻塞。 |
| #256 | feat(skills): 新增 科研进度跟踪与预警的skill : gitlink-research-progress | 建议进入维护者内容确认 | 低 | P3 | static_only | not_required | 主要为文档/skill 变更，改动范围较小，静态检查未见阻塞。 |
| #255 |  feat(skills): 新增 科研协作智能匹配的skill : gitlink-research-match | 建议进入维护者内容确认 | 低 | P3 | static_only | not_required | 主要为文档/skill 变更，改动范围较小，静态检查未见阻塞。 |
| #254 | fix(api): resolve :owner/:repo and --var placeholders in sin | 建议进入合并前人工确认 | 低 | P2 | passed | unverified_unrelated_histories | PR head 的 `go build ./...` 与 `go test ./...` 均通过，改动范围可控。 |
| #167 | feat(account): add email verification shortcuts | 建议作者先 rebase/解决冲突后再审 | 中 | P2 | blocked | blocked | GitLink 元数据显示不可合并或存在冲突，集成验证被阻塞。 |
| #151 | feat(transfer): add transfer request shortcuts | 建议作者先 rebase/解决冲突后再审 | 中 | P2 | blocked | blocked | GitLink 元数据显示不可合并或存在冲突，集成验证被阻塞。 |
| #135 | feat(dev): add developer resource shortcuts | 建议作者先 rebase/解决冲突后再审 | 高 | P2 | blocked | blocked | GitLink 元数据显示不可合并或存在冲突，集成验证被阻塞。 |
| #82 | feat(meta): add attachment and metadata shortcuts | 建议作者先 rebase/解决冲突后再审 | 高 | P2 | blocked | blocked | GitLink 元数据显示不可合并或存在冲突，集成验证被阻塞。 |
| #76 | feat(notification): add OpenAPI shortcuts | 建议作者先 rebase/解决冲突后再审 | 中 | P2 | blocked | blocked | GitLink 元数据显示不可合并或存在冲突，集成验证被阻塞。 |
| #72 | feat(template): add project template shortcuts | 建议作者先 rebase/解决冲突后再审 | 中 | P2 | blocked | blocked | GitLink 元数据显示不可合并或存在冲突，集成验证被阻塞。 |
| #70 | feat(user): add account and stats shortcuts | 建议作者先 rebase/解决冲突后再审 | 中 | P2 | blocked | blocked | GitLink 元数据显示不可合并或存在冲突，集成验证被阻塞。 |
| #65 | feat(wiki): add OpenAPI shortcuts | 建议作者先 rebase/解决冲突后再审 | 中 | P2 | blocked | blocked | GitLink 元数据显示不可合并或存在冲突，集成验证被阻塞。 |
| #64 | feat(dataset): add OpenAPI shortcuts | 建议作者先 rebase/解决冲突后再审 | 中 | P2 | blocked | blocked | GitLink 元数据显示不可合并或存在冲突，集成验证被阻塞。 |
| #63 | feat(code): add repository code OpenAPI shortcuts | 建议作者先 rebase/解决冲突后再审 | 中 | P2 | blocked | blocked | GitLink 元数据显示不可合并或存在冲突，集成验证被阻塞。 |
| #107 | feat(star): add starred project shortcuts | 建议作者先 rebase/解决冲突后再审 | 中 | P2 | blocked | blocked | GitLink 元数据显示不可合并或存在冲突，集成验证被阻塞。 |
| #83 | feat(org): add team project binding shortcuts | 建议进入合并前人工确认 | 低 | P2 | passed | unverified_unrelated_histories | PR head 的 `go build ./...` 与 `go test ./...` 均通过，改动范围可控。 |
| #78 | feat(branch): complete OpenAPI shortcuts | 建议进入合并前人工确认 | 低 | P2 | passed | unverified_unrelated_histories | PR head 的 `go build ./...` 与 `go test ./...` 均通过，改动范围可控。 |
| #77 | feat(journal): add issue and PR comment shortcuts | 建议作者先 rebase/解决冲突后再审 | 高 | P2 | blocked | blocked | GitLink 元数据显示不可合并或存在冲突，集成验证被阻塞。 |
| #179 | feat(dataset): add research dataset shortcuts | 建议作者先 rebase/解决冲突后再审 | 中 | P2 | blocked | blocked | GitLink 元数据显示不可合并或存在冲突，集成验证被阻塞。 |
| #178 | feat(contents): add repository content shortcuts | 建议作者先 rebase/解决冲突后再审 | 中 | P2 | blocked | blocked | GitLink 元数据显示不可合并或存在冲突，集成验证被阻塞。 |
| #177 | feat(wiki): add wiki management shortcuts | 建议作者先 rebase/解决冲突后再审 | 中 | P2 | blocked | blocked | GitLink 元数据显示不可合并或存在冲突，集成验证被阻塞。 |
| #176 | feat(user): add dashboard shortcuts | 建议作者先 rebase/解决冲突后再审 | 中 | P2 | blocked | blocked | GitLink 元数据显示不可合并或存在冲突，集成验证被阻塞。 |

## 跳过项

- #225：feat(repo): add scaffold creation options；原因：already-approved-or-rejected

## 逐条维护者报告

<!-- gitlink-pr-assessor:report v1 -->
### PR #276 维护者评估报告

**标题：** 注册重编激活 25 域、新增 6 个 Skill、补全索引与示例、产出任务三/四的端到端工作流
**作者：** Surponess
**结论：** 建议拒绝当前形态或要求拆分重做
**风险等级：** 高
**优先级：** P1
**证据强度：** 高
**执行验证：** blocked；集成状态：blocked

#### 核心判断
1. 先移除或重做部署自动化中的硬编码 IP、root 登录和破坏性命令。
1. 改动面过大，建议拆分为独立功能/文档/skill PR。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：127 文件，+12513 / -3288，90 commits；Go 文件 52，测试文件 23，文档/skill 文件 64。
风险信号：hardcoded_root, hardcoded_ip, destructive_shell, secret_or_password, external_ci。
冲突文件：internal/client/client.go, shortcuts/issue/issue.go, shortcuts/register.go, shortcuts/register_test.go, shortcuts/wiki/wiki.go, shortcuts/wiki/wiki_test.go, skills/README.md, skills/gitlink-issue/SKILL.md ...
样例文件：.devops/自动构建部署.yml, .dockerignore, .github/workflows/ci.yml, .gitignore, .golangci.yml, Dockerfile

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 弱 |
| 代码质量 | 弱 |
| 安全性 | 高风险 |
| 维护成本 | 高 |
| 协作质量 | 弱 |
| 回归风险 | 高 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：注册重编激活 25 域、新增 6 个 Skill、补全索引与示例、产出任务三/四的端到端工作流 | blocked | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 集成 | mergeability/conflict 检查 | 阻塞 | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 建议 review 文案

建议暂不合入。请先拆分 PR，并移除部署自动化中的硬编码服务器/root 登录/破坏性 shell 命令；解决与 master 的冲突后，再针对单一能力补充测试和说明。

<!-- gitlink-pr-assessor:report v1 -->
### PR #275 维护者评估报告

**标题：** feat(shortcut): add repo upload
**作者：** co63oc
**结论：** 建议修复测试失败后再进入人工评审
**风险等级：** 中
**优先级：** P1
**证据强度：** 高
**执行验证：** failed；集成状态：unverified_unrelated_histories

#### 核心判断
1. 当前 `go test ./...` 已失败，先让作者修复再继续评审。
1. 重点检查 repo upload 的 i18n key 是否在所有语言包中补齐。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：10 文件，+466 / -0，1 commits；Go 文件 5，测试文件 2，文档/skill 文件 3。
样例文件：README.md, README.zh-CN.md, doc/changes/repo-upload-shortcut.md, internal/client/client.go, internal/client/client_test.go, internal/i18n/locales/en-US.json

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 中 |
| 代码质量 | 弱 |
| 安全性 | 低-中 |
| 维护成本 | 中 |
| 协作质量 | 中 |
| 回归风险 | 中 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(shortcut): add repo upload | failed | PR head: go test ./... failed |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 构建 | `go build ./...` | 通过 | 在 PR head 临时 clone 中执行。 |
| 测试 | `go test ./...` | 失败 | i18n 检查失败：存在缺失翻译 key。 |

#### 建议 review 文案

当前 PR head 上 `go build ./...` 通过，但 `go test ./...` 失败。请先修复失败测试并补齐相关断言/翻译后再请求复审。

<!-- gitlink-pr-assessor:report v1 -->
### PR #274 维护者评估报告

**标题：** feat(shortcuts): add template shortcuts
**作者：** co63oc
**结论：** 建议修复测试失败后再进入人工评审
**风险等级：** 中
**优先级：** P1
**证据强度：** 高
**执行验证：** failed；集成状态：unverified_unrelated_histories

#### 核心判断
1. 当前 `go test ./...` 已失败，先让作者修复再继续评审。
1. 新增 template 命令后需要同步更新 shortcuts 注册测试期望。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：8 文件，+542 / -0，1 commits；Go 文件 3，测试文件 1，文档/skill 文件 3。
样例文件：README.md, README.zh-CN.md, doc/changes/template-shortcut.md, internal/i18n/locales/en-US.json, internal/i18n/locales/zh-CN.json, shortcuts/register.go

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 中 |
| 代码质量 | 弱 |
| 安全性 | 低-中 |
| 维护成本 | 中 |
| 协作质量 | 中 |
| 回归风险 | 中 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(shortcuts): add template shortcuts | failed | PR head: go test ./... failed |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 构建 | `go build ./...` | 通过 | 在 PR head 临时 clone 中执行。 |
| 测试 | `go test ./...` | 失败 | 注册表测试失败：命令组数量断言未更新。 |

#### 建议 review 文案

当前 PR head 上 `go build ./...` 通过，但 `go test ./...` 失败。请先修复失败测试并补齐相关断言/翻译后再请求复审。

<!-- gitlink-pr-assessor:report v1 -->
### PR #262 维护者评估报告

**标题：** feat(shortcuts): add user key shortcuts
**作者：** co63oc
**结论：** 建议进入合并前人工确认
**风险等级：** 低
**优先级：** P2
**证据强度：** 中
**执行验证：** passed；集成状态：unverified_unrelated_histories

#### 核心判断
1. PR head 构建与测试通过，可进入维护者对功能入口和 API 语义的人工确认。
1. 涉及密码/Token/认证语义，需额外确认 dry-run、日志和错误输出不会泄露敏感值。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：9 文件，+3629 / -7，1 commits；Go 文件 3，测试文件 1，文档/skill 文件 4。
风险信号：secret_or_password。
样例文件：README.md, README.zh-CN.md, doc/changes/user-key-shortcuts.md, internal/i18n/locales/en-US.json, internal/i18n/locales/zh-CN.json, shortcuts/common/prompt.go

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中-强 |
| 实现可行性 | 中 |
| 代码质量 | 中-强 |
| 安全性 | 中 |
| 维护成本 | 低 |
| 协作质量 | 中 |
| 回归风险 | 低 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(shortcuts): add user key shortcuts | passed | PR head: go build ./... and go test ./... passed |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 构建 | `go build ./...` | 通过 | 在 PR head 临时 clone 中执行；集成 merge commit 未验证。 |
| 测试 | `go test ./...` | 通过 | 在 PR head 临时 clone 中执行。 |

#### 建议 review 文案

PR head 上 `go build ./...` 和 `go test ./...` 均通过。建议维护者聚焦新增命令/API 参数、i18n、文档和测试覆盖做最终人工确认。

<!-- gitlink-pr-assessor:report v1 -->
### PR #273 维护者评估报告

**标题：** feat(repo): 补齐仓库治理中的转移与模块配置能力
**作者：** Mengz
**结论：** 建议进入合并前人工确认
**风险等级：** 低
**优先级：** P2
**证据强度：** 中
**执行验证：** passed；集成状态：unverified_unrelated_histories

#### 核心判断
1. PR head 构建与测试通过，可进入维护者对功能入口和 API 语义的人工确认。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：10 文件，+485 / -1，2 commits；Go 文件 2，测试文件 1，文档/skill 文件 6。
样例文件：README.md, README.zh-CN.md, doc/changes/repo-transfer-shortcuts.md, doc/changes/repo-units-shortcut.md, internal/i18n/locales/en-US.json, internal/i18n/locales/zh-CN.json

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中-强 |
| 实现可行性 | 中 |
| 代码质量 | 中-强 |
| 安全性 | 低 |
| 维护成本 | 低 |
| 协作质量 | 中 |
| 回归风险 | 低 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(repo): 补齐仓库治理中的转移与模块配置能力 | passed | PR head: go build ./... and go test ./... passed |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 构建 | `go build ./...` | 通过 | 在 PR head 临时 clone 中执行；集成 merge commit 未验证。 |
| 测试 | `go test ./...` | 通过 | 在 PR head 临时 clone 中执行。 |

#### 建议 review 文案

PR head 上 `go build ./...` 和 `go test ./...` 均通过。建议维护者聚焦新增命令/API 参数、i18n、文档和测试覆盖做最终人工确认。

<!-- gitlink-pr-assessor:report v1 -->
### PR #272 维护者评估报告

**标题：** feat(message): 补齐通知中心与消息设置管理工作流
**作者：** Mengz
**结论：** 建议聚焦代码路径人工确认后推进
**风险等级：** 中
**优先级：** P2
**证据强度：** 中
**执行验证：** passed；集成状态：unverified_unrelated_histories

#### 核心判断
1. PR head 构建与测试通过，可进入维护者对功能入口和 API 语义的人工确认。
1. 改动跨多个文件，建议重点看新增命令注册、i18n、文档与测试是否同步。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：13 文件，+1909 / -46，3 commits；Go 文件 6，测试文件 3，文档/skill 文件 5。
样例文件：README.md, README.zh-CN.md, doc/changes/notification-shortcut.md, internal/i18n/locales/en-US.json, internal/i18n/locales/zh-CN.json, shortcuts/messagesetting/messagesetting.go

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中-强 |
| 实现可行性 | 中 |
| 代码质量 | 中-强 |
| 安全性 | 低 |
| 维护成本 | 中 |
| 协作质量 | 中 |
| 回归风险 | 中 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(message): 补齐通知中心与消息设置管理工作流 | passed | PR head: go build ./... and go test ./... passed |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 构建 | `go build ./...` | 通过 | 在 PR head 临时 clone 中执行；集成 merge commit 未验证。 |
| 测试 | `go test ./...` | 通过 | 在 PR head 临时 clone 中执行。 |

#### 建议 review 文案

PR head 上 `go build ./...` 和 `go test ./...` 均通过。建议维护者聚焦新增命令/API 参数、i18n、文档和测试覆盖做最终人工确认。

<!-- gitlink-pr-assessor:report v1 -->
### PR #271 维护者评估报告

**标题：** feat(issue): 补齐 Issue 批量导出与评论管理工作流
**作者：** Mengz
**结论：** 建议聚焦代码路径人工确认后推进
**风险等级：** 中
**优先级：** P2
**证据强度：** 中
**执行验证：** passed；集成状态：unverified_unrelated_histories

#### 核心判断
1. PR head 构建与测试通过，可进入维护者对功能入口和 API 语义的人工确认。
1. 改动跨多个文件，建议重点看新增命令注册、i18n、文档与测试是否同步。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：12 文件，+1263 / -22，2 commits；Go 文件 3，测试文件 1，文档/skill 文件 7。
样例文件：README.md, README.zh-CN.md, doc/changes/issue-comment-management.md, doc/changes/issue-export-shortcut.md, internal/i18n/locales/en-US.json, internal/i18n/locales/zh-CN.json

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中-强 |
| 实现可行性 | 中 |
| 代码质量 | 中-强 |
| 安全性 | 低 |
| 维护成本 | 中 |
| 协作质量 | 中 |
| 回归风险 | 中 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(issue): 补齐 Issue 批量导出与评论管理工作流 | passed | PR head: go build ./... and go test ./... passed |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 构建 | `go build ./...` | 通过 | 在 PR head 临时 clone 中执行；集成 merge commit 未验证。 |
| 测试 | `go test ./...` | 通过 | 在 PR head 临时 clone 中执行。 |

#### 建议 review 文案

PR head 上 `go build ./...` 和 `go test ./...` 均通过。建议维护者聚焦新增命令/API 参数、i18n、文档和测试覆盖做最终人工确认。

<!-- gitlink-pr-assessor:report v1 -->
### PR #270 维护者评估报告

**标题：** feat(message-settings): 增加消息通知设置快捷命令
**作者：** Mengz
**结论：** 建议进入合并前人工确认
**风险等级：** 低
**优先级：** P2
**证据强度：** 中
**执行验证：** passed；集成状态：unverified_unrelated_histories

#### 核心判断
1. PR head 构建与测试通过，可进入维护者对功能入口和 API 语义的人工确认。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：9 文件，+1345 / -44，2 commits；Go 文件 4，测试文件 2，文档/skill 文件 3。
样例文件：README.md, README.zh-CN.md, doc/changes/message-settings-shortcut.md, internal/i18n/locales/en-US.json, internal/i18n/locales/zh-CN.json, shortcuts/messagesetting/messagesetting.go

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中-强 |
| 实现可行性 | 中 |
| 代码质量 | 中-强 |
| 安全性 | 低 |
| 维护成本 | 低 |
| 协作质量 | 中 |
| 回归风险 | 低 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(message-settings): 增加消息通知设置快捷命令 | passed | PR head: go build ./... and go test ./... passed |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 构建 | `go build ./...` | 通过 | 在 PR head 临时 clone 中执行；集成 merge commit 未验证。 |
| 测试 | `go test ./...` | 通过 | 在 PR head 临时 clone 中执行。 |

#### 建议 review 文案

PR head 上 `go build ./...` 和 `go test ./...` 均通过。建议维护者聚焦新增命令/API 参数、i18n、文档和测试覆盖做最终人工确认。

<!-- gitlink-pr-assessor:report v1 -->
### PR #269 维护者评估报告

**标题：** 新增gitlink-dev-full-cycle技能
**作者：** wdgde
**结论：** 建议维护者确认安全表述后再合入
**风险等级：** 中
**优先级：** P3
**证据强度：** 中
**执行验证：** static_only；集成状态：not_required

#### 核心判断
1. 确认新增 skill/文档是否符合仓库技能索引、命名和安全边界。
1. 凭据/Token/密码相关表述需要避免鼓励明文收集或持久化。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：3 文件，+767 / -0，1 commits；Go 文件 0，测试文件 0，文档/skill 文件 3。
风险信号：secret_or_password。
样例文件：skills/gitlink-dev-full-cycle/SKILL.md, skills/gitlink-dev-full-cycle/references/checklist.md, skills/gitlink-dev-full-cycle/references/templates.md

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 中 |
| 代码质量 | 不适用 |
| 安全性 | 中 |
| 维护成本 | 低-中 |
| 协作质量 | 中 |
| 回归风险 | 低 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：新增gitlink-dev-full-cycle技能 | insufficient | no Go code changed |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 静态检查 | 文件/声明检查 | 完成 | 未改 Go 代码，未运行 Go 构建/测试。 |

#### 建议 review 文案

改动主要是文档/skill，建议维护者确认命名、索引、示例和安全表述是否符合仓库规范；确认后可继续推进。

<!-- gitlink-pr-assessor:report v1 -->
### PR #268 维护者评估报告

**标题：** feat(branch): add lifecycle shortcuts
**作者：** wangyue111
**结论：** 建议进入合并前人工确认
**风险等级：** 低
**优先级：** P2
**证据强度：** 中
**执行验证：** passed；集成状态：unverified_unrelated_histories

#### 核心判断
1. PR head 构建与测试通过，可进入维护者对功能入口和 API 语义的人工确认。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：6 文件，+363 / -2，1 commits；Go 文件 2，测试文件 1，文档/skill 文件 4。
样例文件：README.md, README.zh-CN.md, doc/changes/branch-lifecycle-shortcuts.md, shortcuts/branch/branch.go, shortcuts/branch/branch_test.go, skills/gitlink-branch/SKILL.md

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中-强 |
| 实现可行性 | 中 |
| 代码质量 | 中-强 |
| 安全性 | 低 |
| 维护成本 | 低 |
| 协作质量 | 中 |
| 回归风险 | 低 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(branch): add lifecycle shortcuts | passed | PR head: go build ./... and go test ./... passed |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 构建 | `go build ./...` | 通过 | 在 PR head 临时 clone 中执行；集成 merge commit 未验证。 |
| 测试 | `go test ./...` | 通过 | 在 PR head 临时 clone 中执行。 |

#### 建议 review 文案

PR head 上 `go build ./...` 和 `go test ./...` 均通过。建议维护者聚焦新增命令/API 参数、i18n、文档和测试覆盖做最终人工确认。

<!-- gitlink-pr-assessor:report v1 -->
### PR #267 维护者评估报告

**标题：** feat(member): add application shortcuts
**作者：** wangyue111
**结论：** 建议进入合并前人工确认
**风险等级：** 低
**优先级：** P2
**证据强度：** 中
**执行验证：** passed；集成状态：unverified_unrelated_histories

#### 核心判断
1. PR head 构建与测试通过，可进入维护者对功能入口和 API 语义的人工确认。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：6 文件，+241 / -1，1 commits；Go 文件 2，测试文件 1，文档/skill 文件 4。
样例文件：README.md, README.zh-CN.md, doc/changes/member-application-shortcuts.md, shortcuts/member/member.go, shortcuts/member/member_test.go, skills/gitlink-member/SKILL.md

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中-强 |
| 实现可行性 | 中 |
| 代码质量 | 中-强 |
| 安全性 | 低 |
| 维护成本 | 低 |
| 协作质量 | 中 |
| 回归风险 | 低 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(member): add application shortcuts | passed | PR head: go build ./... and go test ./... passed |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 构建 | `go build ./...` | 通过 | 在 PR head 临时 clone 中执行；集成 merge commit 未验证。 |
| 测试 | `go test ./...` | 通过 | 在 PR head 临时 clone 中执行。 |

#### 建议 review 文案

PR head 上 `go build ./...` 和 `go test ./...` 均通过。建议维护者聚焦新增命令/API 参数、i18n、文档和测试覆盖做最终人工确认。

<!-- gitlink-pr-assessor:report v1 -->
### PR #265 维护者评估报告

**标题：** feat(skills): 新增 PR审查效率看板 的skill : gitlink-pr-review-quality
**作者：** yangsai
**结论：** 建议进入维护者内容确认
**风险等级：** 低
**优先级：** P3
**证据强度：** 中
**执行验证：** static_only；集成状态：not_required

#### 核心判断
1. 确认新增 skill/文档是否符合仓库技能索引、命名和安全边界。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：2 文件，+449 / -1，1 commits；Go 文件 0，测试文件 0，文档/skill 文件 2。
样例文件：skills/gitlink-pr-review-quality/SKILL.md, skills/gitlink-shared/SKILL.md

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 中 |
| 代码质量 | 不适用 |
| 安全性 | 低 |
| 维护成本 | 低-中 |
| 协作质量 | 中 |
| 回归风险 | 低 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(skills): 新增 PR审查效率看板 的skill : gitlink-pr-review-quality | insufficient | no Go code changed |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 静态检查 | 文件/声明检查 | 完成 | 未改 Go 代码，未运行 Go 构建/测试。 |

#### 建议 review 文案

改动主要是文档/skill，建议维护者确认命名、索引、示例和安全表述是否符合仓库规范；确认后可继续推进。

<!-- gitlink-pr-assessor:report v1 -->
### PR #263 维护者评估报告

**标题：** 新增 template 命令，支持 GitLink 平台项目模板的增删改查操作
**作者：** wdgde
**结论：** 建议作者先 rebase/解决冲突后再审
**风险等级：** 中
**优先级：** P2
**证据强度：** 中
**执行验证：** blocked；集成状态：blocked

#### 核心判断
1. 当前不可合并，维护者优先要求作者 rebase 到最新 master。
1. 解决冲突后再重新运行构建/测试和声明验证。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：6 文件，+305 / -1，1 commits；Go 文件 4，测试文件 2，文档/skill 文件 2。
冲突文件：shortcuts/register_test.go
样例文件：README.md, README.zh-CN.md, shortcuts/register.go, shortcuts/register_test.go, shortcuts/template/template.go, shortcuts/template/template_test.go

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 弱 |
| 代码质量 | 证据不足 |
| 安全性 | 低-中 |
| 维护成本 | 高 |
| 协作质量 | 中 |
| 回归风险 | 高 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：新增 template 命令，支持 GitLink 平台项目模板的增删改查操作 | blocked | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 集成 | mergeability/conflict 检查 | 阻塞 | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 建议 review 文案

当前 PR 与 master 不可直接集成，建议先 rebase/解决冲突。冲突解决后请重新跑 `go build ./...` 和 `go test ./...`，再进入功能评审。

<!-- gitlink-pr-assessor:report v1 -->
### PR #261 维护者评估报告

**标题：** 添加 gitlink-cli auth checkin 命令，用于定时刷新认证会话，防止用户登录状态过期
**作者：** wdgde
**结论：** 建议作者先 rebase/解决冲突后再审
**风险等级：** 高
**优先级：** P2
**证据强度：** 中
**执行验证：** blocked；集成状态：blocked

#### 核心判断
1. 当前不可合并，维护者优先要求作者 rebase 到最新 master。
1. 解决冲突后再重新运行构建/测试和声明验证。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：10 文件，+433 / -2，2 commits；Go 文件 6，测试文件 3，文档/skill 文件 2。
风险信号：secret_or_password。
冲突文件：shortcuts/register_test.go
样例文件：README.md, README.zh-CN.md, cmd/auth/auth.go, cmd/auth/auth_test.go, internal/i18n/locales/en-US.json, internal/i18n/locales/zh-CN.json

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 弱 |
| 代码质量 | 证据不足 |
| 安全性 | 低-中 |
| 维护成本 | 高 |
| 协作质量 | 中 |
| 回归风险 | 高 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：添加 gitlink-cli auth checkin 命令，用于定时刷新认证会话，防止用户登录状态过期 | blocked | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 集成 | mergeability/conflict 检查 | 阻塞 | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 建议 review 文案

当前 PR 与 master 不可直接集成，建议先 rebase/解决冲突。冲突解决后请重新跑 `go build ./...` 和 `go test ./...`，再进入功能评审。

<!-- gitlink-pr-assessor:report v1 -->
### PR #238 维护者评估报告

**标题：** feat(user): add statistics shortcuts
**作者：** wangyue111
**结论：** 建议进入合并前人工确认
**风险等级：** 低
**优先级：** P2
**证据强度：** 中
**执行验证：** passed；集成状态：unverified_unrelated_histories

#### 核心判断
1. PR head 构建与测试通过，可进入维护者对功能入口和 API 语义的人工确认。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：6 文件，+392 / -7，1 commits；Go 文件 2，测试文件 1，文档/skill 文件 4。
样例文件：README.md, README.zh-CN.md, doc/changes/user-statistics-shortcuts.md, shortcuts/user/user.go, shortcuts/user/user_test.go, skills/gitlink-user/SKILL.md

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中-强 |
| 实现可行性 | 中 |
| 代码质量 | 中-强 |
| 安全性 | 低 |
| 维护成本 | 低 |
| 协作质量 | 中 |
| 回归风险 | 低 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(user): add statistics shortcuts | passed | PR head: go build ./... and go test ./... passed |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 构建 | `go build ./...` | 通过 | 在 PR head 临时 clone 中执行；集成 merge commit 未验证。 |
| 测试 | `go test ./...` | 通过 | 在 PR head 临时 clone 中执行。 |

#### 建议 review 文案

PR head 上 `go build ./...` 和 `go test ./...` 均通过。建议维护者聚焦新增命令/API 参数、i18n、文档和测试覆盖做最终人工确认。

<!-- gitlink-pr-assessor:report v1 -->
### PR #259 维护者评估报告

**标题：** 修改skill
**作者：** nudt_zk
**结论：** 建议拒绝当前形态或要求拆分重做
**风险等级：** 高
**优先级：** P1
**证据强度：** 高
**执行验证：** blocked；集成状态：blocked

#### 核心判断
1. 先移除或重做部署自动化中的硬编码 IP、root 登录和破坏性命令。
1. 改动面过大，建议拆分为独立功能/文档/skill PR。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：156 文件，+37949 / -285，67 commits；Go 文件 44，测试文件 9，文档/skill 文件 98。
风险信号：hardcoded_root, hardcoded_ip, secret_or_password, external_ci。
冲突文件：README.md, README.zh-CN.md, cmd/api/api.go, cmd/auth/auth.go, doc/design.md, internal/client/client.go, internal/config/config.go, shortcuts/branch/branch.go ...
样例文件：.devops/gitlink-cli-autodeploy.yml, README.md, README.zh-CN.md, cmd/api/api.go, cmd/auth/auth.go, doc/INSTALL.md

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 弱 |
| 代码质量 | 弱 |
| 安全性 | 高风险 |
| 维护成本 | 高 |
| 协作质量 | 弱 |
| 回归风险 | 高 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 按标题声明完成变更：修改skill | blocked | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 集成 | mergeability/conflict 检查 | 阻塞 | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 建议 review 文案

建议暂不合入。请先拆分 PR，并移除部署自动化中的硬编码服务器/root 登录/破坏性 shell 命令；解决与 master 的冲突后，再针对单一能力补充测试和说明。

<!-- gitlink-pr-assessor:report v1 -->
### PR #258 维护者评估报告

**标题：** fix(output): deterministic --format table ordering and show error status code
**作者：** luwanzhou
**结论：** 建议进入合并前人工确认
**风险等级：** 低
**优先级：** P2
**证据强度：** 中
**执行验证：** passed；集成状态：unverified_unrelated_histories

#### 核心判断
1. PR head 构建与测试通过，可进入维护者对功能入口和 API 语义的人工确认。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：3 文件，+119 / -6，1 commits；Go 文件 2，测试文件 1，文档/skill 文件 1。
样例文件：doc/changes/output-table-determinism.md, internal/output/formatter.go, internal/output/formatter_test.go

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中-强 |
| 实现可行性 | 中 |
| 代码质量 | 中-强 |
| 安全性 | 低 |
| 维护成本 | 低 |
| 协作质量 | 中 |
| 回归风险 | 低 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 修复/改进：fix(output): deterministic --format table ordering and show error status code | passed | PR head: go build ./... and go test ./... passed |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 构建 | `go build ./...` | 通过 | 在 PR head 临时 clone 中执行；集成 merge commit 未验证。 |
| 测试 | `go test ./...` | 通过 | 在 PR head 临时 clone 中执行。 |

#### 建议 review 文案

PR head 上 `go build ./...` 和 `go test ./...` 均通过。建议维护者聚焦新增命令/API 参数、i18n、文档和测试覆盖做最终人工确认。

<!-- gitlink-pr-assessor:report v1 -->
### PR #257 维护者评估报告

**标题：** feat(skills): 新增 科研复现性评估的skill : gitlink-research-reproducibility
**作者：** luwanzhou
**结论：** 建议进入维护者内容确认
**风险等级：** 低
**优先级：** P3
**证据强度：** 中
**执行验证：** static_only；集成状态：not_required

#### 核心判断
1. 确认新增 skill/文档是否符合仓库技能索引、命名和安全边界。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：3 文件，+597 / -0，1 commits；Go 文件 0，测试文件 0，文档/skill 文件 3。
风险信号：external_ci。
样例文件：skills/gitlink-research-reproducibility/SKILL.md, skills/gitlink-research-reproducibility/examples/real-eval-pdeep.md, skills/gitlink-research-reproducibility/examples/reproducibility-workflow.md

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 中 |
| 代码质量 | 不适用 |
| 安全性 | 低 |
| 维护成本 | 低-中 |
| 协作质量 | 中 |
| 回归风险 | 低 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(skills): 新增 科研复现性评估的skill : gitlink-research-reproducibility | insufficient | no Go code changed |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 静态检查 | 文件/声明检查 | 完成 | 未改 Go 代码，未运行 Go 构建/测试。 |

#### 建议 review 文案

改动主要是文档/skill，建议维护者确认命名、索引、示例和安全表述是否符合仓库规范；确认后可继续推进。

<!-- gitlink-pr-assessor:report v1 -->
### PR #256 维护者评估报告

**标题：** feat(skills): 新增 科研进度跟踪与预警的skill : gitlink-research-progress
**作者：** luwanzhou
**结论：** 建议进入维护者内容确认
**风险等级：** 低
**优先级：** P3
**证据强度：** 中
**执行验证：** static_only；集成状态：not_required

#### 核心判断
1. 确认新增 skill/文档是否符合仓库技能索引、命名和安全边界。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：2 文件，+371 / -0，1 commits；Go 文件 0，测试文件 0，文档/skill 文件 2。
样例文件：skills/gitlink-research-progress/SKILL.md, skills/gitlink-research-progress/examples/progress-workflow.md

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 中 |
| 代码质量 | 不适用 |
| 安全性 | 低 |
| 维护成本 | 低-中 |
| 协作质量 | 中 |
| 回归风险 | 低 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(skills): 新增 科研进度跟踪与预警的skill : gitlink-research-progress | insufficient | no Go code changed |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 静态检查 | 文件/声明检查 | 完成 | 未改 Go 代码，未运行 Go 构建/测试。 |

#### 建议 review 文案

改动主要是文档/skill，建议维护者确认命名、索引、示例和安全表述是否符合仓库规范；确认后可继续推进。

<!-- gitlink-pr-assessor:report v1 -->
### PR #255 维护者评估报告

**标题：**  feat(skills): 新增 科研协作智能匹配的skill : gitlink-research-match
**作者：** luwanzhou
**结论：** 建议进入维护者内容确认
**风险等级：** 低
**优先级：** P3
**证据强度：** 中
**执行验证：** static_only；集成状态：not_required

#### 核心判断
1. 确认新增 skill/文档是否符合仓库技能索引、命名和安全边界。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：2 文件，+328 / -0，1 commits；Go 文件 0，测试文件 0，文档/skill 文件 2。
样例文件：skills/gitlink-research-match/SKILL.md, skills/gitlink-research-match/examples/match-workflow.md

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 中 |
| 代码质量 | 不适用 |
| 安全性 | 低 |
| 维护成本 | 低-中 |
| 协作质量 | 中 |
| 回归风险 | 低 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(skills): 新增 科研协作智能匹配的skill : gitlink-research-match | insufficient | no Go code changed |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 静态检查 | 文件/声明检查 | 完成 | 未改 Go 代码，未运行 Go 构建/测试。 |

#### 建议 review 文案

改动主要是文档/skill，建议维护者确认命名、索引、示例和安全表述是否符合仓库规范；确认后可继续推进。

<!-- gitlink-pr-assessor:report v1 -->
### PR #254 维护者评估报告

**标题：** fix(api): resolve :owner/:repo and --var placeholders in single calls
**作者：** luwanzhou
**结论：** 建议进入合并前人工确认
**风险等级：** 低
**优先级：** P2
**证据强度：** 中
**执行验证：** passed；集成状态：unverified_unrelated_histories

#### 核心判断
1. PR head 构建与测试通过，可进入维护者对功能入口和 API 语义的人工确认。
1. 涉及密码/Token/认证语义，需额外确认 dry-run、日志和错误输出不会泄露敏感值。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：3 文件，+139 / -4，1 commits；Go 文件 2，测试文件 1，文档/skill 文件 1。
风险信号：secret_or_password。
样例文件：cmd/api/api.go, cmd/api/api_test.go, doc/changes/api-path-placeholders.md

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中-强 |
| 实现可行性 | 中 |
| 代码质量 | 中-强 |
| 安全性 | 中 |
| 维护成本 | 低 |
| 协作质量 | 中 |
| 回归风险 | 低 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 修复/改进：fix(api): resolve :owner/:repo and --var placeholders in single calls | passed | PR head: go build ./... and go test ./... passed |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 构建 | `go build ./...` | 通过 | 在 PR head 临时 clone 中执行；集成 merge commit 未验证。 |
| 测试 | `go test ./...` | 通过 | 在 PR head 临时 clone 中执行。 |

#### 建议 review 文案

PR head 上 `go build ./...` 和 `go test ./...` 均通过。建议维护者聚焦新增命令/API 参数、i18n、文档和测试覆盖做最终人工确认。

<!-- gitlink-pr-assessor:report v1 -->
### PR #167 维护者评估报告

**标题：** feat(account): add email verification shortcuts
**作者：** wangyue111
**结论：** 建议作者先 rebase/解决冲突后再审
**风险等级：** 中
**优先级：** P2
**证据强度：** 中
**执行验证：** blocked；集成状态：blocked

#### 核心判断
1. 当前不可合并，维护者优先要求作者 rebase 到最新 master。
1. 解决冲突后再重新运行构建/测试和声明验证。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：9 文件，+611 / -1，1 commits；Go 文件 4，测试文件 2，文档/skill 文件 5。
风险信号：secret_or_password。
冲突文件：shortcuts/register_test.go
样例文件：README.md, README.zh-CN.md, doc/changes/account-email-shortcuts.md, shortcuts/account/account.go, shortcuts/account/account_test.go, shortcuts/register.go

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 弱 |
| 代码质量 | 证据不足 |
| 安全性 | 低-中 |
| 维护成本 | 高 |
| 协作质量 | 中 |
| 回归风险 | 高 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(account): add email verification shortcuts | blocked | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 集成 | mergeability/conflict 检查 | 阻塞 | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 建议 review 文案

当前 PR 与 master 不可直接集成，建议先 rebase/解决冲突。冲突解决后请重新跑 `go build ./...` 和 `go test ./...`，再进入功能评审。

<!-- gitlink-pr-assessor:report v1 -->
### PR #151 维护者评估报告

**标题：** feat(transfer): add transfer request shortcuts
**作者：** wangyue111
**结论：** 建议作者先 rebase/解决冲突后再审
**风险等级：** 中
**优先级：** P2
**证据强度：** 中
**执行验证：** blocked；集成状态：blocked

#### 核心判断
1. 当前不可合并，维护者优先要求作者 rebase 到最新 master。
1. 解决冲突后再重新运行构建/测试和声明验证。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：9 文件，+452 / -50，1 commits；Go 文件 4，测试文件 2，文档/skill 文件 5。
冲突文件：README.md, README.zh-CN.md, shortcuts/register.go, shortcuts/register_test.go
样例文件：README.md, README.zh-CN.md, doc/changes/transfer-request-shortcuts.md, shortcuts/register.go, shortcuts/register_test.go, shortcuts/transferrequest/transferrequest.go

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 弱 |
| 代码质量 | 证据不足 |
| 安全性 | 低-中 |
| 维护成本 | 高 |
| 协作质量 | 中 |
| 回归风险 | 高 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(transfer): add transfer request shortcuts | blocked | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 集成 | mergeability/conflict 检查 | 阻塞 | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 建议 review 文案

当前 PR 与 master 不可直接集成，建议先 rebase/解决冲突。冲突解决后请重新跑 `go build ./...` 和 `go test ./...`，再进入功能评审。

<!-- gitlink-pr-assessor:report v1 -->
### PR #135 维护者评估报告

**标题：** feat(dev): add developer resource shortcuts
**作者：** wangyue111
**结论：** 建议作者先 rebase/解决冲突后再审
**风险等级：** 高
**优先级：** P2
**证据强度：** 中
**执行验证：** blocked；集成状态：blocked

#### 核心判断
1. 当前不可合并，维护者优先要求作者 rebase 到最新 master。
1. 解决冲突后再重新运行构建/测试和声明验证。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：11 文件，+774 / -50，1 commits；Go 文件 6，测试文件 3，文档/skill 文件 5。
冲突文件：README.md, README.zh-CN.md, shortcuts/register.go, shortcuts/register_test.go
样例文件：README.md, README.zh-CN.md, doc/changes/developer-resource-shortcuts.md, shortcuts/feedback/feedback.go, shortcuts/feedback/feedback_test.go, shortcuts/key/key.go

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 弱 |
| 代码质量 | 证据不足 |
| 安全性 | 低-中 |
| 维护成本 | 高 |
| 协作质量 | 中 |
| 回归风险 | 高 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(dev): add developer resource shortcuts | blocked | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 集成 | mergeability/conflict 检查 | 阻塞 | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 建议 review 文案

当前 PR 与 master 不可直接集成，建议先 rebase/解决冲突。冲突解决后请重新跑 `go build ./...` 和 `go test ./...`，再进入功能评审。

<!-- gitlink-pr-assessor:report v1 -->
### PR #82 维护者评估报告

**标题：** feat(meta): add attachment and metadata shortcuts
**作者：** wangyue111
**结论：** 建议作者先 rebase/解决冲突后再审
**风险等级：** 高
**优先级：** P2
**证据强度：** 中
**执行验证：** blocked；集成状态：blocked

#### 核心判断
1. 当前不可合并，维护者优先要求作者 rebase 到最新 master。
1. 解决冲突后再重新运行构建/测试和声明验证。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：12 文件，+780 / -42，1 commits；Go 文件 6，测试文件 3，文档/skill 文件 6。
冲突文件：README.md, README.zh-CN.md, shortcuts/register.go, shortcuts/register_test.go, skills/README.md
样例文件：README.md, README.zh-CN.md, doc/changes/meta-attachment-shortcuts.md, shortcuts/attachment/attachment.go, shortcuts/attachment/attachment_test.go, shortcuts/meta/meta.go

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 弱 |
| 代码质量 | 证据不足 |
| 安全性 | 低-中 |
| 维护成本 | 高 |
| 协作质量 | 中 |
| 回归风险 | 高 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(meta): add attachment and metadata shortcuts | blocked | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 集成 | mergeability/conflict 检查 | 阻塞 | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 建议 review 文案

当前 PR 与 master 不可直接集成，建议先 rebase/解决冲突。冲突解决后请重新跑 `go build ./...` 和 `go test ./...`，再进入功能评审。

<!-- gitlink-pr-assessor:report v1 -->
### PR #76 维护者评估报告

**标题：** feat(notification): add OpenAPI shortcuts
**作者：** wangyue111
**结论：** 建议作者先 rebase/解决冲突后再审
**风险等级：** 中
**优先级：** P2
**证据强度：** 中
**执行验证：** blocked；集成状态：blocked

#### 核心判断
1. 当前不可合并，维护者优先要求作者 rebase 到最新 master。
1. 解决冲突后再重新运行构建/测试和声明验证。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：9 文件，+1042 / -42，1 commits；Go 文件 4，测试文件 2，文档/skill 文件 5。
冲突文件：README.md, README.zh-CN.md, shortcuts/register.go, shortcuts/register_test.go, skills/README.md
样例文件：README.md, README.zh-CN.md, doc/changes/notification-shortcut.md, shortcuts/notification/notification.go, shortcuts/notification/notification_test.go, shortcuts/register.go

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 弱 |
| 代码质量 | 证据不足 |
| 安全性 | 低-中 |
| 维护成本 | 高 |
| 协作质量 | 中 |
| 回归风险 | 高 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(notification): add OpenAPI shortcuts | blocked | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 集成 | mergeability/conflict 检查 | 阻塞 | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 建议 review 文案

当前 PR 与 master 不可直接集成，建议先 rebase/解决冲突。冲突解决后请重新跑 `go build ./...` 和 `go test ./...`，再进入功能评审。

<!-- gitlink-pr-assessor:report v1 -->
### PR #72 维护者评估报告

**标题：** feat(template): add project template shortcuts
**作者：** wangyue111
**结论：** 建议作者先 rebase/解决冲突后再审
**风险等级：** 中
**优先级：** P2
**证据强度：** 中
**执行验证：** blocked；集成状态：blocked

#### 核心判断
1. 当前不可合并，维护者优先要求作者 rebase 到最新 master。
1. 解决冲突后再重新运行构建/测试和声明验证。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：9 文件，+682 / -1，1 commits；Go 文件 4，测试文件 2，文档/skill 文件 5。
冲突文件：README.md, README.zh-CN.md, shortcuts/register_test.go, skills/README.md
样例文件：README.md, README.zh-CN.md, doc/changes/template-shortcuts.md, shortcuts/register.go, shortcuts/register_test.go, shortcuts/template/template.go

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 弱 |
| 代码质量 | 证据不足 |
| 安全性 | 低-中 |
| 维护成本 | 高 |
| 协作质量 | 中 |
| 回归风险 | 高 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(template): add project template shortcuts | blocked | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 集成 | mergeability/conflict 检查 | 阻塞 | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 建议 review 文案

当前 PR 与 master 不可直接集成，建议先 rebase/解决冲突。冲突解决后请重新跑 `go build ./...` 和 `go test ./...`，再进入功能评审。

<!-- gitlink-pr-assessor:report v1 -->
### PR #70 维护者评估报告

**标题：** feat(user): add account and stats shortcuts
**作者：** wangyue111
**结论：** 建议作者先 rebase/解决冲突后再审
**风险等级：** 中
**优先级：** P2
**证据强度：** 中
**执行验证：** blocked；集成状态：blocked

#### 核心判断
1. 当前不可合并，维护者优先要求作者 rebase 到最新 master。
1. 解决冲突后再重新运行构建/测试和声明验证。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：7 文件，+544 / -81，1 commits；Go 文件 2，测试文件 1，文档/skill 文件 5。
冲突文件：README.md, README.zh-CN.md
样例文件：README.md, README.zh-CN.md, doc/changes/user-account-stats-shortcuts.md, shortcuts/user/user.go, shortcuts/user/user_test.go, skills/README.md

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 弱 |
| 代码质量 | 证据不足 |
| 安全性 | 低-中 |
| 维护成本 | 高 |
| 协作质量 | 中 |
| 回归风险 | 高 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(user): add account and stats shortcuts | blocked | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 集成 | mergeability/conflict 检查 | 阻塞 | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 建议 review 文案

当前 PR 与 master 不可直接集成，建议先 rebase/解决冲突。冲突解决后请重新跑 `go build ./...` 和 `go test ./...`，再进入功能评审。

<!-- gitlink-pr-assessor:report v1 -->
### PR #65 维护者评估报告

**标题：** feat(wiki): add OpenAPI shortcuts
**作者：** wangyue111
**结论：** 建议作者先 rebase/解决冲突后再审
**风险等级：** 中
**优先级：** P2
**证据强度：** 中
**执行验证：** blocked；集成状态：blocked

#### 核心判断
1. 当前不可合并，维护者优先要求作者 rebase 到最新 master。
1. 解决冲突后再重新运行构建/测试和声明验证。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：8 文件，+580 / -1，1 commits；Go 文件 4，测试文件 2，文档/skill 文件 4。
冲突文件：README.md, README.zh-CN.md, shortcuts/register.go, shortcuts/register_test.go, shortcuts/wiki/wiki.go, shortcuts/wiki/wiki_test.go, skills/gitlink-wiki/SKILL.md
样例文件：README.md, README.zh-CN.md, doc/changes/wiki-openapi-shortcuts.md, shortcuts/register.go, shortcuts/register_test.go, shortcuts/wiki/wiki.go

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 弱 |
| 代码质量 | 证据不足 |
| 安全性 | 低-中 |
| 维护成本 | 高 |
| 协作质量 | 中 |
| 回归风险 | 高 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(wiki): add OpenAPI shortcuts | blocked | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 集成 | mergeability/conflict 检查 | 阻塞 | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 建议 review 文案

当前 PR 与 master 不可直接集成，建议先 rebase/解决冲突。冲突解决后请重新跑 `go build ./...` 和 `go test ./...`，再进入功能评审。

<!-- gitlink-pr-assessor:report v1 -->
### PR #64 维护者评估报告

**标题：** feat(dataset): add OpenAPI shortcuts
**作者：** wangyue111
**结论：** 建议作者先 rebase/解决冲突后再审
**风险等级：** 中
**优先级：** P2
**证据强度：** 中
**执行验证：** blocked；集成状态：blocked

#### 核心判断
1. 当前不可合并，维护者优先要求作者 rebase 到最新 master。
1. 解决冲突后再重新运行构建/测试和声明验证。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：8 文件，+501 / -1，1 commits；Go 文件 4，测试文件 2，文档/skill 文件 4。
冲突文件：README.md, README.zh-CN.md, shortcuts/dataset/dataset.go, shortcuts/dataset/dataset_test.go, shortcuts/register.go, shortcuts/register_test.go
样例文件：README.md, README.zh-CN.md, doc/changes/dataset-openapi-shortcuts.md, shortcuts/dataset/dataset.go, shortcuts/dataset/dataset_test.go, shortcuts/register.go

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 弱 |
| 代码质量 | 证据不足 |
| 安全性 | 低-中 |
| 维护成本 | 高 |
| 协作质量 | 中 |
| 回归风险 | 高 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(dataset): add OpenAPI shortcuts | blocked | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 集成 | mergeability/conflict 检查 | 阻塞 | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 建议 review 文案

当前 PR 与 master 不可直接集成，建议先 rebase/解决冲突。冲突解决后请重新跑 `go build ./...` 和 `go test ./...`，再进入功能评审。

<!-- gitlink-pr-assessor:report v1 -->
### PR #63 维护者评估报告

**标题：** feat(code): add repository code OpenAPI shortcuts
**作者：** wangyue111
**结论：** 建议作者先 rebase/解决冲突后再审
**风险等级：** 中
**优先级：** P2
**证据强度：** 中
**执行验证：** blocked；集成状态：blocked

#### 核心判断
1. 当前不可合并，维护者优先要求作者 rebase 到最新 master。
1. 解决冲突后再重新运行构建/测试和声明验证。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：8 文件，+697 / -1，1 commits；Go 文件 4，测试文件 2，文档/skill 文件 4。
冲突文件：README.md, README.zh-CN.md, shortcuts/register_test.go
样例文件：README.md, README.zh-CN.md, doc/changes/code-openapi-shortcuts.md, shortcuts/code/code.go, shortcuts/code/code_test.go, shortcuts/register.go

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 弱 |
| 代码质量 | 证据不足 |
| 安全性 | 低-中 |
| 维护成本 | 高 |
| 协作质量 | 中 |
| 回归风险 | 高 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(code): add repository code OpenAPI shortcuts | blocked | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 集成 | mergeability/conflict 检查 | 阻塞 | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 建议 review 文案

当前 PR 与 master 不可直接集成，建议先 rebase/解决冲突。冲突解决后请重新跑 `go build ./...` 和 `go test ./...`，再进入功能评审。

<!-- gitlink-pr-assessor:report v1 -->
### PR #107 维护者评估报告

**标题：** feat(star): add starred project shortcuts
**作者：** wangyue111
**结论：** 建议作者先 rebase/解决冲突后再审
**风险等级：** 中
**优先级：** P2
**证据强度：** 中
**执行验证：** blocked；集成状态：blocked

#### 核心判断
1. 当前不可合并，维护者优先要求作者 rebase 到最新 master。
1. 解决冲突后再重新运行构建/测试和声明验证。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：9 文件，+576 / -20，1 commits；Go 文件 4，测试文件 2，文档/skill 文件 5。
冲突文件：skills/README.md
样例文件：README.md, README.zh-CN.md, doc/changes/star-openapi-shortcuts.md, shortcuts/register.go, shortcuts/register_test.go, shortcuts/star/star.go

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 弱 |
| 代码质量 | 证据不足 |
| 安全性 | 低-中 |
| 维护成本 | 高 |
| 协作质量 | 中 |
| 回归风险 | 高 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(star): add starred project shortcuts | blocked | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 集成 | mergeability/conflict 检查 | 阻塞 | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 建议 review 文案

当前 PR 与 master 不可直接集成，建议先 rebase/解决冲突。冲突解决后请重新跑 `go build ./...` 和 `go test ./...`，再进入功能评审。

<!-- gitlink-pr-assessor:report v1 -->
### PR #83 维护者评估报告

**标题：** feat(org): add team project binding shortcuts
**作者：** wangyue111
**结论：** 建议进入合并前人工确认
**风险等级：** 低
**优先级：** P2
**证据强度：** 中
**执行验证：** passed；集成状态：unverified_unrelated_histories

#### 核心判断
1. PR head 构建与测试通过，可进入维护者对功能入口和 API 语义的人工确认。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：8 文件，+376 / -96，1 commits；Go 文件 2，测试文件 1，文档/skill 文件 6。
样例文件：README.md, README.zh-CN.md, doc/changes/org-team-projects.md, shortcuts/org/org.go, shortcuts/org/org_test.go, skills/README.md

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中-强 |
| 实现可行性 | 中 |
| 代码质量 | 中-强 |
| 安全性 | 低 |
| 维护成本 | 低 |
| 协作质量 | 中 |
| 回归风险 | 低 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(org): add team project binding shortcuts | passed | PR head: go build ./... and go test ./... passed |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 构建 | `go build ./...` | 通过 | 在 PR head 临时 clone 中执行；集成 merge commit 未验证。 |
| 测试 | `go test ./...` | 通过 | 在 PR head 临时 clone 中执行。 |

#### 建议 review 文案

PR head 上 `go build ./...` 和 `go test ./...` 均通过。建议维护者聚焦新增命令/API 参数、i18n、文档和测试覆盖做最终人工确认。

<!-- gitlink-pr-assessor:report v1 -->
### PR #78 维护者评估报告

**标题：** feat(branch): complete OpenAPI shortcuts
**作者：** wangyue111
**结论：** 建议进入合并前人工确认
**风险等级：** 低
**优先级：** P2
**证据强度：** 中
**执行验证：** passed；集成状态：unverified_unrelated_histories

#### 核心判断
1. PR head 构建与测试通过，可进入维护者对功能入口和 API 语义的人工确认。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：7 文件，+535 / -127，1 commits；Go 文件 2，测试文件 1，文档/skill 文件 5。
样例文件：README.md, README.zh-CN.md, doc/changes/branch-openapi-shortcuts.md, shortcuts/branch/branch.go, shortcuts/branch/branch_test.go, skills/README.md

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中-强 |
| 实现可行性 | 中 |
| 代码质量 | 中-强 |
| 安全性 | 低 |
| 维护成本 | 低 |
| 协作质量 | 中 |
| 回归风险 | 低 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(branch): complete OpenAPI shortcuts | passed | PR head: go build ./... and go test ./... passed |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 构建 | `go build ./...` | 通过 | 在 PR head 临时 clone 中执行；集成 merge commit 未验证。 |
| 测试 | `go test ./...` | 通过 | 在 PR head 临时 clone 中执行。 |

#### 建议 review 文案

PR head 上 `go build ./...` 和 `go test ./...` 均通过。建议维护者聚焦新增命令/API 参数、i18n、文档和测试覆盖做最终人工确认。

<!-- gitlink-pr-assessor:report v1 -->
### PR #77 维护者评估报告

**标题：** feat(journal): add issue and PR comment shortcuts
**作者：** wangyue111
**结论：** 建议作者先 rebase/解决冲突后再审
**风险等级：** 高
**优先级：** P2
**证据强度：** 中
**执行验证：** blocked；集成状态：blocked

#### 核心判断
1. 当前不可合并，维护者优先要求作者 rebase 到最新 master。
1. 解决冲突后再重新运行构建/测试和声明验证。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：12 文件，+1191 / -37，1 commits；Go 文件 6，测试文件 2，文档/skill 文件 6。
冲突文件：skills/README.md, skills/gitlink-issue/SKILL.md
样例文件：README.md, README.zh-CN.md, doc/changes/journal-shortcuts.md, shortcuts/issue/issue.go, shortcuts/issue/issue_test.go, shortcuts/issue/journal.go

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 弱 |
| 代码质量 | 证据不足 |
| 安全性 | 低-中 |
| 维护成本 | 高 |
| 协作质量 | 中 |
| 回归风险 | 高 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(journal): add issue and PR comment shortcuts | blocked | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 集成 | mergeability/conflict 检查 | 阻塞 | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 建议 review 文案

当前 PR 与 master 不可直接集成，建议先 rebase/解决冲突。冲突解决后请重新跑 `go build ./...` 和 `go test ./...`，再进入功能评审。

<!-- gitlink-pr-assessor:report v1 -->
### PR #179 维护者评估报告

**标题：** feat(dataset): add research dataset shortcuts
**作者：** wangyue111
**结论：** 建议作者先 rebase/解决冲突后再审
**风险等级：** 中
**优先级：** P2
**证据强度：** 中
**执行验证：** blocked；集成状态：blocked

#### 核心判断
1. 当前不可合并，维护者优先要求作者 rebase 到最新 master。
1. 解决冲突后再重新运行构建/测试和声明验证。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：9 文件，+305 / -1，1 commits；Go 文件 4，测试文件 2，文档/skill 文件 5。
冲突文件：README.md, README.zh-CN.md, doc/changes/dataset-shortcuts.md, shortcuts/dataset/dataset.go, shortcuts/dataset/dataset_test.go, shortcuts/register.go, shortcuts/register_test.go
样例文件：README.md, README.zh-CN.md, doc/changes/dataset-shortcuts.md, shortcuts/dataset/dataset.go, shortcuts/dataset/dataset_test.go, shortcuts/register.go

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 弱 |
| 代码质量 | 证据不足 |
| 安全性 | 低-中 |
| 维护成本 | 高 |
| 协作质量 | 中 |
| 回归风险 | 高 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(dataset): add research dataset shortcuts | blocked | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 集成 | mergeability/conflict 检查 | 阻塞 | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 建议 review 文案

当前 PR 与 master 不可直接集成，建议先 rebase/解决冲突。冲突解决后请重新跑 `go build ./...` 和 `go test ./...`，再进入功能评审。

<!-- gitlink-pr-assessor:report v1 -->
### PR #178 维护者评估报告

**标题：** feat(contents): add repository content shortcuts
**作者：** wangyue111
**结论：** 建议作者先 rebase/解决冲突后再审
**风险等级：** 中
**优先级：** P2
**证据强度：** 中
**执行验证：** blocked；集成状态：blocked

#### 核心判断
1. 当前不可合并，维护者优先要求作者 rebase 到最新 master。
1. 解决冲突后再重新运行构建/测试和声明验证。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：9 文件，+466 / -52，1 commits；Go 文件 4，测试文件 2，文档/skill 文件 5。
冲突文件：README.md, README.zh-CN.md, shortcuts/register.go, shortcuts/register_test.go, skills/README.md
样例文件：README.md, README.zh-CN.md, doc/changes/contents-shortcuts.md, shortcuts/contents/contents.go, shortcuts/contents/contents_test.go, shortcuts/register.go

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 弱 |
| 代码质量 | 证据不足 |
| 安全性 | 低-中 |
| 维护成本 | 高 |
| 协作质量 | 中 |
| 回归风险 | 高 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(contents): add repository content shortcuts | blocked | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 集成 | mergeability/conflict 检查 | 阻塞 | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 建议 review 文案

当前 PR 与 master 不可直接集成，建议先 rebase/解决冲突。冲突解决后请重新跑 `go build ./...` 和 `go test ./...`，再进入功能评审。

<!-- gitlink-pr-assessor:report v1 -->
### PR #177 维护者评估报告

**标题：** feat(wiki): add wiki management shortcuts
**作者：** wangyue111
**结论：** 建议作者先 rebase/解决冲突后再审
**风险等级：** 中
**优先级：** P2
**证据强度：** 中
**执行验证：** blocked；集成状态：blocked

#### 核心判断
1. 当前不可合并，维护者优先要求作者 rebase 到最新 master。
1. 解决冲突后再重新运行构建/测试和声明验证。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：9 文件，+425 / -1，1 commits；Go 文件 4，测试文件 2，文档/skill 文件 5。
冲突文件：shortcuts/wiki/wiki.go, shortcuts/wiki/wiki_test.go, skills/gitlink-wiki/SKILL.md
样例文件：README.md, README.zh-CN.md, doc/changes/wiki-shortcuts.md, shortcuts/register.go, shortcuts/register_test.go, shortcuts/wiki/wiki.go

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 弱 |
| 代码质量 | 证据不足 |
| 安全性 | 低-中 |
| 维护成本 | 高 |
| 协作质量 | 中 |
| 回归风险 | 高 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(wiki): add wiki management shortcuts | blocked | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 集成 | mergeability/conflict 检查 | 阻塞 | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 建议 review 文案

当前 PR 与 master 不可直接集成，建议先 rebase/解决冲突。冲突解决后请重新跑 `go build ./...` 和 `go test ./...`，再进入功能评审。

<!-- gitlink-pr-assessor:report v1 -->
### PR #176 维护者评估报告

**标题：** feat(user): add dashboard shortcuts
**作者：** wangyue111
**结论：** 建议作者先 rebase/解决冲突后再审
**风险等级：** 中
**优先级：** P2
**证据强度：** 中
**执行验证：** blocked；集成状态：blocked

#### 核心判断
1. 当前不可合并，维护者优先要求作者 rebase 到最新 master。
1. 解决冲突后再重新运行构建/测试和声明验证。
1. 本地无法形成真实 merge commit；报告中的执行验证基于 PR head。

变更规模：9 文件，+356 / -52，1 commits；Go 文件 4，测试文件 2，文档/skill 文件 5。
冲突文件：shortcuts/register.go, shortcuts/register_test.go
样例文件：README.md, README.zh-CN.md, doc/changes/user-dashboard-shortcuts.md, shortcuts/register.go, shortcuts/register_test.go, shortcuts/userdashboard/userdashboard.go

#### 维度评估

| 维度 | 结论 |
|------|------|
| 贡献价值 | 中 |
| 实现可行性 | 弱 |
| 代码质量 | 证据不足 |
| 安全性 | 低-中 |
| 维护成本 | 高 |
| 协作质量 | 中 |
| 回归风险 | 高 |

#### 声明验证

| 声明 | 结果 | 证据 |
|------|------|------|
| 新增/补齐能力：feat(user): add dashboard shortcuts | blocked | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 执行验证记录

| 类型 | 命令/动作 | 结果 | 备注 |
|------|-----------|------|------|
| 集成 | mergeability/conflict 检查 | 阻塞 | mergeable=false or conflicts present; skipped execution because integration is blocked |

#### 建议 review 文案

当前 PR 与 master 不可直接集成，建议先 rebase/解决冲突。冲突解决后请重新跑 `go build ./...` 和 `go test ./...`，再进入功能评审。
