$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent $PSScriptRoot
$Expected = @(
  "docs/images/feishu-bot-card.png",
  "docs/images/feishu-weekly-report.png",
  "docs/images/feishu-owner-digest.png",
  "docs/images/feishu-contributor-digest.png",
  "docs/images/feishu-bitable-preview.png",
  "docs/images/feishu-bitable-sync.png",
  "docs/images/feishu-docx-wiki.png",
  "docs/images/feishu-task-create.png",
  "docs/images/feishu-smoke-terminal.png",
  "docs/images/feishu-env-redacted.png"
)

$missing = @()
foreach ($relative in $Expected) {
  $path = Join-Path $RepoRoot $relative
  if (Test-Path $path) {
    Write-Host "Found screenshot: $relative"
  } else {
    Write-Host "Missing screenshot: $relative"
    Write-Host "Open Feishu or terminal and capture this screenshot manually."
    $missing += $relative
  }
}

if ($missing.Count -gt 0) {
  Write-Host ""
  Write-Host "Missing screenshots: $($missing.Count)"
  exit 1
}

Write-Host "All expected screenshots are present."
exit 0
