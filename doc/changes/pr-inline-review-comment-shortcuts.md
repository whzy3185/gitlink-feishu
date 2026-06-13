# PR 行级审查评论快捷命令

这次改动把 `pull request` 里的行级审查评论补成了一套完整的快捷命令，而不只是停留在普通会话评论或 review 总览。

- 新增 `pr +review-comments`，可以按 `review_id`、`state`、`path`、`need_respond` 等条件筛选 inline review comments。
- 新增 `pr +review-comment`、`pr +update-review-comment`、`pr +delete-review-comment`，把创建、更新、删除行级评论的常用操作补齐。
- `pr +review-comment` 默认会自动调用 PR files 接口，按 `--path` 提取对应文件的 diff 并转换成评论接口可用的结构，减少手工拼接大段 `diff` JSON 的负担；如果需要完全自定义，也可以通过 `--diff-file` 直接提供 diff JSON。
- 创建、更新、删除都支持 `--dry-run`，方便在脚本或 Agent 场景里先预览最终请求内容。

这条能力比较适合代码审查自动化、Agent 辅助 review、或者把外部静态分析结果回写到具体变更行上，比单纯暴露原始接口更容易直接落到实际工作流里。

本地验证：

```bash
go test ./shortcuts/pr
go test ./...
go build ./...
git diff --check
go run . pr --help
```
