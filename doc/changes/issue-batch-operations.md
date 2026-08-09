# issue 批量运维能力增强

本次变更把 Issue 的批量运维能力从“只能批量关闭”扩展为更完整的日常工作流：

- 新增 `issue +batch-comment`，支持按 `--numbers` 或 `--from issues.csv` 给多个 Issue 统一追加评论。
- 新增 `issue +batch-update`，支持批量更新状态、优先级、标签、负责人、关联分支、开始日期和截止日期。
- `issue +create`、`issue +update`、`issue +comment` 现在支持 `--body-file`，适合读取 Markdown 文件中的长文本。

设计上延续了现有 `issue +batch-close` 的安全思路：

- 批量命令统一支持 `--dry-run`；
- `--body` 与 `--body-file` 互斥；
- 批量更新会先读取当前 Issue，再保留已有标题、描述和元数据，避免误清空字段；
- 输出统一包含逐条结果汇总，便于 Agent 或脚本继续处理。

相关文档已同步更新：

- `README.md`
- `skills/gitlink-issue/SKILL.md`

本地验证：

```bash
go test ./shortcuts/issue/...
go test ./shortcuts/...
go build ./...
git diff --check
go run . issue +batch-comment --help
go run . issue +batch-update --help
```
