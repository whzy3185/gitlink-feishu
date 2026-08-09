# 各阶段检查清单

> **核心规则：所有 GitLink 平台操作必须使用 `gitlink-cli` 命令，禁止绕过直接使用 Web 界面或 API。**
> **禁止使用 `gitlink-cli api` 命令**（原始 API 请求易误操作，应使用对应的专用子命令）。
> **推送/拉取代码使用原生 `git` 命令，若需认证则向用户询问账号密码或 Token。**
> **🔴 执行 gitlink-cli 命令前，必须先通过 `--help` 查询确认子命令格式！部分子命令有 `+` 前缀（如 `repo +create`），部分没有（如 `auth status`）。不确定时先查 `gitlink-cli <模块> --help`，照抄列表中的格式，禁止凭猜测省略或添加 `+`。**
>
> **分支规则：master 严禁直接提交；feature-xxx 从 master 拉，开发完 PR 合到 release-xxx；发版后 release-xxx PR 合回 master。**

## 前置步骤：认证检查

- [ ] `gitlink-cli auth status` 认证状态已确认
- [ ] 若未认证 → `gitlink-cli auth login` 已完成登录

## ⚠ 阶段零：项目初始化（必须第一个执行！）

> **未完成阶段零，禁止进入后续任何阶段。** 只有项目已有远程仓库且本地已克隆时，经用户确认后方可跳过。

- [ ] 项目信息已确认（仓库名、owner、公开/私有、技术栈）
- [ ] `gitlink-cli repo +create` 仓库已在 GitLink 远程创建
- [ ] `gitlink-cli repo +info` 已确认仓库信息正确
- [ ] 本地已克隆并初始化项目骨架，`npm run dev`/`python main.py` 可运行
- [ ] 初始代码已推送 `git push origin master`
- [ ] `gitlink-cli branch +protect -n master` master 已保护（严禁直接推送）
- [ ] `gitlink-cli workflow +health` 健康评分已检查
- [ ] **`README.md`** 已写入（项目简介、技术栈、启动步骤、目录结构）
- [ ] **`.gitignore`** 已生成（匹配当前技术栈）
- [ ] **`docs/`** 目录已创建

---

## 阶段一：需求分析

- [ ] 需求背景、核心功能点、验收标准（AC）已确认
- [ ] 结构化需求已输出：用户故事 + 功能清单 + AC + Out-of-scope
- [ ] **`docs/requirements.md`** 已按模板写入（含背景、目标用户、用户故事、功能清单、验收标准、Out-of-scope、依赖与风险）
- [ ] 用户已确认需求文档

---

## 阶段二：技术设计

- [ ] 技术栈、数据模型、接口契约、部署方案已确定
- [ ] 技术风险已识别，有解法或备选
- [ ] **`docs/design.md`** 已按模板写入（含技术栈选型表、架构图、数据模型DDL、接口契约、关键实现思路、风险与缓解措施）
- [ ] 用户已确认设计文档

---

## 阶段三：编码开发

- [ ] `gitlink-cli branch +create -n "feature-<功能描述>" -f master` feature 开发分支已从 master 创建
- [ ] 本地已 checkout feature 分支
- [ ] 功能模块已拆分任务（TaskCreate），逐任务完成
- [ ] **代码注释** 已补全（公共函数/类有 docstring/JSDoc：用途、参数、返回值）
- [ ] **`docs/api.md`** 已写入（如有 HTTP API：路径、方法、参数、响应、错误码）
- [ ] 配置项新增已文档化
- [ ] 代码已提交推送至 feature 分支

---

## 阶段四：测试验证

- [ ] 单元测试编写并通过，Lint 无 Error
- [ ] release-xxx 分支已从 master 创建（`gitlink-cli branch +create -n "release-<版本号>" -f master`）
- [ ] feature 已合并到 release-xxx（`git merge feature-<功能描述>`）
- [ ] CI 流水线已触发（`gitlink-cli pipeline +run`）并运行成功
- [ ] 集成测试通过，AC 逐项验证通过
- [ ] `gitlink-cli pr +create --head "feature-<功能描述>" --base "release-<版本号>"` PR 已创建
- [ ] `gitlink-cli workflow +pr-summary -n <pr-number>` AI Review 已生成
- [ ] `gitlink-cli pr +review` Review 已提交（approved）
- [ ] `gitlink-cli pr +merge` PR 已合并（feature-xxx → release-xxx）
- [ ] 确认未出现 feature-xxx 直接向 master 发 PR（违反分支规则）
- [ ] **`docs/test-report.md`** 已按模板写入（含测试范围、环境、AC逐条结果、覆盖率、遗留问题）

---

## ⚠ 阶段五：发布上线（必须严格按顺序执行，禁止跳步！）

> 🔴 **必须按 第1步→第5步 顺序逐步执行，禁止跳过任何步骤、禁止调换顺序。**

### 第 1 步：发布前确认
- [ ] 所有测试通过 + PR Review 通过（feature 已合入 release-xxx）
- [ ] 配置已更新，回滚方案已确认
- [ ] **用户明确确认"可以发布"**

### 第 2 步：触发生产流水线，执行发布
- [ ] 生产流水线已触发（`gitlink-cli pipeline +run -w <prod-workflow> -r release-<版本号>`）⚠ 需用户确认
- [ ] 生产流水线运行成功，冒烟测试通过，监控指标正常

### 第 3 步：release-xxx 合并回 master（⚠ 不可逆，需用户确认）
- [ ] `gitlink-cli pr +create --head "release-<版本号>" --base "master"` PR 已创建
- [ ] **用户已确认** → `gitlink-cli pr +merge` PR 已合并
- [ ] release-xxx 分支已保留（不删除，保留最近一个版本）

### 第 4 步：输出文档
- [ ] **`CHANGELOG.md`** 已追加本次版本条目（Added/Changed/Fixed/Breaking Changes）
- [ ] **`docs/release-notes.md`** 已按模板写入（版本号、上线时间、更新内容、验证方式、回滚方案）

### 第 5 步：创建 Release 并清理
- [ ] `gitlink-cli release +create` Release 已创建（基于 master）
- [ ] feature 分支已删除（本地 + 远程）（**release 分支保留，不删除**）
- [ ] 相关方已通知

### ❌ 禁止的行为
- [ ] 确认未出现 feature-xxx 直接向 master 发 PR（必须经过 release-xxx）
- [ ] 确认未经用户确认就触发生产流水线或执行 release → master 合并
- [ ] 确认未直接 `git push origin master`
- [ ] 确认未用 `git merge` 直接操作 master 绕过 PR 流程
- [ ] 确认合并 PR 后已创建 Release
- [ ] 确认创建 Release 后已清理 feature 分支（release 保留）
