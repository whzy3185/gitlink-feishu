# 新人引导完整工作流示例

**场景**：一个新贡献者想参与 GitLink 上的开源项目，AI Agent 引导其完成从了解项目到提交首次贡献的全过程。

## 前置条件

- `gitlink-cli` 已安装并登录
- 目标仓库为公开项目

## 工作流步骤

### Step 1：获取项目概览

```bash
# 获取仓库基本信息
gitlink-cli repo +info --owner Gitlink --repo forgeplus --format json
```

**输出示例：**
```json
{
  "ok": true,
  "data": {
    "identifier": "forgeplus",
    "name": "ForgePlus",
    "description": "开源研发创新平台",
    "language": "Ruby",
    "default_branch": "master",
    "license_name": "Apache-2.0"
  }
}
```

```bash
# 获取 README
gitlink-cli repo +readme --owner Gitlink --repo forgeplus

# 获取语言统计
gitlink-cli repo +languages --owner Gitlink --repo forgeplus --format json

# 查看贡献者
gitlink-cli repo +contributors --owner Gitlink --repo forgeplus --format json
```

### Step 2：搜索适合新人的 Issue

```bash
# 搜索 good-first-issue
gitlink-cli search +issues --owner Gitlink --repo forgeplus --keyword "good first issue" --category opened

# 如果没有 good-first-issue 标签，查看所有开放的 Issue
gitlink-cli issue +list --state open --format json
```

**AI 分析 Issue 列表，推荐最适合新人的任务：**

```markdown
## 🎯 推荐新手任务

### ⭐ 强烈推荐

1. **Issue #156** — 修复 README 中的过期链接
   - 📁 涉及文件：`README.md`
   - 📝 改动范围：约 3 行
   - 💡 提示：找到文档中的链接，替换为新地址即可

2. **Issue #203** — 添加缺失的 API 参数文档
   - 📁 涉及文件：`doc/api_reference.md`
   - 📝 改动范围：约 15 行
   - 💡 提示：参考代码中的参数定义补充文档
```

### Step 3：为推荐 Issue 生成引导评论

```bash
# 查看 Issue 详情
gitlink-cli issue +view --number 156 --format json
```

**AI 生成引导评论：**

```markdown
## 🌟 欢迎贡献！

这是一个很适合首次贡献的任务。

### 🗺️ 代码定位
- 需要修改的文件：`README.md`（项目根目录）
- 相关位置：第 45 行附近的"相关链接"章节

### ✏️ 修改步骤
1. Fork 项目
2. 创建分支 `git checkout -b fix/readme-links`
3. 打开 `README.md`，找到第 45 行的链接
4. 将 `https://old-domain.com/docs` 替换为 `https://new-domain.com/docs`
5. 本地预览确认链接正确
6. 提交 PR

### 💡 提示
- 只需修改一处，改动量极小
- 不需要运行测试
- PR 标题建议：`docs: update broken link in README`
```

### Step 4：发布引导评论

```bash
gitlink-cli issue +comment \
  --number 156 \
  --body "## 🌟 欢迎贡献！\n\n这是一个很适合首次贡献的任务。\n\n### 🗺️ 代码定位\n- 需要修改的文件：README.md\n\n### ✏️ 修改步骤\n1. Fork 项目\n2. 修改 README.md 中的链接\n3. 提交 PR\n\n### 💡 提示\n只需修改一处链接即可"
```

### Step 5：引导新人完成 Fork-PR 流程

```bash
# 1. Fork 仓库
gitlink-cli repo +fork --owner Gitlink --repo forgeplus

# 2. Clone
git clone https://www.gitlink.org.cn/MyUser/forgeplus.git
cd forgeplus

# 3. 配置 upstream
git remote add upstream https://www.gitlink.org.cn/Gitlink/forgeplus.git

# 4. 创建分支
git checkout -b fix/readme-links

# 5. 修改代码（用户手动操作）
# ...

# 6. 提交
git add README.md
git commit -m "docs: update broken link in README"

# 7. 推送到 Fork
git push origin fix/readme-links

# 8. 创建 PR
gitlink-cli pr +create \
  --owner Gitlink --repo forgeplus \
  --head MyUser:fix/readme-links --base master \
  --title "docs: update broken link in README" \
  --body "## 变更说明\n\nFixes #156\n\n更新了 README 中的过期链接。"
```

### Step 6：生成新人上手指南

**AI 输出完整的新人上手指南：**

```markdown
# 🚀 ForgePlus 新人上手指南

## 项目简介
ForgePlus 是一个开源研发创新平台，主要使用 Ruby 开发。

## 环境搭建
1. Fork 项目：`gitlink-cli repo +fork --owner Gitlink --repo forgeplus`
2. Clone：`git clone https://www.gitlink.org.cn/<you>/forgeplus.git`
3. 安装依赖：`bundle install`
4. 配置数据库：`cp config/database.yml.example config/database.yml`
5. 运行测试：`bundle exec rspec`

## 推荐的第一个任务
- Issue #156：修复 README 过期链接（难度：⭐）
- Issue #203：补充 API 文档（难度：⭐⭐）

## 贡献流程
Fork → Branch → Code → Test → PR

## 获取帮助
在 Issue 中 @maintainer 或在 Wiki 中查阅贡献指南
```

---

## 完整命令速览

```bash
# 项目信息
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
gitlink-cli repo +readme --owner <owner> --repo <repo>
gitlink-cli repo +languages --owner <owner> --repo <repo> --format json
gitlink-cli repo +contributors --owner <owner> --repo <repo> --format json

# 搜索 Issue
gitlink-cli search +issues --owner <owner> --repo <repo> --keyword "good first issue" --category opened
gitlink-cli issue +list --state open --format json
gitlink-cli issue +view --number <n> --format json

# 发布引导评论
gitlink-cli issue +comment --number <n> --body "<引导内容>"

# Fork 和 PR
gitlink-cli repo +fork --owner <owner> --repo <repo>
gitlink-cli pr +create --owner <owner> --repo <repo> --head <you>:<branch> --base master --title "..." --body "..."
```
