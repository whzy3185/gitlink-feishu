# gitlink-contributor-insight 使用样例

## 样例 1：直接调用 Skill（手动执行）

**日期**：2026-06-03
**仓库**：jiangtx/gitlink-cli（Fork from Gitlink/gitlink-cli）
**CLI 版本**：gitlink-cli 0.1.18

### 执行流程

```bash
# Step 1: 获取仓库信息
gitlink-cli repo +info --owner jiangtx --repo gitlink-cli --format json
# → contributor_users_count: 2, pull_requests_count: 9, issues_count: 0
# → fork_info: { fork_project_user_login: "Gitlink" }

# Step 2: 获取 PR 列表（替代不存在的 repo +contributors）
gitlink-cli pr +list --owner jiangtx --repo gitlink-cli --format json
# → 9 个 PR，全部已合并
# → 唯一 author_login: lindiwen23 (5 PRs), jiangtx (4 PRs)

# Step 3: 获取用户信息
gitlink-cli user +info --login jiangtx --format json
# → 注册于 2026-04-28，3 个项目，身份"专业人士"

gitlink-cli user +info --login lindiwen23 --format json
# → 注册于 2025-05-26，6 个项目，1 个组织，身份"专业人士"

# Step 4: 获取 Issue 列表（补充数据）
gitlink-cli issue +list --owner jiangtx --repo gitlink-cli --format json
# → 0 个 Issue
```

### 不可用命令确认

| 命令 | 结果 |
|------|------|
| `gitlink-cli repo +contributors` | 命令不存在，返回 repo 帮助文本 |
| `gitlink-cli user +heatmap` | 命令不存在（user 子命令仅 `+info` / `+me`） |
| `gitlink-cli user +stats` | 命令不存在 |
| `gitlink-cli user +trends` | 命令不存在 |
| `gitlink-cli api GET "/api/v1/repos/.../contributors"` | 返回 HTML 页面，非 JSON |
| `gitlink-cli api GET "/api/v1/users/.../heatmap"` | 返回 HTML 页面，非 JSON |

### 关键发现

**贡献者数据（从 PR 列表提取）：**

| 贡献者 | PR 数 | 活跃日期 | 活跃天数 | 趋势 |
|--------|-------|----------|----------|------|
| lindiwen23 | 5 | 06-01, 06-03 | 2 天 | ↑ |
| jiangtx | 4 | 06-02, 06-03 | 2 天 | ↑ |

**PR 时间分布：**
- 06-01: 1 PR（lindiwen23 #1）
- 06-02: 2 PRs（jiangtx #2, #3）
- 06-03: 6 PRs（jiangtx #4, #5 + lindiwen23 #6, #7, #8, #9）

**分级处理：** 项目仅 3 天历史，适用"年轻项目放宽标准"规则，2 人均标记为 🔥 核心贡献者。

### 生成的报告

