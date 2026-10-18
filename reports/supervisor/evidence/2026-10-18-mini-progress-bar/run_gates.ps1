$ErrorActionPreference = 'Stop'
$repoRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..\..\..\..')).Path
$reviewedCommit = '4da49ab2073ee1920d3dce47085342ccc7901acf'
$gates = @(
    @{ ID = 'E02'; Tool = 'go'; Name = 'workspace-build'; Args = @('build', './...') },
    @{ ID = 'E03'; Tool = 'go'; Name = 'native-gui-build'; Args = @('build', '-tags', 'gui', './cmd/switcher') },
    @{ ID = 'E04'; Tool = 'go'; Name = 'go-test-verbose'; Args = @('test', '-v', '-count=1', './...') },
    @{ ID = 'E05'; Tool = 'staticcheck'; Name = 'staticcheck'; Args = @('-tags', 'gui', './...') }
)
$results = @()
$failed = $false
Push-Location -LiteralPath $repoRoot
try {
    foreach ($gate in $gates) {
        $started = [DateTimeOffset]::Now.ToOffset([TimeSpan]::FromHours(7)).ToString('o')
        $timer = [Diagnostics.Stopwatch]::StartNew()
        $commandArgs = $gate.Args
        $lines = @(& $gate.Tool @commandArgs 2>&1)
        $code = $LASTEXITCODE
        $timer.Stop()
        $textLines = @($lines | ForEach-Object { "$_" })
        ($textLines -join "`n") | Set-Content -LiteralPath (Join-Path $PSScriptRoot ($gate.ID + '-' + $gate.Name + '.log'))
        $record = [ordered]@{
            id = $gate.ID
            command = $gate.Tool + ' ' + ($commandArgs -join ' ')
            reviewed_commit = $reviewedCommit
            started_at_bangkok = $started
            exit = $code
            elapsed_ms = $timer.ElapsedMilliseconds
        }
        if ($gate.ID -eq 'E03' -and $code -eq 0) {
            $record.binary_bytes = (Get-Item -LiteralPath (Join-Path $repoRoot 'switcher.exe')).Length
        }
        if ($gate.ID -eq 'E04') {
            $summary = [ordered]@{
                packages = @($textLines | Where-Object { $_ -match '^ok\s+' })
                pass = @($textLines | Where-Object { $_ -match '^\s*--- PASS:' }).Count
                top_level_pass = @($textLines | Where-Object { $_ -match '^--- PASS:' }).Count
                fail = @($textLines | Where-Object { $_ -match '^\s*--- FAIL:' }).Count
                skip = @($textLines | Where-Object { $_ -match '^\s*--- SKIP:' }).Count
                exit = $code
            }
            $summary | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $PSScriptRoot 'E04-go-test-summary.json')
            Write-Output ($summary | ConvertTo-Json -Depth 5)
        }
        if ($gate.ID -eq 'E05') {
            $record.diagnostic_count = @($textLines | Where-Object { $_.Trim() -ne '' }).Count
        }
        $results += [pscustomobject]$record
        Write-Output ($record | ConvertTo-Json -Compress)
        if ($code -ne 0) { $failed = $true }
    }
} finally {
    $results | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $PSScriptRoot 'command-results.json')
    Pop-Location
}
if ($failed) { exit 1 }
