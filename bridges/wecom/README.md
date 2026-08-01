# GitLink Enterprise WeChat Review Bridge

This sidecar uses the official `@wecom/aibot-node-sdk` long connection and
normalizes WeCom text frames into `gitlink.collab-inbound/v1`.

The stable P4 boundary is GET-only:

- one process per bot, enforced with a local instance lock;
- group and user allowlists are fail-closed when configured;
- raw identifiers are hashed in observations;
- message IDs are stored only as SHA-256 keys in a bounded durable dedupe
  journal, so SDK retries remain idempotent across restarts;
- the bridge calls a fixed local Review Core HTTP endpoint and never builds or
  executes shell commands;
- no GitLink token or WeCom bot secret is sent to the Review Core;
- merge, approval, rejection, line comment, and reviewer mutation commands are
  not accepted.

Run:

```powershell
npm install
$env:WECOM_BOT_ID = "..."
$env:WECOM_BOT_SECRET = "..."
$env:WECOM_ALLOWED_CHAT_IDS = "..."
$env:WECOM_ALLOWED_USER_IDS = "..."
$env:GITLINK_REVIEW_CORE_URL = "http://127.0.0.1:8765/v1/review/inbound"
$env:GITLINK_REVIEW_CORE_TOKEN = "one-random-local-bridge-token"
npm start
```

At least one chat/user allowlist is required by default. `WECOM_ALLOW_ALL=true`
is available only as an explicit development override.

Without `GITLINK_REVIEW_CORE_URL`, the bridge is an observe-only SDK smoke
environment and replies with a controlled boundary message.

Start the repository-provided loopback Review Core in another terminal with the
same token:

```powershell
$env:GITLINK_REVIEW_CORE_TOKEN = "one-random-local-bridge-token"
go run . wecom +review-core --repository Gitlink/gitlink-cli
```

For multiple explicitly authorized repositories:

```powershell
go run . wecom +review-core `
  --repositories Gitlink/gitlink-cli,owner/second `
  --default-repository Gitlink/gitlink-cli
```

Qualified commands use `查看 owner/repo PR #42` or
`查看 owner/repo 待审查`. If multiple repositories are allowed and no default
is configured, unqualified commands fail with `repository_required`.

The dedupe journal defaults to `.local/wecom-review-dedupe.json`. Override its
location with `WECOM_DEDUPE_JOURNAL`; TTL and capacity are bounded through
`WECOM_DEDUPE_TTL_SECONDS` and `WECOM_DEDUPE_MAX_ENTRIES`. A journal read or
write failure denies the event instead of risking a duplicate Review query.
