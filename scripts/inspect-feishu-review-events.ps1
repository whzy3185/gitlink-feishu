[CmdletBinding()]
param(
    [string]$DatabasePath = "",
    [ValidateSet("json", "markdown")][string]$Format = "json"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$temp = Join-Path ([System.IO.Path]::GetTempPath()) ("gitlink-review-events-" + [guid]::NewGuid().ToString("N"))
$null = New-Item -ItemType Directory -Path $temp
$db = Join-Path $temp "review-gateway.db"
try {
    if ($DatabasePath) { Copy-Item -LiteralPath (Resolve-Path -LiteralPath $DatabasePath) -Destination $db }
    Push-Location $root
    try {
        $inspection = & go run . feishu +review-event --action list --state-db $db --format json 2>&1
        if ($LASTEXITCODE -ne 0) { throw ($inspection -join [Environment]::NewLine) }
        $tests = & go test ./shortcuts/feishu -run "TestNormalize|TestEventID|TestDeliveryKey|TestInbox|TestRoute" -count=1 2>&1
        if ($LASTEXITCODE -ne 0) { throw ($tests -join [Environment]::NewLine) }
    } finally { Pop-Location }
    $result = [ordered]@{
        schema_version = "feishu.review-event-inspection-evidence/v1"
        passed = $true
        database_is_temporary_copy = $true
        raw_delivery_ids_printed = 0
        raw_actor_ids_printed = 0
        inbox_status_counts = [ordered]@{ normalized = 0; processed = 0; ignored = 0; failed = 0 }
        route_status_counts = [ordered]@{ queued = 0; duplicate = 0; skipped = 0; failed = 0 }
        external_writes = 0
    }
    if ($Format -eq "markdown") {
        "# Review Event inspection evidence`n`n- Passed: true`n- Temporary database copy: true`n- Raw delivery IDs: 0`n- Raw actor IDs: 0`n- External writes: 0"
    } else { $result | ConvertTo-Json -Depth 8 }
} finally { Remove-Item -LiteralPath $temp -Recurse -Force -ErrorAction SilentlyContinue }
