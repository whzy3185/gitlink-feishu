[CmdletBinding()]
param([string]$OutputFile = '')

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$cases = @(
    [ordered]@{ id='F01'; name='card create succeeds and local save fails'; pattern='TestCanonicalCardCreateSuccessPersistFailureEntersUnknown'; invariant='unknown and no blind create' },
    [ordered]@{ id='F02'; name='card patch timeout'; pattern='TestFinalCardPatchTimeoutSchedulesSafeRetry'; invariant='idempotent patch schedules a safe retry' },
    [ordered]@{ id='F03'; name='reply send timeout'; pattern='TestReplyTimeoutBecomesUnknown'; invariant='unknown and no blind reply' },
    [ordered]@{ id='F04'; name='base create succeeds and local save fails'; pattern='TestBaseCreateSuccessPersistFailureBecomesUnknown'; invariant='unknown and reconcilable by unique key' },
    [ordered]@{ id='F05'; name='base multiple record conflict'; pattern='TestBaseMultipleMatchesRequireReconciliation'; invariant='manual reconciliation' },
    [ordered]@{ id='F06'; name='doc append uncertain'; pattern='TestDocAppendUnknownDoesNotBlindlyAppendAgain'; invariant='append not repeated' },
    [ordered]@{ id='F07'; name='task create uncertain'; pattern='TestTaskCreateUnknownRequiresReconciliation'; invariant='task create not repeated' },
    [ordered]@{ id='F08'; name='rate limit and retry-after'; pattern='TestBase429UsesRetryAfter'; invariant='retry schedule honors server hint' },
    [ordered]@{ id='F09'; name='upstream 503'; pattern='TestOperation503IsTransient'; invariant='classified transient' },
    [ordered]@{ id='F10'; name='sqlite busy'; pattern='TestMetricsUseCachedValuesWhenSQLiteBusy'; invariant='cached metrics and no corruption' },
    [ordered]@{ id='F11'; name='worker lease expires'; pattern='TestWorkerLeaseRecoversAfterRestart'; invariant='expired lease recovers' },
    [ordered]@{ id='F12'; name='stale worker'; pattern='TestStaleOperationWorkerCannotCompleteNewLease'; invariant='stale completion fenced' },
    [ordered]@{ id='F13'; name='process terminates after remote request'; pattern='TestReplySendSuccessAndLocalPersistFailureDoesNotBlindlyResend'; invariant='remote possible enters reconciliation' },
    [ordered]@{ id='F14'; name='graceful shutdown with non-idempotent operation'; pattern='TestGracefulShutdownKeepsNonIdempotentInFlightUnknown'; invariant='unknown is retained' },
    [ordered]@{ id='F15'; name='wal grows during backup'; pattern='TestBackupIncludesCommittedWALData'; invariant='consistent backup includes committed WAL data' },
    [ordered]@{ id='F16'; name='restore verification fails'; pattern='TestRestoreRejectsChecksumMismatch|TestRestoreRejectsInvalidSQLite'; invariant='target database not replaced' },
    [ordered]@{ id='F17'; name='retention interrupted'; pattern='TestMaintenanceSchedulerStopsOnContextCancel'; invariant='scheduler stops on cancellation' },
    [ordered]@{ id='F18'; name='event processor restarts'; pattern='TestInboxSurvivesSQLiteRestart|TestInboxProcessorRoutesOnce'; invariant='inbox survives and route is unique' },
    [ordered]@{ id='F19'; name='replay request duplicated'; pattern='TestReplayRequestIDIsIdempotent'; invariant='one replay request' },
    [ordered]@{ id='F20'; name='high frequency same-PR events'; pattern='TestSamePRRefreshesAreCoalesced'; invariant='one read job with retained consumers' }
)
$results = @()
Push-Location $root
try {
    foreach ($case in $cases) {
        $timer = [Diagnostics.Stopwatch]::StartNew()
        $output = & go test ./shortcuts/feishu -run "^($($case.pattern))$" -count=1 2>&1
        $exit = $LASTEXITCODE
        $timer.Stop()
        if ($exit -ne 0) { throw "failure injection $($case.id) failed: $($output -join [Environment]::NewLine)" }
        $results += [ordered]@{ id=$case.id; name=$case.name; passed=$true; invariant=$case.invariant; duration_ms=$timer.ElapsedMilliseconds }
    }
} finally { Pop-Location }
$result = [ordered]@{
    schema_version = 'feishu.review-final-failure-injection/v1'; passed = ($results.Count -eq 20)
    validation_mode = 'offline'; cases_expected = 20; cases_passed = @($results | Where-Object passed).Count
    unknown_blind_retries = 0; non_idempotent_duplicates = 0; real_external_writes = 0; cases = $results
}
$json = $result | ConvertTo-Json -Depth 12
if ($OutputFile) { $directory=Split-Path -Parent ([IO.Path]::GetFullPath($OutputFile)); $null=New-Item -ItemType Directory -Force $directory; $json | Set-Content -LiteralPath $OutputFile -Encoding utf8 }
$json
