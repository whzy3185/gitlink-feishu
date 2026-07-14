---
name: gitlink-newcomer-nanny
version: 1.0.0
description: "新人全流程保姆编排 Skill：串联 gitlink-member + gitlink-onboarding + notification + gitlink-insight，完成「邀请加入 → good first 识别 → 个性化引导评论 → 通知跟踪 → 首次贡献追踪」的新人运营闭环。当用户需要邀请仓库新人、识别 good first Issue、给新人写引导评论、还原新人首次贡献过程时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli member --help"
---

# gitlink-newcomer-nanny（新人全流程保姆 · 端到端编排）

**CRITICAL — 开始前先阅读任务二的 `gitlink-shared/SKILL.md`，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有写入操作前（发邀请、发评论、打标签），务必先确认用户意图。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

> **定位**：本 Skill 是一个"总指挥"编排 Skill，依次调用任务二的 3 个子 Skill（member + onboarding + insight）+ notification 自查，完成新人从加入到首次贡献的全流程陪伴。

---

## 工作流概览（串联 3 个子 Skill + notification）

| Step | 调用子 Skill / 命令 | 做什么 | 写入 |
|:----:|------------------|--------|:----:|
| 1 | gitlink-member | 邀请能力演示（list / invite-link） | 否（演示） |
| 2 | gitlink-onboarding 工作流 1 | 扫描开放 Issue → 按新人友好标准识别 good first | 否（或打标签时写入） |
| 3 | gitlink-onboarding 工作流 2 | 对每个 good first 写**完全个性化**引导评论 | 是 |
| 4 | notification +list | 查看新人动态通知 | 否 |
| 5 | gitlink-insight 工作流 3 | 交叉验证多源数据，还原新人首次贡献全过程 | 否 |

**串联满足 PDF「≥3 命令/Skill 串联」要求**：5 个 Step 串 3 个子 Skill + notification。

---

## 详细工作流

### Step 1：邀请能力演示（调用 gitlink-member）

```bash
# 看仓库现有成员
gitlink-cli member +list --owner <o> --repo <r> --format json

# 生成邀请链接（有过期时间）
gitlink-cli member +invite-link --owner <o> --repo <r> --format json
```

返回字段含 `id` / `role` / `expired_at` / `sign`。链接需手动发给新人。

**故事设定**：若仓库无真实"新人"测试账号，本步用 `+invite-link` 演示命令能力，后续 Step 设定为"假设新人已通过邀请链接入仓"。

### Step 2：good first Issue 识别（调用 gitlink-onboarding 工作流 1）

严格按 `skills/gitlink-onboarding/SKILL.md` 工作流 1（**第 50-95 行**）：

| 子动作 | 蓝图行号 | 做什么 |
|--------|---------|------|
| 取开放 Issue | 第 54-58 行 Step 1 | `gitlink-cli issue +list --state open` |
| 按识别标准评估 | 第 31-46 行 + 第 60-62 行 Step 2 | **5 条友好信号 + 3 条排除标准**（详见下表） |
| 复用现有标签 | 第 64-74 行 Step 3 | 优先复用仓库已有 `good first` label，无则建 |
| 输出标记报告 | 第 84-95 行 Step 5 | 表格汇总识别结果 |

**新人友好识别标准**（蓝图第 31-46 行）：

| ✅ 5 条友好信号（命中即入选） | ❌ 3 条排除标准（命中即剔除） |
|------------------------------|------------------------------|
| 1. 任务边界清晰 | 1. 性能优化（需底层知识） |
| 2. 描述含具体改动点 | 2. 描述模糊（无明确目标） |
| 3. 标注"docs / 文档"类 | 3. CI/CD 部署（需环境权限） |
| 4. 影响范围小（单文件/单命令） | |
| 5. 不依赖历史上下文 | |

### Step 3：个性化引导评论（调用 gitlink-onboarding 工作流 2）⚠️写入

严格按 `skills/gitlink-onboarding/SKILL.md` 工作流 2（**第 99-145 行**）：

| 子动作 | 蓝图行号 | 做什么 |
|--------|---------|------|
| 读 Issue 详情 | 第 103-107 行 Step 1 | `gitlink-cli issue +view --number <n>` |
| 生成个性化评论 | 第 109-117 行 Step 2 | **4 要素**：任务目标 + 相关文件 + 本地准备 + 提交 PR 规范 |
| 发布评论 | 第 118-123 行 Step 3 | 见下方 Raw API 命令 |
| **禁止模板** | 第 205 行 注意事项 | "避免对所有 Issue 用同一句" —— 每条评论必须**指向不同文件** |

