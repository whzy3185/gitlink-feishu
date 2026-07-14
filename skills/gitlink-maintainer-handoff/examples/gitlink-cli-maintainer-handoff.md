# gitlink-maintainer-handoff 使用示例

> 触发方式：自然语言自动匹配
> 日期：2026-06-11
> Agent 平台：Codex
> 验证状态：✅ 通过（命令全部真实执行，生成了可复用交接摘要）

---

## 用户输入

```text
帮我给 Gitlink/gitlink-cli 生成一份今天的维护者交接摘要，我晚点要把仓库交给别人继续跟。
```

## Skill 触发

Agent 识别到关键词“维护者”“交接摘要”“继续跟进”，自动匹配 `gitlink-maintainer-handoff`，并先读取 `gitlink-shared` 的认证和只读规则。

---

## 执行过程摘要

### Step 1：确认当前账号

```bash
gitlink-cli user +me --format json
```

结果：当前账号为 `Mengz`。

### Step 2：检查站内消息积压

```bash
gitlink-cli api GET "users/Mengz/messages.json" --query "status=1&limit=20" --format json
```

结果：未读消息为 `0`，说明当前没有站内消息积压。

### Step 3：获取仓库治理总览

```bash
gitlink-cli workflow +repo-report --owner Gitlink --repo gitlink-cli --lang zh-CN --format markdown
```

关键结果：

- 仓库综合分 `49`
- 健康分 `58`
- 风险等级 `high`
- 建议优先审查长期未处理 PR，并补齐 README / CONTRIBUTING / LICENSE 相关治理项

### Step 4：提取开放 PR 焦点

```bash
gitlink-cli pr +list --owner Gitlink --repo gitlink-cli --state open --limit 10 --format json
```

抽取出的近期 PR 样本：

| PR | 标题 | 创建时间 |
|----|------|----------|
| #209 | feat(milestone): 增加里程碑进度分析快捷命令 | 2026-06-11 02:00 |
| #208 | feat(user): add pinned project shortcuts | 2026-06-10 13:06 |
| #207 | feat(user): add statistics shortcuts | 2026-06-10 13:06 |
| #206 | feat(commit): add commit inspection shortcuts | 2026-06-10 13:05 |
| #205 | fix(issue): 修复详情缺失并保护更新元数据 | 2026-06-10 11:32 |

对最新的 PR 再补充一份审阅摘要：

```bash
gitlink-cli workflow +pr-summary --owner Gitlink --repo gitlink-cli --number 209 --lang zh-CN --format markdown
```

结果：规则引擎将 `#209` 标为高风险关注项，建议重点复查命令兼容性、文档一致性和测试覆盖。

### Step 5：提取开放 Issue 焦点

```bash
gitlink-cli issue +list --owner Gitlink --repo gitlink-cli --state open --limit 10 --format json
```

关键结果：

- 当前开放 Issue 共 `8` 条
- 最近更新的条目包括 `#6`、`#19`
- `#15` 和 `#2` 的 `status_id=0`，属于需要人工确认的异常状态 Issue

### Step 6：查看最近发布

```bash
gitlink-cli release +list --owner Gitlink --repo gitlink-cli --limit 3 --format json
```

关键结果：

- 最近一次发布为 `v0.2.0`
- 发布时间 `2026-06-09 01:23`
- 目标分支 `master`
- 发布资产数量 `6`

---

## 生成的交接摘要

```markdown
# Gitlink/gitlink-cli 维护者交接摘要

> 生成时间：2026-06-11
> 接手账号：Mengz

## 一、当前状态

- 站内消息：当前无未读消息积压，可以直接从仓库治理数据开始接手
- 仓库风险：high，健康分 58，综合分 49
- 最近发布：v0.2.0 已于 2026-06-09 发布到 master，附件 6 个，发布资产完整

## 二、优先跟进的 PR

| PR | 标题 | 建议动作 |
|----|------|----------|
| #209 | feat(milestone): 增加里程碑进度分析快捷命令 | 先复查分页边界和 README 示例是否一致 |
| #208 | feat(user): add pinned project shortcuts | 确认批量更新和排序的 dry-run 语义 |
| #205 | fix(issue): 修复详情缺失并保护更新元数据 | 重点看 edit 元数据保护逻辑是否覆盖失败路径 |

## 三、优先跟进的 Issue

| Issue | 标题 | 风险提示 | 建议动作 |
|-------|------|----------|----------|
| #19 | windows环境下部分Agent执行gitlink-cli命令失败 | 新近提交，可能影响 Agent 赛道可用性 | 尽快复现并确认影响范围 |
| #18 | giklink-cli issue +update后issue状态框变红色 | 影响 Issue 更新可靠性 | 核对修复分支和回归测试 |
| #15 | issue +view 返回的数据与网页显示不一致 | status_id=0，状态异常 | 人工确认服务端状态，再决定是否继续跟进 |

## 四、下一位维护者第一小时建议

1. 先看 `workflow +repo-report` 里提到的高风险项，把长期未处理 PR 列出跟进顺序。
2. 然后处理 `#19` 和 `#18` 这类直接影响 CLI 使用体验的问题。
3. 最后回看最近发布后的资产和文档项，确认后续 PR 是否补齐治理缺口。
```

---

## 验证结论

| 检查项 | 结果 |
|--------|------|
| Skill 自动触发 | ✅ |
| 能读取当前登录账号 | ✅ |
| 能处理“无未读消息”场景 | ✅ |
| 能汇总仓库健康度和风险等级 | ✅ |
| 能提取近期 PR / Issue 焦点 | ✅ |
| 能补充最近发布状态 | ✅ |
| 交接摘要可直接交给下一位维护者使用 | ✅ |
