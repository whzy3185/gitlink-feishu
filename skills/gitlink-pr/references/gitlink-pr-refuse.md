# pr +refuse

> **前置条件：** 先阅读 [`../../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md) 了解认证、全局参数和安全规则。

拒绝一个 Pull Request（调用 `refuse_merge`，拒绝合并并关闭 PR）。

## 命令

```bash
# 拒绝 PR
gitlink-cli pr +refuse --id 3

# 简写
gitlink-cli pr +refuse -i 3
```

## 参数

| 参数 | 必填 | 说明 |
|------|------|------|
| `--id` / `-i` | 是 | PR 序号（`pull_request_number`） |

## API

```
POST /{owner}/{repo}/pulls/{number}/refuse_merge
```

## Workflow

> [!CAUTION]
> This is a **destructive Write Operation** -- confirm user intent before executing.

1. 使用 `pr +view -i <id>` 确认 PR 状态为 open（`pull_request_status: 0`）
2. 确认用户确实要**拒绝**此 PR（此操作不可逆，会关闭 PR 并标记为已拒绝）
3. 执行 `pr +refuse -i <id>`

## 注意事项

- 此操作调用 `refuse_merge` 端点，即**拒绝合并**
- 拒绝后 PR 状态变为 closed（`pull_request_status: 2`）
- ⛔ **不要用 `pr +refuse` 代替 `pr +merge`**：如果 PR 已被手动合并到 master，不要使用此命令来关闭 PR
- 需要对仓库有相应权限

## References

- [gitlink-shared SKILL.md](../../gitlink-shared/SKILL.md) -- 认证与全局参数
- [gitlink-pr SKILL.md](../SKILL.md) -- PR 操作总览
