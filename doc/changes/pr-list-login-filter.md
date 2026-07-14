# PR 列表按作者过滤

## Summary

`pr +list` 新增 `--login` 参数，可按 Issue 作者（`issue.author.login`）过滤返回的 Pull Request 列表。

## 新增 Flag

| Flag | 说明 |
|------|------|
| `--login` | 按 Issue 作者的 login 过滤 PR（不区分大小写） |

## 实现细节

GitLink PR 列表 API 不支持服务端按作者过滤，因此本功能在客户端对返回结果进行过滤。过滤逻辑：

- 匹配 `issue.author.login` 字段
- 不区分大小写
- 过滤后自动更新 `total_count` 和 `meta.total_count` 以反映实际数量

## 示例

```bash
# 列出 alice 提交的所有 PR
gitlink-cli pr +list --owner Gitlink --repo gitlink-cli --login alice

# 结合其他筛选条件
gitlink-cli pr +list --owner Gitlink --repo gitlink-cli --login alice --state open

# 列出已合并的 PR
gitlink-cli pr +list --login bob --state merged
```

