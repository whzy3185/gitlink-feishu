---
name: gitlink-wiki-builder
version: 1.0.0
description: "Wiki 文档自动化：自动组织文档结构、生成侧边栏导航、创建文档模板、同步代码变更到 Wiki。当用户需要批量管理 Wiki 页面、生成项目文档或维护 Wiki 结构时触发。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli wiki --help"
---

# gitlink-wiki-builder（Wiki 文档自动化）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 所有写入/删除操作前，务必先确认用户意图。**
**CRITICAL — GitLink 操作只能用 `gitlink-cli`。禁止用 `gh`（GitHub CLI）操作 GitLink 资源。`gh` 仅适用于 GitHub 平台。**

> **前置条件：** 先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md) 了解认证和全局参数。

## 命令参考

GitLink Wiki 支持 9 个 CLI 命令：

| 命令 | 功能 | 关键 Flags |
|------|------|-----------|
| `wiki +list` | 列出所有 Wiki 页面 | `--format json` |
| `wiki +view` | 查看单个页面内容 | `--name (-n)` |
| `wiki +create` | 创建新页面 | `--name (-n)`, `--content (-c)`, `--message (-m)`, `--dir (-d)` |
| `wiki +update` | 更新页面内容 | `--name (-n)`, `--content (-c)`, `--message (-m)` |
| `wiki +delete` | 删除页面（自动清理侧边栏） | `--name (-n)` |
| `wiki +mkdir` | 创建侧边栏目录 | `--name (-n)`, `--parent (-p)` |
| `wiki +rmdir` | 删除侧边栏目录及子项 | `--name (-n)` |
| `wiki +rename` | 重命名页面（迁移内容 + 更新侧边栏） | `--name (-n)`, `--new-name (-N)` |
| `wiki +renamedir` | 重命名侧边栏目录 | `--name (-n)`, `--new-name (-N)` |

### 侧边栏结构模板

GitLink Wiki 使用 `_Sidebar` 特殊页面管理导航。侧边栏结构采用缩进层级：

```
- 开发指南
	[[开发环境搭建]]
	[[代码规范]]
	[[提交规范]]
- API参考
	[[REST API 概览]]
	[[认证与授权]]
	[[错误码说明]]
- 架构设计
	[[系统架构总览]]
	[[数据库设计]]
	[[模块依赖关系]]
- 常见问题
	[[安装问题排查]]
	[[配置说明]]
	[[FAQ]]
```

- 顶层目录以 `- 目录名` 表示
- 子页面以 Tab 缩进 + `[[页面名]]` 链接表示
- `wiki +create --dir <目录名>` 自动在对应目录下插入页面链接
- `wiki +mkdir` 创建新目录，`wiki +rmdir` 删除目录及其子项

---

## 工作流概览

| 阶段 | 工作流 | 说明 |
|------|--------|------|
| 1 | 文档结构初始化 | 创建完整的 Wiki 目录结构和索引页面 |
| 2 | 侧边栏导航维护 | 自动生成和更新侧边栏导航 |
| 3 | 文档模板生成 | 创建标准文档模板（贡献指南、开发环境等） |
| 4 | 批量页面更新 | 批量更新 Wiki 页面内容 |

---

## 详细工作流

### 工作流 1：文档结构初始化

**场景**：项目新建或重构时，一键创建标准化的 Wiki 文档结构。

#### Step 1：创建侧边栏顶级目录

```bash
# 创建四大核心目录
gitlink-cli wiki +mkdir --name "开发指南"
gitlink-cli wiki +mkdir --name "API参考"
gitlink-cli wiki +mkdir --name "架构设计"
gitlink-cli wiki +mkdir --name "常见问题"
```

> 目录创建后会在 `_Sidebar` 页面自动添加对应的顶级条目。

#### Step 2：在各目录下创建索引页面

