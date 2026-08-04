[CmdletBinding()]
param([string]$OutputFile = '')

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'feishu-review-final-validation-common.ps1')
Assert-ReviewFinalRealValidation -RequireExternalWrites
$null = & (Join-Path $PSScriptRoot 'test-real-review-validation-preflight.ps1') -RequireExternalWrites
$counts = New-ReviewFinalCounts
$token = Get-ReviewFinalTenantToken $counts
$headers = Get-ReviewFinalBearerHeader $token
$chat = Get-ReviewFinalEnvironmentValue 'FEISHU_REVIEW_TEST_CHAT_A'
$runID = Get-ReviewFinalRunID
$card = @{ config = @{ wide_screen_mode = $true }; header = @{ title = @{ tag = 'plain_text'; content = "[$runID] GitLink Review validation" }; template = 'blue' }; elements = @(@{ tag = 'markdown'; content = "**Test-only card**`nRun: `$runID`nGitLink writes: 0" }) }
$create = Invoke-ReviewFinalRequest -Target feishu -Method POST -Uri 'https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=chat_id' -Counts $counts -Headers $headers -Body @{ receive_id = $chat; msg_type = 'interactive'; content = ($card | ConvertTo-Json -Depth 10 -Compress) }
$created = $create.Content | ConvertFrom-Json
$messageID = $created.data.message_id
if ([string]::IsNullOrWhiteSpace($messageID)) { throw 'card create response missing message_id' }
$card.header.template = 'green'
$patch = Invoke-ReviewFinalRequest -Target feishu -Method PATCH -Uri "https://open.feishu.cn/open-apis/im/v1/messages/$([uri]::EscapeDataString($messageID))" -Counts $counts -Headers $headers -Body @{ content = ($card | ConvertTo-Json -Depth 10 -Compress) }
$reply = Invoke-ReviewFinalRequest -Target feishu -Method POST -Uri "https://open.feishu.cn/open-apis/im/v1/messages/$([uri]::EscapeDataString($messageID))/reply" -Counts $counts -Headers $headers -Body @{ msg_type = 'text'; content = (@{ text = "[$runID] lightweight reply; GitLink writes: 0" } | ConvertTo-Json -Compress) }
Write-ReviewFinalResult -OutputFile $OutputFile -Result ([ordered]@{
    schema_version = 'feishu.review-real-card-reply/v1'; passed = $true; validation_mode = 'real_platform'; run_id = $runID
    chat_id_hash = Get-ReviewFinalHash $chat; message_id_hash = Get-ReviewFinalHash $messageID
    card_create_status = $create.StatusCode; card_patch_status = $patch.StatusCode; reply_status = $reply.StatusCode
    external_writes = $counts; gitlink_writes = 0
})
