$ErrorActionPreference = 'Stop'
$repoRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..\..\..\..')).Path
$overlayPath = Join-Path $repoRoot '.tmp\supervisor-r2-rpc-overlay.json'
$virtualTest = Join-Path $repoRoot 'internal\telemetry\supervisor_r2_probe_test.go'
$probeSource = Join-Path $PSScriptRoot 'rpc_pipeline_probe_test.go.txt'
$replacement = @{}
$replacement[$virtualTest] = $probeSource
@{ Replace = $replacement } | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $overlayPath
Push-Location -LiteralPath $repoRoot
try {
    $probeOutput = @(& go test -v -count=1 -overlay $overlayPath ./internal/telemetry -run '^TestSupervisorR2ActualRPCPoolCache$' 2>&1)
    $probeExit = $LASTEXITCODE
    $probeOutput | Tee-Object -FilePath (Join-Path $PSScriptRoot 'E07-rpc-pipeline.log')
    "Exit: $probeExit" | Add-Content -LiteralPath (Join-Path $PSScriptRoot 'E07-rpc-pipeline.log')
} finally {
    Pop-Location
    Remove-Item -LiteralPath $overlayPath -ErrorAction SilentlyContinue
}
exit $probeExit
