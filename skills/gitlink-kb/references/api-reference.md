# gitlink-kb API 参考

> **前置条件：** 先阅读 [`../../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md)。

本技能索引仓库文档所依赖的接口与字段。全程只读。

## 采集的接口

### README

```
GET /:owner/:repo/readme.json?ref={ref}
# 或经 gitlink-cli：
gitlink-cli repo +readme --owner <owner> --repo <repo> --ref master --format json
```

返回 `content` 字段。注意：GitLink 的 readme 接口虽将 `encoding` 标为 base64，
实测 `content` 多为**明文** Markdown，本技能会先探测明文特征，必要时再做 base64 解码。

### 列目录与读取文档

```
GET /:owner/:repo/sub_entries.json?filepath={dir}&ref={ref}     # 列目录
GET /:owner/:repo/sub_entries.json?filepath={file}&ref={ref}    # 读单文件（entries.content 为明文）
```

本技能在根目录、`docs/`、`doc/`、`.gitlink/`、`wiki/` 中查找文档文件。

## 使用的字段

| 字段 | 说明 | 用途 |
|------|------|------|
| readme `content` | README 内容 | 索引 |
| `entries[].name` | 文件名 | 筛选文档扩展名 |
| `entries[].type` | file / dir | 只索引 file |
| `entries[].content` | 单文件明文内容 | 索引正文 |

## 输出字段（JSON）

检索：

```json
{"query": "如何安装",
 "results": [{"doc": "README", "title": "安装", "score": 7, "snippet": "..."}]}
```

文档地图：`{"README": [{"title": "安装", "level": 2}, ...]}`

FAQ：`{"faq": [{"question": "...", "answer": "...", "doc": "README"}]}`

## 错误处理

沿用 gitlink-shared 错误码。某目录不存在时跳过，不中断索引。
