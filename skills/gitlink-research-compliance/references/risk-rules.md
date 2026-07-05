# Compliance risk rules

| 风险 | 等级 | 判定 | 建议 |
|---|---|---|---|
| 缺少 LICENSE | 高 | 根目录无 LICENSE 文件 | 补充许可证并在 README 声明 |
| 疑似私钥 | 高 | 文件名包含 `id_rsa`、`.pem`、`.key` | 移除文件、轮换密钥、加入 `.gitignore` |
| 疑似环境密钥 | 高 | 文件名包含 `.env`、`credential`、`secret`、`token` | 移除真实配置，保留 `.env.example` |
| 缺少依赖声明 | 中 | 无常见依赖清单 | 补充依赖文件，便于许可证和漏洞审计 |
| 缺少 SECURITY | 中 | 无 `SECURITY.md` | 补充漏洞披露流程 |
| 缺少 CONTRIBUTING | 低 | 无 `CONTRIBUTING.md` | 补充贡献流程和代码规范 |

