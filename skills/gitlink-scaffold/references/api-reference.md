# gitlink-scaffold API 参考

> **前置条件：** 先阅读 [`../../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md)。

本技能检测社区健康文件所依赖的接口与字段。采集只读；模板仅生成到本地。

## 采集的接口

### 列目录（检测文件是否存在）

```
GET /:owner/:repo/sub_entries.json?filepath={dir}&ref={ref}
# 或经 gitlink-cli：
gitlink-cli api GET /:owner/:repo/sub_entries --query 'filepath={dir}&ref=master' --format json
```

- 查询**目录**时，`entries` 为条目数组，每个含 `name` / `type`(file|dir) / `sha` / `size`。
- 查询**单文件**时，`entries` 可能为单个对象（本技能已做归一化处理）。

本技能会在以下目录查找社区文件：根目录、`.gitlink/`、`.github/`、`docs/`。

## 检测的字段

| 字段 | 说明 | 用途 |
|------|------|------|
| `entries[].name` | 条目名 | 与候选文件名（不区分大小写）匹配 |
| `entries[].type` | file / dir | 只匹配 file |

## 写操作（提交模板）

把生成的模板提交到仓库属于写操作。GitLink 创建文件接口需 base64 编码内容：

```
gitlink-cli api POST /:owner/:repo/create_file --body '{
  "filepath": "CONTRIBUTING.md",
  "content": "<base64编码>",
  "branch": "<分支>",
  "message": "docs: add CONTRIBUTING"
}'
```

> 注意：gitlink-shared 记录了 Create File 接口的已知问题，提交前请参考其说明，并务必征得用户确认。本技能默认只在本地生成模板。

## 错误处理

沿用 gitlink-shared 错误码。某目录不存在（404）时本技能视为该目录无文件，继续检查其他目录，不中断。
