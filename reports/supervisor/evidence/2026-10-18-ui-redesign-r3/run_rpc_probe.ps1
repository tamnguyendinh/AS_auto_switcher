$ErrorActionPreference = 'Stop'
$repoRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..\..\..\..')).Path
$overlayPath = Join-Path $repoRoot '.tmp\supervisor-r3-rpc-overlay.json'
$virtualTest = Join-Path $repoRoot 'internal\telemetry\supervisor_r3_probe_test.go'
$probeSource = Join-Path $PSScriptRoot 'rpc_pipeline_probe_test.go.txt'
$replacement = @{}
$replacement[$virtualTest] = $probeSource
@{ Replace = $replacement } | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $overlayPath
Push-Location -LiteralPath $repoRoot
try {
    $timer = [Diagnostics.Stopwatch]::StartNew()
    $output = @(& go test -v -count=1 -overlay $overlayPath ./internal/telemetry -run '^TestSupervisorR3ActualRPCPoolCache$' 2>&1)
    $code = $LASTEXITCODE
    $timer.Stop()
    $output | Tee-Object -FilePath (Join-Path $PSScriptRoot 'E07-rpc-pipeline.log')
    [pscustomobject]@{command='go test -v -count=1 -overlay .tmp/supervisor-r3-rpc-overlay.json ./internal/telemetry -run ^TestSupervisorR3ActualRPCPoolCache$';exit=$code;elapsed_ms=$timer.ElapsedMilliseconds} | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $PSScriptRoot 'E07-rpc-command.json')
} finally {
    Pop-Location
    Remove-Item -LiteralPath $overlayPath -ErrorAction SilentlyContinue
}
exit $code
