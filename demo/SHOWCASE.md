# Skills 功能验收演示文稿

> 用法：照此 5 分钟流程演示。**Demo 1 可现场实跑**（零依赖、最稳），其余讲解设计。
> 配套：`snippet-live-demo.sh`（实演脚本）、`../Skills工作总结.md`（完整成果）

---

## 演示总览（5 分钟）

| 环节 | 时长 | 形式 | 目的 |
|------|:----:|------|------|
| 开场：Skills 是什么 | 30s | 口述 + 成果速览 | 讲清价值定位 |
| **Demo 1 · snippet 实演** | 1.5min | **跑脚本** | 证明 Skill 真能驱动 CLI |
| Demo 2 · onboarding 设计 | 1.5min | 打开 SKILL.md 讲 | 展示 AI 工作流设计深度 |
| Demo 3 · digest/todo 体验优化 | 1min | 讲设计 + 分工 | 展示体验优化与去重思考 |
| 收尾：成果 + 验证 | 30s | 数字 | 强化贡献 |

---

## 开场（30 秒）

> 一句话：**Skills 是写给 AI 的「菜谱」**——告诉 AI「什么场景、按什么顺序、调哪些 gitlink-cli 命令」。我们把 gitlink-cli 从「开发者工具」升级为「AI 可驱动的平台」。
>
> 本次新增 **5 个 Skill** + 补全 **28 个 examples** + snippet **7 命令端到端实测通过**。

---

## Demo 1 · snippet 现场实演（核心，必演）

```bash
bash demo/snippet-live-demo.sh
```

**脚本会演示的闭环**（每个场景都展示「🧑用户提问 → 🤖AI 读 SKILL.md 决策 → 执行命令 → 输出」）：

| 场景 | 命令 | SKILL.md 规则 |
|------|------|--------------|
| 保存代码 | `snippet +create` | --title 必填、--tags 逗号分隔 |
| 浏览 | `snippet +list` | 可按 tag/language 过滤 |
| 检索 | `snippet +search` | 全文匹配 |
| 详情 | `snippet +view` | 按 id |
| 导出 | `snippet +export` | -o 写文件 |
| 更新 | `snippet +update` | 至少一个字段 |
| 删除 | `snippet +delete` | 不可逆，先确认 |

**讲解要点**：注意每个场景 AI 都先「读 SKILL.md 决策」再执行——这就是 Skills 的核心价值，**AI 不是瞎调命令，而是按菜谱编排**。输出严格符合 `{"ok":true,"data":{...}}` 格式。

---

## Demo 2 · onboarding 设计深度（展示 B 类工作流）

**操作**：打开 `gitlink-cli/gitlink-cli/skills/gitlink-onboarding/SKILL.md`

**重点讲三处**（评分重点）：

1. **5 维度友好度评估表**（决策规则章节）——把「哪个 Issue 适合新人」从主观判断变成可量化打分：标题清晰度 / 描述完整度 / 代码定位 / 改动范围 / 难度标签。
2. **4 个工作流**——项目概览 → 找任务 → 生成引导评论 → 贡献全流程（Fork→Branch→PR）。
3. **引导评论输出模板**——AI 能自动生成「欢迎贡献 + 代码定位 + 修改步骤」的个性化评论。

> 一句话：A 类（命令包装）做不到「智能推荐 + 生成评论」，所以选 B 类（AI 工作流）。

---

## Demo 3 · digest / todo 体验优化（展示第二批 + 去重思考）

| Skill | 解决的痛点 | 与团队已有 Skill 的关系 |
|-------|-----------|----------------------|
| `gitlink-digest` | 信息太分散，看动态要挨个刷 | 与团队 `notification-digest` **分工**：它做通知中心，我做项目全景日报（Issue+PR+CI+活跃度） |
| `gitlink-todo` | 没有「我的」视角，不知哪些在等我 | 团队**无对应**，真缺口 |

**去重思考（加分点）**：曾设计「僵尸唤醒 stale」，核查发现团队已有完整的 `gitlink-stale-issue-manager`（563 行），为避免重复造已删除——**体现对项目整体的理解和工程素养**。

---

## 收尾：成果 + 验证（30 秒）

| 指标 | 数据 |
|------|------|
| 新增 Skill | **5 个**（onboarding / auth / snippet / digest / todo） |
| 补充 examples | **28 个**（23 个补已有 Skill + 5 个新增自带） |
| 端到端实测 | snippet 全 7 命令通过 |
| 命令可调用性 | 新增 Skill 全用已注册命令域，可真实调用 |

> 演示结束。完整设计详见 `Skills工作总结.md`。

---

## 答辩 Q&A 预备

| 可能的提问 | 回答要点 |
|-----------|---------|
| 工作边界？ | 新增 5 个 Skill + 补 23 个 examples；团队原有 42 个（见总结第二节） |
| 怎么证明 Skill 真能用？ | 刚跑的 snippet 7 命令闭环；其余 4 个登录后可按 SKILL.md 工作流验证 |
| 为什么 onboarding 选 B 类？ | 需智能推荐 + 生成评论，A 类命令包装做不到 |
| digest 和团队 notification-digest 重复吗？ | 不重复，分工明确：通知中心 vs 项目全景日报 |
| Skill 遵循什么规范？ | 项目模板：YAML frontmatter + CRITICAL 三连 + 引用 gitlink-shared |
