[CmdletBinding(SupportsShouldProcess = $true)]
param([string]$ServiceName = "GitLinkFeishuReview", [int]$TimeoutSeconds = 45)
if ($PSCmdlet.ShouldProcess($ServiceName, "Stop Review service")) {
    Stop-Service -Name $ServiceName
    (Get-Service -Name $ServiceName).WaitForStatus('Stopped', [TimeSpan]::FromSeconds($TimeoutSeconds))
}
