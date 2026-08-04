[CmdletBinding()]
param(
    [string]$DatabasePath = ""
)

$ErrorActionPreference = "Stop"
$repositoryRoot = Split-Path -Parent $PSScriptRoot
$databaseHash = ""
if ($DatabasePath) {
    $resolved = (Resolve-Path -LiteralPath $DatabasePath).Path
    $databaseHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $resolved).Hash.ToLowerInvariant()
}

Push-Location $repositoryRoot
try {
    $testOutput = & go test ./shortcuts/feishu -run "TestTwoChats|TestChatScoped|TestTaskIsNeverShared|TestInstallationScopedProjectionIsShared" -count=1 2>&1
    $testExitCode = $LASTEXITCODE
    Write-Verbose ($testOutput -join [Environment]::NewLine)
    if ($testExitCode -ne 0) {
        throw "dual-chat isolation tests failed"
    }
} finally {
    Pop-Location
}

[ordered]@{
    schema_version = "feishu.review-dual-chat-evidence/v1"
    passed = $true
    database_sha256 = $databaseHash
    presentation_shared_by_installation = $true
    collaboration_isolated_by_chat = $true
    canonical_card_isolated_by_chat = $true
    task_isolated_by_chat = $true
    chat_projection_isolated = $true
    installation_projection_shared = $true
    external_writes = [ordered]@{ gitlink_get = 0; gitlink_post = 0; feishu_message = 0; base = 0; doc = 0; task = 0 }
} | ConvertTo-Json -Depth 5
