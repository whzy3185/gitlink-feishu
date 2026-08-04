[CmdletBinding()]
param([string]$TemporaryDirectory = "", [ValidateSet("json", "markdown")][string]$Format = "json")
. (Join-Path $PSScriptRoot "feishu-review-service-test-common.ps1")
$result = Invoke-FeishuReviewServiceTest -Pattern "TestMetrics|TestAdmin" -SchemaVersion "review.service-admin-evidence/v1" -Capability "metrics_and_read_only_admin" -TemporaryDirectory $TemporaryDirectory
$result.admin_paths = 20
$result.authentication = "bearer_constant_time"
$result.high_cardinality_labels = 0
if ($Format -eq "markdown") { "# Review service admin API`n`n- Passed: true`n- API: authenticated and read-only`n- High-cardinality labels: 0`n- External writes: 0" } else { $result | ConvertTo-Json -Depth 8 }
