[CmdletBinding(SupportsShouldProcess = $true)]
param([string]$ServiceName = "GitLinkFeishuReview")

$ErrorActionPreference = "Stop"
$service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($null -ne $service -and $PSCmdlet.ShouldProcess($ServiceName, "Uninstall Review service while preserving state and backups")) {
    if ($service.Status -ne "Stopped") { Stop-Service -Name $ServiceName -Force }
    & sc.exe delete $ServiceName | Out-Null
    Write-Output "Service removed. Database, backups, bindings, and environment files were preserved."
}
