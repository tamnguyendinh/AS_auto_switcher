[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$r3Workspace = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..\..\..'))
$r3Scratch = Join-Path $r3Workspace '.tmp\supervisor-2026-10-18-4-step-r3\acceptance'
New-Item -ItemType Directory -Force -Path $r3Scratch | Out-Null
$r1Evidence = Join-Path $PSScriptRoot '..\2026-10-18-4-step-hotswap'
$r2Evidence = Join-Path $PSScriptRoot '..\2026-10-18-4-step-hotswap-r2'
python (Join-Path $r1Evidence 'make_sqlite_fixtures.py') $r3Scratch
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

# Preserve prior assertions and evidence; only rename R1 test entry points to avoid collisions.
$r3PreviousSource = Join-Path $r3Scratch 'previous_reloader_acceptance_test.go'
$r3PreviousText = [System.IO.File]::ReadAllText((Join-Path $r1Evidence 'reloader_acceptance_test.go.txt')).Replace('TestSupervisor', 'TestR3Previous')
[System.IO.File]::WriteAllText($r3PreviousSource, $r3PreviousText)
$r3Overlay = @{}
$r3Overlay[(Join-Path $r3Workspace 'internal\reloader\zz_r3_previous_acceptance_test.go')] = $r3PreviousSource
$r3Overlay[(Join-Path $r3Workspace 'internal\reloader\zz_r3_wal_acceptance_test.go')] = Join-Path $r2Evidence 'reloader_extra_acceptance_test.go.txt'
$r3Overlay[(Join-Path $r3Workspace 'internal\manager\zz_r3_previous_acceptance_test.go')] = Join-Path $r2Evidence 'manager_acceptance_test.go.txt'
$r3Overlay[(Join-Path $r3Workspace 'internal\manager\zz_r3_nil_acceptance_test.go')] = Join-Path $PSScriptRoot 'nil_client_acceptance_test.go.txt'
$r3OverlayPath = Join-Path $r3Scratch 'overlay.json'
@{Replace=$r3Overlay} | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $r3OverlayPath -Encoding utf8
$env:SUPERVISOR_SQLITE_FIXTURE = Join-Path $r3Scratch 'conversation_summaries.db'
$r3Log = Join-Path $PSScriptRoot 'independent-acceptance.log'
Push-Location $r3Workspace
try {
    go test -v -count=1 -overlay $r3OverlayPath -run '^(TestR2|TestR3)' ./internal/reloader ./internal/manager 2>&1 | Tee-Object -FilePath $r3Log
    $r3Exit = $LASTEXITCODE
    $r3Lines = Get-Content -LiteralPath $r3Log
    [ordered]@{
        command="go test -v -count=1 -overlay <generated-overlay.json> -run '^(TestR2|TestR3)' ./internal/reloader ./internal/manager"
        exit_code=$r3Exit
        passed_events=@($r3Lines | Where-Object { $_ -match '^\s*--- PASS:' }).Count
        failed_events=@($r3Lines | Where-Object { $_ -match '^\s*--- FAIL:' }).Count
        skipped_events=@($r3Lines | Where-Object { $_ -match '^\s*--- SKIP:' }).Count
        reviewed_commit='6c5116386639be2bd0ed833f5bb96d7087cfb8a7'
        note='R1/R2 assertions retained; additional nil-client matrix uses default RPC continuation and the discovery identity selected by production code.'
    } | ConvertTo-Json | Tee-Object -FilePath (Join-Path $PSScriptRoot 'independent-acceptance-result.json')
} finally { Pop-Location }
exit $r3Exit
