# scripts/build_windows.ps1
# Antigravity Multi-Account Switcher - Windows Build Automation
# Enforces pure Go standalone compilation (CLI) and CGo Fyne GUI compilation (GUI)

[CmdletBinding()]
param(
    [ValidateSet("cli", "gui", "all")]
    [string]$Mode = "cli",
    [switch]$SkipTests,
    [switch]$Clean,
    [string]$OutDir = "bin"
)

[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8
$ErrorActionPreference = "Stop"

Write-Host "================================================================================" -ForegroundColor Cyan
Write-Host "⚡ Antigravity Multi-Account Switcher - Windows Build System" -ForegroundColor Cyan
Write-Host "   Mode: $Mode | OutDir: $OutDir | SkipTests: $SkipTests | Clean: $Clean" -ForegroundColor DarkCyan
Write-Host "================================================================================" -ForegroundColor Cyan

# 1. Clean Stage
if ($Clean) {
    Write-Host "[1/4] Cleaning output artifacts..." -ForegroundColor Yellow
    if (Test-Path $OutDir) {
        Remove-Item -Recurse -Force $OutDir
        Write-Host "  Removed directory: $OutDir" -ForegroundColor DarkGray
    }
    Get-ChildItem -Path . -Filter "antigravity-switch*.exe" -File | Remove-Item -Force
    Write-Host "  Clean complete." -ForegroundColor Green
}

# Ensure output directory exists
if (-not (Test-Path $OutDir)) {
    New-Item -ItemType Directory -Path $OutDir -Force | Out-Null
}

# Set universal target OS and Arch
$env:GOOS = "windows"
$env:GOARCH = "amd64"

# 2. Pre-Flight Validation & Tests
if (-not $SkipTests) {
    Write-Host "[2/4] Running pre-flight tests & vet..." -ForegroundColor Yellow

    if ($Mode -eq "cli" -or $Mode -eq "all") {
        Write-Host "  -> Verifying pure Go packages (CGO_ENABLED=0)..." -ForegroundColor DarkYellow
        $env:CGO_ENABLED = "0"
        
        # go vet
        go vet ./cmd/switcher/...
        if ($LASTEXITCODE -ne 0) {
            Write-Error "go vet failed for cmd/switcher"
        }

        # go test
        go test ./internal/model/... ./internal/wincred/... ./internal/store/... ./internal/scanner/... ./internal/connector/... ./internal/reloader/... ./internal/manager/... ./cmd/switcher/...
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Pure Go test suite failed"
        }
        Write-Host "  [PASS] Pure Go verification succeeded." -ForegroundColor Green
    }

    if ($Mode -eq "gui" -or $Mode -eq "all") {
        Write-Host "  -> Verifying GUI packages with CGo & Fyne (CGO_ENABLED=1)..." -ForegroundColor DarkYellow
        $env:CGO_ENABLED = "1"
        
        # go vet
        go vet -tags gui ./cmd/switcher/... ./internal/ui/...
        if ($LASTEXITCODE -ne 0) {
            Write-Error "go vet failed for GUI packages"
        }

        # go test
        go test -tags gui ./cmd/switcher/... ./internal/ui/...
        if ($LASTEXITCODE -ne 0) {
            Write-Error "GUI test suite failed"
        }
        Write-Host "  [PASS] GUI verification succeeded." -ForegroundColor Green
    }
} else {
    Write-Host "[2/4] Skipping tests (-SkipTests specified)." -ForegroundColor DarkGray
}

# 3. Compilation Pipeline
Write-Host "[3/4] Compiling target binaries..." -ForegroundColor Yellow

function Build-CLI {
    param([string]$TargetDir)
    Write-Host "  -> Building Standalone CLI Binary (Pure Go, CGO_ENABLED=0)..." -ForegroundColor Cyan
    $env:CGO_ENABLED = "0"
    $outFile = Join-Path $TargetDir "antigravity-switch.exe"
    if (Test-Path $outFile) { Remove-Item -Force $outFile }
    
    $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
    go build -ldflags="-s -w" -o $outFile .\cmd\switcher
    $stopwatch.Stop()
    
    if ($LASTEXITCODE -ne 0 -or -not (Test-Path $outFile)) {
        Write-Error "Failed building CLI binary: $outFile"
    }

    $item = Get-Item $outFile
    $sizeMB = [math]::Round($item.Length / 1MB, 2)
    Write-Host "  [SUCCESS] Created: $outFile" -ForegroundColor Green
    Write-Host "  Compile Duration: $($stopwatch.ElapsedMilliseconds) ms | Size: $sizeMB MB ($($item.Length) bytes)" -ForegroundColor Green

    if ($sizeMB -gt 15.0) {
        Write-Error "Binary size check failed! $sizeMB MB exceeds 15MB threshold."
    }
}

function Build-GUI {
    param([string]$TargetDir)
    Write-Host "  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)..." -ForegroundColor Cyan
    $env:CGO_ENABLED = "1"
    $outFile = Join-Path $TargetDir "antigravity-switch-gui.exe"
    if (Test-Path $outFile) { Remove-Item -Force $outFile }

    $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
    go build -tags gui -ldflags="-H windowsgui -s -w" -o $outFile .\cmd\switcher
    $stopwatch.Stop()

    if ($LASTEXITCODE -ne 0 -or -not (Test-Path $outFile)) {
        Write-Error "Failed building GUI binary: $outFile"
    }

    $item = Get-Item $outFile
    $sizeMB = [math]::Round($item.Length / 1MB, 2)
    Write-Host "  [SUCCESS] Created: $outFile" -ForegroundColor Green
    Write-Host "  Compile Duration: $($stopwatch.ElapsedMilliseconds) ms | Size: $sizeMB MB ($($item.Length) bytes)" -ForegroundColor Green
}

switch ($Mode) {
    "cli" { Build-CLI -TargetDir $OutDir }
    "gui" { Build-GUI -TargetDir $OutDir }
    "all" {
        Build-CLI -TargetDir $OutDir
        Build-GUI -TargetDir $OutDir
    }
}

# 4. Quick Verification Gate
Write-Host "[4/4] Executing quick verification..." -ForegroundColor Yellow
$verifyExe = Join-Path $OutDir "antigravity-switch.exe"
if (Test-Path $verifyExe) {
    Write-Host "  -> Running '$verifyExe --status'..." -ForegroundColor Cyan
    & $verifyExe --status
} elseif (Test-Path (Join-Path $OutDir "antigravity-switch-gui.exe")) {
    Write-Host "  [INFO] GUI binary built at $(Join-Path $OutDir "antigravity-switch-gui.exe")." -ForegroundColor Green
}

Write-Host "================================================================================" -ForegroundColor Cyan
Write-Host "⚡ BUILD SUCCESSFUL" -ForegroundColor Green
Write-Host "================================================================================" -ForegroundColor Cyan
