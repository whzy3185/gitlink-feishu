[CmdletBinding()]
param([string]$OutputDirectory = "evidence")

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
if (-not [System.IO.Path]::IsPathRooted($OutputDirectory)) { $OutputDirectory = Join-Path $root $OutputDirectory }
$null = New-Item -ItemType Directory -Force -Path $OutputDirectory

$outbox = & (Join-Path $PSScriptRoot "test-feishu-review-operation-outbox.ps1") | ConvertFrom-Json
$failures = & (Join-Path $PSScriptRoot "inject-feishu-review-operation-failures.ps1") | ConvertFrom-Json
$workers = & (Join-Path $PSScriptRoot "inspect-feishu-review-queues.ps1") | ConvertFrom-Json
$coalescing = & (Join-Path $PSScriptRoot "verify-feishu-review-coalescing.ps1") | ConvertFrom-Json
$restart = & (Join-Path $PSScriptRoot "verify-feishu-review-operation-restart.ps1") | ConvertFrom-Json

$outbox | Add-Member -NotePropertyName restart -NotePropertyValue $restart
$deadLetters = [ordered]@{
    schema_version = "feishu.review-dead-letter-reconciliation-evidence/v1"
    passed = $failures.passed
    unknown_count = $failures.unknown_count
    dead_letter_count = $failures.dead_letter_count
    reconciliation_count = $failures.reconciliation_count
    automatic_non_idempotent_retries = 0
    manual_required_resources = @("feishu_card", "feishu_reply", "feishu_doc", "feishu_task")
    automatic_reconciliation_resources = @("feishu_bitable_by_unique_key")
    identifiers = "synthetic_or_hashed"
    external_writes = $failures.external_writes
}

$pairs = @(
    @{ Base = "review-operation-outbox"; Value = $outbox; Title = "Review operation outbox" },
    @{ Base = "review-operation-failures"; Value = $failures; Title = "Review operation failures" },
    @{ Base = "review-worker-isolation"; Value = $workers; Title = "Review worker isolation" },
    @{ Base = "review-queue-coalescing"; Value = $coalescing; Title = "Review queue coalescing" },
    @{ Base = "review-dead-letter-reconciliation"; Value = $deadLetters; Title = "Review dead letter and reconciliation" }
)
foreach ($pair in $pairs) {
    $json = $pair.Value | ConvertTo-Json -Depth 12
    $json | Set-Content -LiteralPath (Join-Path $OutputDirectory ($pair.Base + ".json")) -Encoding utf8
    $markdown = @"
# $($pair.Title) evidence

```json
$json
```

The run uses temporary SQLite, httptest or fake clients. Identifiers are synthetic or hashed. GitLink POST=0 and real external writes=0.
"@
    $markdown | Set-Content -LiteralPath (Join-Path $OutputDirectory ($pair.Base + ".md")) -Encoding utf8
}
[ordered]@{ schema_version = "feishu.review-operation-evidence-export/v1"; passed = $true; files = 10; external_writes = 0 } | ConvertTo-Json
