[CmdletBinding(SupportsShouldProcess = $true)]
param([string]$ServiceName = "GitLinkFeishuReview")
if ($PSCmdlet.ShouldProcess($ServiceName, "Start Review service")) { Start-Service -Name $ServiceName }
