# 合规预审示例

用户请求：

```text
请检查 Gitconomy/Git4Research 是否具备开源发布合规基础。
```

Agent 步骤：

```bash
gitlink-cli repo +info --owner Gitconomy --repo Git4Research --format json
gitlink-cli repo +tree --owner Gitconomy --repo Git4Research --ref main --format json
```

预期回答：

- 报告许可证、安全策略、贡献指南、依赖清单和敏感文件风险。
- 说明审计范围是仅根目录，还是包含选定子目录。
- 避免引用或暴露敏感文件内容。
