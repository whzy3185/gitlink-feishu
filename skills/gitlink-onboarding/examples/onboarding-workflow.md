# gitlink-onboarding — 使用示例

## 场景 1：good-first-issue 自动标记

**用户 Prompt：**
> 请阅读 skills/gitlink-onboarding/SKILL.md，帮我找出 ylly/gitlink-cli 中适合新人的 Issue 并打上标签。

**Agent 执行过程：**

```bash
# Step 1：获取开放 Issue
gitlink-cli issue +list --owner ylly --repo gitlink-cli --format json
# 返回 7 个 Issue，AI 按新人友好标准筛选 → #8 适合新人

# Step 2：查找现有标签
gitlink-cli label +list --owner ylly --repo gitlink-cli --format json
# 无 good-first-issue 标签

# Step 3：创建标签
gitlink-cli label +create --owner ylly --repo gitlink-cli \
  --name "good first" --color "#7057ff"
# → id 382660（注：GitLink 标签限 15 字符，"good first issue" 会被截断，故用 "good first"）

# Step 4：打标签
gitlink-cli issue +update --owner ylly --repo gitlink-cli \
  --number 8 --label 382660
```

**Agent 输出：**

```markdown
🌱 新人友好 Issue 标记报告 — ylly/gitlink-cli

| Issue | 标题 | 适合新人理由 | 标签 |
|-------|------|------------|:----:|
| #8 | docs: 补充 wiki 命令的使用示例文档 | 文档类、范围明确 | ✅ |

共标记 1 个 good-first-issue。
```

---

## 场景 2：引导评论生成

**用户 Prompt：**
> 帮我给 #8 写一个引导评论，让新人知道怎么入手。

**Agent 执行过程：**

```bash
# 读取 Issue 详情
gitlink-cli issue +view --owner ylly --repo gitlink-cli --number 8 --format json

# AI 根据 Issue 内容生成个性化评论并发布
gitlink-cli issue +comment --owner ylly --repo gitlink-cli \
  --number 8 \
  --body "👋 欢迎贡献！任务目标：为 wiki 命令补充示例。
建议入手位置：README.md、shortcuts/wiki/wiki.go
本地准备：Fork → 克隆 → go build → 建分支 docs/wiki-examples
提交 PR 关联 #8。"
```

**输出：** 评论已发布（comment id 476548）

---

## 场景 3：项目入门指南

**用户 Prompt：**
> 帮我生成一份 gitlink-cli 的新人入门指南。

**Agent 执行过程：**

```bash
gitlink-cli repo +info --owner ylly --repo gitlink-cli --format json
gitlink-cli repo +readme --owner ylly --repo gitlink-cli
```

**Agent 输出入门指南 Markdown：**

```markdown
🚀 gitlink-cli 新人入门指南

## 环境准备
- Go 1.26+，GitLink 账号 + auth login

## 项目结构
| 目录 | 作用 |
|------|------|
| cmd/ | 命令定义 |
| shortcuts/ | Shortcut 实现（核心）|
| skills/ | AI Agent Skills |

## 第一个贡献
1. 找带 good-first-issue 标签的 Issue
2. Fork + 克隆 + 建分支
3. go build && go test ./...
4. 提交 PR 关联 Issue

## 提交规范
- feat / fix / docs / refactor
- 新增模块需在 shortcuts/register.go 注册
```
