[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Bindings,

    [Parameter(Mandatory = $true)]
    [string[]]$Repositories,

    [string]$InstallationID
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

$bindingsPath = [IO.Path]::GetFullPath($Bindings)
if (-not (Test-Path -LiteralPath $bindingsPath -PathType Leaf)) {
    throw "bindings file does not exist"
}

$configuration = Get-Content -LiteralPath $bindingsPath -Raw | ConvertFrom-Json
if ([string]$configuration.schema_version -ne "feishu.review-bindings/v2") {
    throw "bindings must use feishu.review-bindings/v2"
}

$installations = @($configuration.installations)
if ($InstallationID.Trim() -eq "") {
    $enabled = @($installations | Where-Object { $_.enabled })
    if ($enabled.Count -ne 1) {
        throw "select --InstallationID when bindings do not have exactly one enabled installation"
    }
    $InstallationID = [string]$enabled[0].installation_id
}

$installation = @($installations | Where-Object { [string]$_.installation_id -eq $InstallationID })
if ($installation.Count -ne 1) {
    throw "installation was not found or is duplicated"
}
$installation = $installation[0]

$normalized = @($Repositories | ForEach-Object { Get-NormalizedRepository $_ })
$installation.allowed_repositories = Get-UniqueStrings (@($installation.allowed_repositories) + $normalized)

$matchedBindings = 0
foreach ($binding in @($configuration.bindings)) {
    if ([string]$binding.installation_id -ne $InstallationID) {
        continue
    }
    $binding.repositories = Get-UniqueStrings (@($binding.repositories) + $normalized)
    $binding.binding_revision = "repository-add-" + [DateTime]::UtcNow.ToString("yyyyMMddHHmmss")
    $matchedBindings++
}
if ($matchedBindings -eq 0) {
    throw "installation has no chat bindings"
}

$json = $configuration | ConvertTo-Json -Depth 16
[IO.File]::WriteAllText($bindingsPath, $json + [Environment]::NewLine, [Text.UTF8Encoding]::new($false))

[pscustomobject]@{
    schema_version      = [string]$configuration.schema_version
    installation_id    = $InstallationID
    operation_mode      = [string]$installation.operation_mode
    matched_bindings    = $matchedBindings
    repository_count    = @($installation.allowed_repositories).Count
    repositories        = @($installation.allowed_repositories)
    credential_ref      = if (([string]$installation.credential_ref).Trim() -eq "") { "not_configured" } else { "configured" }
    gitlink_write       = $false
} | ConvertTo-Json -Depth 5
