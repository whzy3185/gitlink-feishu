[CmdletBinding()]
param(
    [string]$Executable = ".\gitlink-cli.exe",

    [Parameter(Mandatory = $true)]
    [string]$Bindings,

    [string]$StateDB = ".local/review-gateway-v2.db",
    [switch]$EnableFeishuResourceSync,
    [string]$BaseAppToken = $(if ([string]::IsNullOrWhiteSpace($env:FEISHU_REVIEW_BASE_APP_TOKEN)) { $env:FEISHU_BASE_APP_TOKEN } else { $env:FEISHU_REVIEW_BASE_APP_TOKEN }),
    [string]$ReviewTableID = $env:FEISHU_REVIEW_TABLE_ID,
    [string]$ReviewDocumentID = $env:FEISHU_REVIEW_DOCUMENT_ID,
    [string]$ReviewDocumentFolderToken = $env:FEISHU_REVIEW_DOCUMENT_FOLDER_TOKEN,
    [switch]$EnableTask,
    [switch]$CheckOnly
)

$ErrorActionPreference = "Stop"

$executablePath = [IO.Path]::GetFullPath($Executable)
$bindingsPath = [IO.Path]::GetFullPath($Bindings)
if (-not (Test-Path -LiteralPath $executablePath -PathType Leaf)) {
    throw "gateway executable does not exist"
}
if (-not (Test-Path -LiteralPath $bindingsPath -PathType Leaf)) {
    throw "bindings file does not exist"
}
if ([string]::IsNullOrWhiteSpace($env:FEISHU_APP_ID) -or [string]::IsNullOrWhiteSpace($env:FEISHU_APP_SECRET)) {
    throw "FEISHU_APP_ID and FEISHU_APP_SECRET are required"
}
if ($EnableFeishuResourceSync -and $ReviewDocumentID.Trim() -ne "" -and $ReviewDocumentFolderToken.Trim() -ne "") {
    throw "configure either ReviewDocumentID or ReviewDocumentFolderToken, not both"
}
if ($EnableFeishuResourceSync -and (($BaseAppToken.Trim() -eq "") -ne ($ReviewTableID.Trim() -eq ""))) {
    throw "Base sync requires both BaseAppToken and ReviewTableID"
}
if ($EnableTask -and -not $EnableFeishuResourceSync) {
    throw "EnableTask requires EnableFeishuResourceSync"
}
if ($EnableFeishuResourceSync -and
    $BaseAppToken.Trim() -eq "" -and
    $ReviewDocumentID.Trim() -eq "" -and
    $ReviewDocumentFolderToken.Trim() -eq "" -and
    -not $EnableTask) {
    throw "resource sync requires at least one Base, Doc, or Task target"
}

$arguments = New-Object System.Collections.Generic.List[string]
foreach ($value in @(
    "feishu",
    "+review-gateway",
    "--listen",
    "--bindings",
    $bindingsPath,
    "--state-db",
    $StateDB
)) {
    $arguments.Add($value)
}

if ($EnableFeishuResourceSync) {
    $arguments.Add("--sync-feishu-resources")
    if ($BaseAppToken.Trim() -ne "") {
        foreach ($value in @("--base-app-token", $BaseAppToken, "--review-table-id", $ReviewTableID)) {
            $arguments.Add($value)
        }
    }
    if ($ReviewDocumentID.Trim() -ne "") {
        foreach ($value in @("--review-document-id", $ReviewDocumentID)) {
            $arguments.Add($value)
        }
    }
    if ($ReviewDocumentFolderToken.Trim() -ne "") {
        foreach ($value in @("--review-document-folder-token", $ReviewDocumentFolderToken)) {
            $arguments.Add($value)
        }
    }
    if ($EnableTask) {
        $arguments.Add("--sync-feishu-task")
    }
}

if ($CheckOnly) {
    [pscustomobject]@{
        mode                    = "check_only"
        bindings_schema         = (Get-Content -LiteralPath $bindingsPath -Raw | ConvertFrom-Json).schema_version
        state_db_configured     = -not [string]::IsNullOrWhiteSpace($StateDB)
        feishu_resource_sync    = [bool]$EnableFeishuResourceSync
        base_target             = ($BaseAppToken.Trim() -ne "" -and $ReviewTableID.Trim() -ne "")
        document_target         = $ReviewDocumentID.Trim() -ne ""
        document_folder_target  = $ReviewDocumentFolderToken.Trim() -ne ""
        task_sync               = [bool]$EnableTask
        gitlink_review_write    = $false
    } | ConvertTo-Json
    return
}

& $executablePath @arguments
exit $LASTEXITCODE
