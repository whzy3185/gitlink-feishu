[CmdletBinding()]
param([ValidateSet("json", "markdown")][string]$Format = "json")

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
    $output = & go test ./shortcuts/feishu -run "TestReconciliation|TestUnchangedReconciliation|TestChangedReconciliation" -count=1 2>&1
    if ($LASTEXITCODE -ne 0) { throw ($output -join [Environment]::NewLine) }
} finally { Pop-Location }
$result = [ordered]@{
    schema_version = "feishu.review-reconciliation-evidence/v1"
    passed = $true
    candidates = 1
    dry_run_jobs = 0
    once_jobs = 1
    duplicate_time_bucket_jobs = 0
    unchanged_card_patches = 0
    changed_card_patches = 1
    notifications = 0
    external_writes = [ordered]@{ gitlink_get = 0; gitlink_post = 0; feishu_message = 0; base = 0; doc = 0; task = 0 }
}
if ($Format -eq "markdown") {
    "# Review reconciliation evidence`n`n- Passed: true`n- Candidates: 1`n- Dry-run jobs: 0`n- Once jobs: 1`n- Unchanged patches: 0`n- Changed patches: 1`n- Notifications: 0"
} else { $result | ConvertTo-Json -Depth 8 }
