[CmdletBinding()]
param([string]$OutputFile = '')

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'feishu-review-final-validation-common.ps1')
Assert-ReviewFinalRealValidation
$null = & (Join-Path $PSScriptRoot 'test-real-review-validation-preflight.ps1')
$url = [uri](Get-ReviewFinalEnvironmentValue 'FEISHU_REVIEW_PUBLIC_WEBHOOK_URL')
$secret = Get-ReviewFinalEnvironmentValue 'FEISHU_REVIEW_WEBHOOK_SECRET'
$runID = Get-ReviewFinalRunID
$delivery = "delivery-$runID"
$body = @{ event_type = 'pull_request'; action = 'synchronize'; repository = (Get-ReviewFinalEnvironmentValue 'GITLINK_REVIEW_TEST_REPOSITORY'); pull_request = @{ number = [int](Get-ReviewFinalEnvironmentValue 'GITLINK_REVIEW_TEST_PR_NUMBER') }; validation_run_id = $runID } | ConvertTo-Json -Depth 8 -Compress
$timestamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds().ToString()
$hmac = [Security.Cryptography.HMACSHA256]::new([Text.Encoding]::UTF8.GetBytes($secret))
try { $signature = (([BitConverter]::ToString($hmac.ComputeHash([Text.Encoding]::UTF8.GetBytes("$timestamp.$body"))) -replace '-', '')).ToLowerInvariant() } finally { $hmac.Dispose() }
$headers = @{ 'X-GitLink-Delivery' = $delivery; 'X-GitLink-Timestamp' = $timestamp; 'X-GitLink-Signature-256' = "sha256=$signature" }
$badHeaders = @{}
foreach ($key in $headers.Keys) { $badHeaders[$key] = $headers[$key] }
$badHeaders['X-GitLink-Signature-256'] = 'sha256=invalid'
$bad = Invoke-WebRequest -Uri $url -Method POST -Headers $badHeaders -Body $body -ContentType 'application/json' -SkipHttpErrorCheck
$wrongType = Invoke-WebRequest -Uri $url -Method POST -Headers $headers -Body $body -ContentType 'text/plain' -SkipHttpErrorCheck
$accepted = Invoke-WebRequest -Uri $url -Method POST -Headers $headers -Body $body -ContentType 'application/json' -SkipHttpErrorCheck
$duplicate = Invoke-WebRequest -Uri $url -Method POST -Headers $headers -Body $body -ContentType 'application/json' -SkipHttpErrorCheck
if ($bad.StatusCode -ne 401 -or $wrongType.StatusCode -ne 415 -or $accepted.StatusCode -ne 202 -or $duplicate.StatusCode -ne 202) { throw 'public webhook status contract failed' }
Write-ReviewFinalResult -OutputFile $OutputFile -Result ([ordered]@{
    schema_version = 'feishu.review-real-webhook/v1'; passed = $true; validation_mode = 'real_platform'; run_id = $runID
    delivery_hash = Get-ReviewFinalHash $delivery; event_type = 'pull_request'; action = 'synchronize'
    bad_signature_status = $bad.StatusCode; wrong_content_type_status = $wrongType.StatusCode
    accepted_status = $accepted.StatusCode; duplicate_status = $duplicate.StatusCode
    platform_delivery = 'not_executed_by_script'; gitlink_write_methods = 0
})
