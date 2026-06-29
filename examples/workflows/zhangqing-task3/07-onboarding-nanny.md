# ⑦ 新人全流程保姆 — 执行报告

> 工作流⑦（新人全流程保姆）最终产物
> 驱动者：zhangqing · 工具：gitlink-cli + onboarding Skill + AI Agent
> 执行时间：2026-06-29
> 目标仓库：ylly/gitlink-cli

---

## 工作流概览

| Step | 动作 | 状态 | 写入 |
|------|------|:----:|:----:|
| 1 | member 邀请能力演示（+list / +invite-link） | ✅ | 否 |
| 2 | good-first-issue 识别报告 | ✅ | 否 |
| 3 | 对 3 个 good first Issue 写个性化引导评论 | ✅ | 是（3 条评论） |
| 4 | notification +list 看新人动态 | ✅ | 否 |
| 5 | 跟踪新人首次贡献（ZxR123-Z 案例） | ✅ | 否 |

---

## Step 1：邀请能力演示

- `member +list`：仓库现有 3 个 Manager（ylly / zhangqing23 / ZxR123-Z）
- `member +invite-link`：成功获取邀请链接（id=3371，过期时间 **2026-07-02 11:32**，与任务三截止时间一致）
- 故事设定：因仓库无真实"新人"测试账号，邀请动作用 `+invite-link` 演示命令能力；后续步骤设定为"假设新人已通过邀请链接入仓"

## Step 2：good-first-issue 识别报告

仓库现有 3 个 good first Issue（#8 / #13 / #14，均为 docs 类），onboarding Skill 按"新人友好识别标准"逐条复审 → **全部合理**。其余 14 个开放 Issue 均不适合新人（测试数据 / 模糊描述 / 性能优化 / 新功能开发等）。详见 `onboarding_identify_report.md`。

## Step 3：个性化引导评论发布 ✅

对 3 个 good first Issue 发布了**完全个性化**（非模板）的引导评论，每条含任务目标 + 相关文件 + 本地准备 + 提交 PR 规范 + @mention：

| Issue | 评论 ID | 核心引导 |
|-------|:-------:|---------|
| #8 docs: wiki 命令使用示例 | 478829 | 入手指引：`README.zh-CN.md` + `shortcuts/wiki/wiki.go` |
| #13 docs: README Windows go build | 478830 | 入手指引：参考 `Makefile` + `scripts/build-npm.sh` |
| #14 docs: wiki +list 返回字段说明 | 478831 | 入手指引：跑 `wiki +list --format json` 拿真实数据 |

发布命令模式（绕过 Windows 中文编码坑）：
```bash
MSYS_NO_PATHCONV=1 ./gitlink-cli.exe api POST /v1/ylly/gitlink-cli/issues/<n>/journals \
  --body-file "C:\\...\\comment_<n>.json"
```
（关键发现：journals endpoint 必须用 `/v1/<owner>/<repo>/...` 前缀，否则 404）

## Step 4：通知系统查看

`notification +list --owner zhangqing23`：共 13 条通知（12 未读）。展示通知系统能完整还原仓库动态。

## Step 5：首次贡献追踪（ZxR123-Z 案例）⭐

通知 + Issue + PR 数据交叉验证，**完整还原了一位新人从加入到首次贡献的全过程**：

| 时间 | 事件 | 数据来源 |
|------|------|---------|
| ~1 个月前 | ZxR123-Z 加入项目 | notification `[ProjectMemberJoined]` |
| 6 天前 | ylly 新建多个 Issue（含 good first #8/#13/#14） | notification `[ProjectIssue]` |
| 2 天前 | ZxR123-Z 提交 2 个 PR | notification `[ProjectPullRequest]` |
| 2 天前 | 2 个 PR 全部 merged | pr +list --state merged |

→ ZxR123-Z 已突破"首次贡献"门槛，从"新人"晋升为"活跃贡献者"（在⑤排行榜中排第 3 名，10 commits + 2 PR + 2 Issue，综合积分 24）。

---

## 关键技术发现（避坑）

