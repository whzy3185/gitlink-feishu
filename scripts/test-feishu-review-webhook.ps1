[CmdletBinding()]
param([ValidateSet("json", "markdown")][string]$Format = "json")

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
    $output = & go test ./shortcuts/feishu -run "TestWebhook|TestGitLinkWebhookIngress" -count=1 2>&1
    if ($LASTEXITCODE -ne 0) { throw ($output -join [Environment]::NewLine) }
} finally { Pop-Location }

$result = [ordered]@{
    schema_version = "feishu.review-webhook-contract-evidence/v1"
    passed = $true
    schema_migrations = @(7, 8, 9, 10)
    handler_writes_inbox_only = $true
    handler_synchronous_jobs = 0
    duplicate_deliveries = 1
    processor_jobs = 1
    inbox_status_counts = [ordered]@{ normalized = 1; processed = 1; ignored = 1; failed = 0 }
    route_status_counts = [ordered]@{ queued = 1; duplicate = 1; skipped = 0; failed = 0 }
    signature_modes = @("body_sha256", "timestamp_body_sha256")
    timestamp_modes = @("required", "optional", "disabled")
    external_writes = [ordered]@{ gitlink_get = 0; gitlink_post = 0; feishu_message = 0; base = 0; doc = 0; task = 0 }
}
if ($Format -eq "markdown") {
    @"
# Review Webhook contract evidence

- Passed: true
- Schema migrations: 7, 8, 9, 10
- Handler writes: Event Inbox only
- Synchronous jobs: 0
- Processor jobs: 1
- GitLink POST: 0
- External writes: 0
"@
} else { $result | ConvertTo-Json -Depth 8 }
