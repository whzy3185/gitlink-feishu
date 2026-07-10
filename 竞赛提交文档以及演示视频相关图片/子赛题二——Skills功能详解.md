# 子赛题二 · Skills 功能详解（提交说明文档）

> 本文为子赛题二的**提交说明文档**，逐一讲清每个 Skill 的**作用、功能、效果**。
> 共 53 个 Skill（+ shared 地基），分三类：**A 类·CLI 包装**（直接封装命令域）/ **B 类·AI 工作流**（多步编排+决策）/ **C 类·数据分析**（采集+指标+报告）。

## 一、Skill 体系总览

| 类型 | 含义 | 数量 | 代表 |
|---|---|:--:|---|
| **A·CLI 包装** | 把一个 GitLink 命令域封装成 AI 可调用的标准化命令 | 21 | repo / issue / pr / wiki / snippet |
| **B·AI 工作流** | 多步骤 AI 编排，含决策树与输出模板，端到端自动化 | 19 | onboarding / code-review / pr-guard |
| **C·数据分析** | 采集数据 + 算指标 + 出报告，科研/健康度洞察 | 13 | research-insight / health / compliance |
| 地基 | 认证/全局参数/安全规则，被所有 Skill 引用 | 1 | shared |

> 命令是「工具」，Skill 是「菜谱」——Skill 告诉 AI「什么场景、用哪些命令、按什么顺序」。本文按**子赛题**分组（与《Skills分类名单》《Skills使用说明》一致）。


---

## 子赛题一 · CLI 命令域（21，A 类·命令包装）

### `gitlink-repo`
> 仓库管理
**类型**：A·CLI 包装　**作用**：把 GitLink 仓库的全量信息查询封装成一组只读命令。

**功能**：仓库详情 +info、贡献者 +contributors、语言占比 +languages、README +readme、文件树 +tree、代码统计 +code-stats、关注/点赞/Fork 等互动，以及 +list/+create/+delete。

**效果**：AI 与用户无需翻文档查 API，一条命令拿到仓库画像所需的全部元数据，是其他分析类 Skill 的数据底座。

### `gitlink-file`
> 仓库文件操作
**类型**：A·CLI 包装　**作用**：封装仓库文件读写，自动处理 base64 编解码与 SHA。

**功能**：+browse 浏览目录、+get 读文件内容（自动解码）、+create/+update/+delete 改文件。

**效果**：让"读 LICENSE/CI 配置/源码"这类需求一条命令完成；写操作自动取 SHA、自动编码，避免手写 base64 出错。

### `gitlink-branch`
> 分支管理
**类型**：A·CLI 包装　**作用**：封装 GitLink 分支管理。

**功能**：+list 查分支、+create 建分支、+delete 删分支、+protect/+unprotect 分支保护。

**效果**：补齐 PR/发布流程的分支操作缺口，支持分支保护策略。

### `gitlink-release`
> 发布管理
**类型**：A·CLI 包装　**作用**：封装版本发布管理，并突破 JSON-only 框架支持二进制下载。

**功能**：+list/+view 查发布、+create/+update/+edit 管理发布、+download 流式下载 Release 附件资源。

**效果**：亮点 +download 用 HTTP 流式写盘，可下载大体积二进制资产，是 CLI 能力扩展的代表性突破。

### `gitlink-search`
> 搜索
**类型**：A·CLI 包装　**作用**：封装 GitLink 全局搜索（仓库/用户/Issue）。

**功能**：+repos 搜仓库、+users 搜用户、+issues 搜 Issue（支持 keyword/category/assignee/author/milestone/tag/sort 等 10 个筛选）。

**效果**：把"找 good-first-issue""按标签过滤 Issue"等高频检索做成带丰富筛选的一站式命令。

### `gitlink-compare`
> Compare GitLink branches
**类型**：A·CLI 包装　**作用**：封装分支/标签/提交的差异对比。

**功能**：+view 看差异概览、+files 看变更文件清单。

**效果**：支撑代码审查、变更影响分析等场景的 diff 数据获取。

### `gitlink-issue`
> Issue 管理
**类型**：A·CLI 包装　**作用**：封装 Issue 管理 + 强大的 CSV 批量引擎。

