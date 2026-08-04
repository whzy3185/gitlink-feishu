# linux-race

- Schema: `unspecified`
- Passed: `False`
- Validation mode: `offline`
- Generated: `2026-08-04T16:51:52.9969771Z`

```json
{
    "schema_version":  "feishu.review-linux-race/v1",
    "passed":  false,
    "validation_mode":  "offline",
    "status":  "pending_final_actions",
    "command":  "CGO_ENABLED=1 go test -race ./internal/collab ./shortcuts/feishu ./shortcuts/wecom ./shortcuts/workflow -count=1"
}
```
