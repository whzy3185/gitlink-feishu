# Review worker isolation evidence

`json
{
    "schema_version":  "feishu.review-worker-isolation-evidence/v1",
    "passed":  true,
    "storage":  "temporary_sqlite",
    "queue_class_counts":  {
                               "gitlink_read":  1,
                               "collaboration":  1,
                               "controlled_write":  1,
                               "agent":  0,
                               "canonical_card":  1,
                               "reply":  1,
                               "resource":  2
                           },
    "default_concurrency":  {
                                "event":  2,
                                "gitlink_read":  3,
                                "collaboration":  1,
                                "planner":  2,
                                "canonical_card":  2,
                                "reply":  2,
                                "resource":  2,
                                "controlled_write":  1,
                                "agent":  0,
                                "operation_reconciliation":  1
                            },
    "capacity_rejections":  4,
    "rate_limited_requests":  2,
    "sqlite_transaction_held_during_network":  false,
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
