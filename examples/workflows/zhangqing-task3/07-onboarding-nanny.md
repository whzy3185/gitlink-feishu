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
