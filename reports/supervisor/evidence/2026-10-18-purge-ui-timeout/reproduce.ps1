[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$reviewWorkspace = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..\..\..'))
$reviewScratch = Join-Path $reviewWorkspace '.tmp\supervisor-2026-10-18-purge-ui-timeout\acceptance'
New-Item -ItemType Directory -Force -Path $reviewScratch | Out-Null
$r1Evidence = Join-Path $PSScriptRoot '..\2026-10-18-4-step-hotswap'
$r2Evidence = Join-Path $PSScriptRoot '..\2026-10-18-4-step-hotswap-r2'
$r3Evidence = Join-Path $PSScriptRoot '..\2026-10-18-4-step-hotswap-r3'
python (Join-Path $r1Evidence 'make_sqlite_fixtures.py') $reviewScratch
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
$previousSource = Join-Path $reviewScratch 'previous_reloader_acceptance_test.go'
$previousText = [System.IO.File]::ReadAllText((Join-Path $r1Evidence 'reloader_acceptance_test.go.txt')).Replace('TestSupervisor', 'TestR3Previous')
[System.IO.File]::WriteAllText($previousSource, $previousText)
$reviewOverlay = @{}
$reviewOverlay[(Join-Path $reviewWorkspace 'internal\reloader\zz_timeout_previous_acceptance_test.go')] = $previousSource
$reviewOverlay[(Join-Path $reviewWorkspace 'internal\reloader\zz_timeout_wal_acceptance_test.go')] = Join-Path $r2Evidence 'reloader_extra_acceptance_test.go.txt'
$reviewOverlay[(Join-Path $reviewWorkspace 'internal\manager\zz_timeout_previous_acceptance_test.go')] = Join-Path $r2Evidence 'manager_acceptance_test.go.txt'
$reviewOverlay[(Join-Path $reviewWorkspace 'internal\manager\zz_timeout_nil_acceptance_test.go')] = Join-Path $r3Evidence 'nil_client_acceptance_test.go.txt'
$reviewOverlay[(Join-Path $reviewWorkspace 'internal\connector\zz_timeout_acceptance_test.go')] = Join-Path $PSScriptRoot 'connector_acceptance_test.go.txt'
$reviewOverlay[(Join-Path $reviewWorkspace 'internal\ui\zz_timeout_acceptance_test.go')] = Join-Path $PSScriptRoot 'ui_acceptance_test.go.txt'
$reviewOverlayPath = Join-Path $reviewScratch 'overlay.json'
@{Replace=$reviewOverlay} | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $reviewOverlayPath -Encoding utf8
$env:SUPERVISOR_SQLITE_FIXTURE = Join-Path $reviewScratch 'conversation_summaries.db'
$env:SUPERVISOR_PURGE_UI_IMAGE = Join-Path $PSScriptRoot 'ui-partial-continuation.png'
$reviewLog = Join-Path $PSScriptRoot 'independent-acceptance.log'
Push-Location $reviewWorkspace
try {
    go test -v -count=1 -overlay $reviewOverlayPath -run '^(TestR2|TestR3|TestSupervisorPurge)' ./internal/reloader ./internal/manager ./internal/connector ./internal/ui 2>&1 | Tee-Object -FilePath $reviewLog
    $reviewExit = $LASTEXITCODE
    $reviewLines = Get-Content -LiteralPath $reviewLog
    [ordered]@{
        command="go test -v -count=1 -overlay <generated-overlay.json> -run '^(TestR2|TestR3|TestSupervisorPurge)' ./internal/reloader ./internal/manager ./internal/connector ./internal/ui"
        exit_code=$reviewExit
        passed_events=@($reviewLines | Where-Object { $_ -match '^\s*--- PASS:' }).Count
        failed_events=@($reviewLines | Where-Object { $_ -match '^\s*--- FAIL:' }).Count
        skipped_events=@($reviewLines | Where-Object { $_ -match '^\s*--- SKIP:' }).Count
        reviewed_commit='0289a73242d60039433273b3093f492cb955e76e'
        note='15 prior hot-swap leaf scenarios retained; new UI deadline/partial-summary test and real Win32 listener discovery checks; virtual sleepers, synthetic credentials and SQLite.'
    } | ConvertTo-Json | Tee-Object -FilePath (Join-Path $PSScriptRoot 'independent-acceptance-result.json')
} finally { Pop-Location }
exit $reviewExit
