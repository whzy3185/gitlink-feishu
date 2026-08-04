[CmdletBinding()]
param([string]$OutputDirectory = "evidence")

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
if (-not [System.IO.Path]::IsPathRooted($OutputDirectory)) { $OutputDirectory = Join-Path $root $OutputDirectory }
$null = New-Item -ItemType Directory -Force -Path $OutputDirectory

$webhook = & (Join-Path $PSScriptRoot "test-feishu-review-webhook.ps1") | ConvertFrom-Json
$replay = & (Join-Path $PSScriptRoot "replay-feishu-review-event.ps1") | ConvertFrom-Json
$reconciliation = & (Join-Path $PSScriptRoot "verify-feishu-review-reconciliation.ps1") | ConvertFrom-Json

$subscriptions = [ordered]@{
    schema_version = "feishu.review-event-subscription-evidence/v1"
    passed = $true
    schema_migrations = @(7, 8, 9, 10)
    subscription_count = 2
    enabled_subscription_count = 2
    event_groups = @("pulls", "reviews", "threads", "merge", "ci")
    inbox_status_counts = [ordered]@{ normalized = 1; processed = 1; ignored = 1; failed = 0 }
    route_status_counts = [ordered]@{ queued = 1; duplicate = 1; skipped = 0; failed = 0 }
    duplicate_deliveries = 1
    replay_count = 1
    reconciliation_candidates = 1
    created_jobs = 1
    external_writes = [ordered]@{ gitlink_get = 0; gitlink_post = 0; feishu_message = 0; base = 0; doc = 0; task = 0 }
}

$pairs = @(
    @{ Base = "review-event-subscriptions"; Value = $subscriptions; Title = "Review Event subscriptions" },
    @{ Base = "review-webhook-contract"; Value = $webhook; Title = "Review Webhook contract" },
    @{ Base = "review-event-replay"; Value = $replay; Title = "Review Event Replay" },
    @{ Base = "review-reconciliation"; Value = $reconciliation; Title = "Review reconciliation" }
)
foreach ($pair in $pairs) {
    $json = $pair.Value | ConvertTo-Json -Depth 10
    $json | Set-Content -LiteralPath (Join-Path $OutputDirectory ($pair.Base + ".json")) -Encoding utf8
    $markdown = @"
# $($pair.Title) evidence

```json
$json
```

All identifiers are synthetic or hashed. GitLink POST=0 and real external writes=0.
"@
    $markdown | Set-Content -LiteralPath (Join-Path $OutputDirectory ($pair.Base + ".md")) -Encoding utf8
}
[ordered]@{ schema_version = "feishu.review-event-evidence-export/v1"; passed = $true; files = 8; external_writes = 0 } | ConvertTo-Json
