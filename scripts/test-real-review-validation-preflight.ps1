[CmdletBinding()]
param([switch]$RequireExternalWrites, [string]$OutputFile = '')

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'feishu-review-final-validation-common.ps1')
Assert-ReviewFinalRealValidation -RequireExternalWrites:$RequireExternalWrites

$required = @(
    'GITLINK_REVIEW_BASE_URL','GITLINK_REVIEW_CREDENTIAL_REF','GITLINK_REVIEW_TEST_REPOSITORY','GITLINK_REVIEW_TEST_PR_NUMBER',
    'FEISHU_APP_ID','FEISHU_APP_SECRET','FEISHU_REVIEW_TEST_CHAT_A','FEISHU_REVIEW_TEST_CHAT_B',
    'FEISHU_REVIEW_WEBHOOK_SECRET','FEISHU_REVIEW_PUBLIC_WEBHOOK_URL','FEISHU_REVIEW_BASE_APP_TOKEN',
    'FEISHU_REVIEW_TABLE_ID','FEISHU_REVIEW_DOCUMENT_FOLDER_TOKEN','FEISHU_REVIEW_TEST_STATE_DB',
    'FEISHU_REVIEW_TEST_BACKUP_DIR','FEISHU_REVIEW_TEST_RESOURCE_PREFIX','FEISHU_REVIEW_TEST_ADMIN_ADDRESS',
    'FEISHU_REVIEW_TEST_WEBHOOK_ADDRESS'
)
$missing = @($required | Where-Object { [string]::IsNullOrWhiteSpace((Get-ReviewFinalEnvironmentValue $_)) })
if ($missing) { throw "missing real validation configuration: $($missing -join ', ')" }

$repository = Get-ReviewFinalEnvironmentValue 'GITLINK_REVIEW_TEST_REPOSITORY'
if ($repository -notmatch '^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$') { throw 'test repository must use owner/repo' }
$number = 0
if (-not [int]::TryParse((Get-ReviewFinalEnvironmentValue 'GITLINK_REVIEW_TEST_PR_NUMBER'), [ref]$number) -or $number -le 0) { throw 'test PR number must be positive' }
$chatA = Get-ReviewFinalEnvironmentValue 'FEISHU_REVIEW_TEST_CHAT_A'
$chatB = Get-ReviewFinalEnvironmentValue 'FEISHU_REVIEW_TEST_CHAT_B'
if ($chatA -eq $chatB) { throw 'test chats A and B must be different' }
$webhook = [uri](Get-ReviewFinalEnvironmentValue 'FEISHU_REVIEW_PUBLIC_WEBHOOK_URL')
$gitlink = [uri](Get-ReviewFinalEnvironmentValue 'GITLINK_REVIEW_BASE_URL')
if ($webhook.Scheme -ne 'https' -or $gitlink.Scheme -ne 'https') { throw 'GitLink and public webhook URLs must use HTTPS' }
$null = Resolve-ReviewFinalCredentialReference (Get-ReviewFinalEnvironmentValue 'GITLINK_REVIEW_CREDENTIAL_REF')
$prefix = Assert-ReviewFinalTestPrefix
$stateDB = Get-ReviewFinalEnvironmentValue 'FEISHU_REVIEW_TEST_STATE_DB'
$backup = Get-ReviewFinalEnvironmentValue 'FEISHU_REVIEW_TEST_BACKUP_DIR'
if (-not $stateDB.ToLowerInvariant().EndsWith('.db') -or -not ([IO.Path]::GetFileName($stateDB).ToLowerInvariant().Contains('test'))) { throw 'state DB must be a dedicated test .db file' }
if (-not ([IO.Path]::GetFileName($backup).ToLowerInvariant().Contains('test'))) { throw 'backup directory must be dedicated to tests' }

$runID = Get-ReviewFinalRunID
if ($runID -notmatch '^review-final-[0-9]{8}T[0-9]{6}Z-[0-9a-f]{7,12}$') { throw 'validation run ID format is invalid' }
$admin = Get-ReviewFinalEnvironmentValue 'FEISHU_REVIEW_TEST_ADMIN_ADDRESS'
$hookAddress = Get-ReviewFinalEnvironmentValue 'FEISHU_REVIEW_TEST_WEBHOOK_ADDRESS'
if ($admin -eq $hookAddress) { throw 'admin and webhook listener addresses must differ' }
if (-not (Test-ReviewFinalListenerAvailable $admin)) { throw 'dedicated test admin listener port is unavailable' }
if (-not (Test-ReviewFinalListenerAvailable $hookAddress)) { throw 'dedicated test webhook listener port is unavailable' }

Write-ReviewFinalResult -OutputFile $OutputFile -Result ([ordered]@{
    schema_version = 'feishu.review-real-preflight/v1'; passed = $true; validation_mode = 'preflight'
    run_id = $runID; repository = $repository; pr_number = $number
    chat_a_hash = Get-ReviewFinalHash $chatA; chat_b_hash = Get-ReviewFinalHash $chatB
    test_resource_prefix = $prefix; state_database_hash = Get-ReviewFinalHash ([IO.Path]::GetFullPath($stateDB))
    backup_directory_hash = Get-ReviewFinalHash ([IO.Path]::GetFullPath($backup)); external_writes_confirmed = [bool]$RequireExternalWrites
})
