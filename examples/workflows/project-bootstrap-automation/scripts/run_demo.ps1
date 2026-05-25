param(
    [string]$Config = "examples/sample_project.json",
    [string]$OutputDir = "outputs",
    [switch]$Apply,
    [switch]$CreateRepo,
    [int]$PublishIssueNumber = 0
)

$ErrorActionPreference = "Stop"

$args = @(
    "run",
    "scripts\bootstrap_project.go",
    "--config", $Config,
    "--output-dir", $OutputDir
)

if ($Apply.IsPresent) {
    $cliCandidates = npm.cmd exec --yes --package=@gitlink-ai/cli -- cmd /c where gitlink-cli 2>$null
    $cliPath = $cliCandidates | Where-Object { $_ -match 'gitlink-cli\.cmd$' } | Select-Object -First 1
    if (-not $cliPath) {
        $cliPath = $cliCandidates | Select-Object -First 1
    }
    if (-not $cliPath) {
        throw "未能通过 npm exec 找到 gitlink-cli"
    }
    $args += @("--cli-bin", $cliPath, "--apply")
}

if ($CreateRepo.IsPresent) {
    $args += "--create-repo"
}

if ($PublishIssueNumber -gt 0) {
    $args += @("--publish-issue-number", "$PublishIssueNumber")
}

go @args
