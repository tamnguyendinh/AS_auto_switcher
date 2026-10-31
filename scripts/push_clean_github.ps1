# scripts/push_clean_github.ps1
# AS Auto Switcher - Clean GitHub Release Publisher
# Single-command mechanism to push a sanitized version of the codebase to GitHub
# remote WITHOUT publishing internal 'docs/' and 'reports/' directories,
# while 100% PRESERVING all docs, reports, and local commit history intact.

[CmdletBinding(SupportsShouldProcess = $true)]
param(
    # The git remote name to push to (default: 'origin')
    [Parameter(Position = 0)]
    [string]$Remote = "origin",

    # Target remote branch(es) to push the clean release to (default: main, master)
    [Parameter(Position = 1)]
    [string[]]$Branches = @("main", "master"),

    # Directories to strip from the published tree (default: docs, reports)
    [string[]]$ExcludeDirs = @("docs", "reports"),

    # Custom commit message for the clean export commit.
    # If omitted, defaults to a release message referencing the HEAD commit.
    [string]$CommitMessage,

    # Produce an orphan release commit (default: $true).
    # Creates a root commit with zero parents, guaranteeing that no past commits
    # containing internal docs/reports exist in GitHub's commit history.
    [bool]$Orphan = $true,

    # Explicit force push switch. When $Orphan is true, force update is applied automatically.
    [switch]$Force,

    # Explicit DryRun switch (in addition to standard -WhatIf)
    [switch]$DryRun
)

