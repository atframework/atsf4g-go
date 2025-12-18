@echo off
REM Uninstall git hooks for this repository
REM This resets the core.hooksPath configuration and cleans up auto-generated hooks

setlocal enabledelayedexpansion

REM Get the directory of this script
set "SCRIPT_DIR=%~dp0"
REM Get the repo root (parent of scripts directory)
for %%I in ("%SCRIPT_DIR%..") do set "REPO_ROOT=%%~fI"

echo 🗑️  Uninstalling git hooks for this repository...

REM Check if we're in a git repository
cd /d "%REPO_ROOT%"
git rev-parse --git-dir >nul 2>&1
if errorlevel 1 (
    echo ❌ Not a git repository: %REPO_ROOT%
    exit /b 1
)

REM 1. Reset core.hooksPath to default
echo 🔄 Resetting core.hooksPath...
git config --unset core.hooksPath >nul 2>&1 || (echo. && echo Note: core.hooksPath was not set)
echo ✅ core.hooksPath reset

REM 2. Clean up auto-generated LFS hooks
echo 🗑️  Removing auto-generated LFS hooks...
for %%H in (post-checkout post-commit post-merge pre-push) do (
    if exist "%SCRIPT_DIR%%%H" (
        del /q "%SCRIPT_DIR%%%H"
        echo ✅ Removed %%H
    )
)

REM 3. Clean up log files
if exist "%SCRIPT_DIR%pre-commit.log" (
    del /q "%SCRIPT_DIR%pre-commit.log"
    echo ✅ Removed pre-commit.log
)

REM 4. Reinstall Git LFS hooks to default location (.git/hooks/)
where git-lfs >nul 2>&1
if errorlevel 0 (
    echo 🔄 Reinstalling Git LFS hooks to .git/hooks/...
    git lfs install --local --force >nul 2>&1
    echo ✅ Git LFS hooks reinstalled to default location
)

echo.
echo ✅ Git hooks uninstalled successfully!
echo    • core.hooksPath has been reset
echo    • Auto-generated LFS hooks have been removed
echo    • Custom hooks remain in script/hooks/ (but are no longer active)
echo.
echo To reinstall, run: .\script\hooks\install.bat
