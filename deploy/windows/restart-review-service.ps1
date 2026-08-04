[CmdletBinding(SupportsShouldProcess = $true)]
param([string]$ServiceName = "GitLinkFeishuReview", [int]$TimeoutSeconds = 45)
if ($PSCmdlet.ShouldProcess($ServiceName, "Restart Review service")) {
    & (Join-Path $PSScriptRoot 'stop-review-service.ps1') -ServiceName $ServiceName -TimeoutSeconds $TimeoutSeconds
    & (Join-Path $PSScriptRoot 'start-review-service.ps1') -ServiceName $ServiceName
}
