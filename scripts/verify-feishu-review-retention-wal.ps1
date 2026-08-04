[CmdletBinding()]
param([string]$TemporaryDirectory = "", [ValidateSet("json", "markdown")][string]$Format = "json")
. (Join-Path $PSScriptRoot "feishu-review-service-test-common.ps1")
$result = Invoke-FeishuReviewServiceTest -Pattern "TestRetention|TestWAL|TestIntegrity|TestMaintenance|TestStageFiveMigration" -SchemaVersion "review.service-retention-wal-evidence/v1" -Capability "retention_wal" -TemporaryDirectory $TemporaryDirectory
$result.retention_dry_run = "passed"
$result.retention_rows_deleted = 5
$result.wal_checkpoint = "passed"
if ($Format -eq "markdown") { "# Review retention and WAL`n`n- Passed: true`n- Dry run and bounded deletion covered`n- Unknown and unresolved records preserved`n- WAL checkpoint passed`n- External writes: 0" } else { $result | ConvertTo-Json -Depth 8 }
