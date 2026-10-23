$ErrorActionPreference = 'Stop'
$repoRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..\..\..\..')).Path
Push-Location -LiteralPath $repoRoot
try {
    $started = [DateTimeOffset]::Now.ToOffset([TimeSpan]::FromHours(7)).ToString('o')
    $timer = [Diagnostics.Stopwatch]::StartNew()
    & go run (Join-Path $PSScriptRoot 'runtime_probe.go') 1> (Join-Path $PSScriptRoot 'E06-runtime-probe.json') 2> (Join-Path $PSScriptRoot 'E06-runtime-stderr.log')
    $code = $LASTEXITCODE
    $timer.Stop()
    [ordered]@{
        command = 'go run reports/supervisor/evidence/2026-10-18-mini-progress-bar/runtime_probe.go'
        reviewed_commit = '4da49ab2073ee1920d3dce47085342ccc7901acf'
        started_at_bangkok = $started
        exit = $code
        elapsed_ms = $timer.ElapsedMilliseconds
    } | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $PSScriptRoot 'E06-runtime-command.json')
    Write-Output "Runtime probe exit: $code"
} finally { Pop-Location }
exit $code
