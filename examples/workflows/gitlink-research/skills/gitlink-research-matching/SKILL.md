---
name: gitlink-research-matching
version: 1.0.0
description: "科研协作智能匹配：从贡献者的 commit/PR 活动推断技能画像，按技能互补/研究方向匹配科研合作者，输出协作匹配建议。当课题组寻找技能互补的合作者、科研工作者寻找协作伙伴时触发。覆盖任务四「科研协作智能匹配」场景。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli repo --help"
---

# gitlink-research-matching（科研协作智能匹配 · 科研辅助 Skill）

**CRITICAL — 开始前先阅读 [`../../../skills/gitlink-shared/SKILL.md`](../../../skills/gitlink-shared/SKILL.md)。**
**CRITICAL — 本 Skill 为只读分析，不写入仓库。**
**CRITICAL — 只用 gitlink-cli，禁止 gh。**

> **定位**：任务四科研辅助 Skill（第 4 个）。科研协作常需找"技能互补"的伙伴。本 Skill 从仓库贡献历史**推断每个贡献者的技能领域**，匹配互补合作者 / 符合研究方向的贡献者。覆盖 PDF「科研协作智能匹配」。

---

## 匹配模型

### 贡献者技能画像（从活动推断）
| commit/PR 涉及 | 推断技能 |
|---------------|---------|
| `shortcuts/wiki` `shortcuts/label` | API 封装 / 命令开发 |
| `skills/` | Skill 设计 / AI Agent / 文档 |
| `internal/` `cmd/` | 核心架构 / Go |
| `.github/` `.devops/` | CI/CD / DevOps |
| `npm/` | 前端 / 跨平台打包 |
| `examples/` | 工作流 / 场景设计 |

### 匹配维度
| 维度 | 方法 |
|------|------|
| 🤝 技能互补 | A 擅长 X、B 擅长 Y，X+Y 覆盖完整需求 → 推荐组队 |
| 🎯 研究方向 | 科研方向关键词 → 匹配相关技能的贡献者 |
| 📊 活跃度匹配 | 活跃度相近（协作节奏匹配）|

---

## 工作流

### Step 1：采集贡献者活动
```bash
git log --format="%an|%s"                # 每人 commit 涉及的模块（从 message/文件推断）
gitlink-cli pr +list --owner <o> --repo <r> --format json   # PR 分支/模块
gitlink-cli issue +list --owner <o> --repo <r> --format json # Issue 关注领域
```

### Step 2：AI 推断技能画像
从每人 commit/PR 涉及的目录/文件，推断技能领域（如 Go核心 / 前端 / Skill设计 / CI）。

### Step 3：匹配
- 技能互补对（A+B 覆盖全栈）
- 研究方向匹配（关键词→技能→人）

### Step 4：输出匹配建议
```markdown
## 🤝 科研协作匹配 — <owner>/<repo>

### 贡献者技能画像
| 贡献者 | 擅长领域 | 活跃度 |
|--------|---------|:------:|
| wbtiger | Go核心/统筹/CI | 高 |
| ylly | Wiki/API/Skill设计 | 高 |

### 推荐协作对（技能互补）
1. wbtiger(Go核心+CI) + ylly(API+Skill) → 全栈 AI 工具开发

### 研究方向匹配
方向"AI Agent 工具" → wbtiger/ylly/ZxR/zhangqing（均有 Skill 经验）
```

---

## 关键避坑
| 坑 | 解决 |
|----|------|
| 技能推断需读 commit 文件 | `git log --name-only --author=<u>` 取涉及文件 |
| 单人项目无法匹配 | 标注"贡献者过少，建议扩充团队" |
| 推断主观 | 结合 commit message + 文件路径双重信号 |

---

## 实测落地参考
**Gitlink/gitlink-cli**：
- wbtiger：涉及 internal/cmd/.github → **Go核心 + CI + 统筹**
- ylly：涉及 shortcuts/wiki + skills → **API + Skill设计 + 文档**
- ZxR：shortcuts/label + skills → **命令开发 + Skill**
- zhangqing：shortcuts/notification + skills → **命令开发 + Skill**
**匹配**：4 人技能互补（核心+API+命令+Skill），适合组队做 AI 工具开发。

详见 verification.md。
