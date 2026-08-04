[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$DatabasePath,
    [ValidateSet("list", "policy-list")]
    [string]$Action = "list",
    [string]$Status = "",
    [string]$Resource = "",
    [string]$Installation = ""
)

$ErrorActionPreference = "Stop"
$repositoryRoot = Split-Path -Parent $PSScriptRoot
$source = (Resolve-Path -LiteralPath $DatabasePath).Path
$temporaryDirectory = Join-Path ([System.IO.Path]::GetTempPath()) ("gitlink-review-scope-" + [guid]::NewGuid().ToString("N"))
$null = New-Item -ItemType Directory -Path $temporaryDirectory
$temporaryDatabase = Join-Path $temporaryDirectory "review-gateway.db"

try {
    Copy-Item -LiteralPath $source -Destination $temporaryDatabase
    foreach ($suffix in @("-wal", "-shm")) {
        if (Test-Path -LiteralPath ($source + $suffix)) {
            Copy-Item -LiteralPath ($source + $suffix) -Destination ($temporaryDatabase + $suffix)
        }
    }
    $arguments = @("run", ".", "feishu", "+review-migration", "--action", $Action, "--state-db", $temporaryDatabase)
    if ($Status) { $arguments += @("--status", $Status) }
    if ($Resource) { $arguments += @("--resource", $Resource) }
    if ($Installation) { $arguments += @("--installation", $Installation) }
    Push-Location $repositoryRoot
    try {
        & go @arguments
        if ($LASTEXITCODE -ne 0) { throw "review migration inspection failed" }
    } finally {
        Pop-Location
    }
} finally {
    Remove-Item -LiteralPath $temporaryDirectory -Recurse -Force -ErrorAction SilentlyContinue
}
