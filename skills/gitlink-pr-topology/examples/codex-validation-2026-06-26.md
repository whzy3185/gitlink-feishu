# Codex 验证记录（2026-06-26）

Agent 平台：Codex

验证目标：确认 `gitlink-pr-topology` 能在 GitLink 仓库的 open PR 队列上产出可读、可执行的关系图谱，识别依赖链、功能重叠、替代关系、冲突热点和建议处理顺序。

验证结果如下：

# Gitlink/gitlink-cli PR 队列关系图谱

扫描时间：2026-06-26 11:46:10 +08:00

open PR 总数：156

本轮纳入分析：最近 40 条 open PR

范围：#292, #291, #290, #285, #284, #283, #282, #281, #280, #279, #277, #276, #275, #274, #262, #273, #272, #271, #270, #269, #268, #267, #265, #263, #261, #238, #225, #259, #258, #257, #256, #255, #254, #167, #151, #135, #82, #76, #72, #70

### 1. 主要关系边

| 关系 | 置信度 | 证据 | 建议动作 |
|---|---:|---|---|
| #281 supersedes #254 | 高 | 都修 `cmd/api/api.go` 的单次 Raw API 占位符；#281 还补 `--header`、body/query/header 模板和 `internal/client` 支持 | 优先评审 #281；#254 可关闭或改成补充测试 |
| #282 supersedes #268 | 高 | 都改 `shortcuts/branch/branch.go` 和 branch lifecycle；#282 覆盖筛选、默认分支切换与恢复，并补 i18n/文档 | 保留 #282，#268 只保留可迁移的小补丁 |
| #272 supersedes #270 | 高 | #270 新增 message settings；#272 同时包含 message settings 和 notification 工作流 | 若 #272 验证通过，#270 可关闭 |
| #272 overlaps_with #76 | 中 | 都新增 notification 相关 shortcut，均改 `shortcuts/notification/notification.go` 与注册入口 | 对比 API 覆盖面，择一实现，避免双实现 |
| #274 overlaps_with #263/#72 | 高 | 三者都新增 template shortcut，并改 `shortcuts/template/template.go`、`shortcuts/register.go` | 选择一个 template 实现；可从其他 PR cherry-pick 文档或 skill |
| #261 conflicts_with #263/#274/#72 | 高 | #261 标题是 auth checkin，但同时新增 template 文件，和 template PR 重叠 | 要求拆分：auth checkin 与 template shortcut 分开 |
| #283 merge_after #258 | 中 | #283 改 `internal/output/formatter.go` 以显示 PR 编号；#258 修 table 顺序和错误码，是输出层基础修复 | 先合 #258，再处理 #283 |
| #284 review_together #283/#285 | 高 | 三者都改 `shortcuts/pr/pr.go`、`pr_test.go`，分别处理编号显示、编号搜索、login 过滤 | 打包 review，建议顺序 #283 -> #284 -> #285 |
| #273 review_together #275/#225 | 中 | 都改 `shortcuts/repo/repo.go`；分别是治理/转移、upload、scaffold | 一起统一 repo 命令命名和帮助文案 |
| #262 review_together #238/#70/#167 | 中 | 都涉及 user/account 能力；#262/#238/#70 改 `shortcuts/user/user.go`，#167 补 account email | 拆分 account、stats、key 三类能力后择序合并 |
| #276 conflicts_with 多数 shortcut PR | 高 | 127 文件，覆盖 register、pr、repo、user、branch、client、README 等热点 | 暂缓，建议拆分后再进队列 |
| #259 conflicts_with 多数 shortcut/skill PR | 高 | 156 文件，覆盖 `cmd/api`、`cmd/auth`、多数 shortcuts、skills、README | 暂缓，要求拆成可审单元 |

### 2. 重叠实现簇

**PR 列表能力簇：#283/#284/#285**

共同点：都在扩展 `pr +list`。#283 显示 PR 编号，#284 支持按编号搜索，#285 支持按作者 login 过滤。三者目标互补，但同改 `shortcuts/pr/pr.go` 的 flag、结果封装和 helper 区域，合并时很容易互相重写。建议一起 review，先确立列表输出契约，再合功能。

**Raw API 占位符簇：#281/#254**

#281 覆盖更完整，已经从 path 扩展到 query/body/header 模板，并补了 client header 支持；#254 更像早期窄修。建议保留 #281，#254 关闭或转为补充用例。

**Branch lifecycle 簇：#282/#268**

两者目标高度一致。#282 范围更完整，且同步 README、中文 README、i18n 和测试。建议以 #282 为主线。

**Notification/message setting 簇：#272/#270/#76**

#270 是 message setting 子集，#272 是 notification + message setting 组合，#76 是较早的 notification OpenAPI shortcut。建议维护者先决定 notification 命令的最终 API 覆盖，再合并一个代表实现。

**Template 簇：#274/#263/#72/#261**

#274/#263/#72 都实现 template shortcut；#72 还带 `skills/gitlink-template/SKILL.md`，#274 带更完整的 `doc/changes` 和 i18n。#261 混入 template 改动但标题是 auth checkin，应拆分。

**大而全冲突簇：#276/#259**

这两条是当前队列最大冲突源。它们和几乎所有 shortcut PR 都抢 README、注册入口、i18n、核心 shortcut 文件。建议不要直接进入合并队列，先拆分。

### 3. 冲突热点

- `README.md`：25 条 PR 修改
- `README.zh-CN.md`：23 条 PR 修改
- `skills/README.md`：14 条 PR 修改
- `shortcuts/register.go`：13 条 PR 修改
- `shortcuts/register_test.go`：11 条 PR 修改
- `internal/i18n/locales/en-US.json` / `zh-CN.json`：各 10 条 PR 修改
- `shortcuts/pr/pr.go`：#285/#284/#283/#276/#259
- `shortcuts/repo/repo.go`：#275/#273/#225/#276/#259
- `shortcuts/user/user.go`：#262/#238/#70/#276/#259
- `cmd/api/api.go`：#281/#254/#259

### 4. 建议处理顺序

1. 先冻结 #276 和 #259：要求拆分，不建议在现状下合并，否则会持续冲突后续小 PR。
2. 先处理明确替代关系：#281 替代 #254；#282 替代 #268；#272 替代或吸收 #270。
3. 先合基础修复：#258 输出稳定性、#281 Raw API 修复，优先级高，能降低后续验证不确定性。
4. 打包评审 PR 列表簇：按 #283 -> #284 -> #285 顺序处理，统一 `pr +list` 输出、编号查询和 login 过滤。
5. 择一处理重复命令簇：template 在 #274/#263/#72 中选主实现；notification 在 #272/#76 中选主实现。
6. 按命令族处理 focused shortcut：branch 用 #282；repo 组一起看 #273/#275/#225；user/account 组一起看 #262/#238/#167/#70。
7. 最后处理 skill-only PR：#292/#291/#290/#277/#280/#257/#256/#255/#265/#269 主要冲突在 `skills/README.md` 或各自 skill 目录，可批量审查索引一致性。

### 5. 需要深挖的对象

- 用 `gitlink-pr-integrator`：#281、#282、#272、#274、#283/#284/#285，重点做合并态和冲突验证。
- 用 `gitlink-pr-assessor`：#276、#259，重点判断是否应拆分、暂缓或拒绝。
- 人工重点比较：#272 vs #76、#274 vs #72/#263、#262 vs #238/#70。
