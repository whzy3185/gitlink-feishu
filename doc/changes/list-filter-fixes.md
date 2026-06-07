# Issue/PR 列表筛选增强

本次变更修正并增强 `issue +list` 与 `pr +list` 的筛选能力。

此前 `issue +list --state open` 会向服务端发送 `state=open`，但 GitLink v1 Issue 列表接口实际使用 `category=opened/closed/all`，因此列表可能仍返回关闭 Issue。`pr +list --state open` 也没有映射到 PR 列表接口实际使用的 `status=0/1/2` 参数，Skill 文档中甚至需要提醒用户该参数可能只影响统计。现在两个命令都会保留原有 `--state` 用户体验，同时转换为服务端真实生效的参数。

新增筛选项：

- `issue +list` 支持 `--keyword`、`--participant`、`--author-id`、`--assignee-id`、`--milestone-id`、`--status-id`、`--tag-ids`、`--sort-by`、`--sort-direction`。
- `pr +list` 支持 `--keyword`、`--priority-id`、`--tag-id`、`--milestone-id`、`--reviewer-id`、`--assignee-id`、`--sort-by`、`--sort-direction`。

单元测试覆盖了状态映射、筛选参数透传和 `all` 状态兼容；README、中文 README、Issue Skill 与 PR Skill 已同步更新。
