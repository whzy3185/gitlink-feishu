---
name: gitlink-research-tracker
version: 1.0.0
description: "科研进度智能跟踪与预警：通过 milestone/issue/pr 采集科研项目进度，AI 分析里程碑完成度/issue 积压/PR 阻塞，对超期/停滞/阻塞预警。当课题组需要跟踪科研项目进度、发现进度风险时触发。覆盖任务四「科研进度智能跟踪与预警」场景。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli milestone --help"
---

# gitlink-research-tracker（科研进度智能跟踪与预警 · 科研辅助 Skill）

**CRITICAL — 开始前先阅读 [`../../../skills/gitlink-shared/SKILL.md`](../../../skills/gitlink-shared/SKILL.md)。**
**CRITICAL — 本 Skill 为只读分析，不写入仓库。**
**CRITICAL — 只用 gitlink-cli，禁止 gh。**

> **定位**：任务四科研辅助 Skill（第 5 个）。科研项目有里程碑和进度，本 Skill 通过 milestone/issue/pr 采集进度，AI 分析完成度、积压、阻塞，**预警超期/停滞/阻塞**。覆盖 PDF「科研进度智能跟踪与预警」。

---

## 跟踪模型

| 跟踪对象 | 指标 | 预警条件 |
|---------|------|---------|
| 🏁 里程碑 | 完成度（closed/total issue）、截止日期 | 临近截止/超期/完成度低 |
| 🐛 Issue | 开放数、积压时间、未分类 | 积压 > 30天 / 大量未分类 |
| 🔀 PR | 开放数、待合并、阻塞 | PR 长期未合并 |
| 📈 整体进度 | issue 关闭速率、PR 合并速率 | 速率下降/停滞 |

---

## 工作流

### Step 1：采集里程碑进度
```bash
gitlink-cli milestone +list --owner <owner> --repo <repo> --format json
# 每个 milestone：名称/截止日期/关联 issue 完成度
```

### Step 2：采集 Issue/PR 状态
```bash
gitlink-cli issue +list --state open --format json    # 开放 issue（积压）
gitlink-cli issue +list --state closed --format json  # 关闭速率
gitlink-cli pr +list --format json                    # 开放 PR（阻塞）
```

### Step 3：AI 分析 + 预警
- 里程碑：完成度 vs 截止日期 → 是否延期风险
- Issue：积压时间 → 停滞预警
- PR：长期未合并 → 阻塞预警
- 整体：关闭速率趋势 → 进度健康度

### Step 4：输出进度跟踪报告
```markdown
## 📊 科研进度跟踪与预警 — <owner>/<repo>

### 🏁 里程碑进度
| 里程碑 | 截止 | 完成度 | 状态 |
|--------|------|:------:|:----:|
| v1.0 | 2026-08 | 60% | 🟢 正常 |
| v2.0 | 2026-06 | 30% | 🔴 超期预警 |

### ⚠️ 预警
- 🔴 里程碑 v2.0 已超期，完成度仅 30%
- 🟡 N 个 Issue 积压 > 30 天
- 🟡 N 个 PR 长期未合并

### 📈 整体进度健康度：🟡/🔴/🟢
### 建议：<优先处理/调整截止/增加人力>
```

---

## 关键避坑
| 坑 | 解决 |
|----|------|
| 无 milestone 的仓库 | 跳过里程碑分析，仅看 issue/pr 进度 |
| issue +list 含已关闭 | 客户端按 status.id=1 过滤开放 |
| 截止日期解析 | milestone.effective_date |
| 速率需历史对比 | 取近 N 周关闭数对比（单次为快照）|

---

## 实测落地参考
**Gitlink/gitlink-cli**：
- 里程碑：无正式 milestone（工具型项目按 release 迭代）→ 改用 Release 节奏评估进度
- Issue：19 个（9 开/10 关），关闭率 53%，无严重积压
- PR：322（活跃合并）
- **进度健康度**：🟢 良好（release 节奏稳定 v0.1.x→v0.2.0，issue 关闭正常，PR 活跃）
- **预警**：无（项目持续迭代，无停滞风险）

详见 verification.md。