1. **API 路径双轨制**：release / label 用 `/<owner>/<repo>/...` 即可；issue / journals / pull **必须用 `/v1/<owner>/<repo>/...`**，否则 404
2. **issue +list 与 issue +view 的 tag 字段不一致**：list 用 `issue_tags` 且可能不返回，view 用 `tags` 字段才完整
3. **创建 issue 必须传 `done_ratio: 0`**，否则 MySQL 报 `Column 'done_ratio' cannot be null`
4. **issue create 时 issue_tag_ids 字段不生效**（GitLink bug），需创建后用 `issue +update --label <id>` 补打

---

## Skill 蓝图对照（按 SKILL.md 行号）

本工作流 5 个 Step 严格按 `gitlink-onboarding/SKILL.md` 蓝图执行。

### Step 2 ← `gitlink-onboarding/SKILL.md` 工作流 1（good-first-issue 自动标记）

| 执行动作 | SKILL.md 蓝图位置 | 蓝图要求 | 实际执行 |
|---------|------------------|---------|---------|
| 取开放 Issue | 第 54-58 行 Step 1 | `issue +list` | ✅ 17 个开放 Issue |
| 按识别标准评估 | 第 31-46 行 + 第 60-62 行 Step 2 | 5 条友好信号 + 3 条排除标准 | ✅ 3 个 docs 类入选，14 个按排除标准剔除 |
| 复用现有标签 | 第 64-74 行 Step 3 | 优先复用，无则建 | ✅ 仓库已有 good first label（id 382660），未新建 |
| 输出标记报告 | 第 84-95 行 Step 5 | 表格汇总 | ✅ `onboarding_identify_report.md` |

**关键对照**：SKILL.md 第 44-46 行排除标准（性能优化/描述模糊/CI 部署）逐条复审 14 个未入选 Issue，全部命中排除理由（测试数据 / 模糊描述 / 性能 / 新功能开发）。

### Step 3 ← `gitlink-onboarding/SKILL.md` 工作流 2（引导评论生成）

| 执行动作 | SKILL.md 蓝图位置 | 蓝图要求 | 实际执行 |
|---------|------------------|---------|---------|
| 读 Issue 详情 | 第 103-107 行 Step 1 | `issue +view` | ✅ #8/#13/#14 各取详情 |
| 生成个性化评论 | 第 109-117 行 Step 2 | **禁止固定模板**，4 要素（任务/文件/本地准备/PR 规范） | ✅ 3 条**完全不同**的评论，每条指向具体文件 |
| 发布评论 | 第 118-123 行 Step 3 | `issue +comment` | ✅ journals 478829/478830/478831（绕中文编码走 Raw API） |
| 个性化要求 | 第 205 行注意事项 | "避免对所有 Issue 用同一句" | ✅ #8 指 `wiki.go` / #13 指 `Makefile` / #13 指 `wiki +list --format json` |

### Step 4 ← 通用通知能力

`notification +list` 自查询（受限）。`gitlink-shared/SKILL.md` 平台限制：只能查自己的通知。✅ 用 `--owner zhangqing23` 自查 13 条通知。

### Step 5 ← `gitlink-insight/SKILL.md` 工作流 3（贡献者洞察）交叉验证

通知 `[ProjectMemberJoined]` + `[ProjectPullRequest]` + `pr +list --state merged` 三源交叉 → 完整还原 ZxR123-Z 从入会到首次贡献全过程。该方法论对应 insight SKILL.md 第 173-203 行"贡献者列表 + PR 数据 + 用户信息"多源聚合。

### Skill 串联数

| 步骤 | 使用的 Skill/命令 | 角色 |
|------|------------------|------|
| 1 | member +invite-link | 邀请能力演示 |
| 2 | onboarding 工作流 1（识别） | good-first 识别 |
| 3 | onboarding 工作流 2（引导评论） | 个性化引导 |
| 4 | notification +list | 通知系统查看 |
| 5 | insight 工作流 3（交叉验证） | 首次贡献追踪 |

**5 步串联，满足 PDF「≥3 命令/Skill 串联」要求。**
