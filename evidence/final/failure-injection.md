# failure-injection

- Schema: `feishu.review-final-failure-injection/v1`
- Passed: `True`
- Validation mode: `offline`
- Generated: `2026-08-04T16:51:52.9969771Z`

```json
{
    "schema_version":  "feishu.review-final-failure-injection/v1",
    "passed":  true,
    "validation_mode":  "offline",
    "cases_expected":  20,
    "cases_passed":  20,
    "unknown_blind_retries":  0,
    "non_idempotent_duplicates":  0,
    "real_external_writes":  0,
    "cases":  [
                  {
                      "id":  "F01",
                      "name":  "card create succeeds and local save fails",
                      "passed":  true,
                      "invariant":  "unknown and no blind create",
                      "duration_ms":  3795
                  },
                  {
                      "id":  "F02",
                      "name":  "card patch timeout",
                      "passed":  true,
                      "invariant":  "idempotent patch schedules a safe retry",
                      "duration_ms":  3073
                  },
                  {
                      "id":  "F03",
                      "name":  "reply send timeout",
                      "passed":  true,
                      "invariant":  "unknown and no blind reply",
                      "duration_ms":  2575
                  },
                  {
                      "id":  "F04",
                      "name":  "base create succeeds and local save fails",
                      "passed":  true,
                      "invariant":  "unknown and reconcilable by unique key",
                      "duration_ms":  2926
                  },
                  {
                      "id":  "F05",
                      "name":  "base multiple record conflict",
                      "passed":  true,
                      "invariant":  "manual reconciliation",
                      "duration_ms":  2410
                  },
                  {
                      "id":  "F06",
                      "name":  "doc append uncertain",
                      "passed":  true,
                      "invariant":  "append not repeated",
                      "duration_ms":  2234
                  },
                  {
                      "id":  "F07",
                      "name":  "task create uncertain",
                      "passed":  true,
                      "invariant":  "task create not repeated",
                      "duration_ms":  2336
                  },
                  {
                      "id":  "F08",
                      "name":  "rate limit and retry-after",
                      "passed":  true,
                      "invariant":  "retry schedule honors server hint",
                      "duration_ms":  2379
                  },
                  {
                      "id":  "F09",
                      "name":  "upstream 503",
                      "passed":  true,
                      "invariant":  "classified transient",
                      "duration_ms":  1735
                  },
                  {
                      "id":  "F10",
                      "name":  "sqlite busy",
                      "passed":  true,
                      "invariant":  "cached metrics and no corruption",
                      "duration_ms":  2164
                  },
                  {
                      "id":  "F11",
                      "name":  "worker lease expires",
                      "passed":  true,
                      "invariant":  "expired lease recovers",
                      "duration_ms":  2319
                  },
                  {
                      "id":  "F12",
                      "name":  "stale worker",
                      "passed":  true,
                      "invariant":  "stale completion fenced",
                      "duration_ms":  2457
                  },
                  {
                      "id":  "F13",
                      "name":  "process terminates after remote request",
                      "passed":  true,
                      "invariant":  "remote possible enters reconciliation",
                      "duration_ms":  1855
                  },
                  {
                      "id":  "F14",
                      "name":  "graceful shutdown with non-idempotent operation",
                      "passed":  true,
                      "invariant":  "unknown is retained",
                      "duration_ms":  3788
                  },
                  {
                      "id":  "F15",
                      "name":  "wal grows during backup",
                      "passed":  true,
                      "invariant":  "consistent backup includes committed WAL data",
                      "duration_ms":  2842
                  },
                  {
                      "id":  "F16",
                      "name":  "restore verification fails",
                      "passed":  true,
                      "invariant":  "target database not replaced",
                      "duration_ms":  3740
                  },
                  {
                      "id":  "F17",
                      "name":  "retention interrupted",
                      "passed":  true,
                      "invariant":  "scheduler stops on cancellation",
                      "duration_ms":  2202
                  },
                  {
                      "id":  "F18",
                      "name":  "event processor restarts",
                      "passed":  true,
                      "invariant":  "inbox survives and route is unique",
                      "duration_ms":  2848
                  },
                  {
                      "id":  "F19",
                      "name":  "replay request duplicated",
                      "passed":  true,
                      "invariant":  "one replay request",
                      "duration_ms":  2686
                  },
                  {
                      "id":  "F20",
                      "name":  "high frequency same-PR events",
                      "passed":  true,
                      "invariant":  "one read job with retained consumers",
                      "duration_ms":  2836
                  }
              ]
}
```
