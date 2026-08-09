# 自动化 Release 管理完整工作流示例

**场景**：项目维护者需要从提交历史自动生成 Release Notes 并发布新版本。

## 前置条件

- `gitlink-cli` 已安装并登录
- 项目有提交历史和已合并的 PR

## 工作流步骤

### Step 1：确认当前版本和推荐新版本号

```bash
# 获取当前最新 Release
gitlink-cli release +list --owner myorg --repo myproject --format json
```

**输出示例：**
```json
{
  "ok": true,
  "data": {
    "releases": [
      { "id": 15, "tag_name": "v1.2.0", "name": "v1.2.0 - 性能优化版" }
    ]
  }
}
```

```bash
# 获取自 v1.2.0 以来的提交历史
git log v1.2.0..HEAD --format="%H %s %an %ad" --date=short 2>&1
```

### Step 2：AI 分析提交并推荐版本号

AI 根据提交类型（Conventional Commits）推荐：

```
当前版本：v1.2.0
分析了 22 次提交：
  - 4 个 feat：全文搜索、暗黑模式、消息通知、批量导入
  - 6 个 fix：修复了 6 个 Bug
  - 2 个 docs：更新 API 文档
  - 无 Breaking Change

推荐版本：v1.3.0（MINOR 升级，有新功能）
```

### Step 3：自动生成 Release Notes

```bash
# 获取本期关闭的 Issue
gitlink-cli issue +list --state closed --format json

# 获取本期合并的 PR
gitlink-cli pr +list --state merged --format json
```

AI 根据提交和 PR/Issue 数据生成 Release Notes：

```markdown
## v1.3.0 (2026-06-12)

### ✨ 新功能
- feat(search): 新增全文搜索功能 (#156)
- feat(ui): 支持暗黑模式 (#178)
- feat(notify): 添加站内消息通知 (#189)
- feat(import): 支持批量导入数据 (#195)

### 🐛 Bug 修复
- fix(login): 修复手机号登录验证码未清除 (#165)
- fix(upload): 修复大文件上传超时 (#172)
- fix(api): 修复并发请求偶发 500 错误 (#180)

### 🤝 贡献者
感谢 @zhangsan、@lisi、@wangwu 的贡献！
```

### Step 4：创建 Release 并通知

```bash
# 创建 Release
gitlink-cli release +create \
  --owner myorg --repo myproject \
  --tag v1.3.0 \
  --name "v1.3.0 - 功能增强版" \
  --body "## v1.3.0 (2026-06-12)

### ✨ 新功能
- feat(search): 新增全文搜索功能 (#156)
- feat(ui): 支持暗黑模式 (#178)

### 🐛 Bug 修复
- fix(login): 修复手机号登录验证码未清除 (#165)
- fix(upload): 修复大文件上传超时 (#172)

### 🤝 贡献者
感谢 @zhangsan、@lisi 的贡献！" \
  --target master

# 验证发布成功
gitlink-cli release +list --owner myorg --repo myproject --format json

# 通知相关 Issue 已发版
gitlink-cli issue +comment --number 165 --body "🎉 此问题已在 v1.3.0 中修复，请更新验证。"
gitlink-cli issue +comment --number 172 --body "🎉 此问题已在 v1.3.0 中修复，请更新验证。"
```

---

## 完整命令速览

```bash
# 版本确认
gitlink-cli release +list --format json
git log v<prev>..HEAD --format="%s"

# 发布
gitlink-cli release +create --tag v<x.y.z> --name "..." --body "..." --target master

# 验证
gitlink-cli release +list --format json
gitlink-cli release +view --id <version_id>

# 通知
gitlink-cli issue +comment --number <n> --body "🎉 已在 v<x.y.z> 修复"
```
