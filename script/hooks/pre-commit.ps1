# Git pre-commit hook for Go code formatting (PowerShell version)
# Auto-format Go files before commit
# Logs output to: script\hooks\pre-commit.log

# Get the directory where this script is located
$SCRIPT_DIR = Split-Path -Parent $MyInvocation.MyCommand.Path
$LOG_FILE = Join-Path $SCRIPT_DIR "pre-commit.log"
$REPO_ROOT = Split-Path -Parent (Split-Path -Parent $SCRIPT_DIR)

$CI_CONFIG_FILE = $null
if (Test-Path (Join-Path $REPO_ROOT ".golangci.yaml")) {
    $CI_CONFIG_FILE = Join-Path $REPO_ROOT ".golangci.yaml"
}

# Function to log output (write to console and log file)
function Log-Output {
    param([string]$message)
    Write-Host $message
    Add-Content -Path $LOG_FILE -Value $message
}

# Clear/create log file
"=== Pre-commit Hook Log - $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') ===" | Set-Content $LOG_FILE
Write-Host "🔧 Running Go code formatter..." -ForegroundColor Cyan
Log-Output "🔧 Running Go code formatter..."

# Get staged Go files
$goFiles = git diff --cached --name-only --diff-filter=ACM | Where-Object { $_ -match '\.go$' }

if (-not $goFiles) {
    Write-Host "✅ No Go files to format" -ForegroundColor Green
    Log-Output "✅ No Go files to format"
    exit 0
}

# Check if goimports is installed
$goimportsPath = Get-Command goimports -ErrorAction SilentlyContinue
if (-not $goimportsPath) {
    Write-Host "⚠️  goimports not found. Installing..." -ForegroundColor Yellow
    Log-Output "⚠️  goimports not found. Installing..."
    go install golang.org/x/tools/cmd/goimports@latest
    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ Failed to install goimports" -ForegroundColor Red
        Log-Output "❌ Failed to install goimports"
        exit 1
    }
}

# Format files with goimports
Write-Host "📝 Formatting Go files:" -ForegroundColor Cyan
Log-Output "📝 Formatting Go files:"
foreach ($file in $goFiles) {
    if (Test-Path $file) {
        Write-Host "  - $file" -ForegroundColor Gray
        Log-Output "  - $file"
        goimports -w $file
        git add $file
    }
}

# Optional: Check golangci-lint on staged files only
$lintPath = Get-Command golangci-lint -ErrorAction SilentlyContinue
if ($lintPath) {
    Write-Host "🔍 Running golangci-lint on staged files..." -ForegroundColor Cyan
    Log-Output "🔍 Running golangci-lint on staged files..."
    
    # Log config file if available
    if ($CI_CONFIG_FILE) {
        Write-Host "    Using config: $CI_CONFIG_FILE" -ForegroundColor DarkGray
        Log-Output "    Using config: $CI_CONFIG_FILE"
    }
    
    # Find the module root for each staged file and run golangci-lint from there
    $failedCheck = $false
    foreach ($file in $goFiles) {
        # Find module root by looking for go.mod
        $dir = Split-Path -Parent $file
        $moduleRoot = $dir
        while ($moduleRoot -ne "" -and -not (Test-Path "$moduleRoot/go.mod")) {
            $moduleRoot = Split-Path -Parent $moduleRoot
        }
        
        if ($moduleRoot -eq "" -or -not (Test-Path "$moduleRoot/go.mod")) {
            Write-Host "⚠️  No go.mod found for $file, skipping lint check" -ForegroundColor Yellow
            Log-Output "⚠️  No go.mod found for $file, skipping lint check"
            continue
        }
        
        # Convert file path to relative path from module root
        $fullFilePath = (Resolve-Path -Path $file).ProviderPath
        $fullModuleRoot = (Resolve-Path -Path $moduleRoot).ProviderPath
        $relativeFile = $fullFilePath.Substring($fullModuleRoot.Length).TrimStart('\', '/')
        
        Write-Host "    Linting: $relativeFile (module: $(Split-Path -Leaf $moduleRoot))" -ForegroundColor DarkGray
        Log-Output "    Linting: $relativeFile (module: $(Split-Path -Leaf $moduleRoot))"
        
        # Run golangci-lint from module root
        Push-Location $moduleRoot
        try {
            $lintArgs = @("run")
            if ($CI_CONFIG_FILE) {
                $lintArgs += "--config=$CI_CONFIG_FILE"
            }
            $lintArgs += $relativeFile
            
            $output = golangci-lint @lintArgs 2>&1
            $exitCode = $LASTEXITCODE
            
            # Output diagnostics and save to log
            if ($output) {
                Write-Host "$output" -ForegroundColor Yellow
                $output | ForEach-Object { Log-Output $_ }
            }
            
            # Check if there are actual issues
            if ($exitCode -ne 0) {
                $failedCheck = $true
            }
        }
        finally {
            Pop-Location
        }
    }
    
    if ($failedCheck) {
        Write-Host "❌ golangci-lint found issues. Commit blocked!" -ForegroundColor Red
        Log-Output "❌ golangci-lint found issues. Commit blocked!"
        Write-Host "Please fix the issues and try again." -ForegroundColor Yellow
        Log-Output "Please fix the issues and try again."
        Write-Host "Or skip with: git commit --no-verify" -ForegroundColor Gray
        Log-Output "Or skip with: git commit --no-verify"
        Log-Output "Log saved to: $LOG_FILE"
        exit 1
    }
    
    Write-Host "✅ golangci-lint passed" -ForegroundColor Green
    Log-Output "✅ golangci-lint passed"
}

Write-Host "✅ Code formatting complete" -ForegroundColor Green
Log-Output "✅ Code formatting complete"
Log-Output "📝 Full log saved to: $LOG_FILE"
exit 0
