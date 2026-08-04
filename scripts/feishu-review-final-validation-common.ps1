Set-StrictMode -Version Latest

function Get-ReviewFinalEnvironmentValue {
    param([Parameter(Mandatory = $true)][string]$Name)
    return [Environment]::GetEnvironmentVariable($Name)
}

function Get-ReviewFinalRunID {
    $existing = Get-ReviewFinalEnvironmentValue "FEISHU_REVIEW_VALIDATION_RUN_ID"
    if (-not [string]::IsNullOrWhiteSpace($existing)) { return $existing.Trim() }
    $root = Split-Path -Parent $PSScriptRoot
    $sha = (& git -C $root rev-parse --short=8 HEAD).Trim()
    if ($LASTEXITCODE -ne 0) { throw "cannot resolve validation commit SHA" }
    return "review-final-$([DateTime]::UtcNow.ToString('yyyyMMddTHHmmssZ'))-$sha"
}

function Assert-ReviewFinalRealValidation {
    param([switch]$RequireExternalWrites)
    if ((Get-ReviewFinalEnvironmentValue "FEISHU_REVIEW_REAL_VALIDATION") -ne "1") {
        throw "real platform validation is disabled; set FEISHU_REVIEW_REAL_VALIDATION=1 explicitly"
    }
    if ($RequireExternalWrites -and (Get-ReviewFinalEnvironmentValue "FEISHU_REVIEW_CONFIRM_EXTERNAL_WRITES") -cne "YES") {
        throw "real Feishu writes are denied; set FEISHU_REVIEW_CONFIRM_EXTERNAL_WRITES=YES explicitly"
    }
}

function Get-ReviewFinalHash {
    param([string]$Value)
    if ([string]::IsNullOrWhiteSpace($Value)) { return "" }
    $bytes = [Text.Encoding]::UTF8.GetBytes($Value.Trim())
    $algorithm = [Security.Cryptography.SHA256]::Create()
    try { $hash = $algorithm.ComputeHash($bytes) } finally { $algorithm.Dispose() }
    return (([BitConverter]::ToString($hash) -replace '-', '').Substring(0, 16)).ToLowerInvariant()
}

function Resolve-ReviewFinalCredentialReference {
    param([Parameter(Mandatory = $true)][string]$Reference)
    if ($Reference -notmatch '^env:([A-Z][A-Z0-9_]*)$') { throw "credential reference must use env:VARIABLE" }
    $value = Get-ReviewFinalEnvironmentValue $Matches[1]
    if ([string]::IsNullOrWhiteSpace($value)) { throw "credential reference does not resolve to a non-empty environment variable" }
    return $value
}

function New-ReviewFinalCounts {
    return [ordered]@{
        gitlink_get = 0; gitlink_head = 0; gitlink_post = 0; gitlink_put = 0; gitlink_patch = 0; gitlink_delete = 0
        message_create = 0; message_patch = 0; reply_send = 0
        base_search = 0; base_create = 0; base_update = 0
        doc_create = 0; doc_append = 0; task_create = 0; task_patch = 0; tenant_token = 0
    }
}

function Add-ReviewFinalCount {
    param([System.Collections.IDictionary]$Counts, [string]$Operation)
    if (-not $Counts.Contains($Operation)) { $Counts[$Operation] = 0 }
    $Counts[$Operation] = [int]$Counts[$Operation] + 1
}

function Get-ReviewFinalOperation {
    param([ValidateSet('gitlink','feishu')][string]$Target, [string]$Method, [string]$Path)
    $methodName = $Method.ToUpperInvariant()
    if ($Target -eq 'gitlink') { return "gitlink_$($methodName.ToLowerInvariant())" }
    $lower = $Path.ToLowerInvariant()
    if ($lower -match '/tenant_access_token') { return 'tenant_token' }
    if ($lower.EndsWith('/reply')) { return 'reply_send' }
    if ($lower -match '/im/v1/messages' -and $methodName -eq 'PATCH') { return 'message_patch' }
    if ($lower -match '/im/v1/messages') { return 'message_create' }
    if ($lower -match '/records/search') { return 'base_search' }
    if ($lower -match '/records/' -and $methodName -in @('PUT','PATCH')) { return 'base_update' }
    if ($lower -match '/records') { return 'base_create' }
    if ($lower -match '/blocks') { return 'doc_append' }
    if ($lower -match '/documents') { return 'doc_create' }
    if ($lower -match '/tasks/' -and $methodName -eq 'PATCH') { return 'task_patch' }
    if ($lower -match '/tasks') { return 'task_create' }
    return "feishu_$($methodName.ToLowerInvariant())"
}