**⚠️ 关键避坑**：`issue +comment` 在 Windows 拼 JSON 中文必乱码，且 journals endpoint **必须用 `/v1/` 前缀**（不带则 404）。改用 Raw API：

```bash
# comment_N.json: {"notes": "<UTF-8 中文评论>"}

MSYS_NO_PATHCONV=1 gitlink-cli api POST /v1/<owner>/<repo>/issues/<n>/journals \
  --body-file comment_N.json --format json
```

返回 journal ID 作为评论落地证据。

**个性化示例**（实测，3 条评论各指不同文件）：
- #8 → 指向 `shortcuts/wiki/wiki.go`
- #13 → 指向 `Makefile` + `scripts/build-npm.sh`
- #14 → 指向 `wiki +list --format json` 真实数据

### Step 4：通知系统查看（通用 notification 能力）

```bash
# ⚠️ --owner 必须填「自己」的 login，查别人会 403
gitlink-cli notification +list --owner <self_login> --limit 50 --format json
```

通知类型含：`[ProjectMemberJoined]`（新人加入）/ `[ProjectIssue]`（建 Issue）/ `[ProjectPullRequest]`（提 PR）等。

**受限说明**（蓝图 `gitlink-shared/SKILL.md`）：只能查自己的通知，无法查他人的。

### Step 5：首次贡献追踪（调用 gitlink-insight 工作流 3 交叉验证）

按 `skills/gitlink-insight/SKILL.md` 工作流 3（**第 173-203 行**）多源聚合：

| 数据源 | 命令 | 提取事件 |
|--------|------|---------|
| notification | `notification +list` | `[ProjectMemberJoined]` 新人入会时间 |
| Issue | `issue +list` | good first Issue 创建时间 |
| PR | `pr +list --state merged` | 新人首次合并 PR 时间 |

**三源交叉**还原新人时间线：加入 → good first Issue 提到 → 提 PR → merged。

---

## 输出

| Step | 产物 |
|:----:|------|
| 1 | 邀请链接 + 过期时间 |
| 2 | good first 识别报告（命中友好信号 / 命中排除标准） |
| 3 | 各 Issue 的 journal ID |
| 4 | 通知条数 + 事件分类 |
| 5 | 新人时间线（加入 → 首次贡献）|

---

## 关键避坑（实测提炼）

| 坑 | 解决 |
|----|------|
| journals endpoint 必须 `/v1/<owner>/<repo>/...` 前缀 | 不带 `/v1` 直接 404（issue.go 的 `v1RepoPath`） |
| `issue +comment` 拼 JSON 中文乱码 | 改用 `api POST /v1/.../journals --body-file comment.json` |
| GET `/journals` endpoint 返回 HTML 不是 JSON | 已知限制，验证评论看网页或用 POST 返回的 journal ID |
| `notification --owner` 查别人 403 | 必须填自己的 login |
| 引导评论被模板化（违反蓝图第 205 行） | 每条评论必须指向**不同文件**，禁止"欢迎贡献"通用模板 |
| `--state merged` 过滤不精确 | 需客户端按 `pull_request_status` 字段判断（0=open, 1=merged, 2=closed） |

---

## 实测落地参考

本工作流已在 `ylly/gitlink-cli` 实测落地（2026-06-29）：

| Step | 实测结果 |
|:----:|---------|
| 1 | `member +invite-link` 返回邀请 id=3371（2026-07-02 过期） |
| 2 | 识别 #8/#13/#14 三个 good first Issue（均为 docs 类），其余 14 个开放 Issue 命中排除标准 |
| 3 | 3 条**完全不同**的引导评论（journal 478829 / 478830 / 478831），分别指向 wiki.go / Makefile / wiki +list |
| 4 | 自查 13 条通知（12 未读）含 MemberJoined / Issue / PullRequest 三类事件 |
| 5 | 完整还原 **ZxR123-Z** 时间线：1 月前加入 → 6 天前见 good first → 2 天前提 2 PR → 全部 merged（综合积分 24，⑤ 排行榜第 3） |

执行报告：仓库内 `examples/workflows/zhangqing-task3/07-onboarding-nanny.md`。
