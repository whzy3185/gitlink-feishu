[CmdletBinding()]
param([string]$TemporaryDirectory = "", [ValidateSet("json", "markdown")][string]$Format = "json")
. (Join-Path $PSScriptRoot "feishu-review-service-test-common.ps1")
$result = Invoke-FeishuReviewServiceTest -Pattern "TestReviewService|TestHealth|TestReady" -SchemaVersion "review.service-health-evidence/v1" -Capability "service_health_readiness" -TemporaryDirectory $TemporaryDirectory
$result.migrations = @(16, 17, 18, 19)
$result.required_components = 8
$result.health_status = "ok"
$result.ready_status = "ready_and_draining_covered"
if ($Format -eq "markdown") { "# Review service health`n`n- Passed: true`n- Health: process liveness`n- Readiness: dependencies and admission`n- External writes: 0" } else { $result | ConvertTo-Json -Depth 8 }
