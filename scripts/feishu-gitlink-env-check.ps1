param(
  [ValidateSet("stable", "open-platform", "all")]
  [string]$Layer = "stable"
)

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent $PSScriptRoot
$LocalEnv = Join-Path $RepoRoot ".local/feishu-gitlink.env.ps1"

function Redact-Value {
  param([string]$Value)
  if ([string]::IsNullOrWhiteSpace($Value)) { return "" }
  if ($Value.Length -le 8) { return "***" }
  return "$($Value.Substring(0, 4))...$($Value.Substring($Value.Length - 4))"
}

function Show-Var {
  param(
    [string]$Name,
    [bool]$Sensitive = $true
  )
  $value = [Environment]::GetEnvironmentVariable($Name)
  if ([string]::IsNullOrWhiteSpace($value)) {
    Write-Host ("{0}: missing" -f $Name)
    return $false
  }
  if ($Sensitive) {
    Write-Host ("{0}: set ({1})" -f $Name, (Redact-Value $value))
  } else {
    Write-Host ("{0}: set ({1})" -f $Name, $value)
  }
  return $true
}

function Test-Group {
  param(
    [string]$Title,
    [string[]]$Names,
    [string[]]$Optional = @(),
    [string[]]$NonSensitive = @()
  )
  Write-Host ""
  Write-Host "[$Title]"
  $missingRequired = @()
  foreach ($name in $Names) {
    $isOptional = $Optional -contains $name
    $isSensitive = -not ($NonSensitive -contains $name)
    $present = Show-Var -Name $name -Sensitive:$isSensitive
    if (-not $present -and -not $isOptional) {
      $missingRequired += $name
    }
  }
  return $missingRequired
}

if (Test-Path $LocalEnv) {
  . $LocalEnv
  Write-Host "Loaded local env: .local/feishu-gitlink.env.ps1"
} else {
  Write-Host "No local env file found. Run scripts/feishu-gitlink-setup.ps1 or copy .local/feishu-gitlink.env.example.ps1."
}

$allMissing = @()

if ($Layer -eq "stable" -or $Layer -eq "all") {
  $allMissing += Test-Group -Title "Stable webhook" -Names @("FEISHU_WEBHOOK_URL", "FEISHU_WEBHOOK_SECRET") -Optional @("FEISHU_WEBHOOK_SECRET")
}

if ($Layer -eq "open-platform" -or $Layer -eq "all") {
  $allMissing += Test-Group -Title "Open Platform app" -Names @("FEISHU_APP_ID", "FEISHU_APP_SECRET")
  $allMissing += Test-Group -Title "DocX / Wiki" -Names @("FEISHU_WIKI_URL", "FEISHU_WIKI_NODE_TOKEN", "FEISHU_FOLDER_TOKEN", "FEISHU_DOCUMENT_ID") -Optional @("FEISHU_WIKI_URL", "FEISHU_WIKI_NODE_TOKEN", "FEISHU_FOLDER_TOKEN", "FEISHU_DOCUMENT_ID")
  $allMissing += Test-Group -Title "Base / Bitable" -Names @("FEISHU_BASE_APP_TOKEN", "FEISHU_REPORT_TABLE_ID", "FEISHU_ISSUE_TABLE_ID", "FEISHU_PR_TABLE_ID", "FEISHU_CONTRIBUTOR_TABLE_ID", "FEISHU_TASK_TABLE_ID") -Optional @("FEISHU_CONTRIBUTOR_TABLE_ID", "FEISHU_TASK_TABLE_ID")
  $allMissing += Test-Group -Title "Feishu Task" -Names @("FEISHU_TASK_PROJECT_ID", "FEISHU_TASK_SECTION_ID") -Optional @("FEISHU_TASK_PROJECT_ID", "FEISHU_TASK_SECTION_ID")
}

if ($Layer -eq "all") {
  $allMissing += Test-Group -Title "GitLink test input" -Names @("GITLINK_OWNER", "GITLINK_REPO", "GITLINK_TEST_PR_IDS", "GITLINK_TOKEN") -Optional @("GITLINK_TEST_PR_IDS", "GITLINK_TOKEN") -NonSensitive @("GITLINK_OWNER", "GITLINK_REPO", "GITLINK_TEST_PR_IDS")
}

if ($allMissing.Count -gt 0) {
  Write-Host ""
  Write-Host "Missing required values for layer '$Layer': $($allMissing -join ', ')"
  exit 1
}

Write-Host ""
Write-Host "Layer '$Layer' is ready."
exit 0
