---
name: gitlink-issue-tag
version: 2.0.0
description: "项目标记管理：查看、创建、修改、删除 GitLink 仓库的项目标记（Issue 标签）。当用户需要管理仓库的 Issue 标签/标记时触发，如添加标签、修改标签颜色、删除标签等。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli api --help"
---

# gitlink-tag（项目标记管理）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有写入/删除操作前，务必先确认用户意图。**
**CRITICAL — 项目标记通过 `gitlink-cli api` 操作，无需本地 git 命令。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)

## 功能概述

本技能覆盖 GitLink 项目标记（Issue 标签）的完整生命周期管理，包括：

1. **查看标记** — 列出仓库所有项目标记，支持关键词搜索和精简模式
2. **创建标记** — 创建新标记，设置名称、描述和颜色
3. **修改标记** — 修改已有标记的名称、描述或颜色
4. **删除标记** — 删除不再需要的标记

---

## API 能力说明

GitLink API 对项目标记（issue_tags）的完整支持：

| 操作 | HTTP 方法 | API 路径 | 说明 |
|------|-----------|---------|------|
| 查询标记列表 | GET | `/v1/{owner}/{repo}/issue_tags` | 支持 keyword/only_name/sort_by/sort_direction/limit/page 参数 |
| 创建标记 | POST | `/v1/{owner}/{repo}/issue_tags` | 请求体：{name, description, color} |
| 修改标记 | PATCH | `/v1/{owner}/{repo}/issue_tags/{id}` | 请求体：{name, description, color}，路径参数 id 为标记 ID |
| 删除标记 | DELETE | `/v1/{owner}/{repo}/issue_tags/{id}` | 路径参数 id 为标记 ID |

> **核心原则：** 所有操作均通过 `gitlink-cli api` 调用，无需本地 git 命令。

---

## 一、查看标记

### 1.1 列出仓库所有项目标记

```bash
# 获取项目标记完整列表（含描述、颜色、关联 Issue 数量等）
gitlink-cli api GET /v1/:owner/:repo/issue_tags --format json
```

**返回数据结构**：

| 字段 | 类型 | 说明 |
|------|------|------|
| total_count | integer | 标记总数 |
| issue_tags | array | 标记列表 |
| issue_tags[].id | integer | 标记 ID（修改/删除时使用） |
| issue_tags[].name | string | 标记名称 |
| issue_tags[].description | string | 标记描述 |
| issue_tags[].color | string | 标记颜色（十六进制色值，如 #F17013） |
| issue_tags[].issues_count | integer | 关联的 Issue 数量 |
| issue_tags[].pull_requests_count | integer | 关联的 PR 数量 |
| issue_tags[].user | object | 创建者信息（id/name/login/image_url） |
| issue_tags[].created_at | string | 创建时间（如 2023-02-15 11:02） |
| issue_tags[].updated_at | string | 更新时间 |

### 1.2 按关键词搜索标记

```bash
# 搜索名称中包含关键词的标记
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'keyword=bug' --format json
```

**支持的查询参数**：

| 参数 | 类型 | 必选 | 说明 |
|------|------|------|------|
| keyword | string | 否 | 搜索关键词，匹配标记名称 |
| only_name | string | 否 | 设为 `true` 时只返回名称和 ID，不返回描述等详细信息 |
| sort_by | string | 否 | 排序字段：`updated_on`（更新时间）/ `created_on`（创建时间）/ `issues_count`（Issue 数量） |
| sort_direction | string | 否 | 排序方向：`desc`（倒序）/ `asc`（正序） |

### 1.3 仅获取标记名称和 ID

```bash
# 仅返回名称和 ID（适用于选择标记、快速浏览等场景）
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'only_name=true' --format json
```

**返回示例**：

```json
{
  "total_count": 3,
  "issue_tags": [
    { "id": 1, "name": "bug" },
    { "id": 2, "name": "feature" },
    { "id": 3, "name": "documentation" }
  ]
}
```

### 1.4 按指定字段排序

