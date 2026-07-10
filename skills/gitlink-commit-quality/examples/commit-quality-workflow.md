# 提交质量检查完整工作流示例

**场景**：项目维护者需要检查提交信息的质量和规范性。

## 工作流步骤

### Step 1：获取提交历史

```bash
# 获取最近的提交记录
gitlink-cli repo +commits --owner myorg --repo myproject --limit 20 --format json
```

### Step 2：分析提交信息

AI 检查提交信息是否符合 Conventional Commits 规范：

```markdown
## 📝 提交质量报告

### 检查项

| 检查项 | 标准 | 结果 |
|--------|------|------|
| 格式规范 | Conventional Commits | 15/20 通过 |
| 描述清晰 | 有具体说明 | 18/20 通过 |
| 关联 Issue | 引用 Issue 编号 | 12/20 通过 |

### 不合规示例

1. `fix bug` → 应改为 `fix: resolve login timeout issue (#165)`
2. `update` → 应改为 `docs: update API reference`
3. `wip` → WIP 提交不应出现在主分支

### 建议
- 遵循 `type(scope): description` 格式
- type 可选：feat / fix / docs / style / refactor / perf / test / ci / chore
- 添加 Breaking Change 标注（如有）
```

---

## 完整命令速览

```bash
gitlink-cli repo +commits --owner <owner> --repo <repo> --limit <n> --format json
```