```bash
# 开发指南目录
gitlink-cli wiki +create \
  --name "开发环境搭建" \
  --content "# 开发环境搭建\n\n## 前置要求\n\n...\n\n## 安装步骤\n\n...\n\n## 常用命令\n\n..." \
  --message "初始化：创建开发环境搭建文档" \
  --dir "开发指南"

gitlink-cli wiki +create \
  --name "代码规范" \
  --content "# 代码规范\n\n## 命名约定\n\n...\n\n## 格式化\n\n..." \
  --message "初始化：创建代码规范文档" \
  --dir "开发指南"

gitlink-cli wiki +create \
  --name "提交规范" \
  --content "# 提交规范\n\n## Commit Message 格式\n\n...\n\n## 分支策略\n\n..." \
  --message "初始化：创建提交规范文档" \
  --dir "开发指南"

# API参考目录
gitlink-cli wiki +create \
  --name "REST API 概览" \
  --content "# REST API 概览\n\n## 基础 URL\n\n...\n\n## 认证方式\n\n...\n\n## 通用响应格式\n\n..." \
  --message "初始化：创建 API 概览文档" \
  --dir "API参考"

gitlink-cli wiki +create \
  --name "认证与授权" \
  --content "# 认证与授权\n\n## OAuth2 流程\n\n...\n\n## Token 管理\n\n..." \
  --message "初始化：创建认证文档" \
  --dir "API参考"

gitlink-cli wiki +create \
  --name "错误码说明" \
  --content "# 错误码说明\n\n## HTTP 状态码\n\n...\n\n## 业务错误码\n\n..." \
  --message "初始化：创建错误码文档" \
  --dir "API参考"

# 架构设计目录
gitlink-cli wiki +create \
  --name "系统架构总览" \
  --content "# 系统架构总览\n\n## 整体架构\n\n...\n\n## 核心模块\n\n..." \
  --message "初始化：创建架构总览文档" \
  --dir "架构设计"

# 常见问题目录
gitlink-cli wiki +create \
  --name "FAQ" \
  --content "# 常见问题\n\n## 安装相关\n\n### Q: 安装失败怎么办？\n\n...\n\n## 使用相关\n\n..." \
  --message "初始化：创建 FAQ 文档" \
  --dir "常见问题"
```

> `--dir` 参数会在侧边栏对应目录下自动添加 `[[页面名]]` 链接。

#### Step 3：验证结构

```bash
# 列出所有 Wiki 页面，确认创建结果
gitlink-cli wiki +list --format json

# 查看侧边栏，确认导航结构正确
gitlink-cli wiki +view --name "_Sidebar"
```

---

### 工作流 2：侧边栏导航维护

**场景**：项目文档结构变更时，自动维护侧边栏导航的一致性。

#### 场景 A：添加新页面到已有目录

```bash
# 创建页面并直接关联到目录
gitlink-cli wiki +create \
  --name "部署指南" \
  --content "# 部署指南\n\n## Docker 部署\n\n...\n\n## 手动部署\n\n..." \
  --message "添加部署指南" \
  --dir "开发指南"
```

> `--dir` 自动将 `[[部署指南]]` 插入到侧边栏 "开发指南" 目录下。

#### 场景 B：添加新的子目录

```bash
# 在已有目录下创建子目录
gitlink-cli wiki +mkdir --name "数据库" --parent "架构设计"

# 在子目录下创建页面
gitlink-cli wiki +create \
  --name "数据库设计" \
  --content "# 数据库设计\n\n## ER 图\n\n...\n\n## 表结构说明\n\n..." \
  --message "添加数据库设计文档" \
  --dir "数据库"
```

> `--parent` 在指定目录下创建缩进的子目录。

#### 场景 C：删除页面和目录

```bash
# 删除页面（自动从侧边栏移除链接）
gitlink-cli wiki +delete --name "旧文档"

# 删除目录（自动移除目录及所有子项）
gitlink-cli wiki +rmdir --name "废弃目录"
```

