@echo off
REM Windows batch script to test git pre-commit hook
REM This script tests the PowerShell pre-commit hook

setlocal enabledelayedexpansion

REM Get the current script directory
set SCRIPT_DIR=%~dp0
REM Get the repository root (2 levels up from script dir: hooks -> scripts -> root)
for %%A in ("%SCRIPT_DIR%..\..\") do set REPO_ROOT=%%~fA

REM Set the path to the pre-commit.ps1 script
set PRE_COMMIT_SCRIPT=%SCRIPT_DIR%pre-commit.ps1

echo.
echo ======================================
echo Git Pre-commit Hook Test (Windows)
echo ======================================
echo.
echo Repository Root: %REPO_ROOT%
echo Pre-commit Script: %PRE_COMMIT_SCRIPT%
echo.

REM Check if pre-commit.ps1 exists
if not exist "%PRE_COMMIT_SCRIPT%" (
    echo ERROR: pre-commit.ps1 not found at %PRE_COMMIT_SCRIPT%
    exit /b 1
)

REM Save current directory and change to repo root
pushd "%REPO_ROOT%"

echo Running pre-commit hook...
echo.

REM Run the PowerShell script
powershell -NoProfile -ExecutionPolicy Bypass -File "%PRE_COMMIT_SCRIPT%"

REM Capture exit code
set EXIT_CODE=%ERRORLEVEL%

REM Return to original directory
popd

echo.
echo ======================================
if %EXIT_CODE% equ 0 (
    echo Status: PRE-COMMIT HOOK PASSED ✓
) else (
    echo Status: PRE-COMMIT HOOK FAILED ✗ (Exit Code: %EXIT_CODE%)
)
echo ======================================
echo.

REM Keep window open to see results
pause

exit /b %EXIT_CODE%
