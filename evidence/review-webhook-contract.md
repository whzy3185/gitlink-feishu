# Review Webhook contract evidence

`json
{
    "schema_version":  "feishu.review-webhook-contract-evidence/v1",
    "passed":  true,
    "schema_migrations":  [
                              7,
                              8,
                              9,
                              10
                          ],
    "handler_writes_inbox_only":  true,
    "handler_synchronous_jobs":  0,
    "duplicate_deliveries":  1,
    "processor_jobs":  1,
    "inbox_status_counts":  {
                                "normalized":  1,
                                "processed":  1,
                                "ignored":  1,
                                "failed":  0
                            },
    "route_status_counts":  {
                                "queued":  1,
                                "duplicate":  1,
                                "skipped":  0,
                                "failed":  0
                            },
    "signature_modes":  [
                            "body_sha256",
                            "timestamp_body_sha256"
                        ],
    "timestamp_modes":  [
                            "required",
                            "optional",
                            "disabled"
                        ],
    "external_writes":  {
                            "gitlink_get":  0,
                            "gitlink_post":  0,
                            "feishu_message":  0,
                            "base":  0,
                            "doc":  0,
                            "task":  0
                        }
}
`

All identifiers are synthetic or hashed. GitLink POST=0 and real external writes=0.
