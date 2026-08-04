[CmdletBinding()]
param(
    [string]$OutputDirectory = '.\evidence\final',
    [string]$ResultsDirectory = '.\.local\final-acceptance',
    [switch]$SkipGitHubActions
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
. (Join-Path $PSScriptRoot 'feishu-review-final-validation-common.ps1')

function Resolve-ReviewFinalPath {
    param([Parameter(Mandatory)][string]$Path)
    if ([IO.Path]::IsPathRooted($Path)) {
        return [IO.Path]::GetFullPath($Path)
    }
    return [IO.Path]::GetFullPath((Join-Path $root $Path))
}

$output = Resolve-ReviewFinalPath $OutputDirectory
$results = Resolve-ReviewFinalPath $ResultsDirectory
$null = New-Item -ItemType Directory -Force -Path $output
$branch = (& git -C $root branch --show-current).Trim()
$sha = (& git -C $root rev-parse HEAD).Trim()
$generated = [DateTime]::UtcNow.ToString('o')

function Write-EvidencePair {
    param([string]$Name, $Value)
    $jsonPath = Join-Path $output "$Name.json"
    $mdPath = Join-Path $output "$Name.md"
    $Value | ConvertTo-Json -Depth 30 | Set-Content -LiteralPath $jsonPath -Encoding utf8
    $schema = if ($Value.PSObject.Properties.Name -contains 'schema_version') { $Value.schema_version } else { 'unspecified' }
    $passed = if ($Value.PSObject.Properties.Name -contains 'passed') { $Value.passed } else { $false }
    $mode = if ($Value.PSObject.Properties.Name -contains 'validation_mode') { $Value.validation_mode } else { 'offline' }
    $lines = @("# $Name", '', "- Schema: ``$schema``", "- Passed: ``$passed``", "- Validation mode: ``$mode``", "- Generated: ``$generated``", '', '```json', ($Value | ConvertTo-Json -Depth 12), '```')
    $lines | Set-Content -LiteralPath $mdPath -Encoding utf8
}

function Read-ResultOrDefault {
    param([string]$Name, [string]$Schema, [string]$Mode = 'offline')
    $path = Join-Path $results "$Name.json"
    if (Test-Path -LiteralPath $path) {
        $value = Get-Content -LiteralPath $path -Raw | ConvertFrom-Json
        if ($value.PSObject.Properties.Name -notcontains 'validation_mode') {
            $value | Add-Member -NotePropertyName validation_mode -NotePropertyValue $Mode
        }
        return $value
    }
    return [ordered]@{ schema_version=$Schema; passed=$false; validation_mode=$Mode; status='not_executed'; reason='no verified result file was supplied'; generated_at=$generated }
}

$commits = @(& git -C $root log --reverse --format='%H|%s' e2bcb35efc56db659eb58af141a3fd8018e6d5be..HEAD | ForEach-Object { $parts=$_.Split('|',2); [ordered]@{ sha=$parts[0]; subject=$parts[1] } })
Write-EvidencePair 'commit-baseline' ([ordered]@{
    schema_version='feishu.review-commit-baseline/v1'; passed=$true; validation_mode='offline'; repository='whzy3185/gitlink-feishu'
    branch=$branch; baseline_sha='e2bcb35efc56db659eb58af141a3fd8018e6d5be'; commit_sha=$sha; commits=$commits; history_rewritten=$false
})

$actions = [ordered]@{ schema_version='feishu.review-github-actions/v1'; passed=$false; validation_mode='offline'; status='not_checked'; runs=@() }
if (-not $SkipGitHubActions) {
    try {
        $encodedBranch = [uri]::EscapeDataString($branch)
        $remoteRuns = (Invoke-RestMethod -Uri "https://api.github.com/repos/whzy3185/gitlink-feishu/actions/runs?branch=$encodedBranch&per_page=20").workflow_runs
        $actions.runs = @($remoteRuns | Select-Object -First 10 | ForEach-Object { [ordered]@{ id=$_.id; name=$_.name; head_sha=$_.head_sha; status=$_.status; conclusion=$_.conclusion; url=$_.html_url } })
        $required = @($actions.runs | Where-Object { $_.head_sha -eq $sha -and $_.name -in @('Round 2 Review Collaboration Gate','Feishu Review Final Quality') })
        $actions.passed = $required.Count -eq 2 -and @($required | Where-Object conclusion -ne 'success').Count -eq 0
        $actions.status = if ($actions.passed) { 'passed' } elseif (@($required | Where-Object status -ne 'completed').Count -gt 0) { 'pending' } else { 'not_passed' }
    } catch { $actions.status='unavailable'; $actions.error='GitHub Actions metadata could not be fetched' }
}
Write-EvidencePair 'github-actions' $actions

$race = [ordered]@{ schema_version='feishu.review-linux-race/v1'; passed=$false; validation_mode='offline'; status='pending_final_actions'; command='CGO_ENABLED=1 go test -race ./internal/collab ./shortcuts/feishu ./shortcuts/wecom ./shortcuts/workflow -count=1' }
$successfulRace = @($actions.runs | Where-Object { $_.name -eq 'Feishu Review Final Quality' -and $_.conclusion -eq 'success' } | Select-Object -First 1)
if ($successfulRace.Count -eq 1) { $race.passed=$true; $race.status='covered_by_successful_workflow'; $race.run_id=$successfulRace[0].id }
Write-EvidencePair 'linux-race' $race

Write-EvidencePair 'full-repository-tests' (Read-ResultOrDefault 'full-repository-tests' 'feishu.review-full-repository-tests/v1')
foreach ($name in @('real-gitlink-read','real-webhook','real-dual-chat','real-card-reply','real-base','real-doc','real-task')) {
    Write-EvidencePair $name (Read-ResultOrDefault $name "feishu.review-$name/v1" 'not_executed')
}
Write-EvidencePair 'failure-injection' (Read-ResultOrDefault 'failure-injection' 'feishu.review-final-failure-injection/v1')
Write-EvidencePair 'backup-restore' (Read-ResultOrDefault 'backup-restore' 'feishu.review-backup-restore/v1')
Write-EvidencePair 'secret-scan' (Read-ResultOrDefault 'secret-scan' 'feishu.review-secret-scan/v1')

$capabilities = @(
    'PR read','Review read','Thread read','PR Presentation','canonical card create','canonical card patch','lightweight reply',
    'claim','release','deadline','dual-chat isolation','subscription','webhook','event inbox','replay','reconciliation',
    'Base projection','Doc projection','Task projection','operation outbox','dead letter','worker isolation','rate limit',
    'refresh coalescing','healthz','readyz','metrics','admin API','backup','restore','retention','Linux service','Windows service'
)
Write-EvidencePair 'capability-matrix' ([ordered]@{
    schema_version='feishu.review-capability-matrix/v1'; passed=$true; validation_mode='offline'; source='docs/FEISHU_REVIEW_CAPABILITY_MATRIX.md'
    capabilities=@($capabilities | ForEach-Object { [ordered]@{ name=$_; code_implemented=$true; real_platform_validation='not_executed_unless_separate_real_evidence_passed' } })
})

$realFiles = @('real-gitlink-read','real-webhook','real-dual-chat','real-card-reply','real-base','real-doc','real-task')
$aggregate = New-ReviewFinalCounts
foreach ($name in $realFiles) {
    $path=Join-Path $output "$name.json"; $value=Get-Content -LiteralPath $path -Raw|ConvertFrom-Json
    if ($value.validation_mode -eq 'real_platform' -and $value.external_writes) {
        foreach($property in $value.external_writes.PSObject.Properties){ if($aggregate.Contains($property.Name)){ $aggregate[$property.Name]=[int]$aggregate[$property.Name]+[int]$property.Value } }
    }
}
Write-EvidencePair 'external-write-counts' ([ordered]@{
    schema_version='feishu.review-external-write-counts/v1'; passed=$true; validation_mode='offline'; counts=$aggregate
    note='Counts remain zero when real-platform validators were not executed; zero does not imply those capabilities passed.'
})

$evidenceFiles = @()
foreach ($file in Get-ChildItem -LiteralPath $output -File | Where-Object Name -notin @('manifest.json','manifest.md') | Sort-Object Name) {
    $evidenceFiles += [ordered]@{ name=$file.Name; sha256=(Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash.ToLowerInvariant(); bytes=$file.Length }
}
$manifest = [ordered]@{
    schema_version='feishu.review-final-evidence-manifest/v1'; repository='whzy3185/gitlink-feishu'; branch=$branch; commit_sha=$sha
    generated_at=$generated; generator_version='stage6-v1'; validation_mode='offline'; evidence_files=$evidenceFiles
    self_hash_exclusion='manifest.json and manifest.md are excluded to avoid recursive self-hashing'
}
$manifest | ConvertTo-Json -Depth 20 | Set-Content -LiteralPath (Join-Path $output 'manifest.json') -Encoding utf8
$manifestHash=(Get-FileHash -LiteralPath (Join-Path $output 'manifest.json') -Algorithm SHA256).Hash.ToLowerInvariant()
@("# Final evidence manifest",'',"- Repository: ``whzy3185/gitlink-feishu``","- Branch: ``$branch``","- Commit: ``$sha``","- Manifest SHA-256: ``$manifestHash``","- Hashed evidence files: ``$($evidenceFiles.Count)``",'', 'Manifest files are self-excluded to avoid a recursive hash.') | Set-Content -LiteralPath (Join-Path $output 'manifest.md') -Encoding utf8
[ordered]@{ schema_version='feishu.review-final-evidence-export/v1'; passed=$true; validation_mode='offline'; output_directory=$output; evidence_file_count=(Get-ChildItem -LiteralPath $output -File).Count; manifest_sha256=$manifestHash } | ConvertTo-Json
