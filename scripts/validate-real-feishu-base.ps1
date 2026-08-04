[CmdletBinding()]
param([string]$OutputFile = '')

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'feishu-review-final-validation-common.ps1')
Assert-ReviewFinalRealValidation -RequireExternalWrites
$null = & (Join-Path $PSScriptRoot 'test-real-review-validation-preflight.ps1') -RequireExternalWrites
$prefix = Assert-ReviewFinalTestPrefix
$runID = Get-ReviewFinalRunID
$uniqueKey = "$prefix-base-$runID"
$counts = New-ReviewFinalCounts
$token = Get-ReviewFinalTenantToken $counts
$headers = Get-ReviewFinalBearerHeader $token
$app = Get-ReviewFinalEnvironmentValue 'FEISHU_REVIEW_BASE_APP_TOKEN'
$table = Get-ReviewFinalEnvironmentValue 'FEISHU_REVIEW_TABLE_ID'
$base = "https://open.feishu.cn/open-apis/bitable/v1/apps/$([uri]::EscapeDataString($app))/tables/$([uri]::EscapeDataString($table))/records"
$filter = "CurrentValue.[unique_key]=`"$uniqueKey`""
$searchBody = @{ filter = @{ conjunction = 'and'; conditions = @(@{ field_name = 'unique_key'; operator = 'is'; value = @($uniqueKey) }) }; page_size = 20 }
$search = Invoke-ReviewFinalRequest -Target feishu -Method POST -Uri "$base/search" -Counts $counts -Headers $headers -Body $searchBody
$decoded = $search.Content | ConvertFrom-Json
$items = @($decoded.data.items)
if ($items.Count -gt 1) { throw 'test Base contains duplicate unique_key records; manual reconciliation required' }
if ($items.Count -eq 0) {
    $create = Invoke-ReviewFinalRequest -Target feishu -Method POST -Uri $base -Counts $counts -Headers $headers -Body @{ fields = @{ unique_key = $uniqueKey; repository = (Get-ReviewFinalEnvironmentValue 'GITLINK_REVIEW_TEST_REPOSITORY'); pr_number = [int](Get-ReviewFinalEnvironmentValue 'GITLINK_REVIEW_TEST_PR_NUMBER'); collaboration_status = 'unassigned'; validation_run_id = $runID } }
    $recordID = ($create.Content | ConvertFrom-Json).data.record.record_id
    $initialAction = 'created'
} else {
    $recordID = $items[0].record_id
    $initialAction = 'found'
}
if ([string]::IsNullOrWhiteSpace($recordID)) { throw 'Base validation response missing record_id' }
$secondSearch = Invoke-ReviewFinalRequest -Target feishu -Method POST -Uri "$base/search" -Counts $counts -Headers $headers -Body $searchBody
$secondItems = @(($secondSearch.Content | ConvertFrom-Json).data.items)
if ($secondItems.Count -ne 1) { throw 'Base idempotency search did not return exactly one record' }
$update = Invoke-ReviewFinalRequest -Target feishu -Method PUT -Uri "$base/$([uri]::EscapeDataString($recordID))" -Counts $counts -Headers $headers -Body @{ fields = @{ collaboration_status = 'reviewing'; validation_run_id = $runID } }
Write-ReviewFinalResult -OutputFile $OutputFile -Result ([ordered]@{
    schema_version = 'feishu.review-real-base/v1'; passed = $true; validation_mode = 'real_platform'; run_id = $runID
    unique_key_hash = Get-ReviewFinalHash $uniqueKey; remote_id_hash = Get-ReviewFinalHash $recordID
    initial_action = $initialAction; duplicate_records = 0; update_status = $update.StatusCode
    human_fields_overwritten = $false; external_writes = $counts; gitlink_writes = 0
})
