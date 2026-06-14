# Wiki 从零初始化工作流示例

本文档展示如何使用 `gitlink-wiki-builder` Skill 为新仓库 `whale_hihihi/test` 从零搭建完整的 Wiki 文档结构。

> **前置条件**：已完成 `gitlink-cli auth login` 认证。详见 [`../../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md)。

---

## Step 1：确认当前 Wiki 状态

初始化前先查看当前 Wiki 页面列表，确认是否为空或存在已有内容。

```bash
# 列出所有 Wiki 页面
gitlink-cli wiki +list --owner whale_hihihi --repo test --format json
```

预期输出（空 Wiki）：

```json
{
  "code": 0,
  "message": "success",
  "data": []
}
```

---

## Step 2：创建顶级目录结构

为项目创建四个核心文档分类目录。

```bash
# 创建顶级目录
gitlink-cli wiki +mkdir --name "开发指南" --owner whale_hihihi --repo test
gitlink-cli wiki +mkdir --name "API参考" --owner whale_hihihi --repo test
gitlink-cli wiki +mkdir --name "架构设计" --owner whale_hihihi --repo test
gitlink-cli wiki +mkdir --name "常见问题" --owner whale_hihihi --repo test
```

预期输出：

```
Directory "开发指南" created.
Directory "API参考" created.
Directory "架构设计" created.
Directory "常见问题" created.
```

---

## Step 3：在"开发指南"目录下创建页面

```bash
# 创建开发环境搭建文档
gitlink-cli wiki +create \
  --name "开发环境搭建" \
  --content "# 开发环境搭建

## 系统要求

| 工具 | 最低版本 |
|------|----------|
| Go | 1.21+ |
| Git | 2.30+ |

## 快速开始

1. 克隆仓库：\`git clone https://gitlink.org.cn/whale_hihihi/test.git\`
2. 安装依赖：\`go mod download\`
3. 构建：\`go build -o test .\`
4. 运行测试：\`go test ./...\`

## IDE 配置

推荐使用 VS Code + Go 扩展，安装后可获得代码补全、跳转定义和调试支持。" \
  --message "初始化：创建开发环境搭建文档" \
  --dir "开发指南" \
  --owner whale_hihihi --repo test

# 创建代码规范文档
gitlink-cli wiki +create \
  --name "代码规范" \
  --content "# 代码规范

## 命名约定

- 包名：小写单词，不使用下划线（如 \`shortcuts\`）
- 导出函数：大驼峰（如 \`CreateWiki\`）
- 内部函数：小驼峰（如 \`fetchProjectID\`）
- 常量：大写 + 下划线（如 \`MAX_RETRIES\`）

## 格式化

使用 \`gofmt\` 或 \`goimports\` 格式化代码，提交前确保通过 \`golangci-lint run\`。

## 注释规范

- 导出标识符必须有文档注释
- 注释以标识符名称开头：\`// Shortcuts returns wiki management shortcuts.\`" \
  --message "初始化：创建代码规范文档" \
  --dir "开发指南" \
  --owner whale_hihihi --repo test

# 创建贡献指南
gitlink-cli wiki +create \
  --name "CONTRIBUTING" \
  --content "# 贡献指南

感谢你对本项目的关注！

## 报告 Bug

1. 搜索已有 Issue，确认未被报告
2. 创建新 Issue，包含：复现步骤、预期行为、实际行为、环境信息

## 提交代码

1. Fork 仓库
2. 创建功能分支：\`git checkout -b feature/my-feature\`
3. 提交更改，遵循 Commit Message 规范
4. 发起 Pull Request

## 代码审查

所有 PR 需要至少一位维护者 Review 通过后方可合并。" \
  --message "初始化：创建贡献指南" \
  --dir "开发指南" \
  --owner whale_hihihi --repo test
```

预期输出（每个页面）：

```
Page "开发环境搭建" added to directory "开发指南" in sidebar.
Page "代码规范" added to directory "开发指南" in sidebar.
Page "CONTRIBUTING" added to directory "开发指南" in sidebar.
```

---

## Step 4：在"API参考"目录下创建页面

```bash
# 创建 API 概览
gitlink-cli wiki +create \
  --name "REST API 概览" \
  --content "# REST API 概览

## 基础 URL

\`https://api.example.com/v1\`

## 认证方式

所有 API 请求需在 Header 中携带 Token：

\`\`\`
Authorization: Bearer <token>
\`\`\`

## 通用响应格式

\`\`\`json
{
  \"status\": 0,
  \"message\": \"success\",
  \"data\": {}
}
\`\`\`

## 速率限制

每个 Token 每分钟最多 60 次请求。" \
  --message "初始化：创建 API 概览文档" \
  --dir "API参考" \
  --owner whale_hihihi --repo test

# 创建错误码文档
gitlink-cli wiki +create \
  --name "错误码说明" \
  --content "# 错误码说明

## HTTP 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 参数错误 |
| 401 | 未认证 |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

## 业务错误码

| 错误码 | 说明 | 处理建议 |
|--------|------|----------|
| 10001 | Token 过期 | 重新获取 Token |
| 10002 | 权限不足 | 联系管理员 |
| 20001 | 资源已存在 | 检查是否重复创建 |" \
  --message "初始化：创建错误码文档" \
  --dir "API参考" \
  --owner whale_hihihi --repo test
```

---

## Step 5：在"架构设计"目录下创建页面和子目录

```bash
# 创建架构总览
gitlink-cli wiki +create \
  --name "系统架构总览" \
  --content "# 系统架构总览

## 整体架构

项目采用分层架构：

- **CLI 层**：命令行解析和用户交互（Cobra 框架）
- **Shortcut 层**：高级命令封装（Issue、PR、Wiki 等）
- **API 层**：GitLink REST API 客户端
- **工具层**：输出格式化、国际化、配置管理

## 核心模块

| 模块 | 路径 | 说明 |
|------|------|------|
| shortcuts | ./shortcuts/ | 高级命令封装 |
| internal | ./internal/ | 内部工具库 |
| cmd | ./cmd/ | CLI 入口 |" \
  --message "初始化：创建架构总览文档" \
  --dir "架构设计" \
  --owner whale_hihihi --repo test

# 创建子目录"数据库"并在其中创建页面
gitlink-cli wiki +mkdir --name "数据库" --parent "架构设计" \
  --owner whale_hihihi --repo test

gitlink-cli wiki +create \
  --name "数据库设计" \
  --content "# 数据库设计

## 设计原则

- 所有表使用自增 ID 作为主键
- 时间字段统一使用 \`datetime\` 类型
- 软删除使用 \`is_deleted\` 标记

## 核心表

### users 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int | 主键 |
| user_name | varchar(50) | 用户名 |
| email | varchar(100) | 邮箱 |
| created_at | datetime | 创建时间 |" \
  --message "初始化：创建数据库设计文档" \
  --dir "数据库" \
  --owner whale_hihihi --repo test
```

---

## Step 6：在"常见问题"目录下创建页面

```bash
gitlink-cli wiki +create \
  --name "FAQ" \
  --content "# 常见问题

## 安装相关

### Q: go build 失败怎么办？

确认 Go 版本 >= 1.21，然后执行：

\`\`\`bash
go clean -cache
go mod tidy
go build -o test .
\`\`\`

### Q: 认证失败怎么办？

1. 确认已执行 \`gitlink-cli auth login\`
2. 检查 Token 是否过期
3. 重新登录：\`gitlink-cli auth login --force\`

## 使用相关

### Q: Wiki 命令返回 404？

Wiki 功能需要在 GitLink 项目中先启用 Wiki 模块。在项目设置中开启后重试。

### Q: 如何查看 API 请求的详细信息？

使用 \`--verbose\` 参数查看请求和响应的详细信息：

\`\`\`bash
gitlink-cli wiki +list --verbose
\`\`\`" \
  --message "初始化：创建 FAQ 文档" \
  --dir "常见问题" \
  --owner whale_hihihi --repo test
```

---

## Step 7：验证最终结构

```bash
# 查看侧边栏，确认目录和页面结构正确
gitlink-cli wiki +view --name "_Sidebar" --owner whale_hihihi --repo test

# 列出所有页面
gitlink-cli wiki +list --owner whale_hihihi --repo test --format json
```

预期侧边栏结构：

```
- 开发指南
	[[开发环境搭建]]
	[[代码规范]]
	[[CONTRIBUTING]]
- API参考
	[[REST API 概览]]
	[[错误码说明]]
- 架构设计
	[[系统架构总览]]
	- 数据库
		[[数据库设计]]
- 常见问题
	[[FAQ]]
```

---

## 执行汇总

| 操作 | 数量 | 命令 |
|------|:----:|------|
| 创建顶级目录 | 4 | `wiki +mkdir` |
| 创建子目录 | 1 | `wiki +mkdir --parent` |
| 创建 Wiki 页面 | 9 | `wiki +create --dir` |

总计执行 14 条 `gitlink-cli` 命令，完成从零到完整 Wiki 文档结构的搭建。

---

*由 gitlink-wiki-builder Skill 示例工作流生成*
