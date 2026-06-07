# gitlink-onboarding 使用示例

> 触发方式：自然语言，无需指定 SKILL.md 路径
> 日期：2026-05-28
> 验证状态：✅ 通过（自动触发 + 全部命令执行成功）

---

## 用户输入

```
我是新手，想给 gitlink-cli 贡献代码，调用相关skill指导我
```

## Skill 触发

Claude Code 自动匹配到 `gitlink-onboarding` skill（关键词匹配："新手" + "贡献代码"），同时自动加载前置依赖 `gitlink-shared`。

---

## 执行过程摘要

### Step 1：定位仓库

当前目录不是 git 仓库，Agent 自动尝试了 `gitlink-cli/gitlink-cli`（404）后切换到 `Gitlink/gitlink-cli`（成功）。

```bash
gitlink-cli repo +info --owner Gitlink --repo gitlink-cli --format json
```

获取到：Go 项目，12 位贡献者，master 分支，17 个 Issue，83 个 PR。

### Step 2：扫描开放 Issue

```bash
gitlink-cli issue +list --owner Gitlink --repo gitlink-cli --state open --format json
```

返回 17 条，Agent 识别到 `--state open` 过滤不准确，客户端按 `status_id` 二次过滤，保留 7 条真正开放的。

### Step 3：深入查看候选

对 5 个高潜力候选执行了 `issue +view --number <N>`：

| Issue | 标题 | 判断 |
|---|---|---|
| #18 | `issue +update` 后状态框变红色 | ⭐ 最推荐 — Bug 已定位 |
| #14 | `pr +view` 缺少 PR 关闭时间 | ⭐ 最推荐 — 需求清晰 |
| #5 | skill 认领任务时描述信息消失 | ⭐ 最推荐 — 有复现步骤 |
| #15 | `issue +view` 返回数据与网页不一致 | 推荐 — status_id=0 需确认 |
| #17 | API 是否支持读取仓库内文件 | 可尝试 — 偏向咨询 |

### Step 4：生成报告

按模板输出完整引导报告（项目概览 + 分级推荐 + 贡献流程 6 步 + 注意事项）。

---

## 完整输出报告

```markdown
# Gitlink/gitlink-cli 新人引导报告

## 项目概览

| 项目 | 信息 |
|------|------|
| 仓库 | Gitlink/gitlink-cli |
| 描述 | GitLink CLI - GitLink 平台命令行工具 |
| 语言 | Go |
| 主分支 | master |
| 贡献者数 | 12 |
| Issue 总数 | 17（7 个开放） |
| PR 总数 | 83 |
| 项目规模 | 2.5 MB |

## 筛选说明

仓库未使用 Issue 标签系统（所有 Issue 均无标签），筛选主要依赖标题关键词、状态和分配情况。
从 17 个 Issue 中，按 status_id 过滤掉已关闭的（status_id=3/5），保留 7 个候选，
再根据可操作性排序。

## 推荐的入门 Issue

### 最推荐（Bug 修复，范围明确）

| # | 标题 | 优先级 | 推荐理由 |
|---|------|--------|----------|
| #18 | gitlink-cli issue +update后issue状态框变红色 | 正常 | Bug 修复，根因已定位：+update 会重置 tracker/issue_tags/priority 字段。修复范围明确，适合入门 |
| #14 | pr +view 的返回中缺少PR关闭时间 | 正常 | 功能增强，需求清晰：在 PR 返回中补充 merged_at/closed_at 字段。有完整的 API 响应示例和文档引用 |
| #5 | skill认领任务时描述信息消失 | 正常 | Bug 修复，有明确复现步骤。可能与 #18 同源（Issue 更新导致字段丢失） |

### 推荐（适合新手，建议先评论确认）

| # | 标题 | 优先级 | 推荐理由 |
|---|------|--------|----------|
| #15 | issue +view 返回的数据与网页显示不一致 | 正常 | 状态未知（status_id=0），根因已分析：--id 是全局 ID 而非项目内序号。建议评论确认后再动手 |

### 可尝试（范围需要先讨论）

| # | 标题 | 优先级 | 推荐理由 |
|---|------|--------|----------|
| #6 | 只要创建Issue就让Agent开始干活 | 正常 | 功能需求，标题比较概括。建议先在 Issue 下评论讨论具体实现方案 |
| #17 | API是否支持自动读取仓库内文件？ | 正常 | 目前更偏向使用咨询，可能转为功能需求。可以先参与讨论 |

## 贡献流程

### 第 1 步：Fork 仓库
打开 https://www.gitlink.org.cn/Gitlink/gitlink-cli ，点击右上角 Fork 按钮。

### 第 2 步：Clone 到本地
git clone https://www.gitlink.org.cn/<你的用户名>/gitlink-cli.git
cd gitlink-cli
git remote add upstream https://www.gitlink.org.cn/Gitlink/gitlink-cli.git

### 第 3 步：创建分支
git checkout -b fix/issue-18-update-reset-fields

### 第 4 步：修改 + 提交
go build ./...
go test ./...
git add -A
git commit -m "fix: 修复 issue +update 重置 tracker/tags/priority 字段 (#18)"

### 第 5 步：Push 并提 PR
git push origin fix/issue-18-update-reset-fields
gitlink-cli pr +create \
  --owner Gitlink \
  --repo gitlink-cli \
  --head <你的用户名>:fix/issue-18-update-reset-fields \
  --base master \
  --title "fix: 修复 issue +update 重置字段问题 (#18)"

### 第 6 步：在 Issue 下留言
在目标 Issue 页面评论 "我来处理这个"，避免与他人重复劳动。

## 注意事项

- 先评论再动手 — 在 Issue 下留言认领，避免重复劳动
- 保持 PR 小 — 一个 PR 只解决一个问题
- 不确定就问 — 对需求有疑问，在 Issue 下直接评论提问
- Fork 流程 — 没有主仓库写权限，必须走 Fork → 分支 → PR
- 该仓库无标签系统 — 筛选主要靠标题和描述判断
```

---

## 验证结论

| 检查项 | 状态 |
|---|---|
| Skill 自动触发（无需指定路径） | ✅ |
| 自动加载 gitlink-shared 前置依赖 | ✅ |
| 仓库未自动检测时主动尝试多个 owner | ✅ |
| `repo +info` 正确提取项目信息 | ✅ |
| `issue +list` 识别到状态过滤 bug 并手动过滤 | ✅ |
| `issue +view` 使用 `--number`（正确参数） | ✅ |
| 输出报告含分级推荐 + 贡献流程 + 注意事项 | ✅ |
| 对 status_id=0 异常状态做了标注 | ✅ |
