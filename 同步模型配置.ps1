# Aurora Model Config Sync Script
# Usage: .\Sync-ModelConfig.ps1 -Profile step-3.7-flash
#        .\Sync-ModelConfig.ps1 -Profile longcat -Workspace "D:\MyAurora"
# Auto-detects workspace if -Workspace not specified.

param(
    [Parameter(Mandatory=$true)]
    [string]$Profile,

    [string]$Workspace
)

$ErrorActionPreference = "Stop"

# Auto-detect workspace: look for .nova directory
if (-not $Workspace) {
    $searchDir = $PSScriptRoot
    if (-not $searchDir) { $searchDir = (Get-Location).Path }

    for ($i = 0; $i -lt 6; $i++) {
        $testPath = Join-Path $searchDir ".nova"
        if (Test-Path $testPath) {
            $Workspace = $searchDir
            break
        }
        # Also check for Aurora.exe or config.toml with model_profiles
        $exePath = Join-Path $searchDir "Aurora.exe"
        if (Test-Path $exePath) { $Workspace = $searchDir; break }
        $exePath = Join-Path $searchDir "denova.exe"
        if (Test-Path $exePath) { $Workspace = $searchDir; break }

        $parent = Split-Path $searchDir -Parent
        if ($parent -eq $searchDir) { break }
        $searchDir = $parent
    }

    if (-not $Workspace) {
        # Fallback: use script location's parent
        $Workspace = Split-Path $PSScriptRoot -Parent
        if (-not $Workspace) { $Workspace = (Get-Location).Path }
    }
}

$NovaDir = Join-Path $Workspace ".nova"

Write-Host "Aurora Model Config Sync" -ForegroundColor Cyan
Write-Host "Workspace: $Workspace" -ForegroundColor DarkGray
Write-Host "Nova dir:  $NovaDir" -ForegroundColor DarkGray
Write-Host "Target profile: $Profile" -ForegroundColor Cyan
Write-Host ""

if (-not (Test-Path $NovaDir)) {
    Write-Host "Error: .nova directory not found at $NovaDir" -ForegroundColor Red
    Write-Host "Specify workspace with -Workspace parameter" -ForegroundColor Yellow
    exit 1
}

# Find all book configs
$books = Get-ChildItem -Path $NovaDir -Directory | Where-Object {
    Test-Path (Join-Path $_.FullName ".nova\config.toml")
}

if ($null -eq $books) {
    Write-Host "No book configurations found in $NovaDir" -ForegroundColor Yellow
    exit 0
}

$pattern = "profile_id\s*=\s*'[^']+'"
$changed = 0

foreach ($book in $books) {
    $configPath = Join-Path $book.FullName ".nova\config.toml"
    $content = Get-Content $configPath -Raw -Encoding UTF8

    # Extract current profiles
    $currentProfiles = [regex]::Matches($content, $pattern) |
        ForEach-Object { $_.Value } | Sort-Object -Unique

    $replacement = "profile_id = '$Profile'"
    $newContent = [regex]::Replace($content, $pattern, $replacement)

    if ($newContent -ne $content) {
        Set-Content -Path $configPath -Value $newContent -Encoding UTF8 -NoNewline
        Write-Host "[OK] $($book.Name)" -ForegroundColor Green
        Write-Host "     Old: $($currentProfiles -join ', ')"
        Write-Host "     New: $Profile"
        $changed++
    } else {
        Write-Host "[SKIP] $($book.Name) (already $Profile)" -ForegroundColor DarkGray
    }
}

Write-Host ""
if ($changed -gt 0) {
    Write-Host "$changed book(s) updated. Restart Aurora to apply." -ForegroundColor Cyan
} else {
    Write-Host "All books already use $Profile. No changes needed." -ForegroundColor Cyan
}
