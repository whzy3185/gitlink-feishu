# 文件操作完整工作流示例

**场景**：开发者需要通过 API 在仓库中创建、更新或删除文件。

## 前置条件

- `gitlink-cli` 已安装并登录
- 熟悉 GitLink 文件操作 API（base64 编码、SHA 校验）

## 工作流步骤

### Step 1：创建文件

```bash
# content 必须 base64 编码
# 方式 1：Linux/macOS
CONTENT=$(echo -n "# 项目文档\n\n这是一个示例文档" | base64)

# 方式 2：直接传递
gitlink-cli api POST /:owner/:repo/create_file --body '{
  "filepath": "docs/guide.md",
  "content": "IyDpobnnm67mlrnlvI8KCuacrOeahOWLvumZpGRvYw==",
  "branch": "master",
  "message": "docs: add project guide"
}'
```

### Step 2：获取文件 SHA（用于更新/删除）

```bash
# 获取文件信息（含 SHA）
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=docs/guide.md&ref=master'
# 从返回结果中取 entries.sha
```

### Step 3：更新文件

```bash
gitlink-cli api PUT /:owner/:repo/update_file --body '{
  "filepath": "docs/guide.md",
  "content": "<base64编码的新内容>",
  "sha": "<从sub_entries获取的sha>",
  "branch": "master",
  "message": "docs: update project guide"
}'
```

### Step 4：删除文件

```bash
gitlink-cli api DELETE /:owner/:repo/delete_file --body '{
  "filepath": "docs/old-guide.md",
  "sha": "<文件sha>",
  "branch": "master",
  "message": "docs: remove outdated guide"
}'
```

---

## 注意事项

- 创建文件时 content 必须使用 base64 编码
- 更新和删除文件需要提供文件的 SHA 值
- SHA 可通过 `sub_entries` 接口获取
- 文件操作会直接在指定分支上生成一个提交
