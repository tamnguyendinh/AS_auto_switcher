[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$r2Workspace = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..\..\..\..'))
$r2Scratch = Join-Path $r2Workspace '.tmp\supervisor-2026-10-18-4-step-r2\acceptance'
New-Item -ItemType Directory -Force -Path $r2Scratch | Out-Null
$r1Evidence = Join-Path $PSScriptRoot '..\2026-10-18-4-step-hotswap'
python (Join-Path $r1Evidence 'make_sqlite_fixtures.py') $r2Scratch
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

# Reuse the original five reloader checks without weakening their assertions.
# Change only test names to avoid a name collision with the new repository test.
$r2PreviousTests = [System.IO.File]::ReadAllText((Join-Path $r1Evidence 'reloader_acceptance_test.go.txt')).Replace('TestSupervisor', 'TestR2Previous')
$r2PreviousSource = Join-Path $r2Scratch 'previous_reloader_acceptance_test.go'
[System.IO.File]::WriteAllText($r2PreviousSource, $r2PreviousTests)
$r2Overlay = @{}
$r2Overlay[(Join-Path $r2Workspace 'internal\reloader\zz_r2_previous_acceptance_test.go')] = $r2PreviousSource
$r2Overlay[(Join-Path $r2Workspace 'internal\reloader\zz_r2_extra_acceptance_test.go')] = Join-Path $PSScriptRoot 'reloader_extra_acceptance_test.go.txt'
$r2Overlay[(Join-Path $r2Workspace 'internal\manager\zz_r2_acceptance_test.go')] = Join-Path $PSScriptRoot 'manager_acceptance_test.go.txt'
$r2OverlayPath = Join-Path $r2Scratch 'overlay.json'
@{Replace=$r2Overlay} | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $r2OverlayPath -Encoding utf8
$env:SUPERVISOR_SQLITE_FIXTURE = Join-Path $r2Scratch 'conversation_summaries.db'
$r2Log = Join-Path $PSScriptRoot 'independent-acceptance.log'
Push-Location $r2Workspace
try {
    go test -v -count=1 -overlay $r2OverlayPath -run '^TestR2' ./internal/reloader ./internal/manager 2>&1 | Tee-Object -FilePath $r2Log
    $r2Exit = $LASTEXITCODE
    $r2Lines = Get-Content -LiteralPath $r2Log
    [ordered]@{
        command="go test -v -count=1 -overlay <generated-overlay.json> -run '^TestR2' ./internal/reloader ./internal/manager"
        exit_code=$r2Exit
        passed_events=@($r2Lines | Where-Object { $_ -match '^\s*--- PASS:' }).Count
        failed_events=@($r2Lines | Where-Object { $_ -match '^\s*--- FAIL:' }).Count
        skipped_events=@($r2Lines | Where-Object { $_ -match '^\s*--- SKIP:' }).Count
        reviewed_commit='1c9e560e1bc004da7fa6eabbc33870d63f12c0b2'
        note='Original reloader assertions reused; new RAM outcome matrix and WAL fixture; virtual Go tests only; no production edits.'
    } | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $PSScriptRoot 'independent-acceptance-result.json') -Encoding utf8
} finally { Pop-Location }
exit $r2Exit