```bash
# 按 Issue 数量倒序排列（找出最常用的标记）
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'order_by=issues_count&order_direction=desc' --format json

# 按创建时间正序排列（最早创建的排前面）
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'order_by=created_on&order_direction=asc' --format json

# 按更新时间倒序排列（最近更新的排前面）
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'order_by=updated_on&order_direction=desc' --format json
```

---

## 二、创建标记

### 2.1 创建单个标记

```bash
# 创建一个项目标记
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"测试11","description":"111","color":"#54ff85"}' --format json
```

**请求体参数**：

| 参数 | 类型 | 必选 | 说明 |
|------|------|------|------|
| name | string | 是 | 标记名称 |
| description | string | 是 | 标记描述 |
| color | string | 是 | 标记颜色（十六进制色值，如 #54ff85） |

**返回示例**：

```json
{
  "status": 0,
  "message": "success"
}
```

> **注意**：创建成功后 API 仅返回 status 和 message，建议立即调用查询接口确认新标记已生效。

### 2.2 标记颜色选择

AI 创建标记时，如用户未指定颜色，可按标记用途推荐默认颜色：

| 标记类型 | 推荐颜色 | 色值 | 示例用途 |
|---------|---------|------|---------|
| 🐛 缺陷 | 红色 | `#ee0701` | bug、critical、security |
| ✨ 新功能 | 蓝色 | `#0075ca` | feature、enhancement |
| 📝 文档 | 深青 | `#0075ca` | documentation、docs |
| ❓ 疑问 | 绿色 | `#008672` | question、help-wanted |
| 🎨 优化 | 紫 | `#5319e7` | refactor、performance |
| ⚠️ 待确认 | 黄色 | `#fbca04` | wontfix、invalid、duplicate |
| 🚀 发布 | 橙色 | `#f17013` | release、milestone |
| 🧪 测试 | 青 | `#54ff85` | testing、experimental |

### 2.3 创建后验证

```bash
# 创建标记后，通过关键词搜索确认
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'keyword=测试11' --format json
```

### 2.4 批量创建标记

当用户需要一次创建多个标记时，逐个调用创建 API：

```bash
# 批量创建标记（逐个调用）
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"bug","description":"Bug 修复","color":"#ee0701"}' --format json
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"feature","description":"新功能","color":"#0075ca"}' --format json
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"documentation","description":"文档相关","color":"#0075ca"}' --format json

# 验证创建结果
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'only_name=true' --format json
```

> **⚠️ 批量创建前，先查询现有标记，避免创建重复名称的标记。**

---

## 三、修改标记

### 3.1 修改标记

修改标记需要使用标记的 **ID**（不是名称）。先查询获取 ID，再调用修改接口。

```bash
# Step 1：查询标记列表，获取目标标记的 ID
gitlink-cli api GET /v1/:owner/:repo/issue_tags --format json
# 在返回结果中找到目标标记的 id 字段

# Step 2：使用 ID 修改标记
gitlink-cli api PATCH /v1/:owner/:repo/issue_tags/:id --body '{"name":"测试11","description":"1112","color":"#54ff85"}' --format json
```

**请求体参数**（与创建相同）：

| 参数 | 类型 | 必选 | 说明 |
|------|------|------|------|
| name | string | 是 | 修改后的标记名称 |
| description | string | 是 | 修改后的标记描述 |
| color | string | 是 | 修改后的标记颜色 |

**返回示例**：

```json
{
  "status": 0,
  "message": "success"
}
```

### 3.2 常见修改场景

**修改标记名称**：

```bash
# 查询获取 ID
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'keyword=旧名称' --format json
# 假设返回 id=5

# 修改名称（保持描述和颜色不变）
gitlink-cli api PATCH /v1/:owner/:repo/issue_tags/5 --body '{"name":"新名称","description":"原描述","color":"#ee0701"}' --format json
```

**修改标记颜色**：

```bash
# 查询获取 ID
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'keyword=bug' --format json
# 假设返回 id=3

# 修改颜色（保持名称和描述不变）
gitlink-cli api PATCH /v1/:owner/:repo/issue_tags/3 --body '{"name":"bug","description":"Bug 修复","color":"#ff0000"}' --format json
```

**修改标记描述**：

