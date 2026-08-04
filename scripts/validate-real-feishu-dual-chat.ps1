[CmdletBinding()]
param([string]$OutputFile = '', [int]$ObservationSeconds = 60)

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'feishu-review-final-validation-common.ps1')
Assert-ReviewFinalRealValidation -RequireExternalWrites
$null = & (Join-Path $PSScriptRoot 'test-real-review-validation-preflight.ps1') -RequireExternalWrites
$stateDB = [IO.Path]::GetFullPath((Get-ReviewFinalEnvironmentValue 'FEISHU_REVIEW_TEST_STATE_DB'))
if (-not (Test-Path -LiteralPath $stateDB)) { throw 'the dedicated test gateway must be running and its test state DB must exist' }
$root = Split-Path -Parent $PSScriptRoot
$pattern = 'TestTwoChats|TestChatScoped|TestTaskIsNeverShared|TestInstallationScopedProjectionIsShared|TestCanonical'
Push-Location $root
try { $offline = & go test ./shortcuts/feishu -run $pattern -count=1 2>&1; if ($LASTEXITCODE -ne 0) { throw ($offline -join [Environment]::NewLine) } } finally { Pop-Location }
$instructions = "In both explicitly configured test chats, send a fresh '$([Environment]::GetEnvironmentVariable('FEISHU_REVIEW_TEST_RESOURCE_PREFIX')) 查看 $([Environment]::GetEnvironmentVariable('GITLINK_REVIEW_TEST_REPOSITORY')) PR #$([Environment]::GetEnvironmentVariable('GITLINK_REVIEW_TEST_PR_NUMBER'))' command, then claim/set a test due date in chat A only."
Write-ReviewFinalResult -OutputFile $OutputFile -Result ([ordered]@{
    schema_version = 'feishu.review-real-dual-chat/v1'; passed = $false; validation_mode = 'real_platform'; run_id = Get-ReviewFinalRunID
    status = 'awaiting_observation'; observation_window_seconds = $ObservationSeconds; operator_instruction = $instructions
    chat_a_hash = Get-ReviewFinalHash (Get-ReviewFinalEnvironmentValue 'FEISHU_REVIEW_TEST_CHAT_A'); chat_b_hash = Get-ReviewFinalHash (Get-ReviewFinalEnvironmentValue 'FEISHU_REVIEW_TEST_CHAT_B')
    offline_isolation_contract = 'passed'; real_dual_chat_evidence = 'not_yet_observed'; gitlink_writes = 0
})
throw 'real dual-chat observations have not been supplied; evidence remains not passed'