> `+delete` 会自动清理侧边栏中对应的 `[[页面名]]` 链接。`+rmdir` 会移除目录行及所有子行。

#### 场景 D：重命名页面和目录

```bash
# 重命名页面（自动迁移内容 + 更新侧边栏链接）
gitlink-cli wiki +rename --name "旧名称" --new-name "新名称"

# 重命名目录（更新侧边栏中的目录标题）
gitlink-cli wiki +renamedir --name "旧目录名" --new-name "新目录名"
```

> `+rename` 执行"获取旧页面内容 → 创建新页面 → 删除旧页面 → 更新侧边栏"的完整流程。

---

### 工作流 3：文档模板生成

**场景**：为新项目或标准化流程批量创建文档模板页面。

#### Step 1：生成贡献指南

```bash
gitlink-cli wiki +create \
  --name "CONTRIBUTING" \
  --content "# 贡献指南\n\n感谢你对本项目的关注！以下是参与贡献的流程。\n\n## 如何贡献\n\n### 报告 Bug\n\n1. 搜索已有 Issue，确认没有被报告过\n2. 创建新 Issue，包含：复现步骤、预期行为、实际行为、环境信息\n\n### 提交代码\n\n1. Fork 本仓库\n2. 创建功能分支：\n\n\`\`\`bash\ngit checkout -b feature/my-feature\n\`\`\`\n\n3. 提交更改，遵循 [提交规范](/提交规范)\n4. 发起 Pull Request\n\n### 代码审查\n\n所有 PR 需要至少一位维护者 Review 通过后方可合并。\n\n## 行为准则\n\n请尊重所有贡献者，保持友善和建设性的交流。\n" \
  --message "创建贡献指南模板" \
  --dir "开发指南"
```

#### Step 2：生成开发环境搭建文档

```bash
gitlink-cli wiki +create \
  --name "开发环境搭建" \
  --content "# 开发环境搭建\n\n## 系统要求\n\n| 工具 | 最低版本 |\n|------|----------|\n| Go | 1.21+ |\n| Git | 2.30+ |\n\n## 快速开始\n\n\`\`\`bash\n# 克隆仓库\ngit clone <repo-url>\ncd <repo-name>\n\n# 安装依赖\ngo mod download\n\n# 构建\ngo build -o gitlink-cli .\n\n# 运行测试\ngo test ./...\n\`\`\`\n\n## IDE 推荐\n\n- VS Code + Go 扩展\n- GoLand\n\n## 常见问题\n\n参见 [[FAQ]]\n" \
  --message "创建开发环境搭建模板" \
  --dir "开发指南"
```

#### Step 3：生成 API 文档模板

```bash
gitlink-cli wiki +create \
  --name "API 文档模板" \
  --content "# API 文档模板\n\n## 接口名称\n\n简要描述接口用途。\n\n### 请求\n\n\`\`\`\nMETHOD /api/v1/endpoint\n\`\`\`\n\n**请求参数：**\n\n| 参数 | 类型 | 必填 | 说明 |\n|------|------|:----:|------|\n| | | | |\n\n### 响应\n\n**成功响应（200）：**\n\n\`\`\`json\n{\n  \"status\": 0,\n  \"message\": \"success\",\n  \"data\": {}\n}\n\`\`\`\n\n**错误响应：**\n\n| 状态码 | 说明 |\n|--------|------|\n| 400 | 参数错误 |\n| 401 | 未认证 |\n| 403 | 无权限 |\n| 404 | 资源不存在 |\n" \
  --message "创建 API 文档模板" \
  --dir "API参考"
```

---

### 工作流 4：批量页面更新

**场景**：版本升级、全局术语变更或批量内容修正时，一次性更新多个 Wiki 页面。

#### Step 1：获取当前所有页面列表

```bash
# 列出所有页面，确定需要更新的范围
gitlink-cli wiki +list --format json
```

#### Step 2：逐个读取并分析页面内容

