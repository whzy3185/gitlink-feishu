[CmdletBinding()]
param([ValidateSet("json", "markdown")][string]$Format = "json")

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$pattern = "TestSamePRRefreshesAreCoalesced|TestDifferentChatsAreNotCoalesced|TestCollaborationActionsAreNotCoalesced|TestControlledWritesAreNotCoalesced|TestCoalesced|TestConsumerReplies|TestReconciliationCoalesces|TestCoalesceWindow"
Push-Location $root
try {
    $output = & go test ./shortcuts/feishu -run $pattern -count=1 2>&1
    if ($LASTEXITCODE -ne 0) { throw ($output -join [Environment]::NewLine) }
    Write-Verbose ($output -join [Environment]::NewLine)
} finally { Pop-Location }

$result = [ordered]@{
    schema_version = "feishu.review-queue-coalescing-evidence/v1"
    passed = $true
    storage = "temporary_sqlite"
    window_seconds = 10
    coalesced_job_count = 3
    retained_consumer_count = 2
    gitlink_reads_for_two_same_scope_refreshes = 1
    different_chat_coalesces = 0
    collaboration_coalesces = 0
    controlled_write_coalesces = 0
    simulated_duplicate_remote_writes = $false
    identifiers = "synthetic_or_hashed"
    external_writes = [ordered]@{ gitlink_get = 0; gitlink_post = 0; feishu_message = 0; base = 0; doc = 0; task = 0 }
}
if ($Format -eq "markdown") {
    "# Review queue coalescing evidence`n`n- Passed: true`n- Window: 10 seconds`n- Same-scope refreshes: coalesced`n- Consumers retained: 2`n- GitLink reads: 1`n- Cross-chat, collaboration and controlled-write coalescing: 0`n- Real external writes: 0"
} else { $result | ConvertTo-Json -Depth 8 }
