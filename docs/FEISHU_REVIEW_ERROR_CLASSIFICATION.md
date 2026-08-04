# Feishu Review error classification

## Classification contract

Every Operation execution error is normalized before a state transition.

| Class | Typical input | Result |
| --- | --- | --- |
| `terminal` | HTTP 400, 401, 403, 404; invalid desired payload; missing configuration | terminal failure and dead letter |
| `rate_limited` | HTTP 429 | safe retry at `Retry-After` when allowed |
| `transient` | HTTP 5xx, SQLite busy, network/timeout before a possible non-idempotent effect | bounded retry |
| `stale` | lease/fingerprint/create ownership no longer current | no remote write and no retry |
| `unknown_side_effect` | timeout/network/5xx after a non-idempotent request may have started | reconciliation, never blind retry |

HTTP status and `Retry-After` are captured structurally. Error text is redacted before persistence or display.

## Retry safety

### Idempotent

Repeated application has the same remote outcome. Existing card patch, Base update and Task patch are treated as idempotent. Transient errors may retry up to the Operation attempt limit.

### Reconcilable

The remote state has a stable lookup method. Base upsert is reconcilable because the stable resource key can be searched. An uncertain create is checked by key before any further write.

### Non-idempotent

Repeated application may create duplicate user-visible resources. Card create, reply send, Doc create/append and Task create are non-idempotent. Once a request might have reached the platform, timeout or transport failure becomes `unknown_side_effect`.

## HTTP behavior

- 400/401/403/404: terminal; no automatic retry.
- 429: `rate_limited`; honor server `Retry-After` when present.
- 500–599: transient for idempotent work; unknown for an in-flight non-idempotent write.
- Missing remote ID after a nominal create response: unknown.

A 401 also invalidates the tenant-token cache for that App ID. The failed Operation remains terminal; token invalidation only affects future work.

## Attempt budget

Default maximum attempts are three. A safe transient failure schedules exponential delay unless the platform supplies Retry-After. When the budget is exhausted, the Operation enters `dead_letter`. The system does not turn an unknown non-idempotent effect into a retry merely because attempts remain.

## Local persistence after remote success

If a remote create reports success but the remote ID or fingerprint cannot be saved locally, the result is `unknown`, not success and not failure. The returned remote ID is retained where possible so an administrator can reconcile it.

## Logging and secrecy

Desired payloads reject secret-shaped keys. Persisted errors are redacted for bearer tokens, cookies, URLs and sensitive response fragments. Administrative views use hashes or redacted identifiers and do not expose full remote IDs.