**功能**：单条 +list/+view/+create/+update/+close/+comment；批量 +batch-create/+update/+close/+open/+assign/+label/+delete（CSV 驱动，含公共引擎、--dry-run 预览）。

**效果**：批量引擎 ~2100 行，一次 CSV 即可批量建/指派/打标签/关闭数十条 Issue，是子赛题一最重的交付。

### `gitlink-pr`
> Pull Request 管理
**类型**：A·CLI 包装　**作用**：封装 PR 管理 + 两步 diff + 评审。

**功能**：+list/+view/+create/+merge/+reopen/+comment；+diff（版本列表→diff 详情两步法）、+files、+reviews/+review、+check-merge。

**效果**：修复了原单步 diff 的假实现，提供真实可用的 PR 变更与评审能力。

### `gitlink-label`
> 标签管理
**类型**：A·CLI 包装　**作用**：封装 Issue 标签（tag）管理。

**功能**：+list/+create/+update/+delete（PATCH 部分更新）。

**效果**：补齐标签维护操作，配合批量打标签使用。

### `gitlink-issue-tag`
> 项目标记管理
**类型**：A·CLI 包装　**作用**：封装 GitLink「项目标记」（Issue 标签体系）管理。

**功能**：+list 查项目标记。

**效果**：对齐 GitLink 平台特有的项目标记概念，区别于普通 label。

### `gitlink-milestone`
> 里程碑管理
**类型**：A·CLI 包装　**作用**：封装里程碑管理。

**功能**：+list/+view（含关联 Issue）/+create/+close/+delete。

**效果**：支持项目阶段规划与进度跟踪，含 +close 路径 bug 修复 + 回归测试。

### `gitlink-member`
> 项目成员管理
**类型**：A·CLI 包装　**作用**：封装项目成员管理。

**功能**：+list/+add/+remove。

**效果**：补齐协作权限管理操作。

### `gitlink-org`
> 组织管理
**类型**：A·CLI 包装　**作用**：封装组织管理。

**功能**：+list 组织列表、+info 组织详情、+members 成员、+create 创建组织。

**效果**：支持组织级协作场景。

### `gitlink-webhook`
> Webhook 管理
**类型**：A·CLI 包装　**作用**：封装仓库 Webhook 全生命周期。

**功能**：+list/+view/+create/+update（GET-then-PUT 保留未指定字段）/+delete/+history（投递任务）/+test（测试投递）。

**效果**：一站式管理 Webhook 并排查投递问题，是事件驱动自动化的基础。

### `gitlink-wiki`
> Wiki 操作
**类型**：A·CLI 包装　**作用**：封装 Wiki 页面与目录全生命周期（对照 GitLink 页面功能一对一实现）。

**功能**：+list/+view/+create/+update/+delete 页面，+mkdir/+rmdir/+rename/+renamedir 目录。

**效果**：解决 Wiki 官方 API 异步重建 Sidebar 导致 CLI 读到旧状态的问题，实现完整 Wiki 自动化。

### `gitlink-pm`
> 项目管理（PM）
**类型**：A·CLI 包装　**作用**：封装 GitLink 项目管理（看板/Sprint/周报）。

**功能**：+boards 看板、+sprints Sprint 议题、+weekly 周报、+tags 标签、+pipelines 流水线、+actions 动作记录。

**效果**：打通 PM 模块数据，支撑项目进度类 Skill。

### `gitlink-ci`
> CI/CD 操作
**类型**：A·CLI 包装　**作用**：封装 CI/CD 操作。

**功能**：+builds 构建列表、+logs 日志、+restart/+stop 重启停止、+enable/+disable 启停、+authorize 授权。

**效果**：让 AI 能查询构建状态、拉日志排查故障，是 pr-guard 等 CI 检查步骤的依赖。

### `gitlink-pipeline`
> Pipeline workflow operations
**类型**：A·CLI 包装　**作用**：封装流水线全生命周期。

**功能**：+list/+runs/+run/+view/+logs/+results/+save-yaml/+enable/+disable/+delete。

**效果**：完整覆盖流水线编排与运维。

### `gitlink-user`
> 用户操作
**类型**：A·CLI 包装　**作用**：封装用户信息与统计。

**功能**：+me 当前用户、+info 用户详情、+headmaps 活跃热力图、+stats-activity/develop/major/role 各维统计、+trends 项目动态。

