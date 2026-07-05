# 通知智能摘要 · 验证记录 — gitlink-notify-digest

**验证用户**：ylly（自查通知）
**验证日期**：2026-07-04

## 采集通知（notification +list --owner ylly）
共 5 条（含 IssueAssigned / ProjectIssue / ProjectMemberJoined / ProjectOpenDevOps 等）。

## AI 分级 + 分类
| 优先级 | 通知 | 理由 |
|:------:|------|------|
| 🔴 高 | IssueAssigned（被分配）| 需本人处理 |
| 🔴 高 | ProjectIssue（含 @/指派）| 可能 @到我 |
| 🟡 中 | ProjectMemberJoined（新成员）| 关注社区 |
| 🟢 低 | ProjectOpenDevOps（流水线）| 系统动态 |

## 📬 每日摘要输出
```markdown
## ylly 通知摘要（5 条）

### 🔴 重要（必看）
- 你被分配了 Issue（IssueAssigned）— 需处理
- ProjectIssue 动态（可能 @你）

### 🟡 关注
- 新成员加入项目

### 🟢 动态
- DevOps 流水线通知

### 建议：优先处理"被分配的 Issue"
```

## 验证结论
分级模型成功把"被分配"（最重要）置顶，让 ylly 一眼抓重点，不用逐条翻通知。解决了通知过载痛点。
