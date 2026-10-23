$ErrorActionPreference = 'Stop'
$repoRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..\..\..\..')).Path
$tempBinary = Join-Path $repoRoot '.tmp\supervisor-ui-r6-gui.exe'
$gates = @(
    @{ ID='E02'; Name='workspace-build'; Args=@('build', './...') },
    @{ ID='E03'; Name='native-gui-build'; Args=@('build', '-tags', 'gui', '-ldflags', '-H=windowsgui', '-o', $tempBinary, './cmd/switcher') },
    @{ ID='E04'; Name='go-test-verbose'; Args=@('test', '-v', '-count=1', './...') }
)
$results = @()
New-Item -ItemType Directory -Path (Join-Path $repoRoot '.tmp') -Force | Out-Null
Push-Location -LiteralPath $repoRoot
try {
    foreach ($gate in $gates) {
        $timer = [Diagnostics.Stopwatch]::StartNew()
        $started = [DateTimeOffset]::Now.ToOffset([TimeSpan]::FromHours(7)).ToString('o')
        $commandArgs = $gate.Args
        $lines = @(& go @commandArgs 2>&1)
        $code = $LASTEXITCODE
        $timer.Stop()
        ($lines -join "`n") | Set-Content -LiteralPath (Join-Path $PSScriptRoot ($gate.ID+'-'+$gate.Name+'.log'))
        $results += [pscustomobject]@{id=$gate.ID; command=('go '+($commandArgs -join ' ')); started_at_bangkok=$started; reviewed_commit='383f56b03ec88f720594ace8f03645695f568b81'; exit=$code; elapsed_ms=$timer.ElapsedMilliseconds}
        Write-Output ($results[-1] | ConvertTo-Json -Compress)
        if ($gate.ID -eq 'E04') {
            $textLines = @($lines | ForEach-Object {"$_"})
            $summary = [pscustomobject]@{
                packages=@($textLines | Where-Object {$_ -match '^ok\s+'})
                pass=@($textLines | Where-Object {$_ -match '^\s*--- PASS:'}).Count
                top_level_pass=@($textLines | Where-Object {$_ -match '^--- PASS:'}).Count
                fail=@($textLines | Where-Object {$_ -match '^\s*--- FAIL:'}).Count
                skip=@($textLines | Where-Object {$_ -match '^\s*--- SKIP:'}).Count
                exit=$code
            }
            $summary | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath (Join-Path $PSScriptRoot 'E04-go-test-summary.json')
            Write-Output ($summary | ConvertTo-Json -Depth 4)
        }
    }
    $timer = [Diagnostics.Stopwatch]::StartNew()
    $started = [DateTimeOffset]::Now.ToOffset([TimeSpan]::FromHours(7)).ToString('o')
    $diagnostics = @(& staticcheck -tags gui ./... 2>&1)
    $code = $LASTEXITCODE
    $timer.Stop()
    ($diagnostics -join "`n") | Set-Content -LiteralPath (Join-Path $PSScriptRoot 'E05-staticcheck.log')
    $results += [pscustomobject]@{id='E05'; command='staticcheck -tags gui ./...'; started_at_bangkok=$started; reviewed_commit='383f56b03ec88f720594ace8f03645695f568b81'; exit=$code; diagnostic_count=@($diagnostics | Where-Object {"$_".Trim() -ne ''}).Count; elapsed_ms=$timer.ElapsedMilliseconds}
    Write-Output ($results[-1] | ConvertTo-Json -Compress)
} finally {
    $results | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $PSScriptRoot 'command-results.json')
    if (Test-Path -LiteralPath $tempBinary) {Remove-Item -LiteralPath $tempBinary}
    Pop-Location
}

