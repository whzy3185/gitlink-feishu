[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$SourceBindings,

    [Parameter(Mandatory = $true)]
    [string]$DestinationBindings,

    [string]$InstallationID = "gitlink-default",
    [string]$GitLinkHost = "https://www.gitlink.org.cn",

    [ValidateSet("observe", "collaborate", "write")]
    [string]$OperationMode = "collaborate",

    [string]$CredentialRef,
    [string[]]$AdditionalRepositories = @(),
    [string]$BindingRevision = "v2-local-migration",
    [switch]$Force
)

$ErrorActionPreference = "Stop"

function Get-NormalizedRepository {
    param([string]$Repository)

    $value = ([string]$Repository).Trim().Trim("/")
    if ($value -notmatch "^[^/\s]+/[^/\s]+$") {
        throw "repository must use owner/repository format"
    }
    return $value
}

function Get-UniqueStrings {
    param([object[]]$Values)

    $seen = @{}
    $result = New-Object System.Collections.Generic.List[string]
    foreach ($item in $Values) {
        $value = ([string]$item).Trim()
        if ($value -eq "") {
            continue
        }
        $key = $value.ToLowerInvariant()
        if (-not $seen.ContainsKey($key)) {
            $seen[$key] = $true
            $result.Add($value)
        }
    }
    return @($result)
}

$sourcePath = [IO.Path]::GetFullPath($SourceBindings)
$destinationPath = [IO.Path]::GetFullPath($DestinationBindings)
if (-not (Test-Path -LiteralPath $sourcePath -PathType Leaf)) {
    throw "source bindings file does not exist"
}
if ((Test-Path -LiteralPath $destinationPath) -and -not $Force) {
    throw "destination bindings file already exists; pass -Force to replace it"
}
if ($sourcePath -eq $destinationPath) {
    throw "source and destination bindings paths must differ"
}

$source = Get-Content -LiteralPath $sourcePath -Raw | ConvertFrom-Json
if ([string]$source.schema_version -ne "feishu.review-bindings/v1") {
    throw "source bindings must use feishu.review-bindings/v1"
}
if (@($source.bindings).Count -eq 0) {
    throw "source bindings contain no chat bindings"
}

$extraRepositories = @()
foreach ($repository in $AdditionalRepositories) {
    $extraRepositories += Get-NormalizedRepository $repository
}

$allRepositories = New-Object System.Collections.Generic.List[string]
$migratedBindings = New-Object System.Collections.Generic.List[object]
foreach ($binding in @($source.bindings)) {
    $repository = Get-NormalizedRepository ([string]$binding.repository)
    $repositories = Get-UniqueStrings (@($repository) + $extraRepositories)
    foreach ($item in $repositories) {
        $allRepositories.Add($item)
    }

    $migrated = [ordered]@{
        chat_id            = [string]$binding.chat_id
        installation_id    = $InstallationID
        repositories       = @($repositories)
        default_repository = $repository
        enabled            = [bool]$binding.enabled
        admin_user_ids     = @($binding.admin_user_ids)
        allowed_user_ids   = @($binding.allowed_user_ids)
        deadline_hours     = if ($null -ne $binding.deadline_hours) { [int]$binding.deadline_hours } else { 24 }
        binding_revision   = if (([string]$binding.binding_revision).Trim() -ne "") { [string]$binding.binding_revision } else { $BindingRevision }
    }
    $migratedBindings.Add([pscustomobject]$migrated)
}

$allowedRepositories = Get-UniqueStrings @($allRepositories | ForEach-Object { $_ })
$installation = [ordered]@{
    installation_id     = $InstallationID
    gitlink_host         = $GitLinkHost.TrimEnd("/")
    owner                = ""
    operation_mode       = $OperationMode
    allowed_repositories = @($allowedRepositories)
    enabled              = $true
}
if ($CredentialRef.Trim() -ne "") {
    if ($CredentialRef -notmatch "^env:[A-Za-z_][A-Za-z0-9_]*$") {
        throw "credential reference must use env:VARIABLE format"
    }
    $installation.credential_ref = $CredentialRef
}
if ($OperationMode -eq "write" -and -not $installation.Contains("credential_ref")) {
    throw "write mode requires -CredentialRef env:VARIABLE"
}

$identityBindings = @()
if ($null -ne $source.identity_bindings) {
    $identityBindings = @($source.identity_bindings)
}

$result = [ordered]@{
    schema_version    = "feishu.review-bindings/v2"
    installations     = @([pscustomobject]$installation)
    bindings          = @($migratedBindings | ForEach-Object { $_ })
    identity_bindings = $identityBindings
}

$directory = Split-Path -Parent $destinationPath
if (-not (Test-Path -LiteralPath $directory)) {
    [IO.Directory]::CreateDirectory($directory) | Out-Null
}
$json = $result | ConvertTo-Json -Depth 12
[IO.File]::WriteAllText($destinationPath, $json + [Environment]::NewLine, [Text.UTF8Encoding]::new($false))

[pscustomobject]@{
    schema_version      = $result.schema_version
    installation_count = @($result.installations).Count
    binding_count      = @($result.bindings).Count
    repository_count   = @($installation.allowed_repositories).Count
    repositories       = @($installation.allowed_repositories)
    operation_mode     = $OperationMode
    credential_ref     = if ($installation.Contains("credential_ref")) { "configured" } else { "not_configured" }
    identity_count     = @($result.identity_bindings).Count
} | ConvertTo-Json -Depth 4