```bash
# 查询获取 ID
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'keyword=feature' --format json
# 假设返回 id=7

# 修改描述
gitlink-cli api PATCH /v1/:owner/:repo/issue_tags/7 --body '{"name":"feature","description":"新的功能需求描述","color":"#0075ca"}' --format json
```

### 3.3 修改后验证

```bash
# 修改后查询确认变更已生效
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'keyword=修改后的名称' --format json
```

> **⚠️ 修改标记名称后，已关联该标记的 Issue 会自动更新为新名称。**

---

## 四、删除标记

### 4.1 删除单个标记

删除标记需要使用标记的 **ID**。先查询获取 ID，再调用删除接口。

```bash
# Step 1：查询标记列表，获取目标标记的 ID
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'keyword=要删除的标记名' --format json
# 在返回结果中找到目标标记的 id 字段

# Step 2：删除标记
gitlink-cli api DELETE /v1/:owner/:repo/issue_tags/:id --format json
```

**返回示例**：

```json
{
  "status": 0,
  "message": "success"
}
```

### 4.2 完整删除流程

```bash
# Step 1：查看所有标记，确认要删除的目标
gitlink-cli api GET /v1/:owner/:repo/issue_tags --format json

# Step 2：记录目标标记的 ID 和关联 Issue 数量
# 假设目标标记 id=5, name="deprecated", issues_count=3

# Step 3：向用户确认删除意图（特别是 issues_count > 0 的标记）
# ⚠️ 删除标记后，关联的 Issue 将失去该标记

# Step 4：执行删除
gitlink-cli api DELETE /v1/:owner/:repo/issue_tags/5 --format json

# Step 5：验证删除结果
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'keyword=deprecated' --format json
# total_count 应为 0
```

### 4.3 批量删除标记

```bash
# Step 1：查询所有标记
gitlink-cli api GET /v1/:owner/:repo/issue_tags --format json

# Step 2：AI 根据用户意图筛选要删除的标记，列出 ID 列表
# 假设要删除 id=3, id=5, id=8

# Step 3：逐个删除
gitlink-cli api DELETE /v1/:owner/:repo/issue_tags/3 --format json
gitlink-cli api DELETE /v1/:owner/:repo/issue_tags/5 --format json
gitlink-cli api DELETE /v1/:owner/:repo/issue_tags/8 --format json

# Step 4：验证删除结果
gitlink-cli api GET /v1/:owner/:repo/issue_tags --format json
```

> **⚠️ 批量删除是危险操作，必须先列出待删除标记清单让用户确认后再执行。**

---

## 五、高级操作

### 5.1 项目标记健康度检查

```bash
# 获取所有标记及关联 Issue 数量
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'order_by=issues_count&order_direction=asc' --format json
```

**AI 分析规则**：

| 检查项 | 条件 | 建议 |
|--------|------|------|
| 未使用标记 | `issues_count == 0` | 考虑删除或合并 |
| 过度使用标记 | `issues_count` 为所有标记中最大值 | 考虑拆分为更细粒度的标记 |
| 标记过少 | `total_count < 3` | 建议补充常见分类标记 |
| 标记过多 | `total_count > 15` | 建议合并相似标记 |
| 无描述标记 | `description` 为空 | 建议补充描述说明 |

### 5.2 标记规范化建议

当项目缺少标准标记时，AI 可推荐创建以下基础标记集：

```bash
# 基础标记集（适用于大多数项目）
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"bug","description":"Bug 修复或问题报告","color":"#ee0701"}' --format json
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"feature","description":"新功能需求","color":"#0075ca"}' --format json
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"enhancement","description":"功能优化或改进","color":"#5319e7"}' --format json
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"documentation","description":"文档相关","color":"#0075ca"}' --format json
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"good-first-issue","description":"适合新贡献者的问题","color":"#008672"}' --format json
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"help-wanted","description":"需要帮助的问题","color":"#008672"}' --format json
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"question","description":"使用疑问","color":"#fbca04"}' --format json
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"wontfix","description":"不会处理的问题","color":"#fbca04"}' --format json
```

### 5.3 合并相似标记

当项目存在语义重复的标记时（如 "bug" 和 "defect"），AI 可建议合并：

