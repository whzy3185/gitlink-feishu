---
name: gitlink-dev-full-cycle
version: 1.0.0
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli --help"
description: "需求开发测试发布全流程。触发词：新需求开发、全流程推进、从需求到上线、启动需求、开发测试发布、feature全流程、需求落地、需求闭环、从0到1交付、从零开始开发、新建项目开发。6阶段：项目初始化→需求分析→技术设计→编码开发→测试验证→发布上线。集成gitlink-cli，分支三分支模型：master主分支/feature-xxx开发分支/release-xxx发布分支，每阶段输出标准文档。"
agent_created: true
---

# gitlink-dev-full-cycle — 需求开发测试发布全流程

## 触发词

新需求开发、全流程推进、从需求到上线、启动需求、开发测试发布、feature全流程、需求落地、需求闭环、从0到1交付、从零开始开发、新建项目开发

---

## ⚠ 核心规则：必须使用 gitlink-cli

本 Skill 与 GitLink 平台的所有交互**必须通过 `gitlink-cli` 命令完成**，禁止绕过 gitlink-cli 直接使用 GitLink Web 界面或 API。**禁止使用 `gitlink-cli api` 命令**（该命令用于发送原始 API 请求，容易误操作，所有操作应使用对应的专用子命令）。

### 🔴 执行 gitlink-cli 命令前，必须先查询 --help 确认正确格式！

gitlink-cli 的部分子命令使用 `+` 前缀（如 `+create`、`+merge`、`+run`），部分不需要（如 `auth status`、`auth login`）。**执行任何 gitlink-cli 命令前，必须先通过 `--help` 查询子命令列表，确认准确的命令格式后再执行。**

**查询方式：**
```bash
# 查看某模块下有哪些子命令（注意哪些带 + 哪些不带）
gitlink-cli repo --help
gitlink-cli branch --help
gitlink-cli pr --help

# 查看某子命令的详细参数
gitlink-cli repo +create --help
gitlink-cli pr +merge --help
```

**常见错误：AI agent 执行时丢掉 `+` 前缀，导致命令失败。**
```
❌ gitlink-cli repo create -n "项目名"     # 丢掉了 +，会失败
✅ gitlink-cli repo +create -n "项目名"    # 保留 +，正确

❌ gitlink-cli branch protect -n master    # 丢掉了 +
✅ gitlink-cli branch +protect -n master   # 保留 +

❌ gitlink-cli pr merge -i 123             # 丢掉了 +
✅ gitlink-cli pr +merge -i 123            # 保留 +
```

> **规则：不确定子命令格式时，先执行 `gitlink-cli <模块> --help` 查看可用子命令列表，照抄列表中的格式（含 `+` 或不含 `+`），禁止凭猜测省略或添加 `+`。**

**必须使用 gitlink-cli 的操作：**
- 创建代码仓库 → `gitlink-cli repo +create / +info`
- 创建/保护分支 → `gitlink-cli branch +create / +protect`
- 创建 PR / Review / 合并 → `gitlink-cli pr +create / +review / +merge`
- 触发流水线 / 查看构建 → `gitlink-cli pipeline +run / +runs`、`gitlink-cli ci +builds / +logs`
- 创建 Release → `gitlink-cli release +create`
- AI Code Review → `gitlink-cli workflow +pr-summary`
- 仓库健康检查 → `gitlink-cli workflow +health / +repo-report`

**本地 Git 操作使用原生 git：** `git add`、`git commit`、`git push`、`git pull`、`git checkout`、`git merge`、`git clone` 等。若 `git push` / `git pull` 需要认证，向用户询问账号密码或 Token，不使用 gitlink-cli auth。

---

## 分支策略

```
master ──────────────────────── 主分支（受保护，严禁直接提交或推送，存放最新稳定代码）
  │                                                       ↑
  ├──→ feature-xxx ──→（PR）──→ release-xxx ──→ 发版后合并回 master
  │
  └──→ feature-yyy ──→（PR）──→ release-zzz ──→ ...
```

**3 种分支说明：**

| 分支 | 命名规范 | 从哪拉 | 说明 |
|------|----------|--------|------|
| `master` | 固定 | — | 主分支，存放最新发布的稳定代码，**严禁直接提交代码或修改内容** |
| `release-xxx` | `release-<版本号或日期>` | `master` | 发布分支，用于版本发布准备，发版后保留所有版本，不删除 |
| `feature-xxx` | `feature-<功能描述>` | `master` | 开发分支，团队日常开发的基准分支，存放最新开发进度代码 |

**分支流转规则（严格遵守）：**
1. 每次开发时从 `master` 拉取 `feature-xxx` 分支进行功能开发和缺陷修复
2. 开发自测完成后，提交 PR 将 `feature-xxx` 合并到 `release-xxx`（发布分支）
3. 发版后将 `release-xxx` 合并回 `master` 分支

**禁止的行为：**
- ❌ 直接向 `master` 提交代码或 `git push origin master`
- ❌ `feature-xxx` 直接向 `master` 发 PR（必须先合入 `release-xxx`，再由 `release-xxx` 合回 `master`）
- ❌ 不同 `feature-xxx` 分支之间相互合并

