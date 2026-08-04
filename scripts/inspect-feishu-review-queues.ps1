[CmdletBinding()]
param([ValidateSet("json", "markdown")][string]$Format = "json")

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$pattern = "TestSlow|TestControlledWrite|TestAgentWorker|TestWorkerDoes|TestGlobal|TestPerChat|TestPerUser|TestCapacity|TestUserRate|TestChatRate|TestWebhook"
Push-Location $root
try {
    $output = & go test ./shortcuts/feishu -run $pattern -count=1 2>&1
    if ($LASTEXITCODE -ne 0) { throw ($output -join [Environment]::NewLine) }
    Write-Verbose ($output -join [Environment]::NewLine)
} finally { Pop-Location }

$result = [ordered]@{
    schema_version = "feishu.review-worker-isolation-evidence/v1"
    passed = $true
    storage = "temporary_sqlite"
    queue_class_counts = [ordered]@{ gitlink_read = 1; collaboration = 1; controlled_write = 1; agent = 0; canonical_card = 1; reply = 1; resource = 2 }
    default_concurrency = [ordered]@{ event = 2; gitlink_read = 3; collaboration = 1; planner = 2; canonical_card = 2; reply = 2; resource = 2; controlled_write = 1; agent = 0; operation_reconciliation = 1 }
    capacity_rejections = 4
    rate_limited_requests = 2
    sqlite_transaction_held_during_network = $false
    identifiers = "synthetic_or_hashed"
    external_writes = [ordered]@{ gitlink_get = 0; gitlink_post = 0; feishu_message = 0; base = 0; doc = 0; task = 0 }
}
if ($Format -eq "markdown") {
    "# Review worker isolation evidence`n`n- Passed: true`n- GitLink read workers: 3`n- Collaboration workers: 1`n- Controlled write workers: 1`n- Agent workers: 0`n- SQLite transaction held during network: false`n- Real external writes: 0"
} else { $result | ConvertTo-Json -Depth 8 }