```markdown
# 👥 贡献者洞察报告：gitlink-cli

> 分析时间：2026-06-03 10:00 (UTC+8)
> 仓库：jiangtx/gitlink-cli（Fork from Gitlink/gitlink-cli）
> 总贡献者：2 人，本次分析：2 人（全量分析）

## 一、团队概览

| 指标 | 数值 |
|------|------|
| 总贡献者 | 2 |
| 🔥 核心贡献者 | 2 |
| 🌟 活跃贡献者 | 2 |
| 🌱 新兴贡献者 | 0 |
| 💤 休眠贡献者 | 0 |
| 近 30 天活跃率 | 100% |
| 仓库总 PR 数 | 9（全部已合并） |
| 仓库总 Issue 数 | 0 |
| 项目启动时间 | 2026-06-01（3 天前） |

## 二、贡献者活跃度排行榜

| 排名 | 贡献者 | 级别 | 类型 | 活跃天数 | 总PR | 总Issue | 趋势 |
|------|--------|------|------|---------|------|---------|------|
| 1 | lindiwen23 | 🔥 | 代码 | 2 天 | 5 | 0 | ↑ |
| 2 | jiangtx | 🔥 | 代码 | 2 天 | 4 | 0 | ↑ |

## 三、重点贡献者分析

### 🔥 lindiwen23（核心贡献者）
- 5 个 PR（3 feat + 1 fix + 1 refactor），集中上午时段
- 6/3 当天 43 分钟内连续提交 4 个 PR，集中爆发型节奏
- 平台老用户（2025-05-26 注册），6 个项目经验

### 🔥 jiangtx（核心贡献者 / Owner）
- 4 个 PR（3 feat + 1 fix），下午至深夜时段
- 从基础设施修复 → 模块补全，有序推进型节奏
- 平台新用户（2026-04-28 注册），项目 Owner

## 四、团队健康度评估

| 指标 | 状态 | 说明 |
|------|------|------|
| 核心贡献者占比 | 100% | 2/2 活跃 |
| 近 30 天活跃率 | 100% | 全部近期有贡献 |
| 知识分散度 | ⚠️ Bus Factor = 2 | 人数偏少 |
| 贡献稳定性 | ⚠️ 仅 3 天数据 | 无法评估长期 |

### 风险提示
- ⚠️ 核心贡献者不足（2 人），Bus Factor = 2
- ℹ️ 项目处于早期阶段（3 天），风险置信度有限
- ⚠️ 缺少新鲜血液，0 个外部 Issue

## 五、社区建设建议

1. 保持当前协作节奏（独立分支 + PR 合并）
2. 建立 Issue 文化，添加 Good First Issue 标签
3. 在 GitLink 平台推广，完善使用文档
4. 完善代码注释，降低 Bus Factor 风险
5. 定期同步上游（Gitlink/gitlink-cli）

## 📋 数据来源与局限性

| 数据维度 | 来源 | 可靠性 |
|----------|------|--------|
| 贡献者数量 | repo +info | ✅ 可靠 |
| PR 贡献数据 | pr +list 全量 | ✅ 可靠 |
| 用户信息 | user +info | ✅ 可靠 |
| 贡献热力图 | 不可用（命令未实现） | ❌ 缺失 |
| 统计信息 | 不可用（命令未实现） | ❌ 缺失 |
| 趋势数据 | PR 时间序列推算 | ⚠️ 推算 |
```

---

## 样例 2：通过 Agent 调用 Skill（自动执行）

**日期**：2026-06-03
**仓库**：jiangtx/gitlink-cli
**调用方式**：`Agent(subagent_type="general-purpose", prompt="调用 gitlink-contributor-insight skill，分析一下 jiangtx/gitlink-cli 的贡献者活跃度。")`

### Agent 自主执行的命令序列

```
工具调用 1: gitlink-cli repo +info --owner jiangtx --repo gitlink-cli --format json
           → contributor_users_count: 2, pull_requests_count: 9

工具调用 2: gitlink-cli pr +list --owner jiangtx --repo gitlink-cli --format json
           → 9 PRs, 提取 author_login: lindiwen23(5), jiangtx(4)

工具调用 3: gitlink-cli issue +list --owner jiangtx --repo gitlink-cli --format json
           → 0 issues

工具调用 4: gitlink-cli user +info --login jiangtx --format json
           → 注册 2026-04-28, 3 个项目

工具调用 5: gitlink-cli user +info --login lindiwen23 --format json
           → 注册 2025-05-26, 6 个项目, 1 个组织
```

### Agent 决策过程

Agent 读取 skill 后，**正确遵循了更新后的工作流**：

1. **未尝试 `repo +contributors`**：skill 的"命令可用性声明"表标注该命令不可用
2. **未尝试 `user +heatmap/+stats/+trends`**：skill 标注不可用，直接从 PR 列表推算
3. **未尝试 Raw API**：skill 不推荐此路径，全程使用 Shortcut 命令
4. **正确应用"年轻项目"规则**：识别项目仅 3 天，放宽分级标准，2 人均标记为 🔥 核心
5. **自主增强分析**：Agent 额外分析了工作时段偏好、PR 类型统计、新老比例

共消耗 **5 次 CLI 调用**，**41,312 tokens**，耗时 **77.5 秒**。

