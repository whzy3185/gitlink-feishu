# Feishu real validation gap

Baseline: `feat/common-review-write-final` at `68c0547c4901a844d8a2d433decb5f18003c29f3`.

| Capability | Existing Evidence | Current Status | This Run Action |
| --- | --- | --- | --- |
| GitLink PR / Review read | Real common Review preparation and read-back used the live PR context | PASS | Preserve evidence; perform read-only preflight |
| Canonical Card / Reply | Independent create, patch and reply returned HTTP 200; canonical PR message was reused | PASS | Preserve `real-card-reply.json` and operation evidence |
| Claim / Release / Deadline | Real chat state persisted through claim, deadline and release; one canonical message was reused | PASS | Preserve `real-collaboration.json` |
| Dual Chat Isolation | Offline isolation tests; only one real chat is configured | BLOCKED_BY_ENVIRONMENT | Do not claim PASS without a second test chat |
| Base Projection | Offline tests; no real Base/Table target is configured | BLOCKED_BY_ENVIRONMENT | Run only after an explicit test Base/Table is supplied |
| Doc Snapshot | Offline tests; no real Doc folder target is configured | BLOCKED_BY_ENVIRONMENT | Run only after an explicit test folder is supplied |
| Task Projection | Offline tests; no real Task assignee is configured | BLOCKED_BY_ENVIRONMENT | Run only after an explicit test assignee is supplied |
| Webhook / Event Refresh | Offline tests; no stable public callback is configured | BLOCKED_BY_ENVIRONMENT | Do not create temporary deployment architecture |
| Common Review | Real Review `130`, GET read-back, duplicate zero and Feishu result evidence | PASS | Preserve `real-common-review.json`; no repeat mutation |
| Approve | Real Review `131`, approved GET read-back and duplicate zero | PASS | Preserve `real-approve.json` |
| Reject Review | Real Review `132`, rejected GET read-back, PR remained open and duplicate zero | PASS | Preserve `real-reject.json` |
| Reject & Close | One native refuse/close, zero Review creation, closed GET read-back and duplicate zero | PASS | Preserve `real-refuse.json` |
| Controlled Merge | One native merge into disposable base; default branch unchanged and duplicate zero | PASS | Preserve `real-merge.json`; CI remained unknown |
| Head Stale | Real old-head ActionPlan was persisted stale with zero POST | PASS | Preserve `real-stale.json` |
| Fingerprint-only Stale | Offline regression only | NEEDS_REAL_TEST | Optional after primary acceptance; zero mutation required |

Real external writes remain gated by both `FEISHU_REVIEW_REAL_VALIDATION=1` and
`FEISHU_REVIEW_CONFIRM_EXTERNAL_WRITES=YES`. Evidence must contain only hashed
or sanitized platform identifiers and must never contain credentials.