---

## 前置步骤：认证检查

在执行任何阶段之前，**必须先确认 gitlink-cli 认证状态**：

```bash
gitlink-cli auth status    # 检查认证状态
```

- 若已认证 → 继续执行阶段零
- 若未认证 → 执行 `gitlink-cli auth login` 登录，登录成功后再继续

---

## ⚠ 阶段零：项目初始化（必须第一个执行！）

> **本阶段是后续所有阶段的前提，必须第一个完成！未完成阶段零，禁止进入阶段一及后续任何阶段。** 只有当项目已有远程仓库且本地已克隆时，才可跳过本阶段（需向用户确认）。

1. 确认项目信息：仓库名、owner、公开/私有、技术栈

2. **先在 GitLink 远程创建仓库：**

```bash
gitlink-cli repo +create -n "项目名" -d "描述" --private "true"
gitlink-cli repo +info --owner "org" --repo "项目名"   # 确认远程仓库创建成功
```

3. **再克隆到本地并初始化项目骨架：**

```bash
git clone <仓库URL> && cd <项目名>
# 按技术栈初始化项目骨架（vite / fastapi / express / go mod init ...）
git add . && git commit -m "init" && git push origin master
```

4. **保护 master 分支（严禁直接推送）：**

```bash
gitlink-cli branch +protect --owner "org" --repo "项目名" -n master
gitlink-cli workflow +health --owner "org" --repo "项目名"
```

5. **输出文档：** 项目根目录下写入以下基础文档：
   - `README.md` — 项目简介、技术栈、本地开发启动步骤、目录结构说明
   - `.gitignore` — 根据技术栈生成标准忽略规则
   - `docs/` 目录 — 创建文档目录，后续各阶段文档统一存放于此

---

## 阶段一：需求分析

1. 向用户确认：需求背景、核心功能点、验收标准（AC）、优先级
2. 输出结构化需求：用户故事 + 功能清单 + AC（Given/When/Then）+ Out-of-scope
3. **输出文档 `docs/requirements.md`：** 按模板写入完整需求文档（含背景、目标用户、用户故事、功能清单、验收标准、Out-of-scope、依赖与风险）
4. 用户确认需求文档后进入下一阶段

---

## 阶段二：技术设计

1. 分析技术选型：技术栈、数据模型、接口契约、第三方集成
2. 识别风险与难点，给出解法
3. **输出文档 `docs/design.md`：** 按模板写入完整设计文档（含技术栈选型表、系统架构图、数据模型DDL、接口契约、关键实现思路、风险与缓解措施）
4. 用户确认设计文档后进入下一阶段

---

## 阶段三：编码开发

1. **从 master 创建 feature 开发分支：**

```bash
gitlink-cli branch +create -n "feature-<功能描述>" -f master
git fetch origin && git checkout feature-<功能描述>
```

2. 用 TaskCreate 拆分开发子任务，逐任务实现（先 Happy Path 再边界）
3. 遵守编码规范：函数职责单一、关键逻辑注释、无硬编码、类型注解
4. **输出文档：**
   - **代码注释** — 公共函数/类必须有 docstring 或 JSDoc，说明用途、参数、返回值
   - **`docs/api.md`**（如有 HTTP API）— 列出所有新增接口的路径、方法、参数、响应、错误码
   - **配置文件说明** — 新增配置项需在文档中注明用途和默认值
5. 完成后提交推送：

```bash
git add . && git commit -m "feat: XXX" && git push origin feature-<功能描述>
```

---

## 阶段四：测试验证

### 单元测试 + Lint
编写单元测试覆盖核心逻辑、边界、异常；运行 Lint 修复 Error。

### 创建发布分支（release-xxx）并集成 feature

```bash
# 若 release-xxx 尚未存在，从 master 创建
gitlink-cli branch +create -n "release-<版本号>" -f master
# 将 feature 分支合并到 release 分支
git checkout release-<版本号> && git merge feature-<功能描述> && git push origin release-<版本号>
```

### 触发流水线，验证集成结果

```bash
gitlink-cli pipeline +run -w <ci-workflow> -r release-<版本号>
gitlink-cli pipeline +runs -r release-<版本号>    # 查看运行状态
gitlink-cli ci +builds && gitlink-cli ci +logs     # 查看构建详情
```

冒烟测试：逐条验证 AC。

### Code Review（PR: feature-xxx → release-xxx）

```bash
gitlink-cli pr +create -t "feat: XXX" --head "feature-<功能描述>" --base "release-<版本号>"
gitlink-cli workflow +pr-summary -n <pr-number>    # AI Review
gitlink-cli pr +diff -i <pr-number>
gitlink-cli pr +review -i <pr-number> -s approved -c "LGTM"
gitlink-cli pr +merge -i <pr-number> -m merge
```

### **输出文档 `docs/test-report.md`：**
按模板写入测试报告（含测试范围、测试环境、AC逐条验证结果、单元测试覆盖率、遗留问题清单）

