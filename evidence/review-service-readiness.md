# Review service readiness evidence

`json
{
    "schema_version":  "review.service-readiness-evidence/v1",
    "passed":  true,
    "capability":  "service_health_readiness",
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
    "migrations":  [
                       16,
                       17,
                       18,
                       19
                   ],
    "required_components":  8,
    "health_status":  "ok",
    "ready_status":  "ready_and_draining_covered"
}
`

This evidence is generated with temporary SQLite, loopback listeners, fake clients, and synthetic or hashed identifiers. Real external writes are zero.
