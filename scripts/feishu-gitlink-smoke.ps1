param(
  [ValidateSet("preview", "stable", "open-platform", "all")]
  [string]$Mode = "preview"
)

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent $PSScriptRoot
$LocalDir = Join-Path $RepoRoot ".local"
$LocalEnv = Join-Path $LocalDir "feishu-gitlink.env.ps1"
$ReportJSON = Join-Path $LocalDir "report.json"
$TerminalLog = Join-Path $RepoRoot "reports/feishu-real-smoke-terminal.log"
$DateStamp = Get-Date -Format "yyyyMMdd"
$SmokeReport = Join-Path $RepoRoot "reports/FEISHU_SMOKE_$DateStamp.md"
$Results = New-Object System.Collections.Generic.List[object]
$Notes = New-Object System.Collections.Generic.List[string]

function Env {
  param([string]$Name)
  return [Environment]::GetEnvironmentVariable($Name)
}

function Has-Env {
  param([string[]]$Names)
  foreach ($name in $Names) {
    if ([string]::IsNullOrWhiteSpace((Env $name))) { return $false }
  }
  return $true
}

function Redact-Value {
  param([string]$Value)
  if ([string]::IsNullOrWhiteSpace($Value)) { return "" }
  if ($Value.Length -le 8) { return "***" }
  return "$($Value.Substring(0, 4))...$($Value.Substring($Value.Length - 4))"
}

function Redact-Text {
  param([string]$Text)
  if ($null -eq $Text) { return "" }
  $redacted = $Text
  $names = @(
    "FEISHU_WEBHOOK_URL",
    "FEISHU_WEBHOOK_SECRET",
    "FEISHU_APP_ID",
    "FEISHU_APP_SECRET",
    "FEISHU_WIKI_URL",
    "FEISHU_WIKI_NODE_TOKEN",
    "FEISHU_FOLDER_TOKEN",
    "FEISHU_BASE_APP_TOKEN",
    "FEISHU_REPORT_TABLE_ID",
    "FEISHU_ISSUE_TABLE_ID",
    "FEISHU_PR_TABLE_ID",
    "FEISHU_CONTRIBUTOR_TABLE_ID",
    "FEISHU_TASK_TABLE_ID",
    "FEISHU_TASK_PROJECT_ID",
    "FEISHU_TASK_SECTION_ID",
    "GITLINK_TOKEN"
  )
  foreach ($name in $names) {
    $value = Env $name
    if (-not [string]::IsNullOrWhiteSpace($value)) {
      $redacted = [regex]::Replace($redacted, [regex]::Escape($value), (Redact-Value $value))
    }
  }
  $redacted = [regex]::Replace($redacted, 'tenant_access_token"\s*:\s*"[^"]+', 'tenant_access_token":"REDACTED')
  return $redacted
}

function Add-Result {
  param(
    [string]$Name,
    [string]$Status,
    [string]$Details
  )
  $Results.Add([pscustomobject]@{
    Name = $Name
    Status = $Status
    Details = (Redact-Text $Details)
  }) | Out-Null
}

function Write-Log {
  param([string]$Text)
  Add-Content -LiteralPath $TerminalLog -Value (Redact-Text $Text)
}

function Invoke-Cmd {
  param(
    [string]$Name,
    [string[]]$CommandArgs,
    [bool]$Required = $false
  )
  $display = $CommandArgs -join " "
  Write-Host "RUN: $Name"
  Write-Log ""
  Write-Log "## $Name"
  Write-Log "COMMAND: $display"
  $exe = $CommandArgs[0]
  $rest = @()
  if ($CommandArgs.Count -gt 1) {
    $rest = $CommandArgs[1..($CommandArgs.Count - 1)]
  }
  $output = & $exe @rest 2>&1 | Out-String
  $exit = $LASTEXITCODE
  Write-Log $output
  if ($exit -eq 0) {
    Add-Result $Name "pass" "exit=0"
  } else {
    Add-Result $Name "fail" ("exit={0}; {1}" -f $exit, (($output -split "`r?`n" | Select-Object -First 4) -join " "))
    if ($Required) {
      throw "$Name failed with exit $exit"
    }
  }
}

