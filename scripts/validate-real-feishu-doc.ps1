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
$folder = Get-ReviewFinalEnvironmentValue 'FEISHU_REVIEW_DOCUMENT_FOLDER_TOKEN'
$create = Invoke-ReviewFinalRequest -Target feishu -Method POST -Uri 'https://open.feishu.cn/open-apis/docx/v1/documents' -Counts $counts -Headers $headers -Body @{ folder_token = $folder; title = "[$runID] GitLink Review test snapshot" }
$document = ($create.Content | ConvertFrom-Json).data.document
$documentID = $document.document_id
if ([string]::IsNullOrWhiteSpace($documentID)) { throw 'Doc create response missing document_id' }
$snapshot = "[$runID] GitLink Review test snapshot`nRepository: $([Environment]::GetEnvironmentVariable('GITLINK_REVIEW_TEST_REPOSITORY'))`nPR: #$([Environment]::GetEnvironmentVariable('GITLINK_REVIEW_TEST_PR_NUMBER'))`nGitLink writes: 0"
if ($snapshot.Length -gt 4000) { throw 'bounded Doc snapshot exceeded 4000 characters' }
$block = @{ children = @(@{ block_type = 2; text = @{ elements = @(@{ text_run = @{ content = $snapshot; text_element_style = @{} } }); style = @{} } }) }
$append = Invoke-ReviewFinalRequest -Target feishu -Method POST -Uri "https://open.feishu.cn/open-apis/docx/v1/documents/$([uri]::EscapeDataString($documentID))/blocks/$([uri]::EscapeDataString($documentID))/children" -Counts $counts -Headers $headers -Body $block
$fingerprint = Get-ReviewFinalHash $snapshot
Write-ReviewFinalResult -OutputFile $OutputFile -Result ([ordered]@{
    schema_version = 'feishu.review-real-doc/v1'; passed = $true; validation_mode = 'real_platform'; run_id = $runID
    document_id_hash = Get-ReviewFinalHash $documentID; snapshot_fingerprint = $fingerprint
    create_status = $create.StatusCode; append_status = $append.StatusCode; snapshot_chars = $snapshot.Length
    repeated_identical_snapshot = 'not_appended_by_validator'; contains_full_chat_id = $false; contains_message_id = $false
    external_writes = $counts; gitlink_writes = 0
})
