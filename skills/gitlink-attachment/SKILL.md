---
name: gitlink-attachment
version: 1.0.0
description: "附件管理：上传文件、删除附件，适用于 Issue/PR/数据集等需要附件 ID 的工作流。"
metadata:
  requires:
    bins: ["gitlink-cli"]
  cliHelp: "gitlink-cli attachment --help"
---

# gitlink-attachment（附件管理）

**CRITICAL — 开始前必须先阅读 [`../gitlink-shared/SKILL.md`](../gitlink-shared/SKILL.md)，其中包含认证、权限处理和 API 注意事项。**
**CRITICAL — 上传/删除附件属于写操作。执行真实写入前，优先使用 `--dry-run` 预览并确认用户意图。**

## Shortcuts

| Shortcut | 说明 | 操作类型 |
|----------|------|----------|
| `attachment +upload` | 上传本地文件，返回附件 UUID/URL 等信息 | ⚠️ Write Operation |
| `attachment +delete` | 按 UUID 删除附件 | 🔴 Destructive Operation |

## 参数参考

### attachment +upload

| 参数 | 必填 | 说明 |
|------|------|------|
| `--file, -f` | 是 | 本地文件路径 |
| `--description, -d` | 否 | 附件描述 |
| `--container-id` | 否 | 归属模型 ID，例如 Issue/PR/数据集记录 ID |
| `--container-type` | 否 | 归属模型类型 |
| `--dry-run` | 否 | 只预览 multipart 字段，不上传文件 |
| `--format` | 否 | 输出格式：`json`/`table`/`yaml` |

### attachment +delete

| 参数 | 必填 | 说明 |
|------|------|------|
| `--uuid, -u` | 是 | 附件 UUID |
| `--dry-run` | 否 | 只预览删除请求，不删除附件 |
| `--format` | 否 | 输出格式：`json`/`table`/`yaml` |

## 使用示例

```bash
# 预览上传，不修改线上数据
gitlink-cli attachment +upload --file screenshot.png --description "复现截图" --dry-run

# 上传附件
gitlink-cli attachment +upload --file screenshot.png --description "复现截图"

# 上传并绑定到业务对象
gitlink-cli attachment +upload --file design.pdf \
  --description "设计文档" \
  --container-id 123 \
  --container-type Issue

# 预览删除
gitlink-cli attachment +delete --uuid f5838d8f-451b-4793-a0f2-0278430e8207 --dry-run

# 删除附件
gitlink-cli attachment +delete --uuid f5838d8f-451b-4793-a0f2-0278430e8207
```

## Agent 工作流建议

1. 确认用户要上传或删除的文件/附件 UUID。
2. 写操作先执行 `--dry-run --format json`，展示将要调用的方法、路径和字段。
3. 用户确认后再执行真实命令。
4. 对上传结果，保存返回的 `id`/`url`，后续可作为 Issue、评论或数据集附件引用。

## References

- [gitlink-shared](../gitlink-shared/SKILL.md) — 认证、全局参数、安全规则
- GitLink OpenAPI：`POST /api/attachments.json`、`DELETE /api/attachments/{uuid}.json`
