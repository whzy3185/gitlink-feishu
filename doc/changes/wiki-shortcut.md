# Wiki Shortcut

新增 `wiki` Shortcut 组，支持 Wiki 页面管理：

- `wiki +list` - 列出 Wiki 页面（目录结构）
- `wiki +view` - 按页面名称查看 Wiki 页面详情
- `wiki +create` - 创建新的 Wiki 页面
- `wiki +update` - 更新 Wiki 页面标题和/或内容
- `wiki +delete` - 删除 Wiki 页面

## 实现要点

- **API 端点**：基于 `/api/wiki/open/{action}` 扁平路径结构，覆盖 5 个 Wiki 管理接口：
  - `GET /api/wiki/open/wikiPages` — 目录列表
  - `GET /api/wiki/open/getWiki` — 查看页面
  - `POST /api/wiki/open/createWiki` — 创建页面
  - `PUT /api/wiki/open/updateWiki` — 更新页面
  - `DELETE /api/wiki/open/deleteWiki` — 删除页面
- **标识方式**：Wiki 页面通过 `pageName`（slug）标识，所有操作需要 `projectId`（GitLink 项目数字 ID）
- **内容编码**：创建和更新时，内容自动进行 base64 编码后以 `content_base64` 字段发送
- **更新保护**：`+update` 要求必须提供 `--title` 和 `--page-name`；`--content` 为可选
- **Shortcut 模式**：使用 `common.Shortcut` + `RuntimeContext` 框架，与其他模块保持一致

