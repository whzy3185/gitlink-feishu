# Review service graceful shutdown evidence

`json
{
    "schema_version":  "review.service-shutdown-evidence/v1",
    "passed":  true,
    "capability":  "graceful_shutdown",
    "storage":  "temporary_sqlite",
    "listener":  "loopback_only",
    "identifiers":  "synthetic_or_hashed",
    "external_writes":  {
                            "gitlink_get":  0,
                            "gitlink_post":  0,
                            "feishu_message":  0,
                            "base":  0,
                            "doc":  0,
                            "task":  0
                        },
    "admission_closed_first":  true,
    "unknown_operation_protected":  true,
    "blind_non_idempotent_retries":  0
}
`

This evidence is generated with temporary SQLite, loopback listeners, fake clients, and synthetic or hashed identifiers. Real external writes are zero.
