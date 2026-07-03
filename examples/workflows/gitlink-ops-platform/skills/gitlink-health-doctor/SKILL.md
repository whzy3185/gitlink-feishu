---
name: gitlink-health-doctor
version: 1.0.0
description: "智能健康巡检编排（AI医生）：依次调度 gitlink-insight → gitlink-issue-triage → gitlink-docs-assistant → gitlink-insight，完成「诊断 → 治疗Issue → 补文档 → 复查」的闭环治理。当用户需要给仓库做全面体检并自动治理、或对比治理前后效果时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli issue --help"
---

# gitlink-health-doctor（智能健康巡检 · AI医生 编排 Skill）

**CRITICAL — 开始前先阅读任务二的 [`gitlink-shared/SKILL.md`](../../../../skills/gitlink-shared/SKILL.md)（认证、权限、API 注意事项）。**
**CRITICAL — 所有写入操作前（打标签、补文档），务必先确认用户意图。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 gh（GitHub CLI）操作 GitLink 资源。**

> **定位**：本 Skill 是"总指挥"编排 Skill，本身不直接跑命令，而是依次调用 4 个子 Skill 完成"AI 医生"闭环：**insight 诊断 → triage 治疗 Issue → docs 补文档 → insight 复查**。把任务二的 3 个 Skill（triage / docs-assistant / insight）串成端到端治理，是任务三的核心创新点。

---

## 编排架构

```
gitlink-health-doctor（AI医生）
  ├── Step 1 → Skill("gitlink-insight")          诊断：体检仓库，找出病灶
  ├── Step 2 → Skill("gitlink-issue-triage")     治疗：给未分类 Issue 打标签
  ├── Step 3 → Skill("gitlink-docs-assistant")   进补：补全缺失文档到 Wiki
  └── Step 4 → Skill("gitlink-insight")          复查：对比治疗前后指标
```

## 子 Skill 依赖

| 顺序 | 子 Skill | 角色 | 写入 |
|:----:|---------|------|:----:|
| 1 | gitlink-insight（工作流1：项目健康度报告）| 诊断：采集数据 → 输出健康度 | 否 |
| 2 | gitlink-issue-triage（工作流1：自动分类打标签）| 治疗：未分类 Issue → 分类打标签 | 是 |
| 3 | gitlink-docs-assistant（工作流1+2：体检+补全）| 进补：缺失文档 → 写入 Wiki | 是 |
| 4 | gitlink-insight（工作流1）| 复查：再次体检 → 治疗前后对比 | 否 |

**串联满足 PDF「≥3 命令/Skill 串联」要求**：4 个 Step 串 3 个子 Skill。

---

## 前置：收集参数

| 参数 | 说明 | 示例 |
|------|------|------|
| owner | 仓库所有者 | `ylly` |
| repo | 仓库名 | `gitlink-cli` |

---

## 工作流

### Step 1：诊断（调用 gitlink-insight 工作流1）

**→ 调用 `Skill("gitlink-insight", args="对 <owner>/<repo> 做项目健康度报告（工作流1，只读采集分析，不要写入）。")`**

按 gitlink-insight 工作流1：采集 Issue/PR/Release/语言/贡献者数据 → 输出 7 维健康度报告。

**记录诊断出的"病灶"**（供后续治疗）：
- Issue 治理差（大量未分类、无标签）
- 文档缺失（CHANGELOG / API 文档等不存在）
- 其他低分维度

关键命令（子 Skill 内部）：
```bash
gitlink-cli repo +info --owner <o> --repo <r> --format json
gitlink-cli issue +list --owner <o> --repo <r> --state open --format json
gitlink-cli issue +list --owner <o> --repo <r> --state closed --format json
MSYS_NO_PATHCONV=1 gitlink-cli api GET /<owner>/<repo>/languages.json --format json
```

### Step 2：治疗 Issue（调用 gitlink-issue-triage 工作流1）⚠️写入

**→ 调用 `Skill("gitlink-issue-triage", args="对 <owner>/<repo> 执行工作流1：扫描未分类开放 Issue，按语义分类，复用仓库现有标签打标。打标签前确认用户意图。")`**

针对诊断出的"Issue 治理差"，按 gitlink-issue-triage 工作流1 治理。

