# issue 详情增强与更新保护

这个变更聚焦修复 `issue` 快捷命令里两个容易影响实际使用的问题。

`issue +view` 之前只读取 v1 详情接口，返回结果里缺少网页端常见的状态、优先级、跟踪器和标签信息，用户很难直接把 CLI 输出和网页上的 issue 页面对应起来。这次调整后，命令会继续以 v1 接口为主，再补充读取旧版详情与编辑接口，在不影响主流程可用性的前提下，把 `number`、`database_id`、`tracker_id`、`issue_type`、`issue_tag_ids`、`issue_tag_names` 等信息一起带出来。

`issue +update`、`issue +close` 和 `issue +batch-close` 之前只保留了部分字段，更新时可能把现有 issue 的 `tracker_id`、`fixed_version_id`、`assigned_to_id`、`issue_type` 等服务端依赖字段丢掉，导致网页上出现状态异常或字段被误清空。现在这些命令会先读取 issue 的编辑元数据，再把关键字段一并回写；如果编辑元数据拉取失败，就直接终止更新，避免发送不完整的 PATCH 请求。

为了防止这类问题回归，这次补充了 `shortcuts/issue` 的单元测试，覆盖了详情增强、元数据保留、编辑元数据失败时停止写入，以及批量关闭复用同一套保护逻辑的场景。
