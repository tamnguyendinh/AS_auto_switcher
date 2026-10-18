[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$r4Workspace = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..\..\..'))
$r4Scratch = Join-Path $r4Workspace '.tmp\supervisor-2026-10-18-4-step-r4\acceptance'
New-Item -ItemType Directory -Force -Path $r4Scratch | Out-Null
$r1Evidence = Join-Path $PSScriptRoot '..\2026-10-18-4-step-hotswap'
$r2Evidence = Join-Path $PSScriptRoot '..\2026-10-18-4-step-hotswap-r2'
$r3Evidence = Join-Path $PSScriptRoot '..\2026-10-18-4-step-hotswap-r3'
python (Join-Path $r1Evidence 'make_sqlite_fixtures.py') $r4Scratch
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

# Preserve all 15 R3 leaf scenarios and assertions; save this run's evidence separately.
$r4PreviousSource = Join-Path $r4Scratch 'previous_reloader_acceptance_test.go'
$r4PreviousText = [System.IO.File]::ReadAllText((Join-Path $r1Evidence 'reloader_acceptance_test.go.txt')).Replace('TestSupervisor', 'TestR3Previous')
[System.IO.File]::WriteAllText($r4PreviousSource, $r4PreviousText)
$r4Overlay = @{}
$r4Overlay[(Join-Path $r4Workspace 'internal\reloader\zz_r4_previous_acceptance_test.go')] = $r4PreviousSource
$r4Overlay[(Join-Path $r4Workspace 'internal\reloader\zz_r4_wal_acceptance_test.go')] = Join-Path $r2Evidence 'reloader_extra_acceptance_test.go.txt'
$r4Overlay[(Join-Path $r4Workspace 'internal\manager\zz_r4_previous_acceptance_test.go')] = Join-Path $r2Evidence 'manager_acceptance_test.go.txt'
$r4Overlay[(Join-Path $r4Workspace 'internal\manager\zz_r4_nil_acceptance_test.go')] = Join-Path $r3Evidence 'nil_client_acceptance_test.go.txt'
$r4OverlayPath = Join-Path $r4Scratch 'overlay.json'
@{Replace=$r4Overlay} | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $r4OverlayPath -Encoding utf8
$env:SUPERVISOR_SQLITE_FIXTURE = Join-Path $r4Scratch 'conversation_summaries.db'
$r4Log = Join-Path $PSScriptRoot 'independent-acceptance.log'
Push-Location $r4Workspace
try {
    go test -v -count=1 -overlay $r4OverlayPath -run '^(TestR2|TestR3)' ./internal/reloader ./internal/manager 2>&1 | Tee-Object -FilePath $r4Log
    $r4Exit = $LASTEXITCODE
    $r4Lines = Get-Content -LiteralPath $r4Log
    [ordered]@{
        command="go test -v -count=1 -overlay <generated-overlay.json> -run '^(TestR2|TestR3)' ./internal/reloader ./internal/manager"
        exit_code=$r4Exit
        passed_events=@($r4Lines | Where-Object { $_ -match '^\s*--- PASS:' }).Count
        failed_events=@($r4Lines | Where-Object { $_ -match '^\s*--- FAIL:' }).Count
        skipped_events=@($r4Lines | Where-Object { $_ -match '^\s*--- SKIP:' }).Count
        reviewed_commit='93308c5e912d68c77843a2d35ed1780346206068'
        note='All 15 leaf scenarios from R3 reused unchanged through virtual Go test files; historical reports and logs preserved.'
    } | ConvertTo-Json | Tee-Object -FilePath (Join-Path $PSScriptRoot 'independent-acceptance-result.json')
} finally { Pop-Location }
exit $r4Exit
