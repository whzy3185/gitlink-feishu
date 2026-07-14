# gitlink-changelog API 参考

> **前置条件：** 先阅读 [`../../gitlink-shared/SKILL.md`](../../gitlink-shared/SKILL.md)。

本技能全程只读。

## 采集的接口

### 提交列表

```
GET /:owner/:repo/commits.json?page=<n>&limit=50
```

每页硬上限 50 条，以 `total_count` 为终止依据。使用字段：

| 字段 | 用途 |
|------|------|
| `commits[].message` | 解析 conventional 类型、scope、描述、BREAKING |
| `commits[].sha` | 报告中标注短 sha |
| `commits[].author.login` / `.name` | 提取贡献者 |

### 版本发布列表

```
GET /:owner/:repo/releases.json
```

用于 `--since`/`--until` 定位版本时间窗口。字段：`tag_name` / `name` / `created_at`。

## 关于 compare 接口

GitLink 的 `GET /:owner/:repo/compare/{base}...{head}` 接口需要鉴权（公开访问返回 401）。
因此本技能不依赖 compare，而是用提交列表分析，无需登录即可处理公开仓库。

## 输出字段（JSON）

```json
{
  "since": "v0.1.17", "until": "v0.1.18",
  "total_commits": 127, "typed_commits": 70, "breaking_count": 0,
  "groups": {"feat": 27, "fix": 16, "docs": 15, ...},
  "detail": {"feat": [{"type","scope","desc","sha","author","breaking"}]},
  "contributors": ["..."]
}
```

## 错误处理

沿用 gitlink-shared 错误码。采集失败返回非零退出码并打印原因。
