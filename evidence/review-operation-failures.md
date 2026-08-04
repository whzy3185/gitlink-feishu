# Review operation failures evidence

`json
{
    "schema_version":  "feishu.review-operation-failure-evidence/v1",
    "passed":  true,
    "storage":  "temporary_sqlite",
    "transport":  "fake_openapi_clients",
    "injected_classes":  [
                             "terminal",
                             "transient",
                             "rate_limited",
                             "unknown_side_effect",
                             "stale"
                         ],
    "http_statuses":  [
                          400,
                          401,
                          403,
                          404,
                          429,
                          503
                      ],
    "unknown_count":  4,
    "dead_letter_count":  2,
    "reconciliation_count":  3,
    "automatic_non_idempotent_retries":  0,
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
