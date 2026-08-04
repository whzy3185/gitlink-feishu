# Review service retention and WAL evidence

`json
{
    "schema_version":  "review.service-retention-wal-evidence/v1",
    "passed":  true,
    "capability":  "retention_wal",
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
    "retention_dry_run":  "passed",
    "retention_rows_deleted":  5,
    "wal_checkpoint":  "passed"
}
`

This evidence is generated with temporary SQLite, loopback listeners, fake clients, and synthetic or hashed identifiers. Real external writes are zero.
