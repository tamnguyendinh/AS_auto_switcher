[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$reviewWorkspace = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..\..\..'))
$reviewTemp = Join-Path $reviewWorkspace '.tmp\supervisor-2026-10-18-4-step\acceptance'
New-Item -ItemType Directory -Force -Path $reviewTemp | Out-Null

python (Join-Path $PSScriptRoot 'make_sqlite_fixtures.py') $reviewTemp
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$reviewOverlay = @{}
$reviewOverlay[(Join-Path $reviewWorkspace 'internal\reloader\zz_supervisor_acceptance_test.go')] = Join-Path $PSScriptRoot 'reloader_acceptance_test.go.txt'
$reviewOverlay[(Join-Path $reviewWorkspace 'internal\manager\zz_supervisor_acceptance_test.go')] = Join-Path $PSScriptRoot 'manager_acceptance_test.go.txt'
$reviewOverlayPath = Join-Path $reviewTemp 'overlay.json'
@{ Replace = $reviewOverlay } | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $reviewOverlayPath -Encoding utf8

$env:SUPERVISOR_SQLITE_FIXTURE = Join-Path $reviewTemp 'conversation_summaries.db'
$reviewLog = Join-Path $PSScriptRoot 'independent-acceptance.log'
Push-Location $reviewWorkspace
try {
    go test -v -count=1 -overlay $reviewOverlayPath -run '^TestSupervisor' ./internal/reloader ./internal/manager 2>&1 | Tee-Object -FilePath $reviewLog
    $reviewExit = $LASTEXITCODE
    $reviewLines = Get-Content -LiteralPath $reviewLog
    [ordered]@{
        command = "go test -v -count=1 -overlay <generated-overlay.json> -run '^TestSupervisor' ./internal/reloader ./internal/manager"
        exit_code = $reviewExit
        passed = @($reviewLines | Where-Object { $_ -match '^--- PASS:' }).Count
        failed = @($reviewLines | Where-Object { $_ -match '^--- FAIL:' }).Count
        skipped = @($reviewLines | Where-Object { $_ -match '^--- SKIP:' }).Count
        production_source_modified = $false
        note = 'Overlay adds virtual _test.go files alongside their Go packages. Credentials/taskkill/sleeps mocked; SQLite fixture synthetic.'
    } | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $PSScriptRoot 'independent-acceptance-result.json') -Encoding utf8
} finally {
    Pop-Location
}
exit $reviewExit
