# zhangqing 任务三工作流（① ⑤ ⑦）

> **任务三**：构建端到端自动化工作流
> **负责人**：zhangqing
> **目标仓库**：[ylly/gitlink-cli](https://gitlink.org.cn/ylly/gitlink-cli)
> **完成时间**：2026-06-29

本目录演示任务三中 zhangqing 负责的 3 个端到端工作流，全部用 **任务一的 gitlink-cli 命令 + 任务二的 Skill** 串联完成，无新代码。

---

## 工作流总览

| # | 工作流 | 串联步骤 | 使用的 Skill | 公开产出 |
|---|--------|---------|-------------|---------|
| ① | 社区运营自动化 | 分类→分配→周报→发版 | issue-triage + insight + release-auto | [Release v0.2.0-beta.1](https://gitlink.org.cn/ylly/gitlink-cli/releases) |
| ⑤ | 贡献者成长体系 | 取数→排行→颁奖 | git log + issue +list + pr +list | [颁奖 Issue #17](https://gitlink.org.cn/ylly/gitlink-cli/issues/17) + 3 个徽章 label |
| ⑦ | 新人全流程保姆 | 邀请→识别→引导→通知→追踪 | onboarding | 引导评论 on #8 / #13 / #14 |

3 个工作流首尾衔接形成完整闭环：**①把任务二 7 个 Skill 发版固化 → ⑤奖励贡献者 → ⑦引导新人接力贡献**。

---

## 文件清单

| 文件 | 内容 |
|------|------|
| `README.md` | 本文件（总览 + 复现指引） |
| `01-community-ops-automation.md` | ① 社区运营自动化执行报告 |
| `05-contributor-growth.md` | ⑤ 贡献者排行榜 + 徽章授予方案 |
| `07-onboarding-nanny.md` | ⑦ 新人保姆 5 步执行报告 |
| `release-notes-v0.2.0-beta.1.md` | ① 发布的 Release Notes 全文 |
| `agent-dialogue-triggers.md` | **Agent 对话触发语**（PDF 首要产物：演示 Skill 驱动 Agent） |
| `reproduce.sh` | **可复现脚本**（评委照着跑） |

---

## 怎么复现

```bash
cd <gitlink-cli 仓库根目录>
# 前置：已构建 gitlink-cli.exe、已 auth login、对 ylly/gitlink-cli 有写权限
bash examples/workflows/zhangqing-task3/reproduce.sh
```

脚本标注了 `[READ]`（只读）和 `[WRITE]`（写入）段，评委可选择只跑只读段验证流程。

---

## 任务二 Skill 串联关系（PDF 要求：≥3 命令/Skill）

| 工作流 | 串联的 Skill/命令 | 步数 |
|--------|------------------|:---:|
| ① | issue-triage → insight → release-auto | 4 |
| ⑤ | git log + issue +list + pr +list → 排行榜 → label +create + issue +create | 3 |
| ⑦ | member +invite-link → onboarding（识别 + 引导评论）→ notification +list | 5 |

---

## 任务二 Skill 作为设计蓝图（PDF 核心：Skill = 设计蓝图）

本目录的 3 份执行报告都新增了「Skill 蓝图对照」段，**逐动作引用 SKILL.md 行号**，证明每步执行都严格按蓝图落地：

| 工作流 | 蓝图 SKILL.md 行号定位 |
|--------|----------------------|
| ① 步骤 1 分类 | `gitlink-issue-triage/SKILL.md` 工作流 1（第 54-117 行），含分类规则表第 35-50 行 |
| ① 步骤 2 分配（跳过）| `gitlink-issue-triage/SKILL.md` 第 125-146 行（蓝图预判个人仓库 assigners 为空）|
| ① 步骤 3 周报 | `gitlink-insight/SKILL.md` 工作流 2（第 116-166 行）|
| ① 步骤 4 发版 | `gitlink-release-auto/SKILL.md` 一/二/三（第 32-189 行），含预发布第 179-189 行 |
| ⑤ 取数 | `gitlink-insight/SKILL.md` 工作流 3（第 169-203 行）|
| ⑤ 发奖 | `gitlink-issue-triage/SKILL.md` 工作流 1 Step 5-6（第 80-117 行）|
| ⑦ Step 2 识别 | `gitlink-onboarding/SKILL.md` 工作流 1（第 50-95 行）+ 识别标准第 31-46 行 |
| ⑦ Step 3 引导 | `gitlink-onboarding/SKILL.md` 工作流 2（第 99-145 行）+ 个性化要求第 205 行 |
| ⑦ Step 5 追踪 | `gitlink-insight/SKILL.md` 工作流 3（第 169-203 行）多源聚合 |

**意义**：评委打开任一报告的「Skill 蓝图对照」段，可拿着 SKILL.md 行号对照检查，验证"Skill 不是摆设，是真正驱动 Agent 执行的设计蓝图"。

---

## 关键技术发现（避坑）

### 1. GitLink API 路径双轨制 ⚠

| 资源类型 | 路径前缀 | 例子 |
|---------|---------|------|
| release / label | `/owner/repo/...` | `/ylly/gitlink-cli/releases` |
| issue / journals / pulls | `/v1/owner/repo/...` | `/v1/ylly/gitlink-cli/issues/8/journals` |

混用会 404。cli 内部 `v1RepoPath()` 见 `shortcuts/issue/issue.go:14`。

### 2. release +create 没有 --body-file

多行中文 markdown 用 `--body` 在 Git Bash 必乱码。改用：

```bash
MSYS_NO_PATHCONV=1 ./gitlink-cli.exe api POST /ylly/gitlink-cli/releases \
  --body-file payload.json
```

### 3. 创建 issue 必须传 done_ratio

否则 MySQL 报 `Column 'done_ratio' cannot be null`。最小 payload：

```json
{
  "subject": "...",
  "description": "...",
  "priority_id": 2,
  "done_ratio": 0,
  "issue_tag_ids": [<label_id>]
}
```

### 4. issue create 的 issue_tag_ids 字段不生效（GitLink bug）

创建时传 label 数组，issue 上没标签。需要创建后补：

```bash
./gitlink-cli.exe issue +update --owner <o> --repo <r> --number <n> --label <id>
```

### 5. Windows Git Bash 中文乱码根因

cli 输出 UTF-8 字节 → bash 按 GBK 解码 → 错字符 → python 转 `\uXXXX` → 再传给 cli 就把错字符写到服务器。**所有写入操作必须用 `--body-file <UTF-8文件>`**，不要 shell 拼接 JSON。

---

## 设计决策

| 决策点 | 选择 | 理由 |
|--------|------|------|
| 发版版本号 | v0.2.0-**beta.1**（预发布） | 等任务三 7 个工作流全部完工后由整合者发正式 v0.2.0，避免版本号冲突 |
| Release Notes 覆盖范围 | 仓库全部 104 个提交（含他人贡献） | 发版是仓库级动作，必须覆盖该版本全部变更，不能只写自己的 |
| ⑦ 的"新人"问题 | onboarding 主体真做，邀请段用 invite-link 演示 | 仓库无真实新人账号；onboarding 核心价值（识别 + 引导）与有无新人无关 |
