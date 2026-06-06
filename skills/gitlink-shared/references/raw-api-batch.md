# Raw API 批处理

当某个 GitLink OpenAPI 尚未封装为 shortcut，但需要重复执行多步请求时，可以使用 `api --batch-file` 读取 JSON 计划文件。批处理会复用 gitlink-cli 的认证、base_url、输出格式和错误处理，并返回结构化汇总。

## 示例

```json
{
  "vars": {
    "owner": "Gitlink",
    "repo": "gitlink-cli",
    "issue": "123"
  },
  "requests": [
    {
      "name": "list-open-issues",
      "method": "GET",
      "path": "/v1/{{owner}}/{{repo}}/issues",
      "query": {
        "state": "open",
        "limit": 20
      }
    },
    {
      "name": "comment-issue",
      "method": "POST",
      "path": "/v1/{{owner}}/{{repo}}/issues/{{issue}}/journals",
      "body": {
        "notes": "批处理自动评论"
      }
    }
  ]
}
```

```bash
gitlink-cli api --batch-file plan.json --dry-run
gitlink-cli api --batch-file plan.json --var issue=456
gitlink-cli api --batch-file plan.json --continue-on-error --format json
```

## 字段

| 字段 | 必填 | 说明 |
|------|------|------|
| `vars` | 否 | 模板变量，支持在 name/path/query/body 字符串中使用 `{{name}}` |
| `requests` | 是 | 请求数组，至少包含一个请求 |
| `requests[].name` | 否 | 步骤名称，会出现在结果汇总中 |
| `requests[].method` | 是 | HTTP 方法，如 GET、POST、PUT、PATCH、DELETE |
| `requests[].path` | 是 | API 路径，可省略开头的 `/` |
| `requests[].query` | 否 | 查询参数对象，值可为字符串、数字、布尔值或数组 |
| `requests[].body` | 否 | JSON 请求体，字符串字段会做模板替换 |

## 注意事项

- 默认遇到失败会停止；需要继续执行后续步骤时传 `--continue-on-error`。
- 写入类操作先用 `--dry-run` 检查渲染后的路径、query 和 body。
- `--var key=value` 可重复传入，并覆盖计划文件里的同名变量，适合在不同仓库或 Issue 上复用同一计划。
- 批处理模式不能和单次请求的 `--body`、`--body-file`、`--body-stdin`、`--query`、`--header` 混用。
