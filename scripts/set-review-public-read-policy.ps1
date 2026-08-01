[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Bindings,

    [Parameter(Mandatory = $true)]
    [ValidateSet("enabled", "disabled")]
    [string]$Mode,

    [string]$InstallationID
)

$ErrorActionPreference = "Stop"

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
    $enabledInstallations = @($installations | Where-Object { $_.enabled })
    if ($enabledInstallations.Count -ne 1) {
        throw "select -InstallationID when bindings do not have exactly one enabled installation"
    }
    $InstallationID = [string]$enabledInstallations[0].installation_id
}

$matchedInstallations = @($installations | Where-Object { [string]$_.installation_id -eq $InstallationID })
if ($matchedInstallations.Count -ne 1) {
    throw "installation was not found or is duplicated"
}
$installation = $matchedInstallations[0]
$allowPublicRead = $Mode -eq "enabled"
$installation | Add-Member -NotePropertyName allow_public_read -NotePropertyValue $allowPublicRead -Force

$matchedBindings = 0
foreach ($binding in @($configuration.bindings)) {
    if ([string]$binding.installation_id -ne $InstallationID) {
        continue
    }
    $binding | Add-Member -NotePropertyName allow_public_read -NotePropertyValue $allowPublicRead -Force
    $binding.binding_revision = "public-read-$Mode-" + [DateTime]::UtcNow.ToString("yyyyMMddHHmmss")
    $matchedBindings++
}
if ($matchedBindings -eq 0) {
    throw "installation has no chat bindings"
}

$json = $configuration | ConvertTo-Json -Depth 16
[IO.File]::WriteAllText($bindingsPath, $json + [Environment]::NewLine, [Text.UTF8Encoding]::new($false))

[pscustomobject]@{
    schema_version           = [string]$configuration.schema_version
    installation_id         = $InstallationID
    public_read              = $allowPublicRead
    public_read_action       = "read_review_context"
    credentialless           = $true
    matched_bindings         = $matchedBindings
    bound_repository_count   = @($installation.allowed_repositories).Count
    gitlink_write            = $false
} | ConvertTo-Json -Depth 4
