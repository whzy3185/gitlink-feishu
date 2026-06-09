---
name: gitlink-feedback
description: "GitLink 反馈建议提交：从命令行提交平台问题、改进建议或 CLI 使用反馈，支持文件、stdin、dry-run 和上下文元数据。"
metadata:
  cliHelp: "gitlink-cli feedback --help"
---

# gitlink-feedback

当用户需要向 GitLink 平台提交问题反馈、体验建议或 CLI 改进意见时使用本 Skill。底层接口只接收 `content`，CLI 会把分类、联系方式和相关仓库作为文本元数据拼入反馈正文，便于平台侧处理。

## 常用命令

| 命令 | 用途 |
|------|------|
| `feedback +create` | 提交反馈建议 |

## 示例

```bash
# 直接提交简短反馈
gitlink-cli feedback +create --content "CLI 安装文档需要补充 Windows 说明。" --category docs

# 提交前预览请求路径、正文长度和 payload
gitlink-cli feedback +create --content "希望支持更多输出格式。" --category feature --dry-run

# 从文件读取长反馈
gitlink-cli feedback +create --from feedback.md --category cli --contact mengz@example.com

# 从管道读取反馈
Get-Content feedback.md | gitlink-cli feedback +create --stdin --category ux --repo-ref Gitlink/gitlink-cli
```

## 参数说明

| 参数 | 说明 |
|------|------|
| `--user` / `-u` | GitLink 用户标识；不传时读取当前登录用户 |
| `--content` / `-c` | 直接传入反馈正文 |
| `--from` / `-f` | 从文本文件读取反馈正文 |
| `--stdin` | 从标准输入读取反馈正文 |
| `--category` | 可选分类，如 `bug`、`feature`、`docs`、`ux`、`cli` |
| `--contact` | 可选联系方式，会写入反馈正文 |
| `--repo-ref` | 可选相关仓库，格式建议为 `owner/repo` |
| `--dry-run` | 只预览请求，不提交反馈 |

## 安全规则

- 反馈内容可能进入平台工单或日志，不要提交 Token、Cookie、密码、私钥等敏感信息。
- 需要提交较长复现信息时优先使用 `--from`，并先用 `--dry-run` 检查最终内容。
- `--contact` 是明文写入反馈正文，只填写愿意公开给平台维护者的信息。
- 反馈涉及具体仓库时传 `--repo-ref owner/repo`，不要把仓库上下文混在正文里导致平台侧难以分拣。
