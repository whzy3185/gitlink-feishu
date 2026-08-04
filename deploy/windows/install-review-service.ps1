[CmdletBinding(SupportsShouldProcess = $true)]
param(
    [string]$ServiceName = "GitLinkFeishuReview",
    [Parameter(Mandatory = $true)][string]$ExecutablePath,
    [Parameter(Mandatory = $true)][string]$WorkingDirectory,
    [Parameter(Mandatory = $true)][string]$BindingsFile,
    [string]$StateDB = "review.db",
    [Parameter(Mandatory = $true)][string]$EnvironmentFile,
    [string]$DisplayName = "GitLink Feishu Review"
)

$ErrorActionPreference = "Stop"
$exe = (Resolve-Path -LiteralPath $ExecutablePath).Path
$work = (Resolve-Path -LiteralPath $WorkingDirectory).Path
$bindings = (Resolve-Path -LiteralPath $BindingsFile).Path
$environment = (Resolve-Path -LiteralPath $EnvironmentFile).Path
$state = if ([IO.Path]::IsPathRooted($StateDB)) { $StateDB } else { Join-Path $work $StateDB }
$binaryPath = ('"{0}" feishu +review-gateway --listen --bindings "{1}" --state-db "{2}"' -f $exe, $bindings, $state)

if ($PSCmdlet.ShouldProcess($ServiceName, "Install Review service")) {
    $service = New-Service -Name $ServiceName -BinaryPathName $binaryPath -DisplayName $DisplayName -StartupType Automatic
    & sc.exe failure $ServiceName reset= 86400 actions= restart/5000/restart/15000/none/0 | Out-Null
    & sc.exe failureflag $ServiceName 1 | Out-Null

    $entries = Get-Content -LiteralPath $environment | Where-Object { $_ -match '^[A-Za-z_][A-Za-z0-9_]*=' }
    $registryPath = "HKLM:\SYSTEM\CurrentControlSet\Services\$ServiceName"
    New-ItemProperty -LiteralPath $registryPath -Name Environment -PropertyType MultiString -Value $entries -Force | Out-Null
    & icacls.exe $environment /inheritance:r /grant:r "SYSTEM:(R)" "Administrators:(R)" | Out-Null
    Write-Output "Installed $($service.Name). State and environment files are preserved by uninstall."
}
