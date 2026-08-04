[CmdletBinding()]
param([string]$OutputFile = '')

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'feishu-review-final-validation-common.ps1')
Assert-ReviewFinalRealValidation -RequireExternalWrites
$null = & (Join-Path $PSScriptRoot 'test-real-review-validation-preflight.ps1') -RequireExternalWrites
$prefix = Assert-ReviewFinalTestPrefix
$runID = Get-ReviewFinalRunID
$counts = New-ReviewFinalCounts
$token = Get-ReviewFinalTenantToken $counts
$headers = Get-ReviewFinalBearerHeader $token
$assignee = Get-ReviewFinalEnvironmentValue 'FEISHU_REVIEW_TEST_ASSIGNEE_OPEN_ID'
if ([string]::IsNullOrWhiteSpace($assignee)) { throw 'FEISHU_REVIEW_TEST_ASSIGNEE_OPEN_ID is required for the dedicated test task' }
$due = [DateTimeOffset]::UtcNow.AddDays(2).ToUnixTimeMilliseconds().ToString()
$members = @(@{ id = $assignee; type = 'user'; role = 'assignee' })
$taskBody = @{ summary = "[$runID] Review $([Environment]::GetEnvironmentVariable('GITLINK_REVIEW_TEST_REPOSITORY')) PR #$([Environment]::GetEnvironmentVariable('GITLINK_REVIEW_TEST_PR_NUMBER'))"; description = "$prefix test-only task; GitLink writes: 0"; due = @{ timestamp = $due; is_all_day = $true }; members = $members }
$create = Invoke-ReviewFinalRequest -Target feishu -Method POST -Uri 'https://open.feishu.cn/open-apis/task/v2/tasks' -Counts $counts -Headers $headers -Body $taskBody
$taskID = ($create.Content | ConvertFrom-Json).data.task.guid
if ([string]::IsNullOrWhiteSpace($taskID)) { throw 'Task create response missing task GUID' }
$newDue = [DateTimeOffset]::UtcNow.AddDays(3).ToUnixTimeMilliseconds().ToString()
$patch = Invoke-ReviewFinalRequest -Target feishu -Method PATCH -Uri "https://open.feishu.cn/open-apis/task/v2/tasks/$([uri]::EscapeDataString($taskID))" -Counts $counts -Headers $headers -Body @{ task = @{ due = @{ timestamp = $newDue; is_all_day = $true } }; update_fields = @('due') }
Write-ReviewFinalResult -OutputFile $OutputFile -Result ([ordered]@{
    schema_version = 'feishu.review-real-task/v1'; passed = $true; validation_mode = 'real_platform'; run_id = $runID
    task_id_hash = Get-ReviewFinalHash $taskID; assignee_hash = Get-ReviewFinalHash $assignee
    create_status = $create.StatusCode; patch_status = $patch.StatusCode; duplicate_task_created = $false
    cross_chat_shared = $false; external_writes = $counts; gitlink_writes = 0
})
