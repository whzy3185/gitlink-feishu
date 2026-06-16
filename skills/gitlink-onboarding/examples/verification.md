# Claude Code 验证记录 — gitlink-onboarding

**验证日期：** 2026-06-16
**验证平台：** Claude Code
**验证仓库：** ylly/gitlink-cli
**验证人：** ylly
**gitlink-cli 版本：** 本地源码构建（go1.26.3, windows/amd64）

---

## 0. 环境确认

```bash
$ gitlink-cli auth status
✓ Logged in as ylly

$ gitlink-cli user +me
{ "ok": true, "data": { "login": "ylly", "user_id": 148899 } }
```

📷 环境截图见 `screenshots/00-环境确认.png`

---

## 1. 验证工作流 1：good-first-issue 自动标记

**喂入 Skill：** 在 Claude Code 中输入
> 请阅读 skills/gitlink-onboarding/SKILL.md，帮我找出 ylly/gitlink-cli 中适合新人的 Issue 并打上标签。

**Agent 执行过程：**

| 步骤 | 命令 | 结果 |
|------|------|:----:|
| 获取开放 Issue | `gitlink-cli issue +list --owner ylly --repo gitlink-cli --format json` | ✅ 返回 7 个 Issue |
| 查找标签 | `gitlink-cli label +list --owner ylly --repo gitlink-cli --format json` | ✅ 现有 10 个标签，无 good-first-issue |
| 创建标签 | `gitlink-cli label +create --name "good first issue" --color "#7057ff"` | ✅ 创建成功（id 382660，名字被截断） |
| 改名修正 | `gitlink-cli label +update --id 382660 --name "good first"` | ✅ 修正为 "good first" |
| 打标签 | `gitlink-cli issue +update --number 8 --label 382660` | ✅ #8 已标记 |

**为演示创建了一个真实的"新人友好"Issue（#8）：**
```
#8 docs: 补充 wiki 命令的使用示例文档
   （文档类、范围明确、不涉及代码逻辑 → 适合新人）
```

**验证 #8 已带标签（issue +view 返回）：**
```json
"tags": [{ "color": "#7057ff", "id": 382660, "name": "good first" }]
```

**真实发现（验证过程中记录，并已修正）：**
- GitLink 标签名长度限制 **15 个字符**，"good first issue"（16 字符）创建时被平台自动截断为 "good first issu"。**已通过 `label +update --id 382660 --name "good first"` 修正为 "good first"**。SKILL.md 注意事项已据此提示：标签名建议 ≤15 字符。
- Issue 的标签字段名是 `tags`（注意：label +list 用的是 `issue_tags`，issue 里是 `tags`，两者不同）。

📷 标记报告截图见 `screenshots/01-标记报告.png`

---

## 2. 验证工作流 2：引导评论生成

**Agent 执行命令（AI 根据 #8 内容生成个性化评论）：**

```bash
$ gitlink-cli issue +comment --owner ylly --repo gitlink-cli \
    --number 8 \
    --body "👋 欢迎贡献！这是一个适合新人入手的任务。\n**任务目标：** 为 wiki 命令补充使用示例...\n**建议入手位置：** README.md ...\n**本地准备：** ...\n**提交 PR：** 关联本 Issue (#8)。"
```

**真实返回：**
```json
{ "ok": true, "data": { "id": 476548, "author": { "id": 148899 } } }
```

**评论特点（证明是个性化生成，非固定模板）：**
- 针对 #8 的"wiki 命令文档"主题，定位到 `README.md` 和 `shortcuts/wiki/wiki.go`
- 提示了 wiki 命令使用独立网关 API 的特殊点
- 给出具体的分支命名 `docs/wiki-examples`

📷 评论发布截图见 `screenshots/02-引导评论.png`
（也可访问 https://gitlink.org.cn/ylly/gitlink-cli/issues/8 查看真实评论）

---

## 3. 验证工作流 3：项目入门指南

**Agent 执行命令（只读采集）：**

```bash
$ gitlink-cli repo +info --owner ylly --repo gitlink-cli --format json
# 返回：Go 项目，默认分支 master，License MulanPSL-2.0，8 个 Issue

$ gitlink-cli repo +readme --owner ylly --repo gitlink-cli
# 返回 README 全文，提取项目结构、命令列表
```

**AI 生成的入门指南（节选）：**

```markdown
🚀 gitlink-cli 新人入门指南

## 环境准备
- Go 1.26+，GitLink 账号 + auth login

## 项目结构
| 目录 | 作用 |
| cmd/ | 命令定义 |
| shortcuts/ | Shortcut 命令实现（核心）|
| skills/ | AI Agent Skills 文档 |

## 第一个贡献
1. 找带「good first」标签的 Issue
2. Fork + 克隆 + 建分支
3. go build && go test ./... 验证
4. 提交 PR 关联 Issue
```

📷 入门指南截图见 `screenshots/03-入门指南.png`

---

## 验证结论

| 工作流 | 结果 | 关键证据 |
|--------|:----:|---------|
| 工作流 1：good-first-issue 标记 | ✅ | #8.tags 含 good first（id 382660） |
| 工作流 2：引导评论 | ✅ | comment id 476548 已发布 |
| 工作流 3：入门指南 | ✅ | 基于 repo +info/readme 生成 |

- **Agent 平台：** Claude Code
- **真实仓库验证：** ylly/gitlink-cli（标签 + 评论已真实写入，可在网页查看）
- **兼容性：** 标准 YAML frontmatter，兼容 Claude Code / Cursor / OpenClaw

---

## 截图清单

| 文件名 | 对应步骤 | 内容 |
|--------|---------|------|
| `screenshots/00-环境确认.png` | 第 0 步 | auth status 登录成功 |
| `screenshots/01-标记报告.png` | 工作流 1 | good-first-issue 标记报告表 |
| `screenshots/02-引导评论.png` | 工作流 2 | issue +comment 返回 ok:true + id:476548 |
| `screenshots/03-入门指南.png` | 工作流 3 | 生成的入门指南 |
| `screenshots/04-issue网页实测.png`（可选） | 综合效果 | GitLink 网页 #8 显示标签+评论 |
