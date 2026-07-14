---
name: gitlink-docs-assistant
version: 1.0.0
description: "文档智能维护：扫描仓库检测缺失文档，AI 自动生成 CONTRIBUTING/CHANGELOG/API 文档并写入 Wiki，同步更新过时内容。当用户需要检查文档完整性或自动补全文档时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli wiki --help"
---

# gitlink-docs-assistant（文档智能维护）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有写入/删除操作前，务必先确认用户意图。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

---

## 工作流概览

| 工作流 | 操作 | AI Agent 角色 | 写入 |
|--------|------|--------------|:----:|
| 工作流 1：文档完整性体检 | 扫描仓库根目录 + Wiki 页面，生成体检报告 | 判断缺失项并评级 | 否 |
| 工作流 2：AI 自动补全文档 | 读取代码/README → 生成缺失文档 → 写入 Wiki | 生成文档内容 | 是 |
| 工作流 3：文档同步更新 | 检测 Wiki 过时内容 → AI 更新 → 提交 | 对比代码与文档差距 | 是 |

---

## 文档体检清单

| 检查项 | 标准 | 严重程度 |
|--------|------|:--------:|
| README | 存在且包含安装/使用说明 | 🔴 必须 |
| CONTRIBUTING | 贡献指南 | 🟡 重要 |
| CHANGELOG | 变更记录 | 🟡 重要 |
| API 文档 | 接口说明 | 🟡 重要 |
| LICENSE | 许可证 | 🔴 必须 |
| 代码规范文档 | 开发规范 | 🔵 可选 |

---

## 工作流 1：文档完整性体检（只读）

**触发场景：** "帮我检查一下这个仓库的文档完整性"

### Step 1：获取仓库基本信息

```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
```

### Step 2：扫描仓库根目录文件

```bash
# 获取根目录文件列表，检查 README/CONTRIBUTING/LICENSE/CHANGELOG 是否存在
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=&ref=master'
```

### Step 3：列出已有 Wiki 页面

```bash
gitlink-cli wiki +list --owner <owner> --repo <repo> --format json
```

### Step 4：输出体检报告

AI 汇总以上数据，按清单逐项判断，输出：

```markdown
## 📋 文档体检报告 — <owner>/<repo>

📅 检查时间：<YYYY-MM-DD>

### 总体评级：<🔴 需立即处理 / 🟡 待完善 / 🟢 健康>

| 检查项 | 状态 | 位置 | 建议 |
|--------|:----:|------|------|
| README | ✅ 存在 | 根目录 | — |
| CONTRIBUTING | ❌ 缺失 | — | 建议创建 Wiki 页面 |
| CHANGELOG | ❌ 缺失 | — | 建议创建 Wiki 页面 |
| API 文档 | ❌ 缺失 | — | 建议创建 Wiki 页面 |
| LICENSE | ✅ 存在 | 根目录 | — |
| 代码规范文档 | ⚠️ 未找到 | — | 可选补充 |

### 🎯 建议优先补充
1. 🔴 CONTRIBUTING —— 降低新贡献者入门门槛
2. 🟡 CHANGELOG —— 方便用户了解版本变更
3. 🟡 API 文档 —— 说明对外接口和参数
```

---

## 工作流 2：AI 自动补全文档

**触发场景：** "帮我自动生成缺失的 CONTRIBUTING 文档"

### Step 1：读取现有内容作为素材

```bash
# 读取 README（了解项目背景）
gitlink-cli api GET /:owner/:repo/readme

# 读取项目根目录结构
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=&ref=master'
```

### Step 2：AI 生成文档内容

根据仓库类型、README 内容、目录结构，生成对应文档的 Markdown 内容。

### Step 3：写入 Wiki

```bash
# 创建新 Wiki 页面（内容由 AI 生成）
gitlink-cli wiki +create \
  --owner <owner> \
  --repo <repo> \
  --name "CONTRIBUTING" \
  --content "# 贡献指南\n\n..." \
  --message "docs: AI 自动生成 CONTRIBUTING 文档"

# 同样方式创建 CHANGELOG、API 文档等
gitlink-cli wiki +create \
  --owner <owner> \
  --repo <repo> \
  --name "CHANGELOG" \
  --content "# 变更记录\n\n..." \
  --message "docs: AI 自动生成 CHANGELOG"
```

### Step 4：验证创建结果

```bash
gitlink-cli wiki +list --owner <owner> --repo <repo> --format json
gitlink-cli wiki +view --owner <owner> --repo <repo> --name "CONTRIBUTING"
```

---

## 工作流 3：文档同步更新

**触发场景：** "帮我检查 Wiki 文档是否和最新代码一致，过时的帮我更新"

### Step 1：获取近期代码变更

```bash
gitlink-cli api GET /:owner/:repo/commits --query 'page=1&limit=20'
```

### Step 2：读取相关 Wiki 页面

```bash
# 列出所有页面
gitlink-cli wiki +list --owner <owner> --repo <repo> --format json

# 读取具体页面内容
gitlink-cli wiki +view --owner <owner> --repo <repo> --name "API 文档"
```

### Step 3：AI 分析差距并生成更新内容

对比 commit 描述（尤其是 `feat:` / `fix:` 类型）与 Wiki 页面内容，找出过时部分，生成更新后的全文。

### Step 4：提交更新

```bash
gitlink-cli wiki +update \
  --owner <owner> \
  --repo <repo> \
  --name "API 文档" \
  --content "# API 文档\n\n（更新后的内容）..." \
  --message "docs: 同步更新 API 文档（关联 commit <hash>）"
```

---

## 注意事项

- `wiki +create` / `wiki +update` 是写入操作，执行前必须确认用户意图
- `wiki +delete` 在 GitLink 平台上只清空内容，不真正删除页面（已知平台限制）
- `--content` 参数直接传 Markdown 文本，CLI 内部会自动处理 base64 编码
- 在 git 仓库目录下运行时，`--owner` 和 `--repo` 可省略（自动从 remote 解析）
- `api GET /:owner/:repo/sub_entries` 端点在某些环境返回 HTML 页面而非文件列表；检测文档是否存在时，优先用更可靠的 `gitlink-cli repo +readme`（README）或 `repo +info`。
