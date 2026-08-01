$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$temporary = Join-Path ([IO.Path]::GetTempPath()) ("gitlink-round2-scripts-" + [Guid]::NewGuid().ToString("N"))
[IO.Directory]::CreateDirectory($temporary) | Out-Null

try {
    $source = Join-Path $temporary "bindings-v1.json"
    $destination = Join-Path $temporary "bindings-v2.json"
    $fakeExecutable = Join-Path $temporary "gitlink-cli-test"
    $fixture = @{
        schema_version = "feishu.review-bindings/v1"
        bindings       = @(
            @{
                chat_id          = "chat_contract_test"
                repository       = "owner/one"
                enabled          = $true
                deadline_hours   = 24
                binding_revision = "contract-test"
            }
        )
    } | ConvertTo-Json -Depth 8
    [IO.File]::WriteAllText($source, $fixture, [Text.UTF8Encoding]::new($false))
    [IO.File]::WriteAllText($fakeExecutable, "test", [Text.UTF8Encoding]::new($false))

    & (Join-Path $PSScriptRoot "migrate-review-bindings-v2.ps1") `
        -SourceBindings $source `
        -DestinationBindings $destination `
        -InstallationID "contract-installation" `
        -AdditionalRepositories "owner/two" | Out-Null

    $migrated = Get-Content -LiteralPath $destination -Raw | ConvertFrom-Json
    if ($migrated.schema_version -ne "feishu.review-bindings/v2") {
        throw "migration schema contract failed"
    }
    if (@($migrated.installations).Count -ne 1 -or @($migrated.bindings).Count -ne 1) {
        throw "migration count contract failed"
    }
    if (@($migrated.installations[0].allowed_repositories).Count -ne 2) {
        throw "migration repository contract failed"
    }
    if ($migrated.installations[0].operation_mode -ne "collaborate" -or $migrated.installations[0].credential_ref) {
        throw "migration safe-mode contract failed"
    }

    $writeModeRejected = $false
    try {
        & (Join-Path $PSScriptRoot "migrate-review-bindings-v2.ps1") `
            -SourceBindings $source `
            -DestinationBindings (Join-Path $temporary "write.json") `
            -OperationMode write | Out-Null
    } catch {
        $writeModeRejected = $_.Exception.Message -like "*requires -CredentialRef*"
    }
    if (-not $writeModeRejected) {
        throw "write mode without a credential reference was not rejected"
    }

    $oldAppID = $env:FEISHU_APP_ID
    $oldAppSecret = $env:FEISHU_APP_SECRET
    $oldBaseToken = $env:FEISHU_BASE_APP_TOKEN
    $oldReviewBaseToken = $env:FEISHU_REVIEW_BASE_APP_TOKEN
    $oldReviewTable = $env:FEISHU_REVIEW_TABLE_ID
    $env:FEISHU_APP_ID = "app_contract"
    $env:FEISHU_APP_SECRET = "secret_contract"
    $env:FEISHU_BASE_APP_TOKEN = "base_contract"
    $env:FEISHU_REVIEW_BASE_APP_TOKEN = ""
    $env:FEISHU_REVIEW_TABLE_ID = ""
    try {
        $readOnly = & (Join-Path $PSScriptRoot "start-round2-feishu-gateway.ps1") `
            -Executable $fakeExecutable `
            -Bindings $destination `
            -StateDB (Join-Path $temporary "state.db") `
            -CheckOnly | ConvertFrom-Json
        if ($readOnly.bindings_schema -ne "feishu.review-bindings/v2" -or $readOnly.gitlink_review_write) {
            throw "read-only launcher contract failed"
        }

        $partialBaseRejected = $false
        try {
            & (Join-Path $PSScriptRoot "start-round2-feishu-gateway.ps1") `
                -Executable $fakeExecutable `
                -Bindings $destination `
                -EnableFeishuResourceSync `
                -CheckOnly | Out-Null
        } catch {
            $partialBaseRejected = $_.Exception.Message -like "*requires both*"
        }
        if (-not $partialBaseRejected) {
            throw "partial Base target was not rejected"
        }

        $resource = & (Join-Path $PSScriptRoot "start-round2-feishu-gateway.ps1") `
            -Executable $fakeExecutable `
            -Bindings $destination `
            -EnableFeishuResourceSync `
            -BaseAppToken "base_contract" `
            -ReviewTableID "table_contract" `
            -ReviewDocumentFolderToken "folder_contract" `
            -CheckOnly | ConvertFrom-Json
        if (-not $resource.feishu_resource_sync -or -not $resource.base_target -or -not $resource.document_folder_target) {
            throw "resource launcher contract failed"
        }
    } finally {
        $env:FEISHU_APP_ID = $oldAppID
        $env:FEISHU_APP_SECRET = $oldAppSecret
        $env:FEISHU_BASE_APP_TOKEN = $oldBaseToken
        $env:FEISHU_REVIEW_BASE_APP_TOKEN = $oldReviewBaseToken
        $env:FEISHU_REVIEW_TABLE_ID = $oldReviewTable
    }

    Write-Host "Round 2 PowerShell deployment tool contracts passed."
} finally {
    if (Test-Path -LiteralPath $temporary) {
        $resolvedTemporary = [IO.Path]::GetFullPath($temporary)
        $resolvedTempRoot = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
        if (-not $resolvedTemporary.StartsWith($resolvedTempRoot, [StringComparison]::OrdinalIgnoreCase)) {
            throw "refusing to remove a test directory outside the system temporary directory"
        }
        Remove-Item -LiteralPath $temporary -Recurse -Force
    }
}
