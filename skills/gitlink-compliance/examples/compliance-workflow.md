# 合规检查完整工作流示例

**场景**：项目维护者需要检查项目的合规性（许可证、代码规范等）。

## 工作流步骤

### Step 1：检查项目许可证

```bash
# 查看 LICENSE 文件
gitlink-cli repo +raw --owner myorg --repo myproject --path LICENSE

# 查看仓库信息中的许可证
gitlink-cli repo +info --owner myorg --repo myproject --format json
```

### Step 2：检查项目结构

```bash
# 查看根目录文件（寻找 .gitignore, CONTRIBUTING.md 等）
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=.&ref=master'

# 检查 CI 配置
gitlink-cli repo +raw --owner myorg --repo myproject --path .gitea/workflows/ci.yml
```

### Step 3：生成合规报告

AI 根据采集到的信息生成合规报告：

```markdown
## 📋 项目合规检查报告

| 检查项 | 状态 | 说明 |
|--------|:----:|------|
| LICENSE 文件 | ✅ | Apache-2.0 |
| README.md | ✅ | 完整 |
| CONTRIBUTING.md | ⚠️ | 缺失，建议补充 |
| .gitignore | ✅ | 已配置 |
| CI 配置 | ✅ | Gitea Actions |
| 代码规范配置 | ⚠️ | 缺少 linter 配置 |
```

---

## 完整命令速览

```bash
gitlink-cli repo +raw --owner <owner> --repo <repo> --path LICENSE
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=.&ref=master'
```
