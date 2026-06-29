# 工作流 ① 社区运营自动化 — 执行报告

> 任务三 zhangqing 分担工作流 · 端到端串联 issue-triage + insight + release-auto
> 执行时间：2026-06-29 · 目标仓库：ylly/gitlink-cli

---

## 工作流目标

新 Issue 进入仓库后，自动完成：**分类 → 分配 → 周报 → 发版**，形成社区运营闭环。

## 四步执行记录

### 步骤 1：Issue 智能分类 ✅

调用 `gitlink-issue-triage` Skill 扫描 16 个开放 Issue，识别 11 个未分类，按仓库现有中文标签全部分类：

| Issue | 标题 | 标签 |
|-------|------|------|
| #1-#6 | 测试类 | 测试 |
| #10 | bug | 缺陷 |
| #11 | 功能完善 | 功能 |
| #12 | 疑问 | 疑问 |
| #15 | issue+list 响应慢 | 性能 |
| #16 | 希望搜索按 star 排序 | 功能 |

验证：重新 `issue +list`，未分类 = 0。

### 步骤 2：责任人分配 ⏭ 跳过（平台限制）

`issue +assigners` 返回空数组（total_count: 0），即便 `zhangqing23` 已是 Manager 也不在候选列表内。Raw API PATCH `/v1/.../issues/16` 带 `assigned_to_id` 返回 ok 但 assigners 仍为 `[]`。

**结论**：GitLink 个人仓库的 assigners 机制对 Manager 角色不开放，跳过此步并在报告中标注。

### 步骤 3：社区周报生成 ✅

`gitlink-insight` Skill 综合采集：open/closed issue / pr / release / repo info。

关键数据：
- 开放 Issue 16 / 已关闭 6 / 合并 PR 2 / 开放 PR 0 / Release **0**（本工作流前）
- 标签分布：测试 6 / 缺陷 3 / 文档 3 / good first 3 / 功能 2 / 性能 1 / 疑问 1
- Issue 作者：@ylly 14, @ZxR123-Z 2
- 风险观察：仓库尚未发布任何 Release；测试类 Issue 6 个可清理

### 步骤 4：自动发版 ✅

调用 `gitlink-release-auto` Skill：

1. 版本号推荐：v0.1.18 之后提交分类为 feat 26 / fix 17 / docs 7 / test 4 / refactor 3 → 按 Semver 推荐升 MINOR
2. Release Notes 自动生成（按 Conventional Commits 分类组织，贡献者去重）
3. 用 `release +create` 的底层 endpoint（POST /releases）+ `--body-file` 绕开 Windows 中文编码坑
4. 发布 **v0.2.0-beta.1**（预发布版本，等任务三全部完工后由整合者发正式 v0.2.0）

**产出**：
- version_id: 2218
- tag: v0.2.0-beta.1
- URL: https://gitlink.org.cn/ylly/gitlink-cli/releases
- name: `v0.2.0 Beta 1 — 任务三阶段版（预发布）`

## 任务二 Skill 蓝图对照（按 SKILL.md 行号）

本工作流每一步都严格按任务二 Skill 蓝图执行，Skill 是**设计蓝图**，本报告是蓝图的**落地执行**。

### 步骤 1 ← `gitlink-issue-triage/SKILL.md` 工作流 1（自动分类打标签）

| 执行动作 | SKILL.md 蓝图位置 | 蓝图要求 | 实际执行 |
|---------|------------------|---------|---------|
| 拉开放 Issue | 第 58-62 行 Step 1 | `issue +list --state open` | ✅ 拉到 16 个 |
| 筛未分类 | 第 64-68 行 Step 2 | 过滤 `tags=[]` | ✅ 11 个未分类 |
| 读详情分类 | 第 70-78 行 Step 3-4 | 标题+描述语义分类 | ✅ 按中文标签语义归类 |
| 复用现有标签 | 第 80-91 行 Step 5 | **优先复用仓库已有中文标签** | ✅ 直接复用「测试/缺陷/功能/性能/疑问」未新建 |
| 打标签 | 第 93-99 行 Step 6 | `issue +update --label`（覆盖语义） | ✅ 11 个全部分类完成 |
| 输出报告 | 第 101-117 行 Step 7 | 表格汇总 | ✅ 见本报告步骤 1 表格 |

**关键决策对照**：SKILL.md 第 91 行明确说"优先复用现有标签（含中文同义词），先匹配再考虑新建英文标签"。本仓库已有「缺陷/功能/文档」等中文标签，**未新建任何英文标签**，完全符合蓝图。

### 步骤 2 ← `gitlink-issue-triage/SKILL.md` 工作流 2（自动分配 + 通知）

SKILL.md 第 125-146 行 Step 1-2 已**预判**此场景：个人仓库 `assigners` 返回空数组（第 133 行红框警告），按第 146 行"列表为空则跳过分配并在报告中标注"处理。✅ 实际执行与蓝图预判一致，跳过并在报告标注。

### 步骤 3 ← `gitlink-insight/SKILL.md` 工作流 2（Sprint 周报）

| 执行动作 | SKILL.md 蓝图位置 | 蓝图要求 | 实际执行 |
|---------|------------------|---------|---------|
| 采集 Issue 数据 | 第 120-122 行 | closed + open issue | ✅ 16 open / 6 closed |
| 采集 PR 数据 | 第 123-125 行 | merged PR | ✅ 2 merged |
| 采集 Release | 第 126-128 行 | `release +list` | ✅ Release=0（发版前） |
| 风险标注 | 第 164-165 行 | 列风险与阻塞 | ✅ 标注"未发布 Release""测试类 Issue 可清理" |

### 步骤 4 ← `gitlink-release-auto/SKILL.md`

| 执行动作 | SKILL.md 蓝图位置 | 蓝图要求 | 实际执行 |
|---------|------------------|---------|---------|
| 版本号推荐 | 第 32-90 行 一、版本号推荐 | 按 Conventional Commits 分类推 Semver | ✅ feat 26/fix 17 → MINOR 升 → v0.2.0 |
| Release Notes 生成 | 第 94-145 行 二、自动生成 | 按类型分组 + 贡献者去重 | ✅ 按 Conventional Commits 分组，贡献者去重 |
| 预发布版本 | 第 179-189 行 三、预发布版本 | `--prerelease true` | ✅ 选 beta.1 预发布（等整合者发正式 v0.2.0） |
| 贡献者去重 | 第 145 行红框警告 | **必须去重** | ✅ 跨提交去重为 14 人 |

**3 个 Skill 串联，满足 PDF「≥3 命令/Skill 串联」要求。**

## 关键避坑（本工作流踩到的）

| 坑 | 解决 |
|----|------|
| `release +create` 无 `--body-file` 参数，`--body` 传多行中文必乱码 | 改用 `api POST /owner/repo/releases --body-file payload.json` |
| GitLink API 路径双轨制 | release 用 `/owner/repo/...` 即可 |
| `assigners` 在个人仓库对 Manager 不开放 | 平台限制，报告中标注跳过 |
| Windows Git Bash 拼 JSON 中文乱码 | 全部用 `--body-file <UTF-8文件>` + `MSYS_NO_PATHCONV=1` |
