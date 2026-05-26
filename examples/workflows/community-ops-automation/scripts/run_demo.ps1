param(
    [string]$Config = "examples/demo_active_config.json",
    [string]$Owner = "",
    [string]$Repo = "",
    [int]$WindowDays = 7,
    [string]$OutputDir = "outputs",
    [int]$PublishIssueId = 0,
    [switch]$SkipReleases
)

$ErrorActionPreference = "Stop"

$cliCandidates = npm.cmd exec --yes --package=@gitlink-ai/cli -- cmd /c where gitlink-cli 2>$null
$cliPath = $cliCandidates | Where-Object { $_ -match 'gitlink-cli\.cmd$' } | Select-Object -First 1
if (-not $cliPath) {
    $cliPath = $cliCandidates | Select-Object -First 1
}
if (-not $cliPath) {
    throw "未能通过 npm exec 找到 gitlink-cli"
}

$cliDir = Split-Path -Parent $cliPath
$env:PATH = "$cliDir;$env:PATH"

$args = @(
    "scripts\gitlink_workflow.py",
    "--config", $Config,
    "--window-days", "$WindowDays",
    "--output-dir", $OutputDir
)

if ($Owner) {
    $args += @("--owner", $Owner)
}
if ($Repo) {
    $args += @("--repo", $Repo)
}
if ($PublishIssueId -gt 0) {
    $args += @("--publish-issue-id", "$PublishIssueId")
}
if ($SkipReleases.IsPresent) {
    $args += "--skip-releases"
}

python @args
