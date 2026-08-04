[CmdletBinding()]
param([string]$ServiceName = "GitLinkFeishuReview")
Get-Service -Name $ServiceName | Select-Object Name, Status, StartType
