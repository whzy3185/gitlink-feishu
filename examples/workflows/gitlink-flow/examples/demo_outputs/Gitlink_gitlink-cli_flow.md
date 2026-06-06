# 社区运营周报 — Gitlink/gitlink-cli

生成时间：2026-06-06 14:40　|　工具：gitlink-flow 端到端工作流

> 本周报由 gitlink-flow 自动串联 Issue 分拣、PR Review、Release Notes、社区健康体检、贡献者致谢等步骤生成，覆盖社区运营全链路。

## 一、仓库概览

- Star 5 / Fork 25 / Issue 17 / PR 129

## 二、Issue 自动分拣

共 17 个 Issue，自动分类：

- bug：1 个
- feature：1 个
- good-first：5 个
- other：10 个

发现 **5** 个适合新人上手的任务，建议打 `good first issue` 标签：

- #142586 API是否支持自动读取仓库内文件（README等）？
- #142303 [test] batch-close 测试 issue 1
- #142304 [test] batch-close 测试 issue 2
- #142124 Test: PR#7 标题已修改
- #142155 [test] PR#11 v1 API 测试 - 已更新

## 三、PR Review 汇总

共 20 个 PR：开放 20 / 已合并 0 / 已关闭 0，合并率 0.0%。

待 Review 的 PR：

- #129  feat(skills): 新增 gitlink-kb 知识库问答 Skill — @Ct201314 （来自 Fork）
- #128 feat(skills): 新增 gitlink-contributor 贡献者致谢与成长 Skill — @Ct201314 （来自 Fork）
- #127 feat(skills): 新增 gitlink-deps 依赖追踪 Skill — @Ct201314 （来自 Fork）
- #126 feat(skills): 新增 gitlink-scaffold 社区健康文件体检 Skill — @Ct201314 （来自 Fork）
- #125 feat(skills): 新增 gitlink-newcomer 新人引导 Skill — @Ct201314 （来自 Fork）
- #124 新增 Raw API 批处理执行器 — @Mengz （来自 Fork）
- #123 新增 Release 资产下载命令 — @Mengz （来自 Fork）
- #122 完善仓库 README 快捷命令 — @Mengz （来自 Fork）
- #121 feat(shortcut): add shortcuts/license — @co63oc （来自 Fork）
- #120 fix(health): use effective list filters — @wangyue111 （来自 Fork）

## 四、社区健康体检

健康度评分：**50/100**
- 已具备：README、LICENSE
- 缺失：CONTRIBUTING、贡献准则

## 五、贡献者致谢

共 20 位贡献者，本周致谢榜前列：

- 🥇 wbtiger（112 次贡献）
- 🥈 wangyue789（44 次贡献）
- 🥉 wauxing（31 次贡献）
- 4. Mengz（26 次贡献）
- 5. whale（24 次贡献）

## 六、Release Notes（自动生成）

基于提交历史，版本 `v0.1.18` 的变更摘要（规范化提交 70/127）：

## v0.1.18

### 新功能（feat）
- remove per-issue detail API, add tag tables and persistence
- rewrite gitlink-health as a Go shortcut
- add metadata lookup shortcuts
- support metadata fields and id alias
- add CLI localization foundation
- add skill gitlink-license-compliance
- add issue label shortcuts
- add OpenAPI shortcuts
- add reopen shortcut
- support body input files
- add milestone, compare, pr reopen shortcuts and api enhancements (#45)
- add PR closed time and repo README shortcuts (#49, #52)
- add repository member shortcuts (#46)
- add readme shortcut
- add repository member shortcuts
- add authors shortcut
- add assigners shortcut
- add community ops automation example
- add review shortcuts
- add skills gitlink-release-auto

### 缺陷修复（fix）
- address code review issues 3-6,8
- pr +review now posts a journal comment alongside the formal review
- align schema and check tool lint
- update TestRegisterAll expected groups for label and pipeline
- treat view id as issue number
- preserve raw file API paths
- include closed time in view output
- improve input mode detection and error hint
- report batch-add partial failures
- improve missing binary diagnostics
- URL-encode branch name in unprotect, add +unprotect to Skill
- Issue 操作切换到 v1 API，统一使用 project_issues_index 替代数据库 ID
- 统一使用 pull_request_number 替代 pull_request_id
- preserve issue description on updates
- require Fork even for admin/owner, unless user explicitly says otherwise
- add tool boundary rules to prevent gh/hub misuse on GitLink

### 性能（perf）
- parallel fetch with errgroup and global rate limiter

### 重构（refactor）
- rename pr +close to pr +refuse

### 文档（docs）
- add missing contributor puygob236
- update contributors section with usernames and new contributors
- trim materials for official PR
- remove continuation file from PR draft
- update final submission checklist placeholders
- align workflow agent competition materials
- add competition submission materials
- add REFERENCE.md for all 3 Skills
- finalize webhook shortcut docs
- update README for v0.1.17 — branch skill, npm one-step install, 12 skills
- remove Gitea reference from branch skill
- 补充 Issue status_id 映射表（1=新增, 2=正在解决, 3=已解决, 5=关闭）
- add Fork-based PR workflow guidelines
- update license from Apache 2.0 to MulanPSL-2.0
- translate README to English, add Chinese README.zh-CN.md

### 测试（test）
- add comprehensive test coverage across all packages (88.5% → 90.3%)
- fix webhook endpoint expectations

### 持续集成（ci）
- add local checks (make check, pre-commit hook) and Gitea Actions workflow skeleton
- add PR checks workflow (build, test, vet, fmt)
- add GitHub Release workflow with npm publish

### 工程（chore）
- ensure trailing newline
- fix CI workflow, golangci-lint config, and minor lint/format issues
- add golangci-lint config and fix lint issues
- ignore local gitlink-cli binary

---

由 gitlink-flow 社区运营自动化工作流生成。所有数据来自 GitLink 平台，分析全程只读。