function Invoke-ReportGeneration {
  $owner = Env "GITLINK_OWNER"
  $repo = Env "GITLINK_REPO"
  if ([string]::IsNullOrWhiteSpace($owner) -or [string]::IsNullOrWhiteSpace($repo)) {
    $owner = "Gitlink"
    $repo = "gitlink-cli"
    $Notes.Add("GITLINK_OWNER/GITLINK_REPO were missing. Preview smoke used public Gitlink/gitlink-cli as a fallback.") | Out-Null
  }
  if (-not [string]::IsNullOrWhiteSpace((Env "GITLINK_TEST_PR_IDS"))) {
    $Notes.Add("GITLINK_TEST_PR_IDS was provided, but workflow +repo-report currently does not support explicit PR ID filtering. The smoke test used the real repository report instead.") | Out-Null
  }
  Write-Host "Generating workflow report for $owner/$repo"
  Write-Log ""
  Write-Log "## Generate workflow report"
  $output = & go run . workflow +repo-report --owner $owner --repo $repo --format json 2>&1 | Out-String
  $exit = $LASTEXITCODE
  if ($exit -ne 0) {
    Write-Log $output
    Add-Result "workflow +repo-report" "fail" ("exit={0}; {1}" -f $exit, (($output -split "`r?`n" | Select-Object -First 4) -join " "))
    throw "workflow +repo-report failed"
  }
  $output | Set-Content -LiteralPath $ReportJSON -Encoding utf8
  Write-Log "Report written to .local/report.json"
  Add-Result "workflow +repo-report" "pass" "report=.local/report.json; owner=$owner; repo=$repo"
}

function Add-Skip {
  param([string]$Name, [string]$Reason)
  Write-Host "SKIP: $Name - $Reason"
  Write-Log ""
  Write-Log "## $Name"
  Write-Log "SKIP: $Reason"
  Add-Result $Name "skip" $Reason
}

function Env-Status {
  param([string]$Name)
  if ([string]::IsNullOrWhiteSpace((Env $Name))) { return "missing" }
  return "present"
}

function Escape-Table {
  param([string]$Value)
  if ($null -eq $Value) { return "" }
  return ($Value -replace '\|', '\|' -replace "`r?`n", " ")
}

function Write-SmokeReport {
  $branch = (git branch --show-current | Out-String).Trim()
  $commit = (git rev-parse HEAD | Out-String).Trim()
  $envNames = @(
    "FEISHU_WEBHOOK_URL",
    "FEISHU_WEBHOOK_SECRET",
    "FEISHU_APP_ID",
    "FEISHU_APP_SECRET",
    "FEISHU_WIKI_URL",
    "FEISHU_WIKI_NODE_TOKEN",
    "FEISHU_FOLDER_TOKEN",
    "FEISHU_BASE_APP_TOKEN",
    "FEISHU_REPORT_TABLE_ID",
    "FEISHU_ISSUE_TABLE_ID",
    "FEISHU_PR_TABLE_ID",
    "FEISHU_CONTRIBUTOR_TABLE_ID",
    "FEISHU_TASK_TABLE_ID",
    "FEISHU_TASK_PROJECT_ID",
    "FEISHU_TASK_SECTION_ID",
    "GITLINK_OWNER",
    "GITLINK_REPO",
    "GITLINK_TEST_PR_IDS",
    "GITLINK_TOKEN"
  )
  $lines = @(
    "# Feishu Smoke Report",
    "",
    "Date: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss zzz')",
    "",
    "## Branch",
    "",
    '```text',
    $branch,
    '```',
    "",
    "## Commit",
    "",
    '```text',
    $commit,
    '```',
    "",
    "## Mode",
    "",
    '```text',
    $Mode,
    '```',
    "",
    "## Redacted Environment Presence",
    "",
    "| Variable | Present? |",
    "| --- | --- |"
  )
  foreach ($name in $envNames) {
    $lines += ('| `{0}` | {1} |' -f $name, (Env-Status $name))
  }
  $lines += @(
    "",
    "## Results",
    "",
    "| Command | Result | Details |",
    "| --- | --- | --- |"
  )
  foreach ($result in $Results) {
    $lines += "| $(Escape-Table $result.Name) | $(Escape-Table $result.Status) | $(Escape-Table $result.Details) |"
  }
  $lines += @(
    "",
    "## Notes",
    ""
  )
  if ($Notes.Count -eq 0) {
    $lines += "- None."
  } else {
    foreach ($note in $Notes) {
      $lines += "- $(Redact-Text $note)"
    }
  }
  $lines += @(
    "",
    "## Terminal Log",
    "",
    'Local redacted terminal log: `reports/feishu-real-smoke-terminal.log`',
    "",
    "This log file is ignored and should not be committed after real runs.",
    "",
    "## Image Evidence",
    "",
    "Image files are intentionally not part of this smoke output.",
    "Use the command results, permission matrix, and redacted terminal log as evidence for this round."
  )
  $lines | Set-Content -LiteralPath $SmokeReport -Encoding utf8
  Write-Host "Smoke report written: $SmokeReport"
}

