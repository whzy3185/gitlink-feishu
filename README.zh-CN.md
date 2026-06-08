# gitlink-cli

[![GitLink](https://img.shields.io/badge/GitLink-Gitlink%2Fgitlink--cli-green)](https://www.gitlink.org.cn/Gitlink/gitlink-cli)
[![License](https://img.shields.io/badge/License-MulanPSL--2.0-blue.svg)](https://license.coscl.org.cn/MulanPSL2)
[![Go Version](https://img.shields.io/badge/Go-1.26%2B-blue.svg)](https://golang.org)
[![npm version](https://img.shields.io/npm/v/@gitlink-ai/cli.svg)](https://www.npmjs.com/package/@gitlink-ai/cli)

[GitLink（确实开源）](https://www.gitlink.org.cn) 官方 CLI 工具 — 为人类和 AI Agent 双重设计。支持 **macOS、Linux、Windows**，覆盖仓库管理、Issue 追踪、Pull Request、Webhook、成员协作、CI/CD 和 AI 自动化工作流，包含 40+ 命令和 AI Agent [Skills](./skills/)。

**[English](./README.md)**

[安装](#安装与快速上手) · [AI Agent Skills](#ai-agent-skills) · [认证](#配置与使用) · [命令](#使用示例) · [贡献](#相关项目)

## 贡献者

<div style="display: flex; gap: 16px; flex-wrap: wrap; align-items: flex-start;">
<div align="center">
  <a href="https://www.gitlink.org.cn/wangyue111" title="wangyue111"><img src="https://www.gitlink.org.cn/system/lets/letter_avatars/2/W/43_254_70/120.png" width="40" height="40" alt="wangyue111" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/wangyue111">wangyue111</a></sub>
</div>
<div align="center">
  <a href="https://www.gitlink.org.cn/wbtiger" title="tigerwang"><img src="https://www.gitlink.org.cn/system/lets/letter_avatars/2/T/14_168_39/120.png" width="40" height="40" alt="wbtiger" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/wbtiger">wbtiger</a></sub>
</div>
<div align="center">
  <a href="https://www.gitlink.org.cn/Mengz" title="Mengz"><img src="https://www.gitlink.org.cn/system/lets/letter_avatars/2/M/166_152_185/120.png" width="40" height="40" alt="Mengz" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/Mengz">Mengz</a></sub>
</div>
<div align="center">
  <a href="https://www.gitlink.org.cn/yangsai" title="杨赛"><img src="https://www.gitlink.org.cn/system/lets/letter_avatars/2/Y/94_150_149/120.png" width="40" height="40" alt="yangsai" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/yangsai">yangsai</a></sub>
</div>
<div align="center">
  <a href="https://www.gitlink.org.cn/mengcheng" title="camelliamc"><img src="https://www.gitlink.org.cn/system/lets/letter_avatars/2/M/206_114_54/120.png" width="40" height="40" alt="mengcheng" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/mengcheng">mengcheng</a></sub>
</div>
<div align="center">
  <a href="https://www.gitlink.org.cn/muel" title="赵奕程"><img src="https://www.gitlink.org.cn/images/avatars/User/149182?t=1779603476" width="40" height="40" alt="muel" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/muel">muel</a></sub>
</div>
<div align="center">
  <a href="https://www.gitlink.org.cn/Leo77" title="Leo77"><img src="https://www.gitlink.org.cn/system/lets/letter_avatars/2/L/173_120_149/120.png" width="40" height="40" alt="Leo77" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/Leo77">Leo77</a></sub>
</div>
<div align="center">
  <a href="https://www.gitlink.org.cn/yingjie" title="yingjie"><img src="https://www.gitlink.org.cn/images/avatars/User/145288?t=1765791899" width="40" height="40" alt="yingjie" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/yingjie">yingjie</a></sub>
</div>
<div align="center">
  <a href="https://www.gitlink.org.cn/topshare" title="Kevin Zhang"><img src="https://www.gitlink.org.cn/system/lets/letter_avatars/2/K/65_152_142/120.png" width="40" height="40" alt="topshare" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/topshare">topshare</a></sub>
</div>
<div align="center">
  <a href="https://www.gitlink.org.cn/dtwdtw" title="dtwdtw"><img src="https://www.gitlink.org.cn/system/lets/letter_avatars/2/D/53_166_51/120.png" width="40" height="40" alt="dtwdtw" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/dtwdtw">dtwdtw</a></sub>
</div>
<div align="center">
  <a href="https://www.gitlink.org.cn/recorder" title="recorder"><img src="https://www.gitlink.org.cn/system/lets/letter_avatars/2/R/141_201_87/120.png" width="40" height="40" alt="recorder" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/recorder">recorder</a></sub>
</div>
<div align="center">
  <a href="https://www.gitlink.org.cn/puygob236" title="Jiachen Li"><img src="https://www.gitlink.org.cn/images/avatars/User/149183?t=1778815174" width="40" height="40" alt="puygob236" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/puygob236">puygob236</a></sub>
</div>
<div align="center">
  <a href="https://www.gitlink.org.cn/co63oc" title="co63oc"><img src="https://www.gitlink.org.cn/system/lets/letter_avatars/2/C/205_201_141/120.png" width="40" height="40" alt="co63oc" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/co63oc">co63oc</a></sub>
</div>
<div align="center">
  <a href="https://www.gitlink.org.cn/lindiwen23" title="lindiwen23"><img src="https://www.gitlink.org.cn/images/avatars/User/141609?t=1748270628" width="40" height="40" alt="lindiwen23" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/lindiwen23">lindiwen23</a></sub>
</div>
</div>

## 为什么选择 gitlink-cli？

- **Agent-Native 设计** — 开箱即用结构化 [Skills](./skills/)，兼容 Claude Code — Agent 零配置即可操作 GitLink
- **广泛覆盖** — 仓库、Issue、PR、Webhook、成员、分支、Release、CI、Pipeline、组织、搜索、用户等常用工作流均提供高层命令
- **AI 友好 & 优化** — 每条命令都经过真实 Agent 测试，简洁参数、智能默认值、结构化输出
- **跨平台** — macOS、Linux、Windows (x64/arm64) 全支持，`npm` 一条命令安装
- **开源零门槛** — 木兰宽松许可证第2版（MulanPSL-2.0），`npm install` 即用
- **3 分钟上手** — 交互式登录或 `GITLINK_TOKEN` 环境变量，从安装到首次 API 调用仅需 3 步
- **安全可控** — OS 原生 keychain 凭证存储，`GITLINK_TOKEN` 环境变量支持 CI/CD 和非交互环境，自动 git remote 上下文解析
- **三层架构** — Shortcuts（人+AI友好）→ Raw API（全覆盖）→ Config（配置管理）

## 功能一览

| 分类 | 能力 |
|------|------|
| 📦 仓库 | 列出、创建、Fork、删除仓库，查看仓库信息、洞察数据和互动状态 |
| 🐛 Issue | 创建、更新、关闭、批量关闭、评论 Issue |
| 🔖 标签 | 创建、列出、更新、删除 Issue 标签 |
| 🔀 PR | 创建、合并、Review Pull Request，查看变更文件 |
| 👥 成员 | 列出、添加、移除仓库成员，调整角色，生成和接受邀请链接 |
| 🌿 分支 | 创建、删除、保护分支 |
| 🏷️ 发布 | 创建、编辑、更新、查看、删除 Release |
| 🏢 组织 | 管理组织、成员、团队 |
| 🔧 CI | 查看构建、日志、CI/CD 操作 |
| ⚙️ Pipeline | 运行、查看、启停、删除流水线工作流并查询日志 |
| 🔍 搜索 | 搜索仓库、用户 |
| 👤 用户 | 查看用户资料和信息 |
| 📋 项目管理 | Sprint 管理、看板、周报 |
| 🤖 工作流 | AI 驱动的 Issue 分类、PR Review、Release Notes |

## 安装与快速上手

### 前置条件

- Node.js 14+（`npm`/`npx`）— 用于 npm 安装
- 支持平台：macOS、Linux、Windows（x64/arm64）
- Go 1.26+ — 仅从源码构建时需要

### 快速上手（人类用户）

> **AI 助手请注意：** 如果你是帮助用户安装的 AI Agent，请直接跳到 [快速上手（AI Agent）](#快速上手ai-agent)，其中包含你需要完成的所有步骤。

#### 安装

选择以下**任一**方式：

**方式 1 — 从 npm 安装（推荐）：**

```bash
# 安装 CLI
npm install -g @gitlink-ai/cli

# 安装 CLI Skill（必须，全平台通用）
gitlink-cli-install-skills

# 也可使用 npx 安装 Skill
npx skills add ccfos/gitlink-cli/skills -y -g
```

**方式 2 — 从源码构建：**

需要 Go 1.26+。

```bash
git clone https://www.gitlink.org.cn/Gitlink/gitlink-cli.git
cd gitlink-cli
make install

# 安装 CLI Skill（必须）
npx skills add ./skills -y -g
```

> **Windows 用户注意：** 请在 PowerShell 或 CMD 中运行 `npm install -g @gitlink-ai/cli`。从源码构建请使用 `go install .` 代替 `make install`。

#### 配置与使用

```bash
# 1. 配置（首次使用，交互式引导）
gitlink-cli config init

# 2. 登录（任选其一）
gitlink-cli auth login            # 用户名密码（推荐）
gitlink-cli auth login --token    # 或粘贴私人令牌
export GITLINK_TOKEN="your-token" # 或设置环境变量（适用于 CI/CD、非交互环境）

# 3. 开始使用
gitlink-cli repo +list
```

### 快速上手（AI Agent）

> 以下步骤面向 AI Agent。部分步骤需要用户在浏览器中完成操作。

**第 1 步 — 安装**

```bash
# 安装 CLI
npm install -g @gitlink-ai/cli

# 安装 CLI Skill（必须，全平台通用）
gitlink-cli-install-skills
```

**第 2 步 — 配置**

```bash
gitlink-cli config init
```

**第 3 步 — 登录**

交互环境：
```bash
gitlink-cli auth login
```

非交互环境（CI/CD、Trae 沙箱、MCP 等）：
```bash
export GITLINK_TOKEN="your-private-token"
```

> 获取私人令牌：GitLink 网页端 → 个人设置 → 私人令牌。

**第 4 步 — 验证**

```bash
gitlink-cli user +me
```

## 使用示例

### 仓库操作

```bash
# 列出仓库
gitlink-cli repo +list

# 查看仓库信息
gitlink-cli repo +info --owner Gitlink --repo forgeplus

# 读取仓库 README
gitlink-cli repo +readme --owner Gitlink --repo forgeplus --ref master

# 查看语言占比
gitlink-cli repo +languages --owner Gitlink --repo forgeplus

# 列出贡献者
gitlink-cli repo +contributors --owner Gitlink --repo forgeplus

# 查看分支、标签或提交的贡献者代码行统计
gitlink-cli repo +contributor-stats --owner Gitlink --repo forgeplus --ref master --pass-year 1

# 查看仓库代码统计
gitlink-cli repo +code-stats --owner Gitlink --repo forgeplus --ref master

# 按时间范围查看关注者和点赞者
gitlink-cli repo +watchers --owner Gitlink --repo forgeplus --start-at 1714521600 --end-at 1717200000
gitlink-cli repo +stargazers --owner Gitlink --repo forgeplus --start-at 1714521600 --end-at 1717200000

# 预览并执行仓库互动操作
gitlink-cli repo +follow --owner Gitlink --repo forgeplus --dry-run
gitlink-cli repo +follow --owner Gitlink --repo forgeplus
gitlink-cli repo +unfollow --owner Gitlink --repo forgeplus --project-id 123
gitlink-cli repo +like --owner Gitlink --repo forgeplus
gitlink-cli repo +unlike --owner Gitlink --repo forgeplus --project-id 123

# 创建仓库
gitlink-cli repo +create -n my-project -d "项目描述"

# Fork 仓库
gitlink-cli repo +fork --owner Gitlink --repo forgeplus
```

### Webhook 管理

```bash
# 列出 webhook
gitlink-cli webhook +list --owner Gitlink --repo forgeplus

# 创建 webhook
gitlink-cli webhook +create --owner Gitlink --repo forgeplus \
  --url https://example.com/hook --events push,create

# 测试 webhook
gitlink-cli webhook +test --owner Gitlink --repo forgeplus --id 68

# 查看 webhook 投递任务
gitlink-cli webhook +tasks --owner Gitlink --repo forgeplus --id 68
```

### 成员管理

```bash
# 列出仓库成员
gitlink-cli member +list --owner Gitlink --repo forgeplus

# 添加成员
gitlink-cli member +add --owner Gitlink --repo forgeplus --user-id 101

# 预览批量添加成员，不修改数据
gitlink-cli member +batch-add --owner Gitlink --repo forgeplus --user-ids 101,102 --dry-run

# 从 CSV 文件批量添加成员
gitlink-cli member +batch-add --owner Gitlink --repo forgeplus --from members.csv

# 调整成员权限
gitlink-cli member +role --owner Gitlink --repo forgeplus --user-id 101 --role Developer

# 生成邀请链接
gitlink-cli member +invite-link --owner Gitlink --repo forgeplus --role developer --apply true
```

### Issue 管理

```bash
# 列出 Issue
gitlink-cli issue +list --owner Gitlink --repo forgeplus

# 创建 Issue
gitlink-cli issue +create --owner Gitlink --repo forgeplus -t "Bug: 登录失败" -b "复现步骤..."

# 创建带元数据的 Issue
gitlink-cli issue +create --owner Gitlink --repo forgeplus -t "Bug: 登录失败" --priority-id 3 --tag-ids 4,5 --assigner-ids 7

# 查看 Issue
gitlink-cli issue +view --owner Gitlink --repo forgeplus -i 123

# 更新 Issue 元数据
gitlink-cli issue +update --owner Gitlink --repo forgeplus --number 123 --priority-id 4 --branch bugfix/login --due-date 2026-06-15

# 关闭 Issue
gitlink-cli issue +close --owner Gitlink --repo forgeplus -i 123

# 预览批量关闭，不修改数据
gitlink-cli issue +batch-close --owner Gitlink --repo forgeplus --numbers 123,124 --dry-run

# 从 CSV 文件批量关闭 Issue
gitlink-cli issue +batch-close --owner Gitlink --repo forgeplus --from issues.csv

# 添加评论
gitlink-cli issue +comment --owner Gitlink --repo forgeplus -i 123 -b "已修复"

# 列出 Issue 负责人
gitlink-cli issue +assigners --owner Gitlink --repo forgeplus

# 列出 Issue 发布人
gitlink-cli issue +authors --owner Gitlink --repo forgeplus

# 列出 Issue 优先级
gitlink-cli issue +priorities --owner Gitlink --repo forgeplus

# 列出 Issue 标签
gitlink-cli issue +tags --owner Gitlink --repo forgeplus --only-name

# 列出 Issue 状态
gitlink-cli issue +statuses --owner Gitlink --repo forgeplus
```

`issue +view`、`issue +update`、`issue +close` 和 `issue +comment` 推荐使用
`--number` / `-n` 传网页 URL 中的 Issue 编号。`--id` / `-i` 是同一网页 Issue
编号的兼容别名，不是数据库内部 ID。

### 标签管理

```bash
# 列出 Issue 标签
gitlink-cli label +list --owner Gitlink --repo forgeplus

# 按关键词筛选标签
gitlink-cli label +list --owner Gitlink --repo forgeplus -k bug

# 创建标签（颜色默认 #1E90FF）
gitlink-cli label +create --owner Gitlink --repo forgeplus -n bug -d "功能缺陷" -c "#FF0000"

# 更新标签（未指定的字段会被保留）
gitlink-cli label +update --owner Gitlink --repo forgeplus -i 42 -c "#00FF00"

# 删除标签
gitlink-cli label +delete --owner Gitlink --repo forgeplus -i 42
```

### Pull Request

```bash
# 列出 PR
gitlink-cli pr +list --owner Gitlink --repo forgeplus

# 创建 PR（同仓库分支）
gitlink-cli pr +create --owner Gitlink --repo forgeplus -t "feat: 搜索功能" --head feature/search --base master

# 创建 PR（从 Fork 仓库）
gitlink-cli pr +create --owner Gitlink --repo forgeplus -t "feat: 新功能" --head your_username/forgeplus:feature/my-feature --base master

# 查看 PR
gitlink-cli pr +view --owner Gitlink --repo forgeplus -i 42

# 合并 PR
gitlink-cli pr +merge --owner Gitlink --repo forgeplus -i 42

# 重开已关闭的 PR
gitlink-cli pr +reopen --owner Gitlink --repo forgeplus -i 42

# 查看 PR 变更文件
gitlink-cli pr +files --owner Gitlink --repo forgeplus -i 42

# 查看 PR patchset/version 列表
gitlink-cli pr +versions --owner Gitlink --repo forgeplus -i 42

# 查看指定 patchset/version diff
gitlink-cli pr +version-diff --owner Gitlink --repo forgeplus -i 42 --version-id 16040

# 查看 PR 审查记录
gitlink-cli pr +reviews --owner Gitlink --repo forgeplus -i 42

# 创建 PR 审查（支持 dry-run 预览）
gitlink-cli pr +review --owner Gitlink --repo forgeplus -i 42 --status approved -c "LGTM" --dry-run
gitlink-cli pr +review --owner Gitlink --repo forgeplus -i 42 --status approved -c "LGTM"
```

### 发布管理

```bash
# 列出 Release
gitlink-cli release +list --owner Gitlink --repo forgeplus

# 创建 Release，可附带附件 ID
gitlink-cli release +create --owner Gitlink --repo forgeplus -t v1.0.0 -n "v1.0.0 正式版" -b "更新内容..." --attachment-ids 12,34

# 查看 Release
gitlink-cli release +view --owner Gitlink --repo forgeplus -i <version_id>

# 获取编辑数据并保留未传字段更新
gitlink-cli release +edit --owner Gitlink --repo forgeplus -i <version_id>
gitlink-cli release +update --owner Gitlink --repo forgeplus -i <version_id> -b "更新后的内容" --dry-run

# 删除前先预览请求
gitlink-cli release +delete --owner Gitlink --repo forgeplus -i <version_id> --dry-run
```

### 流水线管理

```bash
# 列出平台流水线
gitlink-cli pipeline +list --owner-id 123 --page 1 --limit 20

# 列出仓库流水线运行记录
gitlink-cli pipeline +runs --owner Gitlink --repo forgeplus --ref master --workflow build.yml

# 运行流水线工作流，先用 dry-run 预览请求
gitlink-cli pipeline +run --owner Gitlink --repo forgeplus --ref master --workflow build.yml --dry-run

# 查看流水线详情、日志和运行结果
gitlink-cli pipeline +view --owner Gitlink --repo forgeplus --id 7
gitlink-cli pipeline +logs --owner Gitlink --repo forgeplus --run-id 99 --id 7 --index 43
gitlink-cli pipeline +results --owner Gitlink --repo forgeplus --run-id 99

# 启停或删除流水线工作流，写入/删除前先预览
gitlink-cli pipeline +disable --owner Gitlink --repo forgeplus --id 7 --workflow build.yml --dry-run
gitlink-cli pipeline +delete --owner Gitlink --repo forgeplus --id 7 --dry-run
```

### 搜索

```bash
# 搜索仓库
gitlink-cli search +repos -k "machine learning"

# 搜索用户
gitlink-cli search +users -k "zhangsan"
```

### Raw API

Shortcuts 未覆盖的接口可通过 Raw API 直接调用：

```bash
# GET 请求
gitlink-cli api GET /users/me

# POST 请求
gitlink-cli api POST /Gitlink/forgeplus/issues --body '{"subject":"test","description":"..."}'

# 从文件读取 JSON body
gitlink-cli api POST /Gitlink/forgeplus/issues --body-file issue.json

# 从 stdin 读取 JSON body
Get-Content issue.json | gitlink-cli api POST /Gitlink/forgeplus/issues --body-stdin

# 带查询参数
gitlink-cli api GET /Gitlink/forgeplus/commits --query 'page=1&limit=5'
```

## 全局参数

| 参数 | 说明 | 示例 |
|------|------|------|
| `--owner` | 仓库所有者 | `--owner Gitlink` |
| `--repo` | 仓库名称 | `--repo forgeplus` |
| `--format` | 输出格式（json/table/yaml） | `--format json` |
| `--debug` | 启用调试输出 | `--debug` |

**自动上下文解析**：在 git 仓库目录下，`--owner` 和 `--repo` 会自动从 `git remote origin` 解析。

## 分支约定

gitlink-cli 支持 GitHub 和 GitLink 的代码双向同步：

| 平台 | 主分支 |
|------|--------|
| GitHub | `main` |
| GitLink | `master` |

**本地 push 到 GitLink**：

```bash
# 方式 1：使用 git 命令
git push gitlink main:master

# 方式 2：配置 git remote
git config remote.gitlink.push refs/heads/main:refs/heads/master
git push gitlink
```

## AI Agent Skills

`skills/` 目录包含 Claude Code Agent Skill 文件，支持 AI 自动化操作 GitLink 平台。

详见 [skills/README.md](skills/README.md)

| Skill | 说明 |
|-------|------|
| `gitlink-shared` | 认证、全局参数、安全规则、API 注意事项 |
| `gitlink-repo` | 仓库操作（创建、查看、删除、Fork、洞察数据等） |
| `gitlink-issue` | Issue 操作（创建、更新、关闭、评论等） |
| `gitlink-pr` | Pull Request 操作（创建、合并、Review 等） |
| `gitlink-member` | 仓库成员与邀请链接管理 |
| `gitlink-release` | 发布管理（创建、编辑、更新、查看、删除等） |
| `gitlink-org` | 组织管理（成员、团队等） |
| `gitlink-ci` | CI/CD 操作（构建、日志等） |
| `gitlink-pipeline` | 流水线工作流操作（运行、日志、启停、删除等） |
| `gitlink-search` | 搜索功能（仓库、用户等） |
| `gitlink-user` | 用户管理（个人信息等） |
| `gitlink-pm` | 项目管理（Sprint、看板、周报等） |
| `gitlink-workflow` | AI 自动化工作流（Issue 分类、PR Review、Release Notes 等） |

## 项目结构

```
gitlink-cli/
├── cmd/                      # Cobra 命令定义
│   ├── root.go               # 根命令 + 全局 flags
│   ├── auth/                 # 认证命令
│   ├── api/                  # Raw API 命令
│   ├── config/               # 配置命令
│   └── cmdutil/              # 全局工具
├── internal/                 # 内部包
│   ├── auth/                 # 登录、Token 存储、Transport
│   ├── client/               # HTTP 客户端 + 分页
│   ├── config/               # 配置文件管理
│   ├── context/              # git remote 解析
│   └── output/               # Envelope + Formatter
├── shortcuts/                # Shortcut 实现
│   ├── common/               # 框架（types, runner）
│   ├── repo/                 # 仓库 shortcuts
│   ├── issue/                # Issue shortcuts
│   ├── pr/                   # PR shortcuts
│   ├── member/               # 仓库成员 shortcuts
│   ├── branch/               # 分支 shortcuts
│   ├── release/              # Release shortcuts
│   ├── org/                  # 组织 shortcuts
│   ├── ci/                   # CI shortcuts
│   ├── pipeline/             # Pipeline shortcuts
│   ├── search/               # 搜索 shortcuts
│   ├── user/                 # 用户 shortcuts
│   └── register.go           # 注册入口
├── skills/                   # AI Agent Skills
│   ├── README.md             # Skills 使用指南
│   ├── gitlink-shared/       # 共享规则
│   ├── gitlink-repo/         # 仓库 Skill
│   ├── gitlink-issue/        # Issue Skill
│   ├── gitlink-pr/           # PR Skill
│   ├── gitlink-pm/           # 项目管理 Skill
│   └── ...
├── doc/                      # 设计文档
│   ├── Design.md
│   ├── CODE_SYNC_STRATEGY_FINAL.md
│   └── ...
├── main.go
├── Makefile
├── go.mod
└── README.md
```

## 文档

- [Skills 使用指南](skills/README.md) — AI Agent Skills 详细说明
- [设计文档](doc/design.md) — 架构设计和开发计划

## 常见问题

### Q: 如何在脚本中使用 gitlink-cli？

使用 `GITLINK_TOKEN` 环境变量 + `--format json` 获取结构化输出：

```bash
export GITLINK_TOKEN="your-private-token"
gitlink-cli repo +list --format json | jq '.data.projects[] | .name'
```

### Q: 如何自动解析 owner/repo？

在 git 仓库目录下运行命令，CLI 会自动从 `git remote origin` 解析：

```bash
cd ~/my-gitlink-project
gitlink-cli issue +list  # 自动使用当前仓库
```

### Q: Token 过期了怎么办？

重新登录：

```bash
# 用户名密码登录
gitlink-cli auth login

# 或使用私人令牌（在 GitLink 网页端 个人设置 → 私人令牌 中生成）
gitlink-cli auth login --token
```

### Q: 如何在 CI/CD 或非交互环境（Trae 沙箱等）中使用？

设置 `GITLINK_TOKEN` 环境变量即可，无需 `auth login`：

```bash
export GITLINK_TOKEN="your-private-token"
gitlink-cli repo +list   # 直接可用
gitlink-cli auth status   # 显示 "✓ Logged in via GITLINK_TOKEN environment variable"
```

Token 优先级：`GITLINK_TOKEN` 环境变量 > keyring/文件存储的 token。不设置环境变量时完全兼容原有交互式登录。

### Q: npm 安装成功但 `gitlink-cli` 提示缺少二进制怎么办？

先尝试重新安装：

```bash
npm install -g @gitlink-ai/cli
```

如果仍然失败，请检查 Release 页面是否包含当前平台的资产，例如 Windows x64 对应 `gitlink-cli_<version>_windows_amd64.zip`。也可以从 Release 页面手动下载二进制，或使用 `go install .` 从源码构建。

### Q: Windows 上凭证存储在哪里？

gitlink-cli 使用 Windows Credential Manager 安全存储 Token。如果 Credential Manager 不可用，会自动降级到文件存储（`~/.config/gitlink-cli/credentials`）。

### Q: 如何查看完整的 API 参考？

查看 [skills/gitlink-shared/REFERENCE.md](skills/gitlink-shared/REFERENCE.md)

## 许可证

[MulanPSL-2.0](https://license.coscl.org.cn/MulanPSL2)
