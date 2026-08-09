# 组织管理完整工作流示例

**场景**：组织管理员需要查看组织信息和成员列表。

## 工作流步骤

### Step 1：查看组织信息

```bash
# 列出用户所在的组织
gitlink-cli org +list --format json
```

### Step 2：查看组织成员

```bash
# 列出组织成员
gitlink-cli org +members --owner myorg --format json
```

### Step 3：查看组织仓库

```bash
# 列出组织下的仓库
gitlink-cli repo +list --user myorg --format json
```

---

## 完整命令速览

```bash
gitlink-cli org +list --format json
gitlink-cli org +members --owner <org> --format json
```
