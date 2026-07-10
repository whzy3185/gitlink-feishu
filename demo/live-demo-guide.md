# 4 个 Skill 登录实演指南（onboarding / digest / todo / auth）

> 用法：登录后，**在 Claude Code 里用自然语言触发**，让真 AI 读 SKILL.md 自主编排 —— 这是最强的实演证据。
> 配套：`live-demo.sh`（辅助采集脚本，不想手敲命令时用）。

---

## 〇、前置准备

1. **登录**：`gitlink-cli auth login`（或 `--token`），用 `gitlink-cli auth status` 确认已登录
2. **准备测试仓库**：用一个你自己的/有权限的公开仓库作为演示对象（避免在团队主仓库留痕）
3. **开着 Claude Code**：在本仓库目录下启动，让 AI 能读到 `skills/*/SKILL.md`
4. **安全原则**：只读命令随便跑；写操作（评论/关闭）让 AI 先 `--dry-run` 或确认

---

## 〇〇、推荐演示方式：自然语言触发真 AI（最有说服力）

> 不要告诉 AI 用哪个命令，只描述需求。看它是否**自主读 SKILL.md → 调对命令 → 输出符合模板**。
> 这是"Skills 让 AI 能驱动 CLI"的活证据，比手敲命令强得多。

每个 Skill 下面都给：**① 触发语**（你对 AI 说这句）→ **② 预期 AI 行为** → **③ 手动备选**（想自己跑时）→ **④ 讲解要点**。

---

## 一、auth 实演（最简单，开场暖身）

**① 触发语**：
> "检查一下我的 gitlink 登录状态"

**② 预期 AI 行为**：读 `auth/SKILL.md` → 调 `gitlink-cli auth status` → 报告登录用户、Token 有效期、存储位置。

**③ 手动备选**：
```bash
gitlink-cli auth status
gitlink-cli auth login --token    # 如需演示登录流程
```

**④ 讲解要点**：
- auth Skill 与 `gitlink-shared` 分工：shared 讲认证原理，auth 讲具体命令操作
- 决策树：遇到 401 → 引导 `auth login`；403 → 查权限；CI 环境 → 用 `--token`
- 演示 `logout` 后提醒：会清凭证，需重新登录

---

## 二、onboarding 实演（核心亮点：5 维度评估）

**① 触发语**：
> "我想参与 <owner>/<repo> 这个项目，帮我找几个适合新手的任务"

**② 预期 AI 行为**：读 `onboarding/SKILL.md` →
1. `search +issues --keyword "good first issue" --category opened`（找新手 Issue）
2. `repo +info` + `repo +readme`（项目概览）
3. 对候选 Issue 做 **5 维度友好度评估**（标题/描述/定位/范围/难度）
4. 输出「推荐新手任务」清单 + 可选生成引导评论

**③ 手动备选**：
```bash
gitlink-cli search +issues --owner <owner> --repo <repo> --keyword "good first issue" --category opened
gitlink-cli repo +info --owner <owner> --repo <repo>
gitlink-cli repo +readme --owner <owner> --repo <repo>
```

**④ 讲解要点**：
- **5 维度评估表**是设计亮点：把"哪个 Issue 适合新人"从主观判断变成可量化打分（指着 AI 输出的评分讲）
- 引导评论模板：AI 能生成「欢迎贡献 + 代码定位 + 修改步骤」个性化评论（写操作，会先确认）
- 若无 good-first-issue 标签：AI 应从开放 Issue 推荐最简单的（决策规则）

---

## 三、digest 实演（亮点：跨源聚合成简报）

**① 触发语**：
> "给我一份 <owner>/<repo> 的项目简报，今天有什么动态"

**② 预期 AI 行为**：读 `digest/SKILL.md` → 并行采集 → 聚合分类 →
1. `issue +list --state open` + `pr +list`（Issue/PR 动态）
2. `ci +builds`（CI 状态）
3. `api GET "users/<me>/messages.json"`（通知）
4. 按 🔴需关注 / 🟢新增 / 🔵进行中 / 📊指标 分类，输出 Markdown 简报

**③ 手动备选**：
```bash
gitlink-cli issue +list --state open --format json
gitlink-cli pr +list --format json
gitlink-cli ci +builds --owner <owner> --repo <repo> --format json
gitlink-cli api GET "users/<me>/messages.json"
```

**④ 讲解要点**：
- **跨源聚合**是亮点：一份简报汇总 Issue/PR/CI/通知，不用挨个刷
- 与团队 `notification-digest` 分工：它做通知中心（标记已读），digest 做项目全景（不做标记已读）—— 体现去重思考
- 纯只读，安全可随时跑

---

## 四、todo 实演（亮点：补上「我的」视角）

**① 触发语**：
> "我的待办有哪些？哪些 Issue/PR 在等我处理"

**② 预期 AI 行为**：读 `todo/SKILL.md` →
1. `api GET "users/me"`（识别身份）
2. `search +issues --assignee <me> --category opened`（分配我的）
3. `api GET "users/<me>/messages.json"`（@我的）
4. `pr +list`（我的 PR 状态）
5. 按紧急度（@我 > 待 review > 指派）排序，输出待办清单

**③ 手动备选**：
```bash
gitlink-cli api GET "users/me" --format json
gitlink-cli search +issues --assignee <me> --category opened
gitlink-cli api GET "users/<me>/messages.json"
```

**④ 讲解要点**：
- **「我的」视角**是 gitlink 最缺的：跨 Issue/PR 汇总个人待办
- 紧急度排序逻辑：@我且停留 >24h → 🔴紧急；待 review 的 PR → 🟡本周
- 团队无对应 Skill，是真正的新增价值

---

## 五、验收串场词（5 分钟版）

```
开场（30s）：Skills 让 AI 能驱动 gitlink-cli。先看 snippet 实演（跑 snippet-live-demo.sh）。

转场：snippet 是本地功能。接下来演示需要平台 API 的 4 个 Skill，
      我用自然语言提问，看 AI 是否自主读 SKILL.md 编排命令。

① auth（30s）：「检查登录状态」→ AI 调 auth status。
② onboarding（1.5min）：「找新手任务」→ AI 5 维度评估出推荐清单。（重点讲评估表）
③ digest（1.5min）：「给我项目简报」→ AI 跨源聚合出报告。（重点讲与团队分工）
④ todo（1min）：「我的待办」→ AI 汇总排序。（重点讲个人视角是缺口）

收尾（30s）：5 个新增 Skill 都能被 AI 正确调用，snippet 7 命令实测通过。
```

---

## 六、安全清单（实演前确认）

- [ ] 用**测试仓库**演示，不用团队主仓库
- [ ] 写操作（onboarding 引导评论、issue close）让 AI **先确认 / --dry-run**
- [ ] 演示完 `auth logout` 的话，记得重新登录
- [ ] 只读命令（list/view/search/info/messages）可放心反复跑