**效果**：提供用户画像数据，支撑个人视角 Skill（todo 等）。

### `gitlink-snippet`
> 本地代码片段管理
**类型**：A·CLI 包装　**作用**：本地代码片段管理（纯本地、不调 API、免登录）。

**功能**：+create（支持 stdin）/+list/+view/+search 全文检索/+update/+delete/+export。

**效果**：CLI 唯一的本地命令域，AI 可零依赖存取常用代码片段，已端到端实测。

### `gitlink-auth`
> 认证管理
**类型**：A·CLI 包装　**作用**：封装认证管理。

**功能**：login（交互/Token）、status 查状态与有效期、logout。

**效果**：与 gitlink-shared 分工，聚焦认证命令操作，处理 401/Token 过期等场景。


---

## 子赛题二 · AI/分析 Skill（20，B/C 类）

### `gitlink-onboarding`
> 新人引导
**类型**：B·AI 工作流　**作用**：为开源项目新贡献者提供从环境搭建到首次提交的完整引导。

**功能**：搜 good-first-issue → 对每个候选做 5 维度友好度评估（标题清晰度/描述完整度/代码定位/改动范围/难度标签）→ 输出推荐清单 + 生成引导评论。

**效果**：把"哪个 Issue 适合新人"从主观判断变可量化打分，降低新贡献者参与门槛（课程明确要求、原缺失）。

### `gitlink-digest`
> 每日简报
**类型**：B·AI 工作流　**作用**：聚合仓库多源动态成一份可读的项目简报。

**功能**：并行采集 Issue/PR/CI/通知/活跃度 → 按 🔴需关注/🟢新增/🔵进行中/📊指标 分级。

**效果**：一份简报掌握项目全局，解决信息分散问题。

### `gitlink-todo`
> 我的待办
**类型**：B·AI 工作流　**作用**：跨 Issue/PR 汇总「分配给我/@我/我的 PR 待 review」的个人待办。

**功能**：搜 assignee=me 的 Issue + @我的通知 + 我的 PR 评审状态 → 按紧急度排序。

**效果**：补上 GitLink 缺失的「我的视角」，知道下一步该做什么。

### `gitlink-code-review`
> 智能代码审查
**类型**：B·AI 工作流　**作用**：智能代码审查：分析 PR diff 输出结构化 Review。

**功能**：取 PR 变更 → 按严重度分级（Critical/Warning/Suggestion）→ 生成 Review 评论 + 摘要报告。

**效果**：把人工 Review 流程自动化，输出可直接贴的评审意见；含安全审查增强示例。

### `gitlink-commit-quality`
> 提交质量守护
**类型**：B·AI 工作流　**作用**：提交质量守护：检查提交规范。

**功能**：校验 Conventional Commits、PR 描述完整性、分支命名、变更合理性。

**效果**：规范团队提交流程，提升可追溯性。

### `gitlink-gatekeeper`
> Policy-as-Code 的 PR 合并门禁
**类型**：B·AI 工作流　**作用**：Policy-as-Code 的 PR 合并门禁：按版本化策略卡裁决。

**功能**：聚合 review_findings/test_coverage/pr_hygiene/commit_quality/ci_status 5 维加权 → 0-100 评分 → 5 硬门禁 + 三态裁决（PASS/COMMENT/REQUEST_CHANGES）。

**效果**：113 个 PR 全仓验证均分 88.5，把"能否合并"变成可版本化、可解释的策略。

### `gitlink-issue-triage`
> Issue 智能分拣
**类型**：B·AI 工作流　**作用**：Issue 智能分拣。

**功能**：分析开放 Issue 列表 → 按类型/紧急度/复杂度分类 → 生成分拣报告与维护建议。

**效果**：自动整理堆积的 Issue，减轻维护者负担。

### `gitlink-issueops`
> IssueOps 事件驱动自动化
**类型**：B·AI 工作流　**作用**：IssueOps 事件驱动自动化：创建 Issue 即触发 Agent。

**功能**：webhook 捕获 issues 事件（Live 回调或 webhook+tasks 轮询）→ Agent 自动处理。

**效果**：让"建 Issue"成为触发自动化流程的入口。

### `gitlink-stale-issue-manager`
> 过期 Issue 管理
**类型**：B·AI 工作流　**作用**：过期 Issue 管理。