```bash
# Step 1：查询所有标记，AI 识别相似标记对
gitlink-cli api GET /v1/:owner/:repo/issue_tags --format json

# Step 2：假设 "bug"(id=3) 和 "defect"(id=7) 需要合并，保留 "bug"

# Step 3：将 "defect" 关联的 Issue 改为 "bug"（需逐个修改 Issue 的标记）
# 先查找 "defect" 关联的 Issue 列表
gitlink-cli issue +list --state open --owner <owner> --repo <repo> --format json
# AI 筛选标记为 "defect" 的 Issue，将其改为 "bug"

# Step 4：删除 "defect" 标记
gitlink-cli api DELETE /v1/:owner/:repo/issue_tags/7 --format json

# Step 5：验证
gitlink-cli api GET /v1/:owner/:repo/issue_tags --format json
```

### 5.4 查询参数组合使用

```bash
# 搜索关键词 + 仅返回名称
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'keyword=bug&only_name=true' --format json

# 按 Issue 数量倒序 + 仅返回名称（快速查看热门标记）
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'order_by=issues_count&order_direction=desc&only_name=true' --format json

# 按更新时间倒序（查看最近活跃的标记）
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'order_by=updated_on&order_direction=desc' --format json
```

---

## 六、执行步骤总览

### 6.1 查看标记流程

```bash
# Step 1：获取项目标记完整列表
gitlink-cli api GET /v1/:owner/:repo/issue_tags --format json

# Step 2（可选）：搜索特定标记
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'keyword=bug' --format json

# Step 3（可选）：查看精简列表
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'only_name=true' --format json
```

### 6.2 创建标记流程

```bash
# Step 1：查看现有标记，避免重复
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'only_name=true' --format json

# Step 2：创建标记
gitlink-cli api POST /v1/:owner/:repo/issue_tags --body '{"name":"新标记","description":"标记描述","color":"#54ff85"}' --format json

# Step 3：验证创建结果
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'keyword=新标记' --format json
```

### 6.3 修改标记流程

```bash
# Step 1：查询目标标记，获取 ID
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'keyword=要修改的标记' --format json
# 记录目标标记的 id

# Step 2：修改标记（使用 ID）
gitlink-cli api PATCH /v1/:owner/:repo/issue_tags/:id --body '{"name":"修改后名称","description":"修改后描述","color":"#ff0000"}' --format json

# Step 3：验证修改结果
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'keyword=修改后名称' --format json
```

### 6.4 删除标记流程

```bash
# Step 1：查询目标标记，获取 ID 和关联 Issue 数量
gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'keyword=要删除的标记' --format json
# 记录目标标记的 id 和 issues_count

# Step 2：确认删除意图（⚠️ 如 issues_count > 0 需特别提醒）
# ⚠️ 删除标记后，关联的 Issue 将失去该标记

# Step 3：删除标记
gitlink-cli api DELETE /v1/:owner/:repo/issue_tags/:id --format json

# Step 4：验证删除结果
gitlink-cli api GET /v1/:owner/:repo/issue_tags --format json
```

---

## 七、操作报告模板

### 7.1 查看标记报告

```markdown
## 📋 项目标记概览

**仓库：** <owner>/<repo>
**标记总数：** 8

| ID | 名称 | 描述 | 颜色 | 关联 Issue | 创建时间 |
|----|------|------|------|-----------|---------|
| 1 | bug | Bug 修复或问题报告 | 🟡 #ee0701 | 12 | 2025-01-10 |
| 2 | feature | 新功能需求 | 🔵 #0075ca | 8 | 2025-01-10 |
| 3 | documentation | 文档相关 | 🔵 #0075ca | 3 | 2025-02-15 |
| ... | ... | ... | ... | ... | ... |

**健康度分析：**
- ⚠️ "deprecated" 标记关联 0 个 Issue，建议删除
- ✅ 标记分类覆盖完整
```

### 7.2 创建标记报告

```markdown
## ➕ 标记创建报告

**仓库：** <owner>/<repo>
**操作时间：** 2025-06-12 16:00:00

| 项目 | 详情 |
|------|------|
| 标记名称 | 测试11 |
| 标记描述 | 111 |
| 标记颜色 | 🟢 #54ff85 |
| 创建结果 | ✅ 成功 |

**验证：**
- 查询确认：✅ 标记已存在于仓库
- 重复检查：✅ 无同名标记
```

