$ErrorActionPreference = 'Stop'
$repoRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..\..\..\..')).Path
Push-Location -LiteralPath $repoRoot
try {
    $timer = [Diagnostics.Stopwatch]::StartNew()
    & go run (Join-Path $PSScriptRoot 'runtime_probe.go') (Join-Path $PSScriptRoot 'quota_matrix.go') 1> (Join-Path $PSScriptRoot 'E06-runtime-probe.json') 2> (Join-Path $PSScriptRoot 'E06-runtime-stderr.log')
    $code = $LASTEXITCODE
    $timer.Stop()
    $command = [pscustomobject]@{command='go run reports/supervisor/evidence/2026-10-18-ui-redesign-r4/runtime_probe.go reports/supervisor/evidence/2026-10-18-ui-redesign-r4/quota_matrix.go';exit=$code;elapsed_ms=$timer.ElapsedMilliseconds}
    $command | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $PSScriptRoot 'E06-runtime-command.json')
    Write-Output ($command | ConvertTo-Json -Compress)
} finally {Pop-Location}
exit $code
