$ErrorActionPreference = 'Stop'
$repoRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..\..\..\..')).Path
Push-Location -LiteralPath $repoRoot
try {
    $started = [DateTimeOffset]::Now.ToOffset([TimeSpan]::FromHours(7)).ToString('o')
    $timer = [Diagnostics.Stopwatch]::StartNew()
    & go run (Join-Path $PSScriptRoot 'runtime_probe.go') (Join-Path $PSScriptRoot 'quota_matrix.go') (Join-Path $PSScriptRoot 'explicit_cycles.go') 1> (Join-Path $PSScriptRoot 'E06-runtime-probe.json') 2> (Join-Path $PSScriptRoot 'E06-runtime-stderr.log')
    $code = $LASTEXITCODE
    $timer.Stop()
    $command = [pscustomobject]@{
        command='go run reports/supervisor/evidence/2026-10-18-ui-redesign-r6/runtime_probe.go reports/supervisor/evidence/2026-10-18-ui-redesign-r6/quota_matrix.go reports/supervisor/evidence/2026-10-18-ui-redesign-r6/explicit_cycles.go'
        reviewed_commit='383f56b03ec88f720594ace8f03645695f568b81'
        started_at_bangkok=$started
        exit=$code
        elapsed_ms=$timer.ElapsedMilliseconds
    }
    $command | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $PSScriptRoot 'E06-runtime-command.json')
    Write-Output ($command | ConvertTo-Json -Compress)
} finally {Pop-Location}
exit $code

