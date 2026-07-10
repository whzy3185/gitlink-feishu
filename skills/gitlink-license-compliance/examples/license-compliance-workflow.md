# 许可证合规检查完整工作流示例

**场景**：项目维护者需要检查项目的依赖许可证是否合规。

## 工作流步骤

### Step 1：获取项目依赖文件

```bash
# Go 项目
gitlink-cli repo +raw --owner myorg --repo myproject --path go.mod

# Node.js 项目
gitlink-cli repo +raw --owner myorg --repo myproject --path package.json

# Python 项目
gitlink-cli repo +raw --owner myorg --repo myproject --path requirements.txt
```

### Step 2：检查项目许可证

```bash
# 查看项目自身许可证
gitlink-cli repo +raw --owner myorg --repo myproject --path LICENSE
```

### Step 3：生成合规报告

AI 分析依赖许可证后生成报告：

```markdown
## 📜 许可证合规检查报告

### 项目许可证：Apache-2.0

### 依赖分析

| 依赖 | 许可证 | 兼容性 | 风险 |
|------|--------|:------:|:----:|
| library-a | MIT | ✅ 兼容 | 低 |
| library-b | Apache-2.0 | ✅ 兼容 | 低 |
| library-c | GPL-3.0 | ⚠️ 注意 | 高 |
| library-d | BSD-3-Clause | ✅ 兼容 | 低 |

### 建议
- ⚠️ library-c 使用 GPL-3.0 许可证，可能要求项目也使用 GPL 许可
- ✅ 其他依赖许可证与 Apache-2.0 兼容
```

---

## 注意事项

- GPL/AGPL 许可证具有传染性，需特别关注
- 建议定期检查依赖许可证变更