### Agent 相对于手动执行的改进

| 维度 | 手动执行 | Agent 执行 |
|------|----------|-----------|
| 工作节奏分析 | 仅按日期统计 | 识别出时段偏好（上午 vs 深夜） |
| PR 类型统计 | 未分类 | feat(6) + fix(3) + refactor(1) |
| 新老比例 | 未计算 | 1:1（jiangtx 1 月 vs lindiwen23 1 年+） |
| 贡献者建议 | 通用建议 | 建议为 lindiwen23 授予更高级别权限 |
| 协作模式 | 分支策略分析 | 新增独立分支 + PR 合并模式分析 |

### Agent 生成的报告摘要

Agent 生成了完整的五段式报告（团队概览 → 排行榜 → 个人分析 → 健康度评估 → 建议），结构与我手动执行一致，但细节更丰富：

- **lindiwen23 分析**：增加了"3 feat + 1 fix + 1 refactor"分类、"上午 9:00-11:30 时段偏好"、"集中爆发型节奏"
- **jiangtx 分析**：增加了"先修复基础设施，再逐步补全功能模块"的有序推进评价
- **健康度评估**：增加了新老比例 1:1 分析，指出两人经验互补
- **建议**：更具体，如"为 lindiwen23 授予更高级别权限"、"编写 CONTRIBUTING.md"

### 验证结论

✅ skill v1.1.0 验证通过：
- Agent 正确遵循了"命令可用性声明"，未尝试不可用命令
- Agent 正确从 `pr +list` 提取贡献者数据（替代不存在的 `repo +contributors`）
- Agent 正确从 PR 时间戳推算活跃天数（替代不存在的 `user +heatmap`）
- Agent 正确从 PR 聚合获得产出量（替代不存在的 `user +stats`）
- Agent 正确从 PR 时间分布判断趋势（替代不存在的 `user +trends`）
- Agent 正确应用"年轻项目放宽标准"规则
- Agent 正确标注数据来源局限性
- Agent 未使用 `gh` 或其他平台工具
- 报告结构完整，覆盖所有必需章节

---

## 异常场景速查

| 场景 | 检测方式 | 数据表现 | 处理 |
|------|----------|----------|------|
| `repo +contributors` 不可用 | 命令返回帮助文本 | 无 `+contributors` 子命令 | 从 `pr +list` 提取 `author_login` |
| `user +heatmap` 不可用 | 命令不存在 | user 仅 `+info`/`+me` | 从 PR 时间戳推算活跃天数 |
| `user +stats` 不可用 | 命令不存在 | 同上 | 从 `pr +list` 聚合 PR/Issue 数 |
| `user +trends` 不可用 | 命令不存在 | 同上 | 从 PR 按日聚合判断趋势 |
| Raw API 返回 HTML | `api GET` 返回 HTML | 非 JSON 响应 | 仅使用 Shortcut 命令 |
| 项目 < 30 天 | PR 时间跨度 < 30 天 | 全部 PR 在近期 | 放宽分级标准，标注"早期阶段" |
| 贡献者 ≤ 2 人 | `contributor_users_count` ≤ 2 | Bus Factor 极低 | 报告标注风险 + 提供吸引新人建议 |
| PR 数为 0 | `pr +list` 空数组 | `issues: []` | 标注"仓库暂无 PR 数据" |
| Issue 数为 0 | `issue +list` 空数组 | `issues_count: 0` | 贡献类型统一标注"代码贡献者" |

---

## 版本兼容性说明

本 skill 基于 `gitlink-cli 0.1.18` 编写。不同版本的可用命令可能有差异：

| CLI 版本 | 贡献者分析可用命令 | 缺失命令 |
|----------|-------------------|----------|
| 0.1.18 | `repo +info`, `pr +list`, `issue +list`, `user +info` | `repo +contributors`, `user +heatmap`, `user +stats`, `user +trends` |
| 未来版本 | 可能新增 `user +heatmap` 等 | — |

当 CLI 版本更新后，重新验证可用命令：
```bash
gitlink-cli repo --help
gitlink-cli user --help
```
