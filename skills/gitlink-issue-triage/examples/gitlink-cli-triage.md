# gitlink-issue-triage 使用示例

> 触发方式：自然语言，Skill 自动匹配
> 日期：2026-05-28
> 验证状态：✅ 通过（自动触发 + 全部命令执行成功 + 分类规则正确应用）

---

## 用户输入

```
帮我把 Gitlink/gitlink-cli 的 Issue 整理分类一下，看看哪些需要优先处理
```

## Skill 触发

Claude Code 自动匹配到 `gitlink-issue-triage` skill（关键词匹配："Issue" + "分类" + "优先处理"），同时自动加载前置依赖 `gitlink-shared`。

---

## 执行过程摘要

### Step 1：项目概览

```bash
gitlink-cli repo +info --owner Gitlink --repo gitlink-cli --format json
```

获取到：17 个 Issue，12 位贡献者，master 分支。

### Step 2：扫描开放 Issue

```bash
gitlink-cli issue +list --owner Gitlink --repo gitlink-cli --state open --format json
```

返回 17 条，按 `status_id` 过滤后保留 7 条开放（含 2 条 status_id=0 状态异常）。

### Step 3：逐条分析

对全部 7 条 Issue 执行了 `issue +view --number <N>`，按 4 个维度分类：

| # | 标题 | type | urgency | complexity | action |
|---|------|------|---------|------------|--------|
| 18 | issue +update后状态框变红色 | bug | high | medium | fix-now |
| 5 | skill认领任务时描述信息消失 | bug | high | medium | fix-now |
| 14 | pr +view 缺少PR关闭时间 | bug | normal | easy | investigate |
| 15 | issue +view 返回数据与网页不一致 | bug | normal | easy | investigate |
| 17 | API是否支持读取仓库内文件 | question | normal | medium | discuss |
| 6 | 创建Issue就让Agent开始干活 | feature | normal | hard | discuss |
| 2 | 使用讨论与反馈收集 | meta | low | easy | discuss |

### 验证要点

| 修复项 | Agent 测试时的问题 | 本次表现 |
|--------|-------------------|----------|
| status_id=0 处理 | 规则矛盾 | ✅ 纳入分析 + 标注"状态异常" |
| #2 元讨论分类 | 无法归类 | ✅ 正确归为 meta |
| #6 纯图片描述 | 未覆盖 | ✅ 标注"描述缺失（仅含图片附件）"，复杂度=hard，建议=discuss |
| 空段占位文本 | 缺失 | ✅ "无长期不活跃的 Issue" |
| 备注列 | 缺失 | ✅ #15 和 #2 正确标注了状态异常 |
| suggested_this_week | 无公式 | ✅ 4 = 建议近期处理段 Issue 数 |

---

## 完整输出报告

```markdown
# 📊 Gitlink/gitlink-cli Issue 分拣报告

> 分析时间：2026-05-28
> Issue 总数：17，开放：7，本次分析：7 条（含 2 条状态异常）

## 总览

| 指标 | 数量 |
|------|------|
| Bug | 4 |
| 功能请求 | 1 |
| 咨询 | 1 |
| 元讨论 | 1 |
| **需立即处理** | 2 |
| **适合入门** | 2 |

## 🔴 需立即处理

| # | 标题 | 类型 | 紧急度 | 复杂度 | 建议 | 备注 |
|---|------|------|--------|--------|------|------|
| #18 | issue +update后issue状态框变红色 | bug | high | medium | fix-now | +update 会重置 tracker、tags、priority 字段，导致界面异常红框。根因已明确 |
| #5 | skill认领任务时描述信息消失 | bug | high | medium | fix-now | Issue 更新流程中 description 丢失，数据完整性严重问题 |

## 🟡 建议近期处理

| # | 标题 | 类型 | 紧急度 | 复杂度 | 建议 | 备注 |
|---|------|------|--------|--------|------|------|
| #14 | pr +view 的返回中缺少PR关闭时间 | bug | normal | easy | investigate | 描述详尽，含完整复现数据 |
| #15 | issue +view 返回的数据与网页显示不一致 | bug | normal | easy | investigate | --id 全局ID vs 序号混淆。status_id=0 状态异常 |

## 🟢 可延迟 / 需讨论

| # | 标题 | 类型 | 紧急度 | 复杂度 | 建议 | 备注 |
|---|------|------|--------|--------|------|------|
| #17 | API是否支持自动读取仓库内文件？ | question | normal | medium | discuss | 需确认是否新增文件读取能力 |
| #6 | 只要创建Iusse就让Agent开始干活 | feature | normal | hard | discuss | 描述缺失（仅含图片附件），需补充文字说明 |

## ⭐ 适合入门（Good First Issue）

| # | 标题 | 类型 | 复杂度 | 推荐理由 |
|---|------|------|--------|----------|
| #14 | pr +view 的返回中缺少PR关闭时间 | bug | easy | 描述完整、范围明确、单点问题 |
| #15 | issue +view 返回的数据与网页显示不一致 | bug | easy | 根因已分析完成，涉及参数UX改进和文档 |

## ⚠️ 候选关闭（90+ 天无活动）

无长期不活跃的 Issue。

## 📋 维护建议

1. 立即行动：2 个 bug 涉及数据丢失/字段重置（#18、#5），应最高优先级修复
2. 本周目标：建议处理 4 个 Issue（2 urgent + 2 easy）
3. 社区引导：#14 和 #15 适合标记为 good first issue
4. 标签体系缺失：建议建立标签体系（bug/feature/docs/question/good-first-issue/help-wanted）
5. 状态异常：2 个 Issue（#2、#15）status_id=0，需手动确认

## 优先级排序总结

| 优先级 | Issue | 核心问题 |
|--------|-------|----------|
| P0 | #18, #5 | 数据丢失/字段重置 — 涉及用户数据完整性 |
| P1 | #14, #15 | API 返回字段缺失 / UX 混淆 — 影响数据分析场景 |
| P2 | #17, #6 | 功能咨询 / 需求不明 — 需先澄清再决定 |
| 长期 | #2 | 反馈收集帖 — 作为长期讨论入口保留 |
```

---

## 验证结论

| 检查项 | 状态 |
|------|------|
| Skill 自动触发（无需指定路径） | ✅ |
| 自动加载 gitlink-shared 前置依赖 | ✅ |
| 状态过滤：排除 closed，保留 status_id=0 并标注异常 | ✅ |
| 4 维分类（type/urgency/complexity/action）正确应用 | ✅ |
| #2 元讨论帖正确归为 meta | ✅ |
| #6 纯图片描述正确标注"描述缺失" | ✅ |
| 空段输出占位文本 | ✅ |
| 备注列标注状态异常 | ✅ |
| P0/P1/P2 优先级排序 | ✅ |
| 标签体系缺失建议 | ✅ |
