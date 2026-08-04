Set-StrictMode -Version Latest

function Invoke-FeishuReviewServiceTest {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)][string]$Pattern,
        [Parameter(Mandatory = $true)][string]$SchemaVersion,
        [Parameter(Mandatory = $true)][string]$Capability,
        [string]$TemporaryDirectory = ""
    )

    $ErrorActionPreference = "Stop"
    $root = Split-Path -Parent $PSScriptRoot
    $base = if ([string]::IsNullOrWhiteSpace($TemporaryDirectory)) { [IO.Path]::GetTempPath() } else { $TemporaryDirectory }
    $runDirectory = Join-Path $base ("gitlink-review-service-" + [Guid]::NewGuid().ToString("N"))
    $null = New-Item -ItemType Directory -Force -Path $runDirectory
    $oldTemp, $oldTmp = $env:TEMP, $env:TMP
    $env:TEMP, $env:TMP = $runDirectory, $runDirectory
    try {
        Push-Location $root
        try {
            $output = & go test ./shortcuts/feishu -run $Pattern -count=1 2>&1
            if ($LASTEXITCODE -ne 0) { throw ($output -join [Environment]::NewLine) }
        } finally { Pop-Location }
        [ordered]@{
            schema_version = $SchemaVersion
            passed = $true
            capability = $Capability
            storage = "temporary_sqlite"
            listener = "loopback_only"
            identifiers = "synthetic_or_hashed"
            external_writes = [ordered]@{ gitlink_get = 0; gitlink_post = 0; feishu_message = 0; base = 0; doc = 0; task = 0 }
        }
    } finally {
        $env:TEMP, $env:TMP = $oldTemp, $oldTmp
        Remove-Item -LiteralPath $runDirectory -Recurse -Force -ErrorAction SilentlyContinue
    }
}
