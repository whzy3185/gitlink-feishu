# Label clone shortcut

新增 `label +clone`，对齐 `gh label clone`：把源仓库的全部 Issue 标签复制到当前仓库。

- 用法：`label +clone --source owner/repo [--force]`。
- 语义与 `gh` 一致：按**名称**判重，目标已存在的同名标签默认**跳过**；仅在 `--force` 下就地**覆盖**（`PATCH` 标签 id，保留标签 id 与其 Issue 关联）。
- 纯组合已有端点：`GET issue_tags` 列举 + `POST` 新建 / `PATCH` 更新，不新增 API。
- 返回 `created` / `updated` / `skipped` 三组名称，便于查看每个标签的去向。

实现要点：

- 新增自包含的 `fetchLabelsForRepo(ctx, owner, repo)`，按 `page`/`limit` 翻页遍历 `issue_tags` 数组（与 `workflow` 的 `fetchAllListItems` 同一翻页范式），源仓库或目标仓库标签超过一页也能完整镜像。
- **未改动既有 `fetchLabel`**：上游 PR #363（`fix/label-update-pagination`）正在为 `fetchLabel` 加翻页，clone 走独立的 `fetchLabelsForRepo` 以避免合并冲突、也不重新引入单页 bug。
- 补充路径辅助 `repoLabelPath` / `repoLabelItemPath` 支持任意 owner/repo，原 `labelPath` / `labelItemPath` 改为其薄封装；`splitOwnerRepo` 解析 `owner/repo`（容忍首尾斜杠与多余尾部路径）。
- 单测覆盖：默认跳过同名、新建缺失标签、`--force` 就地 `PATCH`，以及 `fetchLabelsForRepo` 翻页遍历两页。
