# Review Event subscriptions evidence

`json
{
    "schema_version":  "feishu.review-event-subscription-evidence/v1",
    "passed":  true,
    "schema_migrations":  [
                              7,
                              8,
                              9,
                              10
                          ],
    "subscription_count":  2,
    "enabled_subscription_count":  2,
    "event_groups":  [
                         "pulls",
                         "reviews",
                         "threads",
                         "merge",
                         "ci"
                     ],
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
    "duplicate_deliveries":  1,
    "replay_count":  1,
    "reconciliation_candidates":  1,
    "created_jobs":  1,
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
