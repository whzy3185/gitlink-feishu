# milestone +create 放宽必填参数

## 背景

`milestone +create` 之前把 `--description` 与 `--due-date` 标记为必填，
但平台 v1 API 创建里程碑仅要求 `name`（已在生产环境
`POST /api/v1/:owner/:repo/milestones` 用仅含 `{"name": ...}` 的请求验证成功）。
CLI 侧的额外限制迫使用户为可选字段编造内容。

## 变更

- `--description` 与 `--due-date` 改为可选；未提供时请求体中省略对应字段。
- `milestonePayload` 的 `requireAll` 参数收敛为 `requireName`，仅校验 `--name`。

## 验证

- `go test ./shortcuts/milestone/`（新增 `TestMilestoneCreateNameOnly`
  断言未提供的可选字段不出现在请求体中）
- `go vet ./...`
- 生产 gitlink.org.cn 实测：`milestone +create -n <name>` 仅带名称创建成功。
