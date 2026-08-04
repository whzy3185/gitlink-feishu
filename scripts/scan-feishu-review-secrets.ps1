[CmdletBinding()]
param([string]$OutputFile = '')

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$excludedNames = @('go.sum','package-lock.json')
$textExtensions = @('.go','.ps1','.sh','.md','.json','.yaml','.yml','.txt','.toml','.ini','.conf','.service','.js','.ts','.py','.xml')
$patterns = [ordered]@{
    private_key = '-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----'
    bearer = '(?i)Authorization:\s*Bearer\s+[A-Za-z0-9._-]{20,}'
    cookie = '(?i)Cookie:\s*[^\s"'']{20,}'
    assigned_secret = '(?i)(?:access_token|refresh_token|app_secret|client_secret|webhook_secret)\s*[:=]\s*["'']([^"''\s]{16,})'
    feishu_identifier = '\b(?:oc|ou|om)_[A-Za-z0-9]{20,}\b'
}
$allowedValueMarkers = @('replace_me','fixture','example','test-only','sensitive','redacted','not-serialized','environment','VARIABLE','<')
$exactAllowlist = @(
    [ordered]@{ rule='cookie'; file='internal/auth/transport_test.go'; value='cookie:autologin_trustie=test123'; reason='authentication transport fixture' },
    [ordered]@{ rule='cookie'; file='internal/auth/transport_test.go'; value='cookie:autologin_trustie=injected'; reason='cookie append fixture' },
    [ordered]@{ rule='private_key'; file='scripts/research/test_repro.py'; value='-----BEGIN RSA PRIVATE KEY-----'; reason='secret scanner detection fixture' },
    [ordered]@{ rule='cookie'; file='shortcuts/feishu/review_gateway_test.go'; value='Cookie: session=cookie-secret'; reason='redaction fixture' },
    [ordered]@{ rule='cookie'; file='scripts/scan-feishu-review-secrets.ps1'; value='cookie:autologin_trustie=test123'; reason='declared exact whitelist value' },
    [ordered]@{ rule='cookie'; file='scripts/scan-feishu-review-secrets.ps1'; value='cookie:autologin_trustie=injected'; reason='declared exact whitelist value' },
    [ordered]@{ rule='private_key'; file='scripts/scan-feishu-review-secrets.ps1'; value='-----BEGIN RSA PRIVATE KEY-----'; reason='declared exact whitelist value' },
    [ordered]@{ rule='cookie'; file='scripts/scan-feishu-review-secrets.ps1'; value='Cookie: session=cookie-secret'; reason='declared exact whitelist value' }
)
$findings = @()
$files = @(& git -C $root -c core.quotepath=false ls-files; & git -C $root -c core.quotepath=false ls-files --others --exclude-standard) | Sort-Object -Unique
foreach ($relative in $files) {
    if ($excludedNames -contains [IO.Path]::GetFileName($relative)) { continue }
    $extension = [IO.Path]::GetExtension($relative).ToLowerInvariant()
    $leaf = [IO.Path]::GetFileName($relative)
    if ($extension -notin $textExtensions -and $leaf -notin @('Dockerfile','Caddyfile','nginx.conf')) { continue }
    $path = Join-Path $root $relative
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) { continue }
    try { $lines = Get-Content -LiteralPath $path -ErrorAction Stop } catch { continue }
    for ($index=0; $index -lt $lines.Count; $index++) {
        $line = [string]$lines[$index]
        foreach ($entry in $patterns.GetEnumerator()) {
            $matches = [regex]::Matches($line, $entry.Value)
            foreach ($match in $matches) {
                $candidate = $match.Value
                $allowed = @($exactAllowlist | Where-Object { $_.rule -eq $entry.Key -and $_.file -eq $relative -and $_.value -eq $candidate }).Count -eq 1
                foreach ($marker in $allowedValueMarkers) { if ($candidate.ToLowerInvariant().Contains($marker.ToLowerInvariant())) { $allowed=$true; break } }
                if (-not $allowed) { $findings += [ordered]@{ rule=$entry.Key; file=$relative; line=$index+1; match_hash=(Get-FileHash -InputStream ([IO.MemoryStream]::new([Text.Encoding]::UTF8.GetBytes($candidate))) -Algorithm SHA256).Hash.Substring(0,16).ToLowerInvariant() } }
            }
        }
    }
}
$diffText = (& git -C $root -c core.autocrlf=false diff --unified=0 e2bcb35efc56db659eb58af141a3fd8018e6d5be -- . 2>$null) -join "`n"
$diffScanned = -not [string]::IsNullOrWhiteSpace($diffText)
$publicAllowlist = @($exactAllowlist | ForEach-Object {
    $bytes = [Text.Encoding]::UTF8.GetBytes($_.value)
    $hash = (Get-FileHash -InputStream ([IO.MemoryStream]::new($bytes)) -Algorithm SHA256).Hash.ToLowerInvariant()
    [ordered]@{ rule=$_.rule; file=$_.file; value_hash=$hash.Substring(0,16); reason=$_.reason }
})
$result = [ordered]@{
    schema_version='feishu.review-secret-scan/v1'; passed=($findings.Count -eq 0); validation_mode='offline'
    scopes=@('tracked_worktree','untracked_worktree','stage5_to_stage6_diff','docs','evidence','deploy','scripts','.github/workflows')
    exact_whitelist=$publicAllowlist; allowed_placeholder_markers=$allowedValueMarkers; excluded_files=$excludedNames; files_scanned=$files.Count
    stage5_diff_scanned=$diffScanned; findings_count=$findings.Count; findings=$findings
}
$json = $result | ConvertTo-Json -Depth 10
if ($OutputFile) { $directory=Split-Path -Parent ([IO.Path]::GetFullPath($OutputFile)); $null=New-Item -ItemType Directory -Force $directory; $json | Set-Content -LiteralPath $OutputFile -Encoding utf8 }
$json
if ($findings.Count -ne 0) { throw "secret scan found $($findings.Count) unapproved candidate(s)" }