**功能**：识别长期无活动 Issue → 按过期等级标记/提醒/批量关闭，支持白名单与 dry-run。

**效果**：清理社区积压、维护仓库活跃度。

### `gitlink-release-auto`
> 自动化 Release 管理
**类型**：B·AI 工作流　**作用**：自动化 Release 管理。

**功能**：从提交历史自动生成 Release Notes → 推荐语义化版本号 → 批量发版。

**效果**：把发版从手工整理变成自动产出。

### `gitlink-wiki-builder`
> Wiki 文档自动化
**类型**：B·AI 工作流　**作用**：Wiki 文档自动化。

**功能**：自动组织文档结构、批量创建页面、生成侧栏导航、同步代码变更到 Wiki。

**效果**：批量初始化与维护项目文档。

### `gitlink-pipeline-guardian`
> 流水线健康守护
**类型**：B·AI 工作流　**作用**：流水线健康守护。

**功能**：监控流水线状态 → 分析失败模式、识别慢构建 → 生成健康度评分与修复建议。

**效果**：快速定位流水线故障、优化构建效率。

### `gitlink-webhook-sentinel`
> Webhook 监控哨兵
**类型**：B·AI 工作流　**作用**：Webhook 监控哨兵。

**功能**：监控投递成功率、检测端点问题、验证安全配置、分析失败原因。

**效果**：保障事件驱动集成的可靠性。

### `gitlink-notification-digest`
> 通知摘要
**类型**：B·AI 工作流　**作用**：通知摘要：分类汇总通知。

**功能**：汇总 GitLink 通知按类型分类 → 生成摘要，支持批量标记已读。

**效果**：快速清理未读、不遗漏关键通知。

### `gitlink-competition-manager`
> 编程竞赛管理
**类型**：B·AI 工作流　**作用**：编程竞赛管理。

**功能**：批量创建队伍仓库、初始化题目与权限。

**效果**：把竞赛组织的手工建仓自动化。

### `gitlink-health`
> 项目健康度分析（专用工作流）
**类型**：C·数据分析　**作用**：项目健康度分析（专用工作流）：采集到 SQLite 算聚合指标。

**功能**：health +fetch 把 PR/Issue 落库 → 算响应时长、PR 合并效率、贡献者活跃度等指标 → 报告。

**效果**：用确定性的 SQL 指标回答"项目维护得好不好"。

### `gitlink-insight`
> 项目健康度与协作洞察
**类型**：C·数据分析　**作用**：项目健康度与协作洞察。

**功能**：分析 Issue/PR 指标 + 贡献者活跃度 → 生成周报和健康度报告。

**效果**：了解项目进展与团队协作状况。

### `gitlink-ci-health`
> CI 健康巡检
**类型**：C·数据分析　**作用**：CI 健康巡检。

**功能**：检查 CI/CD 授权状态、构建历史和成功率 → CI 健康度报告。

**效果**：排查 CI 故障、分析构建成功率。

### `gitlink-contributor-insight`
> 贡献者活跃度分析
**类型**：C·数据分析　**作用**：贡献者活跃度分析。

**功能**：分析贡献者活跃度、贡献趋势、工作节奏 → 洞察报告。

**效果**：评估成员参与度、发现核心贡献者。

### `gitlink-license-compliance`
> 许可证合规检查
**类型**：C·数据分析　**作用**：许可证合规检查。

**功能**：扫描许可证兼容性、依赖合规性、敏感信息泄露 → 结构化合规报告。

**效果**：排查开源风险、准备合规开源。


---

## 子赛题三 · 工作流（2，B 类）

### `gitlink-pr-guard`
> 代码质量看门人
**类型**：B·AI 工作流　**作用**：代码质量看门人：PR 提交后跑完 5 步门禁闭环。

**功能**：采集(pr)→AI Review(code-review)→CI(ci)→汇总评论(api)→质量判定/合并(pr)；门禁规则 0 Critical+CI 成功→合并。

**效果**：区别于 code-review 只审查，pr-guard 是会下结论、会合并的完整闭环（子赛题三旗舰）。

### `gitlink-workflow`
> AI 自动化工作流
**类型**：B·AI 工作流　**作用**：AI 自动化工作流域：提供可复用的多步编排命令。