---

## ⚠ 阶段五：发布上线（必须严格按顺序执行，禁止跳步！）

> **🔴 发布上线阶段的所有步骤必须严格按下方顺序逐步执行，禁止跳过任何步骤、禁止调换顺序、禁止自行发明其他发布方式。**
>
> **禁止的行为：**
> - ❌ feature-xxx 直接向 master 发 PR（必须经过 release-xxx）
> - ❌ 未经用户确认就触发生产流水线或执行 release-xxx 合并回 master
> - ❌ 直接 `git push origin master`（master 受保护，只能通过 PR 合入）
> - ❌ 合并 PR 后不创建 Release
> - ❌ 用 `git merge` 直接操作 master 绕过 PR 流程

### 第 1 步：发布前确认（全部满足才能继续）

- [ ] 所有测试通过
- [ ] PR Review 通过（feature-xxx 已合并到 release-xxx）
- [ ] 配置已更新
- [ ] 回滚方案已确认
- [ ] **用户明确确认"可以发布"**

### 第 2 步：触发生产流水线，执行发布

```bash
gitlink-cli pipeline +run -w <prod-workflow> -r release-<版本号>    # ⚠ 需用户确认
gitlink-cli ci +builds && gitlink-cli ci +logs                       # 监控构建
```

生产冒烟验证 + 监控指标检查。

### 第 3 步：release-xxx 合并回 master（⚠ 不可逆，需用户确认）

```bash
gitlink-cli pr +create -t "release-<版本号> 合并回 master" --head "release-<版本号>" --base "master"
gitlink-cli pr +merge -i <pr-number> -m merge    # ⚠ 需用户确认
```

> **保留所有 release-xxx 分支，不删除。**

### 第 4 步：输出文档

- **`CHANGELOG.md`** — 在文件顶部追加本次版本条目（Added / Changed / Fixed / Breaking Changes）
- **`docs/release-notes.md`** — 按模板写入发布通知（版本号、上线时间、更新内容、验证方式、回滚方案）

### 第 5 步：创建 Release

```bash
gitlink-cli release +create -t "v1.x.0" -n "v1.x.0-需求名" --target master --prerelease false
```

> **所有 feature-xxx 和 release-xxx 分支均保留，不删除。**

---

## 文档清单总览

| 阶段 | 输出文档 | 存放位置 |
|------|----------|----------|
| 阶段零 | README.md、.gitignore | 项目根目录 |
| 阶段一 | requirements.md | `docs/` |
| 阶段二 | design.md | `docs/` |
| 阶段三 | api.md（如有API）、代码注释 | `docs/` + 源码 |
| 阶段四 | test-report.md | `docs/` |
| 阶段五 | CHANGELOG.md、release-notes.md | 项目根目录 + `docs/` |

> 所有文档模板见 `references/templates.md`，按模板格式写入，确保结构统一。

---

## 进度追踪

执行时用 TaskCreate 建立任务列表：

```
[ ] 阶段零：项目初始化 → README.md + .gitignore
[ ] 阶段一：需求分析 → docs/requirements.md
[ ] 阶段二：技术设计 → docs/design.md
[ ] 阶段三：编码开发 → feature-xxx 分支 + api.md + 代码注释
[ ] 阶段四：测试验证 → docs/test-report.md + PR Review（feature-xxx → release-xxx）
[ ] 阶段五：发布上线 → CHANGELOG.md + release-notes.md + release-xxx合并回master + Release
```

---

## 注意事项

- **阶段零必须第一个执行，未完成禁止进入后续阶段**（已有仓库需用户确认后方可跳过）
- 小需求可裁剪阶段一至五，但阶段零不可裁剪
- **所有平台操作必须使用 `gitlink-cli`，禁止绕过直接使用 Web 界面或 API**
- **禁止使用 `gitlink-cli api` 命令**（原始 API 请求易误操作，应使用对应的专用子命令）
- **执行 gitlink-cli 命令前，先通过 `--help` 查询确认子命令格式（部分子命令有 `+` 前缀，部分没有），禁止凭猜测省略或添加 `+`**
- **分支规则：feature-xxx 从 master 拉；开发完 PR 合到 release-xxx；发版后 release-xxx 合回 master；master 严禁直接提交**
- **发布上线阶段必须严格按照第1步→第5步顺序执行，禁止跳步、禁止调换顺序、禁止自行发明其他发布方式**
- 不可逆操作（release-xxx 合并回 master、触发生产流水线）执行前必须获得用户确认
- `git push` / `git pull` 需认证时向用户询问，不使用 gitlink-cli auth
- 每阶段输出文档后才算阶段完成，文档需用户确认后才能推进
- `--owner` / `--repo` 在 Git 仓库目录下可自动探测
- 项目已有规范时优先遵循，不覆盖

---

## 参考资源

- `references/checklist.md` — 各阶段检查清单
- `references/templates.md` — 需求文档、设计文档、测试报告、Changelog 等模板
