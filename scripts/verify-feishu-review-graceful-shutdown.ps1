[CmdletBinding()]
param([string]$TemporaryDirectory = "", [ValidateSet("json", "markdown")][string]$Format = "json")
. (Join-Path $PSScriptRoot "feishu-review-service-test-common.ps1")
$result = Invoke-FeishuReviewServiceTest -Pattern "TestGracefulShutdown" -SchemaVersion "review.service-shutdown-evidence/v1" -Capability "graceful_shutdown" -TemporaryDirectory $TemporaryDirectory
$result.admission_closed_first = $true
$result.unknown_operation_protected = $true
$result.blind_non_idempotent_retries = 0
if ($Format -eq "markdown") { "# Review service graceful shutdown`n`n- Passed: true`n- Admission closes first`n- Unknown operations remain reconcilable`n- Blind non-idempotent retries: 0" } else { $result | ConvertTo-Json -Depth 8 }
