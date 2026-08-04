# Review dead letter and reconciliation evidence

`json
{
    "schema_version":  "feishu.review-dead-letter-reconciliation-evidence/v1",
    "passed":  true,
    "unknown_count":  4,
    "dead_letter_count":  2,
    "reconciliation_count":  3,
    "automatic_non_idempotent_retries":  0,
    "manual_required_resources":  [
                                      "feishu_card",
                                      "feishu_reply",
                                      "feishu_doc",
                                      "feishu_task"
                                  ],
    "automatic_reconciliation_resources":  [
                                               "feishu_bitable_by_unique_key"
                                           ],
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
