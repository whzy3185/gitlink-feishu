# gitlink-issue-triage — 使用示例

## 场景 1：自动分类打标签

**用户 Prompt：**
> 请阅读 skills/gitlink-issue-triage/SKILL.md，帮我给 ylly/gitlink-cli 里所有没标签的开放 Issue 自动分类打标签。

**Agent 执行过程：**

```bash
# Step 1：获取开放 Issue
gitlink-cli issue +list --owner ylly --repo gitlink-cli --state open --format json
# 返回 8 个 Issue，AI 客户端过滤出 tags 为空的 → #9 #10 #11 未分类

# Step 2：读取每个未分类 Issue 的详情
gitlink-cli issue +view --owner ylly --repo gitlink-cli --number 9 --format json
# subject: "登录后偶发 500 报错"  description: "刷新页面后控制台出现 crash..."
# → AI 判定：bug（关键词 500/crash + 语义"报错"）

# Step 3：查找现有标签
gitlink-cli label +list --owner ylly --repo gitlink-cli --format json
# 已有 "good first"，无 bug 标签

# Step 4：创建 bug 标签
gitlink-cli label +create --owner ylly --repo gitlink-cli \
  --name "bug" --color "#ee0701"
# → id 382700

# Step 5：打标签（issue +view 显示 #9 无原标签，直接传新 ID）
gitlink-cli issue +update --owner ylly --repo gitlink-cli \
  --number 9 --label 382700
```

**Agent 输出：**

```markdown
🏷️ Issue 分类报告 — ylly/gitlink-cli

| Issue | 标题（节选） | 分类 | 依据 | 标签 ID |
|-------|------------|:----:|------|:------:|
| #9 | 登录后偶发 500 报错 | bug | "500/crash/报错" 语义 | 382700 |
| #10 | 希望支持 webhook 自定义 header | enhancement | "希望支持" 句式 | 382701 |
| #11 | README 里 install 命令拼写错误 | documentation | "README/拼写" 语义 | 382702 |

### 📊 汇总
- 处理：3 个未分类 Issue
- bug × 1（🔴 高）｜enhancement × 1（🔵 低）｜documentation × 1（🔵 低）
```

---

## 场景 2：自动分配 + 通知

**用户 Prompt：**
> 帮我把刚分类的 #9 分配给最合适的成员，并通知他。

**Agent 执行过程：**

```bash
# Step 1：列出可分配成员
gitlink-cli issue +assigners --owner ylly --repo gitlink-cli --format json
# 返回: [{id: 148899, login: "ylly"}, {id: 149000, login: "zhangqing"}]

# Step 2：AI 推荐 → #9 是 bug，派给 zhangqing（最近修过相关模块）
#   决策依据：分类为 bug，需核心成员跟进

# Step 3a：先 GET #9 拿原 subject/description（避免 Raw PATCH 清空字段）
gitlink-cli issue +view --owner ylly --repo gitlink-cli --number 9 --format json

# Step 3b：PATCH 分配
gitlink-cli api PATCH /v1/ylly/gitlink-cli/issues/9 --body '{
  "subject": "登录后偶发 500 报错",
  "description": "刷新页面后控制台出现 crash...",
  "assigned_to_id": 149000
}'

# Step 4：验证通知到位（--owner 填被分配人 zhangqing 的 login）
gitlink-cli notification +list --owner zhangqing --format json
# 在返回中找到 source=Issue、subject 匹配 #9 的条目 → ✅ 通知已生成
```

**输出：**

```markdown
👥 责任人分配报告 — ylly/gitlink-cli

| Issue | 分类 | 责任人 | 通知状态 |
|-------|:----:|--------|:------:|
| #9 | bug | @zhangqing | ✅ 已通知 |

> GitLink 平台分配责任人时自动生成站内消息，无需手动 send。
> 网页验证：https://gitlink.org.cn/ylly/gitlink-cli/issues/9 右侧"负责人"栏。
```

---

## 场景 3：批量分拣 + 报告

**用户 Prompt：**
> 把 ylly/gitlink-cli 里所有没分类的开放 Issue 一次性处理掉，给我一份汇总。

**Agent 执行过程：**

```bash
# Step 1：批量拉取
gitlink-cli issue +list --owner ylly --repo gitlink-cli --state open --format json
# AI 客户端过滤 tags 为空 → 共 5 个未分类 Issue

# Step 2：批量预览（先生成清单给用户确认，不直接写）
#   AI 输出：
#     #9  bug            → 分配 zhangqing
#     #10 enhancement    → 分配 ylly
#     #11 documentation  → 分配 ylly
#     #12 question       → 分配 ylly（owner 处理）
#     #13 模糊 → triage  → 不分配，等人工

# Step 3：得到用户"确认"后，循环执行分类 + 分配
for n in 9 10 11 12 13; do
  gitlink-cli issue +view    --owner ylly --repo gitlink-cli --number $n --format json
  gitlink-cli label +list    --owner ylly --repo gitlink-cli --format json  # 复用已建标签
  gitlink-cli issue +update  --owner ylly --repo gitlink-cli --number $n --label <tag_id>
  gitlink-cli api PATCH /v1/ylly/gitlink-cli/issues/$n --body '{...}'
done

# Step 4：验证关键通知
gitlink-cli notification +list --owner zhangqing --format json
gitlink-cli notification +list --owner ylly      --format json
```

**Agent 输出（节选）：**

```markdown
📋 Issue 智能分拣总报告 — ylly/gitlink-cli

📅 处理时间：2026-06-16 15:30
🎯 处理范围：所有开放且未分类的 Issue

### 分类分布
| 类型 | 数量 | 占比 |
|------|:----:|:----:|
| 🔴 bug | 1 | 20% |
| 🔵 enhancement | 1 | 20% |
| 🔵 documentation | 1 | 20% |
| 🟡 question | 1 | 20% |
| ⚪ triage（待人工） | 1 | 20% |
| **合计** | **5** | **100%** |

### 责任人分配
| 责任人 | 分到 | 涉及 Issue |
|--------|:----:|-----------|
| @zhangqing | 1 | #9 |
| @ylly | 3 | #10 #11 #12 |
| 待分配 | 1 | #13（已打 triage，等 owner 复核） |

### ⚠️ 需人工跟进
- #13 描述过于模糊 → 已打 `triage`，建议 owner @ylly 复核后手动归类

### 🔗 网页验证
- Issue 列表：https://gitlink.org.cn/ylly/gitlink-cli/issues
- 标签视图：https://gitlink.org.cn/ylly/gitlink-cli/issues/tags
```

---

## 与其它 Skill 的协作

| 场景 | 推荐 Skill |
|------|-----------|
| 找适合新人的 Issue + 打 good-first | `gitlink-onboarding` |
| 给 Issue 自动分类、分配责任人 | **本 Skill（gitlink-issue-triage）** |
| 仓库文档体检、自动补全 Wiki | `gitlink-docs-assistant` |

三个 Skill 串成"**新人入门 → Issue 治理 → 文档维护**"的社区运营闭环。
