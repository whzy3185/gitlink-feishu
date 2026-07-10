# 仓库管理完整工作流示例

**场景**：开发者需要创建仓库、查看信息、管理文件和贡献者。

## 工作流步骤

### Step 1：创建新仓库

```bash
gitlink-cli repo +create --name my-project --description "我的新项目" --private false
```

### Step 2：查看仓库信息

```bash
gitlink-cli repo +info --owner myuser --repo my-project --format json

# 查看 README
gitlink-cli repo +readme --owner myuser --repo my-project

# 查看贡献者
gitlink-cli repo +contributors --owner myuser --repo my-project --format json

# 查看语言统计
gitlink-cli repo +languages --owner myuser --repo my-project --format json
```

### Step 3：管理仓库文件

```bash
# 获取文件内容
gitlink-cli repo +raw --owner myuser --repo my-project --path README.md

# 查看目录结构
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath=src&ref=master'

# 查看提交历史
gitlink-cli repo +commits --owner myuser --repo myproject --limit 10 --format json
```

### Step 4：Fork 和删除

```bash
# Fork 仓库
gitlink-cli repo +fork --owner target-user --repo target-repo

# 删除自己的仓库（⚠️ 危险操作，需确认）
gitlink-cli repo +delete --owner myuser --repo old-project
```

---

## 完整命令速览

```bash
gitlink-cli repo +list --user <user> --format json
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
gitlink-cli repo +create --name <name> --description "..."
gitlink-cli repo +fork --owner <owner> --repo <repo>
gitlink-cli repo +readme --owner <owner> --repo <repo>
gitlink-cli repo +contributors --owner <owner> --repo <repo> --format json
gitlink-cli repo +commits --owner <owner> --repo <repo> --limit <n> --format json
gitlink-cli repo +raw --owner <owner> --repo <repo> --path <path>
```