**功能**：+triage Issue 分拣、+health 健康度、+pr-summary PR 摘要、+repo-report 仓库报告。

**效果**：把高频分析场景封装成单命令，供其他工作流复用。


---

## 子赛题四 · 科研辅助（9，C 类）

### `gitlink-research-insight`
> 科研仓库洞悉
**类型**：C·数据分析　**作用**：科研仓库洞悉（S1/S3）：四维科研画像 + 谱系。

**功能**：挖掘提交时间线/PR 演进/创新点；四维评分（可复现性 5 项/活跃度/引用价值/协作健康）+ 巴士因子 + fork 检测。

**效果**：回答"能不能复现/引用/合作"；对真实仓库验证：识别 fork、巴士因子 17%、可复现性 8/10。

### `gitlink-research-graph`
> 科研热点追踪与知识图谱（子赛题四·S2）
**类型**：C·数据分析　**作用**：科研热点追踪与知识图谱（S2）。

**功能**：按关键词/分类抓取仓库 → 构建「仓库-学者-主题」networkx 图谱 → 热度榜 + 飙升项目。

**效果**：辅助科研选题与前沿跟踪（呼应"知识图谱"要求）。

### `gitlink-collab-match`
> 科研协作智能匹配（子赛题四·S4）
**类型**：C·数据分析　**作用**：科研协作智能匹配（S4）。

**功能**：分析仓库技术缺口 + 候选人科研画像 → TF-IDF+余弦匹配 → 协作推荐方案。

**效果**：智能找到跨团队/跨学者的合作伙伴。

### `gitlink-research-progress`
> 科研进度智能跟踪与预警（子赛题四·S5）
**类型**：C·数据分析　**作用**：科研进度智能跟踪与预警（S5）。

**功能**：统计提交/Issue/里程碑 → 阈值规则产出 stale/逾期/bus factor 风险预警 + 进度周报。

**效果**：辅助课题组项目管理、提前预警风险。

### `gitlink-research-visual`
> 科研成果可视化沉淀（子赛题四·S6）
**类型**：C·数据分析　**作用**：科研成果可视化沉淀。

**功能**：把时间线/贡献者热力/语言占比/里程碑甘特沉淀成交互 HTML。

**效果**：支持学术分享与成果梳理。

### `gitlink-research-fork-impact`
> 科研 Fork 影响力分析
**类型**：C·数据分析　**作用**：科研 Fork 影响力分析。

**功能**：分析 fork 的改进方向与影响力 → 科研想法传播图谱。

**效果**：揭示科研想法的传播路径。

### `gitlink-research-tracker`
> 技术评估与调研报告
**类型**：C·数据分析　**作用**：技术评估与调研报告。

**功能**：对技术项目多维度评估（社区活跃度/成熟度/技术趋势）→ 含选型建议的调研报告。

**效果**：辅助科研选题分析、竞品对比研究。

### `gitlink-scholar-profile`
> 学者/团队科研画像
**类型**：C·数据分析　**作用**：学者/团队科研画像。

**功能**：跨仓库聚合分析用户/组织的科研产出 → 影响力雷达 + 代表性成果报告。

**效果**：评估某学者/团队的科研产出全貌。

### `gitlink-compliance`
> 开源合规检查
**类型**：C·数据分析　**作用**：开源合规与复现性检查（S2/S3）。

**功能**：扫描许可证、密钥、依赖、隐私 → 验证实验可复现性 → 合规评估报告。

**效果**：保障学术规范、确认可复现。

---

## 三、整体效果与价值

- **覆盖广**：53 个 Skill 覆盖课程 6 大场景 + 个人效率/文档自动化/监控守护/科研辅助等扩展，examples 覆盖率 52/53。
- **可被 AI 驱动**：每个 Skill 遵循 frontmatter + CRITICAL 三连 + 引用 gitlink-shared 的规范，Claude Code 等 Agent 读 SKILL.md 即可按工作流编排命令。
- **三类协同**：A 类提供标准化命令、B 类编排端到端场景、C 类产出数据洞察——从「能用」到「好用」到「有洞见」逐层提升。
- **真实可用**：snippet 7 命令、research-insight 四维画像等已端到端实测，输出符合各自 SKILL.md 模板。