Push-Location $RepoRoot
try {
  New-Item -ItemType Directory -Force -Path $LocalDir | Out-Null
  New-Item -ItemType Directory -Force -Path (Join-Path $RepoRoot "reports") | Out-Null
  if (Test-Path $LocalEnv) {
    . $LocalEnv
    Write-Host "Loaded local env: .local/feishu-gitlink.env.ps1"
  } else {
    $Notes.Add("No .local/feishu-gitlink.env.ps1 file found. Preview smoke can run with public fallback data; real sends are skipped.") | Out-Null
  }
  "# Feishu/GitLink smoke terminal log $(Get-Date -Format o)" | Set-Content -LiteralPath $TerminalLog -Encoding utf8

  $helpCommands = @(
    "+owner-digest",
    "+contributor-digest",
    "+app-check",
    "+doc-check",
    "+bitable-check",
    "+bitable-sync",
    "+task-preview",
    "+task-check",
    "+task-create"
  )
  Invoke-Cmd "feishu help" @("go", "run", ".", "feishu", "--help") $true
  foreach ($command in $helpCommands) {
    Invoke-Cmd "feishu $command help" @("go", "run", ".", "feishu", $command, "--help") $true
  }

  Invoke-ReportGeneration

  $runPreview = $Mode -in @("preview", "stable", "open-platform", "all")
  $runStable = $Mode -in @("stable", "all")
  $runOpenPlatform = $Mode -in @("open-platform", "all")

  if ($runPreview) {
    Invoke-Cmd "notify preview" @("go", "run", ".", "feishu", "+notify", "--from-workflow-json", $ReportJSON, "--format", "table") $true
    Invoke-Cmd "weekly report preview" @("go", "run", ".", "feishu", "+weekly-report", "--from-workflow-json", $ReportJSON, "--format", "markdown") $true
    Invoke-Cmd "owner digest preview" @("go", "run", ".", "feishu", "+owner-digest", "--from-workflow-json", $ReportJSON, "--format", "table") $true
    Invoke-Cmd "contributor digest preview" @("go", "run", ".", "feishu", "+contributor-digest", "--from-workflow-json", $ReportJSON, "--format", "table") $true
    Invoke-Cmd "bitable records preview" @("go", "run", ".", "feishu", "+bitable-records", "--from-workflow-json", $ReportJSON, "--tables", "reports,issues,prs,contributors,tasks", "--format", "table") $true
    Invoke-Cmd "task preview" @("go", "run", ".", "feishu", "+task-preview", "--from-workflow-json", $ReportJSON, "--format", "table") $true
  }

  if ($runStable) {
    if (Has-Env @("FEISHU_WEBHOOK_URL")) {
      Invoke-Cmd "bot-test send" @("go", "run", ".", "feishu", "+bot-test", "--send", "--format", "table") $false
      Invoke-Cmd "notify send" @("go", "run", ".", "feishu", "+notify", "--from-workflow-json", $ReportJSON, "--send", "--format", "table") $false
      Invoke-Cmd "weekly report send" @("go", "run", ".", "feishu", "+weekly-report", "--from-workflow-json", $ReportJSON, "--send", "--format", "table") $false
      Invoke-Cmd "owner digest send" @("go", "run", ".", "feishu", "+owner-digest", "--from-workflow-json", $ReportJSON, "--send", "--format", "table") $false
      Invoke-Cmd "contributor digest send" @("go", "run", ".", "feishu", "+contributor-digest", "--from-workflow-json", $ReportJSON, "--send", "--format", "table") $false
    } else {
      Add-Skip "stable webhook send" "FEISHU_WEBHOOK_URL missing"
    }
  }

  if ($runOpenPlatform) {
    if (Has-Env @("FEISHU_APP_ID", "FEISHU_APP_SECRET")) {
      Invoke-Cmd "app-check remote" @("go", "run", ".", "feishu", "+app-check", "--remote", "--format", "table") $false
      Invoke-Cmd "task-check remote" @("go", "run", ".", "feishu", "+task-check", "--remote", "--format", "table") $false
    } else {
      Add-Skip "app/task diagnostics" "missing FEISHU_APP_ID/FEISHU_APP_SECRET"
    }

    if (Has-Env @("FEISHU_APP_ID", "FEISHU_APP_SECRET") -and (Has-Env @("FEISHU_WIKI_URL") -or Has-Env @("FEISHU_WIKI_NODE_TOKEN") -or Has-Env @("FEISHU_FOLDER_TOKEN") -or Has-Env @("FEISHU_DOCUMENT_ID"))) {
      Invoke-Cmd "doc-check remote" @("go", "run", ".", "feishu", "+doc-check", "--remote", "--format", "table") $false
    } else {
      Add-Skip "doc diagnostics" "missing FEISHU_APP_ID/FEISHU_APP_SECRET or DocX/Wiki target"
    }

    if (Has-Env @("FEISHU_APP_ID", "FEISHU_APP_SECRET", "FEISHU_BASE_APP_TOKEN", "FEISHU_REPORT_TABLE_ID", "FEISHU_ISSUE_TABLE_ID", "FEISHU_PR_TABLE_ID")) {
      Invoke-Cmd "bitable-check remote" @("go", "run", ".", "feishu", "+bitable-check", "--tables", "reports,issues,prs,contributors,tasks", "--remote", "--format", "table") $false
    } else {
      Add-Skip "bitable diagnostics" "missing Feishu app credentials, base app token, or required table IDs"
    }

    if (Has-Env @("FEISHU_APP_ID", "FEISHU_APP_SECRET") -and (Has-Env @("FEISHU_WIKI_URL") -or Has-Env @("FEISHU_WIKI_NODE_TOKEN") -or Has-Env @("FEISHU_FOLDER_TOKEN") -or Has-Env @("FEISHU_DOCUMENT_ID"))) {
      Invoke-Cmd "doc-export send" @("go", "run", ".", "feishu", "+doc-export", "--from-workflow-json", $ReportJSON, "--send", "--format", "table") $false
    } else {
      Add-Skip "doc-export send" "missing FEISHU_APP_ID/FEISHU_APP_SECRET or DocX/Wiki target"
    }

    if (Has-Env @("FEISHU_APP_ID", "FEISHU_APP_SECRET", "FEISHU_BASE_APP_TOKEN", "FEISHU_REPORT_TABLE_ID", "FEISHU_ISSUE_TABLE_ID", "FEISHU_PR_TABLE_ID")) {
      Invoke-Cmd "bitable-sync send" @("go", "run", ".", "feishu", "+bitable-sync", "--from-workflow-json", $ReportJSON, "--tables", "reports,issues,prs,contributors,tasks", "--send", "--format", "table") $false
    } else {
      Add-Skip "bitable-sync send" "missing app/base/table variables"
    }

    if (Has-Env @("FEISHU_APP_ID", "FEISHU_APP_SECRET")) {
      Invoke-Cmd "task-create send" @("go", "run", ".", "feishu", "+task-create", "--from-workflow-json", $ReportJSON, "--send", "--format", "table") $false
    } else {
      Add-Skip "task-create send" "missing FEISHU_APP_ID/FEISHU_APP_SECRET"
    }
  }

  Write-SmokeReport
} finally {
  Pop-Location
}
