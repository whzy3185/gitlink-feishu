# gitlink-docs-assistant — 使用示例

## 场景 1：文档完整性体检

**用户 Prompt：**
> 请阅读 skills/gitlink-docs-assistant/SKILL.md，帮我检查 ylly/gitlink-cli 仓库的文档完整性。

**Agent 执行过程：**

```bash
# Step 1：获取仓库信息
gitlink-cli repo +info --owner ylly --repo gitlink-cli --format json
```

```bash
# Step 2：扫描根目录文件
gitlink-cli api GET /ylly/gitlink-cli/sub_entries --query 'filepath=&ref=master'
```

根目录文件（节选）：
```
README.md, README.zh-CN.md, LICENSE, go.mod, go.sum, main.go, ...
```
→ 有 README ✅、有 LICENSE ✅，无 CONTRIBUTING ❌，无 CHANGELOG ❌

```bash
# Step 3：列出 Wiki 页面
gitlink-cli wiki +list --owner ylly --repo gitlink-cli --format json
```

返回：`["HOME", "demopage"]` → 无 API 文档 ❌

**Agent 输出报告：**

```markdown
## 📋 文档体检报告 — ylly/gitlink-cli

📅 检查时间：2026-06-15

### 总体评级：🟡 待完善

| 检查项 | 状态 | 建议 |
|--------|:----:|------|
| README | ✅ 存在 | — |
| CONTRIBUTING | ❌ 缺失 | 建议创建 Wiki 页面 |
| CHANGELOG | ❌ 缺失 | 建议创建 Wiki 页面 |
| API 文档 | ❌ 缺失 | 建议创建 Wiki 页面 |
| LICENSE | ✅ 存在 | — |
| 代码规范文档 | ⚠️ 未找到 | 可选补充 |

### 🎯 建议优先补充
1. 🔴 CONTRIBUTING —— 降低新贡献者入门门槛
2. 🟡 CHANGELOG —— 方便用户了解版本变更
3. 🟡 API 文档 —— 说明 wiki/label/notification 等命令接口
```

---

## 场景 2：AI 自动生成 CONTRIBUTING 文档

**用户 Prompt：**
> 帮我自动生成 ylly/gitlink-cli 的 CONTRIBUTING 文档并写入 Wiki。

**Agent 执行过程：**

```bash
# 读取 README 了解项目背景
gitlink-cli api GET /ylly/gitlink-cli/readme

# 读取目录结构了解项目技术栈
gitlink-cli api GET /ylly/gitlink-cli/sub_entries --query 'filepath=&ref=master'
```

AI 分析：Go 项目，有 shortcuts/ 目录，有单元测试，CLI 工具。

```bash
# 写入 Wiki
gitlink-cli wiki +create \
  --owner ylly \
  --repo gitlink-cli \
  --name "CONTRIBUTING" \
  --content "# 贡献指南

欢迎为 gitlink-cli 做贡献！

## 环境准备
- Go 1.21+
- gitlink-cli 已配置认证

## 开发流程
1. Fork 仓库
2. 创建功能分支
3. 编写代码和测试
4. 提交 PR

## 代码规范
- 运行 \`go test ./...\` 确保测试通过
- 新增 shortcut 需在 \`shortcuts/register.go\` 注册
" \
  --message "docs: AI 自动生成 CONTRIBUTING 文档"
```

**Agent 输出：**
> ✅ 已创建 Wiki 页面「CONTRIBUTING」。

---

## 场景 3：更新过时的 Wiki 文档

**用户 Prompt：**
> wiki 里的 API 文档还没有 wiki/label 命令的说明，帮我更新一下。

**Agent 执行过程：**

```bash
# 读取现有页面
gitlink-cli wiki +view --owner ylly --repo gitlink-cli --name "API 文档"

# 读取最近提交确认变更范围
gitlink-cli api GET /ylly/gitlink-cli/commits --query 'page=1&limit=10'
```

```bash
# 更新页面（追加 wiki/label 命令说明）
gitlink-cli wiki +update \
  --owner ylly \
  --repo gitlink-cli \
  --name "API 文档" \
  --content "# API 文档

（原有内容）...

## Wiki 命令
- \`wiki +list\` — 列出所有 Wiki 页面
- \`wiki +create\` — 创建新页面
- \`wiki +update\` — 更新页面内容
- \`wiki +view\` — 查看页面内容
- \`wiki +delete\` — 删除页面

## Label 命令
- \`label +list\` — 列出标签
- \`label +create\` — 创建标签
- \`label +update\` — 更新标签
- \`label +delete\` — 删除标签
" \
  --message "docs: 补充 wiki/label 命令说明"
```

**Agent 输出：**
> ✅ 已更新「API 文档」页面，新增 wiki 和 label 命令说明。