```bash
# 查看需要更新的页面
gitlink-cli wiki +view --name "开发环境搭建"
gitlink-cli wiki +view --name "部署指南"
gitlink-cli wiki +view --name "REST API 概览"
```

> Agent 应解析 `wiki +view` 的输出，提取 `content_base64` 字段，解码后分析需要修改的部分。

#### Step 3：批量更新页面

```bash
# 更新版本号（示例：全局升级 v1.0 → v2.0）
gitlink-cli wiki +update \
  --name "开发环境搭建" \
  --content "# 开发环境搭建\n\n## 系统要求\n\n| 工具 | 最低版本 |\n|------|----------|\n| Go | 1.22+ |\n| Git | 2.40+ |\n\n..." \
  --message "更新：升级系统要求版本"

gitlink-cli wiki +update \
  --name "部署指南" \
  --content "# 部署指南（v2.0）\n\n## 变更说明\n\nv2.0 新增以下部署要求：\n\n..." \
  --message "更新：同步 v2.0 部署变更"

gitlink-cli wiki +update \
  --name "REST API 概览" \
  --content "# REST API 概览\n\n## 基础 URL\n\n`https://api.example.com/v2`\n\n..." \
  --message "更新：API 基础 URL 升级至 v2"
```

#### 批量更新注意事项

- 所有写入操作前**必须确认用户意图**，特别是涉及多个页面的批量更新
- 建议先在单个页面上验证更新效果，确认无误后再批量执行
- 每次更新提供清晰的 `--message`，便于后续追溯变更历史
- 如果更新过程中某个页面失败，记录失败的页面名称和错误信息，继续处理剩余页面

---

## 侧边栏操作速查

| 操作 | 命令 | 侧边栏效果 |
|------|------|-----------|
| 创建顶级目录 | `wiki +mkdir --name "目录名"` | 添加 `- 目录名` |
| 创建子目录 | `wiki +mkdir --name "子目录" --parent "父目录"` | 在父目录下缩进添加 `- 子目录` |
| 删除目录 | `wiki +rmdir --name "目录名"` | 移除目录及所有子行 |
| 重命名目录 | `wiki +renamedir --name "旧名" --new-name "新名"` | 替换目录标题 |
| 创建页面到目录 | `wiki +create --name "页面" --content "..." --dir "目录"` | 在目录下添加 `[[页面]]` |
| 删除页面 | `wiki +delete --name "页面"` | 自动移除 `[[页面]]` 链接 |
| 重命名页面 | `wiki +rename --name "旧名" --new-name "新名"` | 自动替换 `[[旧名]]` 为 `[[新名]]` |

---

## 注意事项

- Wiki 操作通过独立的网关 API（`gateway.gitlink.org.cn/api`）执行，与仓库 API 不同
- `wiki +create --dir` 要求目录已存在于侧边栏中，否则会报错；应先 `wiki +mkdir` 再 `wiki +create --dir`
- `wiki +rename` 执行"获取内容 → 创建新页面 → 删除旧页面 → 更新侧边栏"的完整流程，操作不可逆
- `wiki +rmdir` 会删除目录及该目录下所有子项（页面链接和子目录），操作不可逆
- 侧边栏使用 Tab 缩进表示层级，手动编辑 `_Sidebar` 页面时请保持缩进一致
- 建议在执行批量操作前先用 `wiki +list` 和 `wiki +view --name "_Sidebar"` 确认当前状态

---

## 相关 Skill 交叉引用

| Skill | 关联场景 |
|-------|----------|
| [`gitlink-shared`](../gitlink-shared/SKILL.md) | 认证、全局参数、安全规则基础 |
| [`gitlink-code-review`](../gitlink-code-review/SKILL.md) | 审查 PR 时同步更新 Wiki 文档 |
| [`gitlink-workflow`](../gitlink-workflow/SKILL.md) | 自动化工作流，可结合 Wiki 更新 |