关键命令（子 Skill 内部）：
```bash
gitlink-cli issue +list --owner <o> --repo <r> --state open --format json   # 找未分类
gitlink-cli label +list --owner <o> --repo <r> --format json               # 复用现有标签
gitlink-cli issue +view --owner <o> --repo <r> --number <n> --format json  # 读详情分类
gitlink-cli issue +update --owner <o> --repo <r> --number <n> --label <id> # 打标签
```

### Step 3：补文档（调用 gitlink-docs-assistant 工作流1+2）⚠️写入

**→ 调用 `Skill("gitlink-docs-assistant", args="对 <owner>/<repo> 做文档体检（工作流1），然后补全缺失文档（工作流2，写入 Wiki，补全前确认用户意图）。")`**

针对诊断出的"文档缺失"，按 gitlink-docs-assistant 体检 + 补全。

关键命令（子 Skill 内部）：
```bash
gitlink-cli repo +readme --owner <o> --repo <r>                       # 体检 README
gitlink-cli wiki +list --owner <o> --repo <r> --format json           # 看缺什么
MSYS_NO_PATHCONV=1 gitlink-cli wiki +create --owner <o> --repo <r> \
  --name "CHANGELOG" --content "# 变更记录\n\n..." \
  --message "docs: AI 自动补全 CHANGELOG"                              # 补全到 Wiki
```

### Step 4：复查（再次调用 gitlink-insight 工作流1）

**→ 调用 `Skill("gitlink-insight", args="对 <owner>/<repo> 再次做项目健康度报告（工作流1，只读）。对比 Step 1 诊断结果，输出治疗前后变化。")`**

重新体检，重点对比：Issue 分类率（未分类 → 已分类）、文档完整度（缺失 → 补全）。

---

## 最终输出

四个子 Skill 执行完毕后，汇总输出"AI 医生诊疗报告"：

```markdown
## 🩺 AI 医生诊疗报告 — <owner>/<repo>

### 🔍 诊断（Step 1）
- 总评：⭐x.x / 5
- 病灶：① Issue 治理差（N 个未分类）② 文档缺失（CHANGELOG/API 文档）

### 💊 治疗（Step 2-3）
- Issue：N 个已分类打标签（复用 缺陷/功能/疑问 等现有标签）
- 文档：补全 CHANGELOG / API 文档到 Wiki

### 📈 复查（Step 4）
| 指标 | 治疗前 | 治疗后 |
|------|:------:|:------:|
| Issue 分类率 | 30% | 90% |
| 文档完整度 | 50% | 85% |
| 健康度评分 | 3.5 | 4.2 |

### 结论
仓库健康度由 🟡 待完善 提升至 🟢 良好。
```

---

## 关键避坑（实测提炼）

| 坑 | 解决 |
|----|------|
| `api GET /v1/...` 路径在 Git Bash 被转成 Windows 路径 → 404 | 命令前加 `MSYS_NO_PATHCONV=1` |
| `api` 命令会自动补 `.json`，路径无需手动加 | 不要画蛇添足手动加 .json（已实测）|
| 个人仓库 `assigners` 返回空（平台限制）| 治疗阶段用"打标签"替代"分配责任人"|
| GitLink 标签名限 15 字符 | 用 "good first" 等短名，不用 "good first issue"|
| `wiki +create` 中文内容 | CLI 内部自动 base64，`--content` 直接传中文 |
| `api GET sub_entries` 返回 HTML 非文件列表 | 文档体检改用 `repo +readme` |
| `notification` 跨用户查询 403 | 只能自查通知，不代查他人 |
| `issue +update --label` 是覆盖语义 | Issue 已有标签时要把原标签 ID 一并传入 |

---

## 实测落地参考

本工作流基于 `ylly/gitlink-cli` 实测（2026-06/07）：

| 步骤 | 实测结果 |
|------|---------|
| Step 1 诊断 | insight 健康度 3.9/5；病灶 = Issue 治理弱 + 缺 CHANGELOG/API 文档 |
| Step 2 治疗 | 给 #9 等 Issue 打"缺陷"标签（复用现有标签 id 327264）|
| Step 3 补文档 | docs-assistant 创建 CONTRIBUTING（code 201，commit_count 1→2）|
| Step 4 复查 | Issue 分类率提升、文档完整度提升 |

详见各子 Skill（gitlink-insight / gitlink-issue-triage / gitlink-docs-assistant）的 verification.md。
