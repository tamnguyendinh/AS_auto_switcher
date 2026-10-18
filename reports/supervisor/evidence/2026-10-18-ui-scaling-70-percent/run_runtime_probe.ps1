$ErrorActionPreference = 'Stop'
$repoRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..\..\..\..')).Path
Push-Location -LiteralPath $repoRoot
try {
    $started = [DateTimeOffset]::Now.ToOffset([TimeSpan]::FromHours(7)).ToString('o')
    $timer = [Diagnostics.Stopwatch]::StartNew()
    & go run (Join-Path $PSScriptRoot 'runtime_probe.go') 1> (Join-Path $PSScriptRoot 'E06-runtime-probe.json') 2> (Join-Path $PSScriptRoot 'E06-runtime-stderr.log')
    $code = $LASTEXITCODE
    $timer.Stop()
    $record = [pscustomobject]@{
        command = 'go run reports/supervisor/evidence/2026-10-18-ui-scaling-70-percent/runtime_probe.go'
        reviewed_commit = '92b324542302f2fe311b886650d89ffbfef78d1b'
        started_at_bangkok = $started
        exit = $code
        elapsed_ms = $timer.ElapsedMilliseconds
    }
    $record | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $PSScriptRoot 'E06-runtime-command.json')
    Write-Output ($record | ConvertTo-Json -Compress)
} finally { Pop-Location }
exit $code
