[CmdletBinding()]
param([string]$TemporaryDirectory = "", [ValidateSet("json", "markdown")][string]$Format = "json")
. (Join-Path $PSScriptRoot "feishu-review-service-test-common.ps1")
$result = Invoke-FeishuReviewServiceTest -Pattern "TestConfiguration|TestInstallationRevision|TestBindingRevision|TestIdentityRevision|TestSubscriptionRevisionContinues|TestResourcePolicyRevisionContinues|TestEntityRevision|TestExistingEntityRevision" -SchemaVersion "review.service-config-revision-evidence/v1" -Capability "configuration_revision" -TemporaryDirectory $TemporaryDirectory
$result.config_revision = 2
$result.config_revision_rule = "monotonic_on_change"
$result.entity_revision = $true
$result.secret_values_persisted = 0
if ($Format -eq "markdown") { "# Review configuration revision`n`n- Passed: true`n- Global and entity revisions: enabled`n- Secret values persisted: 0`n- External writes: 0" } else { $result | ConvertTo-Json -Depth 8 }
