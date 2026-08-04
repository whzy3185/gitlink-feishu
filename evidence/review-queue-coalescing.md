# Review queue coalescing evidence

`json
{
    "schema_version":  "feishu.review-queue-coalescing-evidence/v1",
    "passed":  true,
    "storage":  "temporary_sqlite",
    "window_seconds":  10,
    "coalesced_job_count":  3,
    "retained_consumer_count":  2,
    "gitlink_reads_for_two_same_scope_refreshes":  1,
    "different_chat_coalesces":  0,
    "collaboration_coalesces":  0,
    "controlled_write_coalesces":  0,
    "simulated_duplicate_remote_writes":  false,
    "identifiers":  "synthetic_or_hashed",
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

The run uses temporary SQLite, httptest or fake clients. Identifiers are synthetic or hashed. GitLink POST=0 and real external writes=0.
