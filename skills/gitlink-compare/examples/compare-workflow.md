# 代码比较完整工作流示例

**场景**：开发者需要比较两个分支或提交之间的差异。

## 工作流步骤

### Step 1：比较两个分支

```bash
# 比较两个分支的差异
gitlink-cli api GET /:owner/:repo/compare/master...develop --format json
```

### Step 2：查看提交差异

```bash
# 获取两个提交之间的差异
gitlink-cli api GET /:owner/:repo/compare/abc123...def456 --format json
```

### Step 3：查看特定文件变更

```bash
# 获取文件的变更历史
gitlink-cli repo +commits --owner myorg --repo myproject --format json

# 获取某个提交的详细内容
gitlink-cli api GET /:owner/:repo/commits/<sha> --format json
```

---

## 完整命令速览

```bash
gitlink-cli api GET /:owner/:repo/compare/<base>...<head> --format json
gitlink-cli repo +commits --owner <owner> --repo <repo> --format json
```
