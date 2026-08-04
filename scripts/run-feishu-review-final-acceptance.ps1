[CmdletBinding(DefaultParameterSetName='Offline')]
param(
    [Parameter(ParameterSetName='Offline')][switch]$OfflineOnly,
    [Parameter(ParameterSetName='Real')][switch]$RealPlatform,
    [switch]$SkipGitHubActions,
    [string]$OutputDirectory='.\evidence\final',
    [Parameter(ParameterSetName='Real')][switch]$ConfirmExternalWrites
)

$ErrorActionPreference='Stop'
$root=Split-Path -Parent $PSScriptRoot
$results=Join-Path $root '.local/final-acceptance'
$null=New-Item -ItemType Directory -Force $results
if(-not $OfflineOnly -and -not $RealPlatform){$OfflineOnly=$true}

function Invoke-Checked {
    param([string]$Name,[scriptblock]$Command)
    $timer=[Diagnostics.Stopwatch]::StartNew()
    & $Command
    if($LASTEXITCODE -ne 0){throw "$Name failed with exit $LASTEXITCODE"}
    $timer.Stop()
    return $timer.ElapsedMilliseconds
}

function Invoke-IsolatedPowerShell {
    param(
        [Parameter(Mandatory)][string]$Name,
        [Parameter(Mandatory)][string]$ScriptPath,
        [string[]]$ArgumentList = @(),
        [string]$CaptureOutput
    )
    $hostPath = (Get-Process -Id $PID).Path
    $resolvedScript = [IO.Path]::GetFullPath((Join-Path $root $ScriptPath))
    if ($CaptureOutput) {
        & $hostPath -NoProfile -File $resolvedScript @ArgumentList |
            Set-Content -LiteralPath $CaptureOutput -Encoding utf8
    } else {
        & $hostPath -NoProfile -File $resolvedScript @ArgumentList | Out-Null
    }
    if ($LASTEXITCODE -ne 0) {
        throw "$Name failed with exit $LASTEXITCODE"
    }
}

Push-Location $root
try {
    if($OfflineOnly){
        Write-Host 'FINAL-ACCEPTANCE offline: full repository test/vet/build'
        $testMS=Invoke-Checked 'full go test' { go test ./... -count=1 | Out-Null }
        $vetMS=Invoke-Checked 'full go vet' { go vet ./... | Out-Null }
        $buildMS=Invoke-Checked 'full go build' { go build ./... | Out-Null }
        [ordered]@{schema_version='feishu.review-full-repository-tests/v1';passed=$true;validation_mode='offline';go_test='passed';go_vet='passed';go_build='passed';test_duration_ms=$testMS;vet_duration_ms=$vetMS;build_duration_ms=$buildMS;commit_sha=(& git rev-parse HEAD).Trim()}|ConvertTo-Json|Set-Content (Join-Path $results 'full-repository-tests.json') -Encoding utf8
        Write-Host 'FINAL-ACCEPTANCE offline: PowerShell contracts'
        Invoke-IsolatedPowerShell 'PowerShell contracts' 'scripts/test-feishu-review-powershell-cross-platform.ps1' @('-OutputFile',(Join-Path $results 'powershell.json'))
        Write-Host 'FINAL-ACCEPTANCE offline: Node and WeCom'
        Push-Location bridges/wecom
        try { npm ci --ignore-scripts | Out-Null; npm test | Out-Null; if($LASTEXITCODE -ne 0){throw 'WeCom Node tests failed'} } finally { Pop-Location }
        Write-Host 'FINAL-ACCEPTANCE offline: failure injection'
        Invoke-IsolatedPowerShell 'failure injection' 'scripts/inject-feishu-review-final-failures.ps1' @('-OutputFile',(Join-Path $results 'failure-injection.json'))
        Write-Host 'FINAL-ACCEPTANCE offline: backup and restore'
        Invoke-IsolatedPowerShell 'backup and restore' 'scripts/verify-feishu-review-backup-restore.ps1' @() (Join-Path $results 'backup-restore.json')
        Write-Host 'FINAL-ACCEPTANCE offline: secret scan'
        Invoke-IsolatedPowerShell 'secret scan' 'scripts/scan-feishu-review-secrets.ps1' @('-OutputFile',(Join-Path $results 'secret-scan.json'))
    } else {
        if(-not $ConfirmExternalWrites){throw '-RealPlatform requires -ConfirmExternalWrites'}
        $offline=Join-Path $results 'full-repository-tests.json'
        if(-not(Test-Path $offline)-or -not(Get-Content $offline -Raw|ConvertFrom-Json).passed){throw 'Offline acceptance must pass before RealPlatform'}
        if($env:FEISHU_REVIEW_REAL_VALIDATION -ne '1' -or $env:FEISHU_REVIEW_CONFIRM_EXTERNAL_WRITES -cne 'YES'){throw 'both explicit real validation environment switches are required'}
        Invoke-IsolatedPowerShell 'real validation preflight' 'scripts/test-real-review-validation-preflight.ps1' @('-RequireExternalWrites','-OutputFile',(Join-Path $results 'preflight.json'))
        Invoke-IsolatedPowerShell 'real GitLink read' 'scripts/validate-real-gitlink-review-read.ps1' @('-OutputFile',(Join-Path $results 'real-gitlink-read.json'))
        Invoke-IsolatedPowerShell 'real webhook' 'scripts/validate-real-gitlink-webhook.ps1' @('-OutputFile',(Join-Path $results 'real-webhook.json'))
        Invoke-IsolatedPowerShell 'real dual chat' 'scripts/validate-real-feishu-dual-chat.ps1' @('-OutputFile',(Join-Path $results 'real-dual-chat.json'))
        Invoke-IsolatedPowerShell 'real card and reply' 'scripts/validate-real-feishu-card-reply.ps1' @('-OutputFile',(Join-Path $results 'real-card-reply.json'))
        Invoke-IsolatedPowerShell 'real Base' 'scripts/validate-real-feishu-base.ps1' @('-OutputFile',(Join-Path $results 'real-base.json'))
        Invoke-IsolatedPowerShell 'real Doc' 'scripts/validate-real-feishu-doc.ps1' @('-OutputFile',(Join-Path $results 'real-doc.json'))
        Invoke-IsolatedPowerShell 'real Task' 'scripts/validate-real-feishu-task.ps1' @('-OutputFile',(Join-Path $results 'real-task.json'))
        Invoke-IsolatedPowerShell 'secret scan' 'scripts/scan-feishu-review-secrets.ps1' @('-OutputFile',(Join-Path $results 'secret-scan.json'))
    }
    Write-Host 'FINAL-ACCEPTANCE: export evidence'
    ./scripts/export-feishu-review-final-evidence.ps1 -OutputDirectory $OutputDirectory -ResultsDirectory $results -SkipGitHubActions:$SkipGitHubActions
    Write-Host 'FINAL-ACCEPTANCE: post-export secret scan'
    Invoke-IsolatedPowerShell 'post-export secret scan' 'scripts/scan-feishu-review-secrets.ps1' @('-OutputFile',(Join-Path $results 'secret-scan.json'))
    ./scripts/export-feishu-review-final-evidence.ps1 -OutputDirectory $OutputDirectory -ResultsDirectory $results -SkipGitHubActions:$SkipGitHubActions
} finally { Pop-Location }
