# Feishu Review queue limits and coalescing

## Default admission limits

| Limit | Default |
| --- | ---: |
| active Jobs globally | 2,000 |
| active Operations globally | 5,000 |
| active Jobs per chat | 100 |
| active Jobs per user | 20 |
| active Operations per chat | 200 |
| user commands per minute | 10 |
| chat user commands per minute | 60 |
| refresh coalescing window | 10 seconds |

Capacity checks happen in the same SQLite transaction as Job creation, Consumer creation and dedupe reservation. A rejected request leaves no orphan reservation.

User/chat rate limits apply only to `user_command`. Webhook/event routes, replay and reconciliation are not charged to a user's command bucket, but all sources still obey global and per-chat capacity.

## Queue classes

- `gitlink_read`: read queue/context, refresh, draft and read-only listing.
- `collaboration`: claim/release/deadline/subscription/default-repository/help operations.
- `controlled_write`: confirmed common Review write path.
- `agent`: reserved and disabled by default.

Unsupported action names are rejected instead of silently falling into a default queue.

## Refresh coalescing

Only `refresh_review_context` Jobs coalesce. The key is derived from:

```text
installation + chat + repository + PR number
```

Within the ten-second window, a queued or running Job is reused. The second admission returns the existing Job ID and appends a Job Consumer. This is coalescing, not dropping: every user message can still receive an independent reply Operation after the shared GitLink read completes.

Different chats never coalesce. Collaboration actions and controlled writes never coalesce. When the window expires, the next refresh creates a new Job. Event-triggered and scheduled-reconciliation refreshes can share the same in-flight refresh for the same scope.

## Dedupe versus coalescing

- Dedupe answers “have we already admitted this delivery?” and uses the delivery's deterministic key.
- Coalescing answers “can these distinct refresh requests share one current read?” and uses scope plus time window.
- Consumer identity answers “which source messages or routes need their own downstream notification?”

These concepts are stored separately so a duplicate delivery does not create a consumer, while a legitimate second request can share execution and retain its reply.

## Backpressure response

Capacity and rate-limit errors are typed. User-facing handling maps them to a bounded “current chat has many Review requests, retry later” response without exposing queue counts, user IDs or database details.

## Operational boundary

Limits are process configuration defaults backed by SQLite. Dynamic configuration revisions, management APIs, metrics and distributed enforcement are not implemented in this stage. Queue evidence is synthetic and offline; it does not establish production throughput for Feishu or GitLink.
