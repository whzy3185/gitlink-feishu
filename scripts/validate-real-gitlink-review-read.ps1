[CmdletBinding()]
param([string]$OutputFile = '', [string]$StateDirectory = '')

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'feishu-review-final-validation-common.ps1')
Assert-ReviewFinalRealValidation
$null = & (Join-Path $PSScriptRoot 'test-real-review-validation-preflight.ps1')
$root = Split-Path -Parent $PSScriptRoot
$repository = Get-ReviewFinalEnvironmentValue 'GITLINK_REVIEW_TEST_REPOSITORY'
$owner, $repo = $repository.Split('/', 2)
$number = Get-ReviewFinalEnvironmentValue 'GITLINK_REVIEW_TEST_PR_NUMBER'
$token = Resolve-ReviewFinalCredentialReference (Get-ReviewFinalEnvironmentValue 'GITLINK_REVIEW_CREDENTIAL_REF')
$runRoot = if ($StateDirectory) { [IO.Path]::GetFullPath($StateDirectory) } else { Join-Path ([IO.Path]::GetTempPath()) ("review-final-gitlink-" + [Guid]::NewGuid().ToString('N')) }
$null = New-Item -ItemType Directory -Force -Path $runRoot
$methodLog = Join-Path $runRoot 'http-methods.json'
$configDir = Join-Path $runRoot 'config'
$null = New-Item -ItemType Directory -Force -Path $configDir
@"
base_url: $(Get-ReviewFinalEnvironmentValue 'GITLINK_REVIEW_BASE_URL')
default_format: json
"@ | Set-Content -LiteralPath (Join-Path $configDir 'config.yaml') -Encoding utf8
$oldToken, $oldConfig, $oldLog = $env:GITLINK_TOKEN, $env:GITLINK_CONFIG_DIR, $env:GITLINK_REVIEW_READ_ONLY_METHOD_LOG
try {
    $env:GITLINK_TOKEN = $token; $env:GITLINK_CONFIG_DIR = $configDir; $env:GITLINK_REVIEW_READ_ONLY_METHOD_LOG = $methodLog
    $firstPath = Join-Path $runRoot 'first.json'; $secondPath = Join-Path $runRoot 'second.json'
    Push-Location $root
    try {
        $first = & go run . workflow +review-context --owner $owner --repo $repo --number $number --format json 2>&1
        if ($LASTEXITCODE -ne 0) { throw 'first GitLink read failed' }
        ($first -join [Environment]::NewLine) | Set-Content -LiteralPath $firstPath -Encoding utf8
        $second = & go run . workflow +review-context --owner $owner --repo $repo --number $number --format json 2>&1
        if ($LASTEXITCODE -ne 0) { throw 'second GitLink read failed' }
        ($second -join [Environment]::NewLine) | Set-Content -LiteralPath $secondPath -Encoding utf8
    } finally { Pop-Location }
    $a = Get-Content -LiteralPath $firstPath -Raw | ConvertFrom-Json
    $b = Get-Content -LiteralPath $secondPath -Raw | ConvertFrom-Json
    $methods = Get-Content -LiteralPath $methodLog -Raw | ConvertFrom-Json
    if ($a.repository -ne $repository -or $a.pull_request -ne [int]$number -or [string]::IsNullOrWhiteSpace($a.current_head_sha)) { throw 'GitLink context identity is incomplete' }
    if ($a.work_item.source_fingerprint -ne $b.work_item.source_fingerprint) { throw 'repeated GitLink read changed source fingerprint' }
    if ($methods.POST -ne 0 -or $methods.PUT -ne 0 -or $methods.PATCH -ne 0 -or $methods.DELETE -ne 0) { throw 'GitLink write method count is nonzero' }
    Write-ReviewFinalResult -OutputFile $OutputFile -Result ([ordered]@{
        schema_version = 'feishu.review-real-gitlink-read/v1'; passed = $true; validation_mode = 'real_platform'; run_id = Get-ReviewFinalRunID
        repository = $repository; pr_number = [int]$number; title_present = -not [string]::IsNullOrWhiteSpace($a.pr.title)
        author = if ([string]::IsNullOrWhiteSpace($a.pr.user.login)) { 'unknown' } else { $a.pr.user.login }
        head_sha = $a.current_head_sha; collection_status = $a.collection_status; partial = [bool]$a.partial
        source_fingerprint = $a.work_item.source_fingerprint; repeated_read_idempotent = $true
        sections = @($a.section_statuses); http_methods = $methods
    })
} finally {
    $env:GITLINK_TOKEN, $env:GITLINK_CONFIG_DIR, $env:GITLINK_REVIEW_READ_ONLY_METHOD_LOG = $oldToken, $oldConfig, $oldLog
    if (-not $StateDirectory) { Remove-Item -LiteralPath $runRoot -Recurse -Force -ErrorAction SilentlyContinue }
}