function Invoke-ReviewFinalRequest {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)][ValidateSet('gitlink','feishu')][string]$Target,
        [Parameter(Mandatory = $true)][ValidateSet('GET','HEAD','POST','PUT','PATCH','DELETE')][string]$Method,
        [Parameter(Mandatory = $true)][uri]$Uri,
        [Parameter(Mandatory = $true)][System.Collections.IDictionary]$Counts,
        [hashtable]$Headers = @{},
        [object]$Body,
        [int[]]$ExpectedStatus = @(200),
        [switch]$AllowHttpError
    )
    $operation = Get-ReviewFinalOperation $Target $Method $Uri.AbsolutePath
    Add-ReviewFinalCount $Counts $operation
    if ($Target -eq 'gitlink' -and $Method -notin @('GET','HEAD')) {
        throw "GitLink validation blocked $Method before network transmission"
    }
    if ($Target -eq 'feishu' -and $Method -notin @('GET','HEAD')) {
        Assert-ReviewFinalRealValidation -RequireExternalWrites
    }
    $parameters = @{
        Uri = $Uri; Method = $Method; Headers = $Headers; SkipHttpErrorCheck = $true
        MaximumRedirection = 0; ConnectionTimeoutSeconds = 15; OperationTimeoutSeconds = 45
    }
    if ($null -ne $Body) {
        $parameters.ContentType = 'application/json; charset=utf-8'
        $parameters.Body = if ($Body -is [string]) { $Body } else { $Body | ConvertTo-Json -Depth 20 -Compress }
    }
    $response = Invoke-WebRequest @parameters
    if (-not $AllowHttpError -and $response.StatusCode -notin $ExpectedStatus) {
        throw "tracked $Target request returned HTTP $($response.StatusCode) for $Method $($Uri.AbsolutePath)"
    }
    return $response
}

function Get-ReviewFinalTenantToken {
    param([System.Collections.IDictionary]$Counts)
    $response = Invoke-ReviewFinalRequest -Target feishu -Method POST `
        -Uri 'https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal' -Counts $Counts `
        -Body @{ app_id = (Get-ReviewFinalEnvironmentValue 'FEISHU_APP_ID'); app_secret = (Get-ReviewFinalEnvironmentValue 'FEISHU_APP_SECRET') }
    $decoded = $response.Content | ConvertFrom-Json
    if ($decoded.code -ne 0 -or [string]::IsNullOrWhiteSpace($decoded.tenant_access_token)) { throw "Feishu tenant token request failed with code $($decoded.code)" }
    return $decoded.tenant_access_token
}

function Get-ReviewFinalBearerHeader {
    param([string]$Token)
    return @{ Authorization = "Bearer $Token" }
}

function Write-ReviewFinalResult {
    param([Parameter(Mandatory = $true)]$Result, [string]$OutputFile = '')
    $json = $Result | ConvertTo-Json -Depth 30
    if (-not [string]::IsNullOrWhiteSpace($OutputFile)) {
        $directory = Split-Path -Parent ([IO.Path]::GetFullPath($OutputFile))
        $null = New-Item -ItemType Directory -Force -Path $directory
        $json | Set-Content -LiteralPath $OutputFile -Encoding utf8
    }
    $json
}

function Assert-ReviewFinalTestPrefix {
    $prefix = Get-ReviewFinalEnvironmentValue 'FEISHU_REVIEW_TEST_RESOURCE_PREFIX'
    if ([string]::IsNullOrWhiteSpace($prefix) -or -not $prefix.StartsWith('review-final-')) {
        throw 'FEISHU_REVIEW_TEST_RESOURCE_PREFIX must start with review-final-'
    }
    return $prefix
}

function Test-ReviewFinalListenerAvailable {
    param([Parameter(Mandatory = $true)][string]$Address)
    if ($Address -notmatch '^([^:]+):(\d+)$') { throw "invalid listener address: $Address" }
    $hostName, $port = $Matches[1], [int]$Matches[2]
    $ip = if ($hostName -in @('localhost','127.0.0.1')) { [Net.IPAddress]::Loopback } elseif ($hostName -eq '::1') { [Net.IPAddress]::IPv6Loopback } else { throw "test listener must use loopback: $Address" }
    $listener = [Net.Sockets.TcpListener]::new($ip, $port)
    try {
        $listener.Start()
        return $true
    } catch {
        $client = [Net.Sockets.TcpClient]::new()
        try {
            $pending = $client.ConnectAsync($ip, $port)
            return $pending.Wait(500) -and $client.Connected
        } catch { return $false } finally { $client.Dispose() }
    } finally { $listener.Stop() }
}
