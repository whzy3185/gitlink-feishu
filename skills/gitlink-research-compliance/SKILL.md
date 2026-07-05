---
name: gitlink-research-compliance
description: "科研开源合规与敏感风险检查：审计 GitLink 科研仓库的许可证、SECURITY、CONTRIBUTING、依赖声明和敏感文件风险。用于开源发布前预检查、论文代码发布合规清单和整改建议。"
---

# gitlink-research-compliance

开始前必须先阅读 `../gitlink-shared/SKILL.md`，确认认证、权限、安全规则和 GitLink API 注意事项。

## 安全规则

- 只读执行，不删除文件、不改 `.gitignore`、不关闭 Issue。
- 所有 GitLink 操作必须使用 `gitlink-cli`。
- 所有命令使用 `--format json`。
- 不输出 Token、Cookie 或认证 Header。
- 报告必须注明“工程合规预审，不构成法律意见”。

## 工作流

1. 采集仓库信息和文件树：

```bash
gitlink-cli repo +info --owner <owner> --repo <repo> --format json
gitlink-cli repo +tree --owner <owner> --repo <repo> --ref <branch> --format json
```

2. 如果 `repo +tree` 不可用，回退：

```bash
gitlink-cli api GET /<owner>/<repo>/sub_entries --query 'filepath=&ref=<branch>' --format json
```

3. 对根目录和高风险目录做规则检查：

- 根目录：许可证、安全策略、贡献指南、依赖清单、`.gitignore`。
- 配置目录：`.github`、`.gitlink`、`.vscode`、`config`。
- 脚本目录：`scripts`、`deploy`、`docker`。

4. 按 `references/risk-rules.md` 输出风险分级。

## 输出格式

```markdown
# 科研开源合规预审：<owner>/<repo>

## 总览

| 项目 | 结果 | 风险等级 | 证据 |
|---|---|---|---|

## 高优先级风险

1. ...

## 整改建议

1. ...

## 说明

本报告是工程合规预审，不构成法律意见。
```

## 检查项

- 许可证：存在 `LICENSE`，README 中最好也声明许可证。
- 安全策略：存在 `SECURITY.md`。
- 贡献指南：存在 `CONTRIBUTING.md`。
- 依赖声明：存在语言对应依赖文件，便于后续许可证和漏洞扫描。
- 敏感文件风险：文件名包含 `.env`、`id_rsa`、`.pem`、`.key`、`credential`、`secret`、`token` 时列为风险。

## 注意事项

- 不要下载或展示疑似密钥内容，只报告路径和风险类型。
- 如果只能获取根目录文件树，报告必须说明深层敏感文件未覆盖。
- 私有仓库需要用户自行完成认证，不能要求用户粘贴 Token。