### 7.3 修改标记报告

```markdown
## ✏️ 标记修改报告

**仓库：** <owner>/<repo>
**操作时间：** 2025-06-12 16:05:00

| 项目 | 修改前 | 修改后 |
|------|--------|--------|
| 标记 ID | 5 | 5 |
| 标记名称 | 测试11 | 测试11 |
| 标记描述 | 111 | 1112 |
| 标记颜色 | 🟢 #54ff85 | 🟢 #54ff85 |
| 修改结果 | — | ✅ 成功 |

**影响范围：** 关联 Issue 3 个，已自动更新标记信息
```

### 7.4 删除标记报告

```markdown
## 🗑️ 标记删除报告

**仓库：** <owner>/<repo>
**操作时间：** 2025-06-12 16:10:00

| 项目 | 详情 |
|------|------|
| 删除标记 ID | 5 |
| 删除标记名称 | deprecated |
| 关联 Issue 数 | 0 |
| 删除结果 | ✅ 成功 |
| 删除原因 | 标记未被使用 |

⚠️ 已关联该标记的 Issue 将失去此标记。
```

---

## 八、常见场景示例

### 场景 1：为新项目创建标准标记

```
用户："帮我的新仓库创建一套 Issue 标签"

AI 执行：
1. gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'only_name=true' --format json
   → 确认当前标记列表为空
2. 逐个创建基础标记（bug/feature/enhancement/documentation/good-first-issue/help-wanted/question/wontfix）
3. 验证创建结果
4. 输出创建报告
```

### 场景 2：修改标记颜色

```
用户："把 bug 标签的颜色改成红色"

AI 执行：
1. gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'keyword=bug' --format json
   → 获取 id 和当前信息
2. gitlink-cli api PATCH /v1/:owner/:repo/issue_tags/:id --body '{"name":"bug","description":"原描述","color":"#ee0701"}' --format json
3. 验证修改结果
4. 输出修改报告
```

### 场景 3：清理未使用的标记

```
用户："删除没有关联任何 Issue 的标签"

AI 执行：
1. gitlink-cli api GET /v1/:owner/:repo/issue_tags --query 'order_by=issues_count&order_direction=asc' --format json
2. AI 筛选 issues_count == 0 的标记
3. 列出待删除标记清单，请用户确认
4. 确认后逐个删除
5. 输出删除报告
```

### 场景 4：搜索并合并重复标记

```
用户："检查有没有重复的标签"

AI 执行：
1. gitlink-cli api GET /v1/:owner/:repo/issue_tags --format json
2. AI 分析语义相似的标记对（如 bug/defect、feature/enhancement）
3. 列出建议合并的标记对，请用户确认
4. 执行合并（迁移 Issue 标记 → 删除冗余标记）
5. 输出合并报告
```

---

## 注意事项

- ✅ **修改和删除需要 ID**：`PATCH` 和 `DELETE` 接口使用标记 ID（非名称），操作前必须先查询获取 ID
- ✅ **创建前检查重复**：先查询现有标记列表，避免创建同名标记
- ✅ **删除前确认影响**：查看标记的 `issues_count`，如大于 0 需提醒用户删除后关联 Issue 会丢失该标记，待用户确认后再执行
- ✅ **请求体三个字段**：创建和修改的请求体均需包含 `name`、`description`、`color` 三个字段
- ✅ **颜色格式**：使用十六进制色值，格式为 `#RRGGBB`（如 `#ee0701`）
- ✅ **创建/修改后验证**：API 仅返回 `{status, message}`，需查询确认操作是否生效
- ⚠️ **标记名称唯一**：同一仓库下标记名称不能重复
- ✅ **排序参数**：`sort_by` 支持 `updated_on`、`created_on`、`issues_count`，`sort_direction` 支持 `desc`、`asc`
- ✅ **所有操作通过 gitlink-cli api**：无需本地 git 命令，所有增删查改均通过 API 完成

