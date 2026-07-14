# Search: Recommended Projects

## Summary

Adds `search +recommend`, which lists the platform's recommended / featured
projects. This wraps `GET /api/projects/recommend`, a discovery endpoint that
previously had no shortcut coverage.

## Command

| Command | Purpose | Endpoint |
|---------|---------|----------|
| `gitlink-cli search +recommend` | List recommended/featured projects | `GET /projects/recommend` |

Each item includes `id`, `identifier`, `name`, `visits`, `author`
(`name`/`login`/`image_url`) and `category`.

## Behaviour

- Takes no arguments. Honors the global `--format` (json/table/yaml).
- The endpoint returns a bare JSON array; the command normalizes it into
  structured data so the output is clean (not an escaped JSON string).

## Tests

Unit tests cover the endpoint path, the bare-array normalization, and HTTP
error handling.

## 中文说明

### 变更内容

- 新增 `search +recommend`，列出平台推荐/精选项目，封装此前无 shortcut 的
  `GET /api/projects/recommend`。
- 无需参数，遵循全局 `--format`；对该接口返回的裸 JSON 数组做结构化归一，输出整洁。

### 价值

为项目发现提供入口（科研选题/技术调研时可快速看到平台精选项目），补全 Raw API 封装。