[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8
$ErrorActionPreference = "Stop"

if ($DryRun) {
    $WhatIfPreference = $true
}

$IconShield   = [char]::ConvertFromUtf32(0x1F6E1)
$IconRocket   = [char]::ConvertFromUtf32(0x1F680)
$IconCheck    = [char]0x2714
$IconCross    = [char]0x2716
$IconFolder   = [char]::ConvertFromUtf32(0x1F4C1)

Write-Host "================================================================================" -ForegroundColor Cyan
Write-Host "$IconRocket AS Auto Switcher - Clean GitHub Release Publisher" -ForegroundColor Cyan
Write-Host "   Remote: $Remote | Branches: $($Branches -join ', ') | Exclude: $($ExcludeDirs -join ', ')" -ForegroundColor DarkCyan
Write-Host "   Orphan Mode: $Orphan (Zero historical docs/reports on remote)" -ForegroundColor DarkCyan
if ($Force -or $Orphan) {
    Write-Host "   Mode: FORCE UPDATE (Guarantees clean remote release branch)" -ForegroundColor Yellow
}
if ($WhatIfPreference) {
    Write-Host "   Mode: DRY RUN (-WhatIf active; no remote modifications will occur)" -ForegroundColor Magenta
}
Write-Host "================================================================================" -ForegroundColor Cyan

# -----------------------------------------------------------------------------
# 1. Check Worktree Hygiene
# -----------------------------------------------------------------------------
Write-Host "[1/5] Checking worktree hygiene..." -ForegroundColor Yellow
$porcelainStatus = cmd.exe /c "git status --porcelain 2>&1"
if ($porcelainStatus) {
    Write-Host "$IconCross [ERROR] Working tree has uncommitted or unstaged changes:" -ForegroundColor Red
    $porcelainStatus | ForEach-Object { Write-Host "    $_" -ForegroundColor DarkRed }
    Write-Error "Worktree is not clean. Please commit or stash all changes before running this script."
}
Write-Host "  $IconCheck Worktree is 100% clean." -ForegroundColor Green

# -----------------------------------------------------------------------------
# 2. Record Current Working Branch & HEAD Commit
# -----------------------------------------------------------------------------
Write-Host "[2/5] Inspecting current branch and commit..." -ForegroundColor Yellow
$currentBranch = (cmd.exe /c "git branch --show-current 2>&1").Trim()
$headCommit    = (cmd.exe /c "git rev-parse HEAD 2>&1").Trim()
$headSubject   = (cmd.exe /c "git log -1 --pretty=%s 2>&1").Trim()

Write-Host "  -> Current working branch : $currentBranch" -ForegroundColor DarkGreen
Write-Host "  -> Local HEAD commit      : $headCommit" -ForegroundColor DarkGreen
Write-Host "  -> Local HEAD subject     : $headSubject" -ForegroundColor DarkGreen

if (-not $currentBranch) {
    Write-Warning "Repository is currently in detached HEAD state."
}

# -----------------------------------------------------------------------------
# 3. Create Sanitized Tree (Plumbing Isolation)
# -----------------------------------------------------------------------------
Write-Host "[3/5] Generating sanitized tree without excluded directories..." -ForegroundColor Yellow
$tempIndex = Join-Path ([System.IO.Path]::GetTempPath()) ("git-clean-idx-" + [Guid]::NewGuid().ToString("N"))
[System.Environment]::SetEnvironmentVariable("GIT_INDEX_FILE", $tempIndex, [System.EnvironmentVariableTarget]::Process)

$cleanTree = $null
try {
    # Read HEAD tree into the isolated temporary index
    $readOutput = cmd.exe /c "git read-tree HEAD 2>&1"
    if ($LASTEXITCODE -ne 0) {
        throw "git read-tree HEAD failed: $readOutput"
    }

    # Strip excluded directories from the isolated index
    foreach ($dir in $ExcludeDirs) {
        $treeEntry = cmd.exe /c "git ls-tree HEAD $dir 2>&1"
        if ($treeEntry -and $LASTEXITCODE -eq 0) {
            $rmOutput = cmd.exe /c "git rm -r --cached --quiet --ignore-unmatch $dir 2>&1"
            Write-Host "  $IconFolder [STRIPPED] Directory '$dir' removed from release tree." -ForegroundColor Cyan
        } else {
            Write-Host "  [SKIP] Directory '$dir' not found in HEAD tree." -ForegroundColor DarkGray
        }
    }

    # Write clean tree object to git database
    $cleanTree = (cmd.exe /c "git write-tree 2>&1").Trim()
    if ($LASTEXITCODE -ne 0 -or -not $cleanTree) {
        throw "git write-tree failed to generate tree object"
    }
    Write-Host "  -> Clean Tree SHA : $cleanTree" -ForegroundColor Green

    # Security check: verify none of the excluded directories exist in $cleanTree
    foreach ($dir in $ExcludeDirs) {
        $checkClean = cmd.exe /c "git ls-tree $cleanTree $dir 2>&1"
        if ($checkClean) {
            throw "Security check failed: directory '$dir' is still present in clean tree ($cleanTree)"
        }
    }
    Write-Host "  $IconCheck [SECURITY PASS] Zero excluded directories present in clean tree." -ForegroundColor Green
}
finally {
    # .NET cleanup is immune to $WhatIfPreference skipping
    [System.Environment]::SetEnvironmentVariable("GIT_INDEX_FILE", $null, [System.EnvironmentVariableTarget]::Process)
    if ([System.IO.File]::Exists($tempIndex)) {
        [System.IO.File]::Delete($tempIndex)
    }
}

# -----------------------------------------------------------------------------
# 4. Formulate Clean Release Commit & Temporary Branch
# -----------------------------------------------------------------------------
Write-Host "[4/5] Creating clean release commit and temporary branch..." -ForegroundColor Yellow
$msg = if ($CommitMessage) {
    $CommitMessage
} else {
    "release(clean): sanitized codebase export (v1.0.3.3) - $headSubject"
}

# Commit tree: either orphan root (zero parents) or with HEAD parent
$cleanCommit = if ($Orphan) {
    (cmd.exe /c "git commit-tree $cleanTree -m `"$msg`" 2>&1").Trim()
} else {
    (cmd.exe /c "git commit-tree $cleanTree -p $headCommit -m `"$msg`" 2>&1").Trim()
}

if ($LASTEXITCODE -ne 0 -or -not $cleanCommit) {
    throw "git commit-tree failed to create release commit: $cleanCommit"
}
Write-Host "  -> Clean Commit SHA: $cleanCommit (Orphan Root: $Orphan)" -ForegroundColor Green

$tempBranch = "temp-clean-publish-" + (Get-Date -Format "yyyyMMdd-HHmmss")
$branchOutput = cmd.exe /c "git branch $tempBranch $cleanCommit 2>&1"
if ($LASTEXITCODE -ne 0) {
    throw "Failed to create temporary branch '$tempBranch': $branchOutput"
}
Write-Host "  -> Created temporary branch: $tempBranch" -ForegroundColor DarkGreen

# -----------------------------------------------------------------------------
# 5. Push to Remote Branches & Cleanup
# -----------------------------------------------------------------------------
try {
    Write-Host "[5/5] Publishing clean commit to remote '$Remote'..." -ForegroundColor Yellow

    $forceFlag = if ($Force -or $Orphan) { "--force" } else { "" }

    foreach ($targetBranch in $Branches) {
        $targetRef = "${cleanCommit}:refs/heads/${targetBranch}"
        $desc = "Push clean commit $cleanCommit to $Remote/$targetBranch"

        if ($PSCmdlet.ShouldProcess("$Remote/$targetBranch", $desc)) {
            Write-Host "  -> Pushing clean release to $Remote/$targetBranch..." -ForegroundColor Cyan
            $pushCmd = "git push $Remote $targetRef $forceFlag 2>&1"
            $pushOutput = cmd.exe /c $pushCmd
            $pushExit = $LASTEXITCODE

            if ($pushExit -ne 0) {
                Write-Host "$IconCross [PUSH FAILED] Output from git push:" -ForegroundColor Red
                $pushOutput | ForEach-Object { Write-Host "    $_" -ForegroundColor DarkRed }
                throw "Failed to push to $Remote/$targetBranch (exit code $pushExit)"
            }
            $pushOutput | ForEach-Object { Write-Host "    $_" -ForegroundColor DarkGray }
            Write-Host "  $IconCheck [SUCCESS] Clean codebase pushed to $Remote/$targetBranch." -ForegroundColor Green
        } else {
            Write-Host "  [WHATIF] Dry-run verification with git push --dry-run:" -ForegroundColor Magenta
            $dryRunCmd = "git push $Remote $targetRef --dry-run $forceFlag 2>&1"
            $dryRunOutput = cmd.exe /c $dryRunCmd
            $dryRunExit = $LASTEXITCODE
            if ($dryRunExit -eq 0) {
                $dryRunOutput | ForEach-Object { Write-Host "    $_" -ForegroundColor DarkGray }
                Write-Host "  $IconCheck [WHATIF PASS] Remote accepts push ref without errors." -ForegroundColor Green
            } else {
                Write-Host "    $dryRunOutput" -ForegroundColor DarkYellow
                Write-Warning "Remote rejected dry-run push. If remote history diverged, add -Force."
            }
        }
    }
}
finally {
    # Clean up temporary local branch
    if ($tempBranch) {
        cmd.exe /c "git branch -D $tempBranch 2>&1" | Out-Null
        Write-Host "  -> Cleaned up temporary branch: $tempBranch" -ForegroundColor DarkGray
    }

    # Verify or restore original working branch
    $finalBranch = (cmd.exe /c "git branch --show-current 2>&1").Trim()
    if ($currentBranch -and $finalBranch -ne $currentBranch) {
        Write-Host "  -> Restoring original branch: $currentBranch" -ForegroundColor Yellow
        cmd.exe /c "git checkout $currentBranch --quiet 2>&1" | Out-Null
    }

    # Verify worktree hygiene preserved
    $finalStatus = cmd.exe /c "git status --porcelain 2>&1"
    if ($finalStatus) {
        Write-Warning "Worktree has changes after execution. Please inspect git status."
    } else {
        Write-Host "  $IconCheck [VERIFIED] Local branch '$currentBranch' and all documentation remain 100% intact." -ForegroundColor Green
    }
}

Write-Host "================================================================================" -ForegroundColor Cyan
Write-Host "$IconCheck Clean export pipeline completed successfully." -ForegroundColor Green
Write-Host "================================================================================" -ForegroundColor Cyan
