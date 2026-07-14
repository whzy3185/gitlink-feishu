# notification shortcut

新增 `notification` 命令组，支持用户通知管理：

| 命令 | 功能 |
|------|------|
| `notification +list` | 列出通知列表 |
| `notification +view --id <id>` | 查看通知详情 |
| `notification +read --id <id>` | 标记通知已读 |
| `notification +delete --id <id>` | 删除通知 |

修复：`issue +create` 命令的 `--label` 参数现在会正确传入请求 body。
