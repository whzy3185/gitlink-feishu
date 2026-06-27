$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent $PSScriptRoot
$LocalDir = Join-Path $RepoRoot ".local"
$LocalEnv = Join-Path $LocalDir "feishu-gitlink.env.ps1"

function Redact {
  param([string]$Value)
  if ([string]::IsNullOrWhiteSpace($Value)) { return "missing" }
  if ($Value.Length -le 8) { return "***" }
  return "$($Value.Substring(0, 4))...$($Value.Substring($Value.Length - 4))"
}

function Escape-PSString {
  param([string]$Value)
  if ($null -eq $Value) { return "" }
  return $Value.Replace('`', '``').Replace('"', '`"')
}

function SecureString-ToPlainText {
  param([System.Security.SecureString]$Secure)
  if ($null -eq $Secure) { return "" }
  $ptr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($Secure)
  try {
    return [Runtime.InteropServices.Marshal]::PtrToStringBSTR($ptr)
  } finally {
    if ($ptr -ne [IntPtr]::Zero) {
      [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($ptr)
    }
  }
}

function Read-OptionalValue {
  param(
    [string]$Name,
    [string]$CurrentValue,
    [string]$Hint
  )
  $currentLabel = if ([string]::IsNullOrWhiteSpace($CurrentValue)) { "empty" } else { Redact $CurrentValue }
  Write-Host ""
  Write-Host "$Name ($currentLabel)"
  if ($Hint) { Write-Host $Hint }
  $value = Read-Host "Enter value, or press Enter to keep current"
  if ([string]::IsNullOrWhiteSpace($value)) { return $CurrentValue }
  return $value.Trim()
}

function Read-SecretValue {
  param(
    [string]$Name,
    [string]$CurrentValue,
    [string]$Hint
  )
  $currentLabel = if ([string]::IsNullOrWhiteSpace($CurrentValue)) { "empty" } else { "set" }
  Write-Host ""
  Write-Host "$Name ($currentLabel)"
  if ($Hint) { Write-Host $Hint }
  $secure = Read-Host "Enter secret, or press Enter to keep current" -AsSecureString
  $plain = SecureString-ToPlainText $secure
  if ([string]::IsNullOrWhiteSpace($plain)) { return $CurrentValue }
  return $plain.Trim()
}

function Open-Url {
  param(
    [string]$Url,
    [string]$Reason
  )
  Write-Host ""
  Write-Host "Opening: $Reason"
  Write-Host $Url
  Start-Process $Url
}

function Pause-User {
  param([string]$Message)
  [void](Read-Host $Message)
}

function Env {
  param([string]$Name)
  return [Environment]::GetEnvironmentVariable($Name)
}

function Missing {
  param([string[]]$Names)
  foreach ($name in $Names) {
    if ([string]::IsNullOrWhiteSpace((Env $name))) { return $true }
  }
  return $false
}

New-Item -ItemType Directory -Force -Path $LocalDir | Out-Null
if (Test-Path $LocalEnv) {
  . $LocalEnv
  Write-Host "Loaded existing local env: .local/feishu-gitlink.env.ps1"
}

Write-Host "This setup writes values only to .local/feishu-gitlink.env.ps1."
Write-Host "Do not paste secrets into chat or tracked docs."

if (Missing @("FEISHU_WEBHOOK_URL")) {
  Open-Url "https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot" "Feishu custom bot guide"
  Pause-User "Add a custom bot to your Feishu test group and copy the webhook URL. Press Enter when ready."
  $env:FEISHU_WEBHOOK_URL = Read-OptionalValue "FEISHU_WEBHOOK_URL" (Env "FEISHU_WEBHOOK_URL") "Used by +bot-test, +notify, +weekly-report, +owner-digest, and +contributor-digest."
  $env:FEISHU_WEBHOOK_SECRET = Read-SecretValue "FEISHU_WEBHOOK_SECRET" (Env "FEISHU_WEBHOOK_SECRET") "Optional signing secret from the custom bot security settings."
}

if (Missing @("FEISHU_APP_ID", "FEISHU_APP_SECRET")) {
  Open-Url "https://open.feishu.cn/app" "Feishu developer console"
  Pause-User "Create/open a self-built app, enable required scopes, and copy app_id/app_secret. Press Enter when ready."
  $env:FEISHU_APP_ID = Read-OptionalValue "FEISHU_APP_ID" (Env "FEISHU_APP_ID") "Used by experimental +doc-export, +bitable-sync, and +task-create."
  $env:FEISHU_APP_SECRET = Read-SecretValue "FEISHU_APP_SECRET" (Env "FEISHU_APP_SECRET") "Self-built app secret."
}

if (Missing @("FEISHU_WIKI_URL", "FEISHU_WIKI_NODE_TOKEN", "FEISHU_FOLDER_TOKEN", "FEISHU_DOCUMENT_ID")) {
  Open-Url "https://www.feishu.cn/" "Feishu Docs / Wiki"
  Pause-User "Open your target Wiki/Doc page or folder and copy the URL/token. Press Enter when ready."
  $env:FEISHU_WIKI_URL = Read-OptionalValue "FEISHU_WIKI_URL" (Env "FEISHU_WIKI_URL") "Existing Wiki or Doc URL for +doc-export."
  $env:FEISHU_WIKI_NODE_TOKEN = Read-OptionalValue "FEISHU_WIKI_NODE_TOKEN" (Env "FEISHU_WIKI_NODE_TOKEN") "Optional. Usually parsed from FEISHU_WIKI_URL."
  $env:FEISHU_FOLDER_TOKEN = Read-OptionalValue "FEISHU_FOLDER_TOKEN" (Env "FEISHU_FOLDER_TOKEN") "Optional. Used when creating a new DocX in a folder."
  $env:FEISHU_DOCUMENT_ID = Read-OptionalValue "FEISHU_DOCUMENT_ID" (Env "FEISHU_DOCUMENT_ID") "Optional. Existing DocX document ID for append."
}

if (Missing @("FEISHU_BASE_APP_TOKEN", "FEISHU_REPORT_TABLE_ID", "FEISHU_ISSUE_TABLE_ID", "FEISHU_PR_TABLE_ID")) {
  Open-Url "https://www.feishu.cn/" "Feishu Base / Bitable"
  Pause-User "Open/create the Base for reports, issues, prs, contributors, and tasks. Press Enter when table IDs are ready."
  Write-Host "Expected tables: reports, issues, prs, contributors, tasks."
  Write-Host "Each selected table should contain at least unique_key and repository."
  $env:FEISHU_BASE_APP_TOKEN = Read-OptionalValue "FEISHU_BASE_APP_TOKEN" (Env "FEISHU_BASE_APP_TOKEN") "Base app token."
  $env:FEISHU_REPORT_TABLE_ID = Read-OptionalValue "FEISHU_REPORT_TABLE_ID" (Env "FEISHU_REPORT_TABLE_ID") "Reports table ID."
  $env:FEISHU_ISSUE_TABLE_ID = Read-OptionalValue "FEISHU_ISSUE_TABLE_ID" (Env "FEISHU_ISSUE_TABLE_ID") "Issues table ID."
  $env:FEISHU_PR_TABLE_ID = Read-OptionalValue "FEISHU_PR_TABLE_ID" (Env "FEISHU_PR_TABLE_ID") "Pull request table ID."
  $env:FEISHU_CONTRIBUTOR_TABLE_ID = Read-OptionalValue "FEISHU_CONTRIBUTOR_TABLE_ID" (Env "FEISHU_CONTRIBUTOR_TABLE_ID") "Optional contributor table ID."
  $env:FEISHU_TASK_TABLE_ID = Read-OptionalValue "FEISHU_TASK_TABLE_ID" (Env "FEISHU_TASK_TABLE_ID") "Optional task candidate table ID."
}

if (Missing @("FEISHU_TASK_PROJECT_ID", "FEISHU_TASK_SECTION_ID")) {
  Open-Url "https://www.feishu.cn/" "Feishu Tasks"
  Pause-User "Open the Feishu Task project/section if needed. Press Enter when IDs are ready."
  $env:FEISHU_TASK_PROJECT_ID = Read-OptionalValue "FEISHU_TASK_PROJECT_ID" (Env "FEISHU_TASK_PROJECT_ID") "Optional. Some Task API paths may not require it."
  $env:FEISHU_TASK_SECTION_ID = Read-OptionalValue "FEISHU_TASK_SECTION_ID" (Env "FEISHU_TASK_SECTION_ID") "Optional."
}

if (Missing @("GITLINK_OWNER", "GITLINK_REPO")) {
  Open-Url "https://www.gitlink.org.cn/" "GitLink"
  Pause-User "Open the target GitLink repository and the previous 3 PRs. Press Enter when ready."
  $env:GITLINK_OWNER = Read-OptionalValue "GITLINK_OWNER" (Env "GITLINK_OWNER") "Repository owner, for example Gitlink."
  $env:GITLINK_REPO = Read-OptionalValue "GITLINK_REPO" (Env "GITLINK_REPO") "Repository name, for example gitlink-cli."
  $env:GITLINK_TEST_PR_IDS = Read-OptionalValue "GITLINK_TEST_PR_IDS" (Env "GITLINK_TEST_PR_IDS") "Comma-separated PR IDs for smoke report references."
  $env:GITLINK_TOKEN = Read-SecretValue "GITLINK_TOKEN" (Env "GITLINK_TOKEN") "Optional if gitlink-cli is already logged in."
}

if (-not [string]::IsNullOrWhiteSpace($env:GITLINK_OWNER) -and -not [string]::IsNullOrWhiteSpace($env:GITLINK_REPO)) {
  Open-Url "https://www.gitlink.org.cn/$env:GITLINK_OWNER/$env:GITLINK_REPO" "Target GitLink repository"
  if (-not [string]::IsNullOrWhiteSpace($env:GITLINK_TEST_PR_IDS)) {
    Write-Host "GITLINK_TEST_PR_IDS set: $env:GITLINK_TEST_PR_IDS"
    Write-Host "Verify the PR page URL pattern manually if needed."
  }
}

$vars = @(
  "FEISHU_WEBHOOK_URL",
  "FEISHU_WEBHOOK_SECRET",
  "FEISHU_APP_ID",
  "FEISHU_APP_SECRET",
  "FEISHU_WIKI_URL",
  "FEISHU_WIKI_NODE_TOKEN",
  "FEISHU_FOLDER_TOKEN",
  "FEISHU_DOCUMENT_ID",
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
  "# Local Feishu/GitLink smoke-test environment.",
  "# Generated by scripts/feishu-gitlink-setup.ps1.",
  "# Do not commit this file."
)
foreach ($name in $vars) {
  $value = [Environment]::GetEnvironmentVariable($name)
  $lines += ('$' + ('env:{0}="{1}"' -f $name, (Escape-PSString $value)))
}
$lines | Set-Content -LiteralPath $LocalEnv -Encoding utf8

Write-Host ""
Write-Host "Saved local env: .local/feishu-gitlink.env.ps1"
Write-Host "Redacted summary:"
foreach ($name in $vars) {
  $value = [Environment]::GetEnvironmentVariable($name)
  Write-Host ("{0}: {1}" -f $name, (Redact $value))
}
