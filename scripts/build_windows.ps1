# scripts/build_windows.ps1
# Antigravity Multi-Account Switcher - Windows Build Automation
# Enforces single GUI artifact compilation (CGo Fyne GUI, CGO_ENABLED=1)

[CmdletBinding()]
param(
    [switch]$SkipTests,
    [switch]$Clean,
    [switch]$SyncOnly,
    [string]$OutDir = "bin"
)

[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8
$ErrorActionPreference = "Stop"

$IconLightning = [char]0x26A1

Write-Host "================================================================================" -ForegroundColor Cyan
Write-Host "$IconLightning Antigravity Multi-Account Switcher - Windows Build System" -ForegroundColor Cyan
Write-Host "   OutDir: $OutDir | SkipTests: $SkipTests | Clean: $Clean | SyncOnly: $SyncOnly" -ForegroundColor DarkCyan
Write-Host "================================================================================" -ForegroundColor Cyan

# 1. Clean Stage
if ($Clean) {
    Write-Host "[1/4] Cleaning output artifacts..." -ForegroundColor Yellow
    Get-Process -Name "antigravity-switch*" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
    Start-Sleep -Milliseconds 300
    if (Test-Path $OutDir) {
        Remove-Item -Recurse -Force $OutDir
        Write-Host "  Removed directory: $OutDir" -ForegroundColor DarkGray
    }
    Get-ChildItem -Path . -Filter "antigravity-switch*.exe" -File -ErrorAction SilentlyContinue | Remove-Item -Force
    Get-ChildItem -Path "cmd\switcher" -Filter "*.syso" -File -ErrorAction SilentlyContinue | Remove-Item -Force
    Write-Host "  Clean complete." -ForegroundColor Green
}

# 0. Version Synchronization Stage (Pre-Flight SSOT Sync)
Write-Host "[0/4] Checking and synchronizing application version..." -ForegroundColor Yellow

function Get-AppVersionFromSource {
    param([string]$SourcePath = "internal\core\version\version.go")
    if (-not (Test-Path $SourcePath)) {
        Write-Error "Version source of truth not found: $SourcePath"
    }
    $content = Get-Content $SourcePath -Raw -Encoding UTF8
    if ($content -match 'Version\s*=\s*"([^"]+)"') {
        $v = $Matches[1].Trim()
        Write-Host "  -> Single Source of Truth detected: Version $v (from $SourcePath)" -ForegroundColor DarkGreen
        return $v
    }
    Write-Error "Failed to parse Version from $SourcePath"
}

function Sync-ReadmeVersion {
    param(
        [string[]]$ReadmePaths = @("README.md", "README_VN.md", "README_EN.md"),
        [string]$TargetVersion
    )
    foreach ($ReadmePath in $ReadmePaths) {
        if (-not (Test-Path $ReadmePath)) {
            Write-Warning "README file not found: $ReadmePath"
            continue
        }

        $content = Get-Content $ReadmePath -Raw -Encoding UTF8
        $pattern = '(?m)^(#.*?\bAntigravity Multi-Account Switcher\s*-\s*\*\*Version\s*)([0-9]+(?:\.[0-9]+){2,3})(\*\*.*)$'

        if ($content -match $pattern) {
            $currentVersion = $Matches[2]
            if ($currentVersion -ne $TargetVersion) {
                Write-Host "  -> Synchronizing $ReadmePath version: $currentVersion -> $TargetVersion" -ForegroundColor Cyan
                # Replace only the version token in the header line
                $replacement = '${1}' + $TargetVersion + '${3}'
                $newContent = [regex]::Replace($content, $pattern, $replacement, [System.Text.RegularExpressions.RegexOptions]::Multiline)
                [System.IO.File]::WriteAllText((Get-Item $ReadmePath).FullName, $newContent, [System.Text.UTF8Encoding]::new($false))
                Write-Host "  [SYNC] $ReadmePath synchronized to Version $TargetVersion" -ForegroundColor Green
            } else {
                Write-Host "  [OK] $ReadmePath version is already in sync ($TargetVersion)" -ForegroundColor DarkGray
            }
        } else {
            Write-Warning "Could not match version header pattern in $ReadmePath"
        }
    }
}

function Sync-AppRcVersion {
    param(
        [string]$RcPath = "cmd\switcher\app.rc",
        [string]$TargetVersion
    )
    if (-not (Test-Path $RcPath)) {
        return
    }

    $parts = $TargetVersion.Split('.')
    while ($parts.Count -lt 4) { $parts += "0" }
    $commaVersion = ($parts[0..3]) -join ','

    $content = Get-Content $RcPath -Raw -Encoding UTF8
    $newContent = $content
    $newContent = [regex]::Replace($newContent, 'FILEVERSION\s+[0-9,]+', "FILEVERSION $commaVersion")
    $newContent = [regex]::Replace($newContent, 'PRODUCTVERSION\s+[0-9,]+', "PRODUCTVERSION $commaVersion")
    $newContent = [regex]::Replace($newContent, 'VALUE\s+"FileVersion",\s*"[^"]*"', "VALUE `"FileVersion`", `"$TargetVersion`"")
    $newContent = [regex]::Replace($newContent, 'VALUE\s+"ProductVersion",\s*"[^"]*"', "VALUE `"ProductVersion`", `"$TargetVersion`"")

    if ($newContent -ne $content) {
        Write-Host "  -> Synchronizing $RcPath to $TargetVersion ($commaVersion)" -ForegroundColor Cyan
        [System.IO.File]::WriteAllText((Get-Item $RcPath).FullName, $newContent, [System.Text.UTF8Encoding]::new($false))
        Write-Host "  [SYNC] $RcPath synchronized." -ForegroundColor Green
    } else {
        Write-Host "  [OK] $RcPath is already in sync ($TargetVersion)" -ForegroundColor DarkGray
    }
}

# Execute SSOT Synchronization
$AppVersion = Get-AppVersionFromSource
Sync-ReadmeVersion -TargetVersion $AppVersion
Sync-AppRcVersion -TargetVersion $AppVersion

if ($SyncOnly) {
    Write-Host "================================================================================" -ForegroundColor Cyan
    Write-Host "$IconLightning VERSION SYNCHRONIZATION COMPLETE (-SyncOnly)" -ForegroundColor Green
    Write-Host "================================================================================" -ForegroundColor Cyan
    exit 0
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
    $env:CGO_ENABLED = "1"
    
    # go vet
    go vet -unsafeptr=false ./cmd/switcher/... ./internal/...
    if ($LASTEXITCODE -ne 0) {
        Write-Error "go vet failed"
    }

    # go test
    go test ./cmd/switcher/... ./internal/...
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Test suite failed"
    }
    Write-Host "  [PASS] Verification succeeded." -ForegroundColor Green
} else {
    Write-Host "[2/4] Skipping tests (-SkipTests specified)." -ForegroundColor DarkGray
}

# 3. Compilation Pipeline
Write-Host "[3/4] Compiling target binaries..." -ForegroundColor Yellow

function Compile-WindowsResource {
    param(
        [string]$RcPath = "cmd\switcher\app.rc",
        [string]$SysoPath = "cmd\switcher\rsrc_windows_amd64.syso",
        [string]$IcoPath = "logo\app.ico"
    )

    Write-Host "  -> Checking Windows application icon and resource..." -ForegroundColor DarkYellow

    # Ensure icon exists; if missing, generate from PNG
    if (-not (Test-Path $IcoPath)) {
        if (Test-Path "logo\01-as-apex.png") {
            Write-Host "  Generating $IcoPath from logo\01-as-apex.png..." -ForegroundColor DarkYellow
            python scripts\generate_ico.py
        } else {
            Write-Error "Icon file not found: $IcoPath"
        }
    }

    if (-not (Test-Path $RcPath)) {
        Write-Error "Resource script not found: $RcPath"
    }

    # Resolve windres
    $windresCmd = Get-Command "windres.exe" -ErrorAction SilentlyContinue
    $windresPath = if ($windresCmd) { $windresCmd.Source } else { "C:\msys64\ucrt64\bin\windres.exe" }

    if (-not (Test-Path $windresPath)) {
        Write-Error "windres.exe not found! Please ensure MSYS2/MinGW is installed and in PATH or at C:\msys64\ucrt64\bin\windres.exe"
    }

    Write-Host "  -> Compiling resource with windres: $RcPath -> $SysoPath" -ForegroundColor Cyan
    # Honor the same explicitly selected C compiler as the Go CGO build.
    $resourceCompilerArgs = @()
    if (-not [string]::IsNullOrWhiteSpace($env:CC)) {
        $resourceCompiler = (Get-Command $env:CC -ErrorAction Stop).Source
        $resourceCompilerArgs = @('--preprocessor', ('"{0}" -E -xc -DRC_INVOKED' -f $resourceCompiler))
    }
    & $windresPath @resourceCompilerArgs -I . -I logo -I cmd\switcher -i $RcPath -O coff -o $SysoPath
    if ($LASTEXITCODE -ne 0 -or -not (Test-Path $SysoPath)) {
        Write-Error "Failed compiling Windows resource COFF object: $SysoPath"
    }

    $sysoItem = Get-Item $SysoPath
    Write-Host "  [SUCCESS] Compiled PE resource object: $SysoPath ($($sysoItem.Length) bytes)" -ForegroundColor Green
}

function Build-GUI {
    param(
        [string]$TargetDir,
        [string]$Version = $AppVersion
    )
    Write-Host "  -> Building Standalone GUI Binary (Fyne, CGO_ENABLED=1)..." -ForegroundColor Cyan
    $env:CGO_ENABLED = "1"
    $outFile = Join-Path $TargetDir "antigravity-switch.exe"
    if (Test-Path $outFile) {
        $baseName = [System.IO.Path]::GetFileNameWithoutExtension($outFile)
        Get-Process -Name $baseName -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
        Start-Sleep -Milliseconds 200
        Remove-Item -Force $outFile
    }

    $GitCommit = ""
    try {
        $GitCommit = (git rev-parse --short HEAD 2>$null).Trim()
    } catch {
        $GitCommit = ""
    }

    $ldflags = "-H windowsgui -s -w -X 'antigravity-switch/internal/core/version.Version=$Version' -X 'antigravity-switch/internal/core/version.GitCommit=$GitCommit'"

    $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
    go build -ldflags="$ldflags" -o $outFile .\cmd\switcher
    $stopwatch.Stop()

    if ($LASTEXITCODE -ne 0 -or -not (Test-Path $outFile)) {
        Write-Error "Failed building GUI binary: $outFile"
    }

    $item = Get-Item $outFile
    $sizeMB = [math]::Round($item.Length / 1MB, 2)
    Write-Host "  [SUCCESS] Created: $outFile" -ForegroundColor Green
    Write-Host "  Compile Duration: $($stopwatch.ElapsedMilliseconds) ms | Size: $sizeMB MB ($($item.Length) bytes)" -ForegroundColor Green
}

# Always compile Windows PE resource first so *.syso is automatically linked by Go compiler
Compile-WindowsResource

# Build GUI artifact
Build-GUI -TargetDir $OutDir -Version $AppVersion

# 4. Quick Verification Gate
Write-Host "[4/4] Executing quick verification..." -ForegroundColor Yellow

# Verify embedded PE icon on generated binaries
Add-Type -AssemblyName System.Drawing -ErrorAction SilentlyContinue
function Verify-BinaryIcon {
    param([string]$FilePath)
    if (Test-Path $FilePath) {
        try {
            $extracted = [System.Drawing.Icon]::ExtractAssociatedIcon((Get-Item $FilePath).FullName)
            if ($extracted -ne $null) {
                Write-Host "  [PASS] Embedded PE application icon verified on $FilePath ($($extracted.Width)x$($extracted.Height))" -ForegroundColor Green
            } else {
                Write-Warning "  [WARN] Could not extract icon from $FilePath"
            }
        } catch {
            Write-Warning "  [WARN] Icon verification encountered error: $_"
        }
    }
}

Verify-BinaryIcon -FilePath (Join-Path $OutDir "antigravity-switch.exe")
Write-Host "  [INFO] GUI binary built at $(Join-Path $OutDir "antigravity-switch.exe")." -ForegroundColor Green

Write-Host "================================================================================" -ForegroundColor Cyan
Write-Host "$IconLightning BUILD SUCCESSFUL" -ForegroundColor Green
Write-Host "================================================================================" -ForegroundColor Cyan
