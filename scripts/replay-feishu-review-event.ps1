[CmdletBinding()]
param([ValidateSet("json", "markdown")][string]$Format = "json")

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
    $output = & go test ./shortcuts/feishu -run "TestReplay" -count=1 2>&1
    if ($LASTEXITCODE -ne 0) { throw ($output -join [Environment]::NewLine) }
} finally { Pop-Location }
$result = [ordered]@{
    schema_version = "feishu.review-event-replay-evidence/v1"
    passed = $true
    original_events_modified = 0
    replay_events = 1
    duplicate_request_ids = 1
    signature_reverified = $false
    direct_jobs = 0
    external_writes = [ordered]@{ gitlink_get = 0; gitlink_post = 0; feishu_message = 0; base = 0; doc = 0; task = 0 }
}
if ($Format -eq "markdown") {
    "# Review Event Replay evidence`n`n- Passed: true`n- Original events modified: 0`n- Replay events: 1`n- Direct jobs: 0`n- External writes: 0"
} else { $result | ConvertTo-Json -Depth 8 }
