@echo off
REM Install git hooks for this repository only
REM This uses Git's core.hooksPath to isolate hooks to this repo

setlocal enabledelayedexpansion

REM Get the directory of this script
set "SCRIPT_DIR=%~dp0"
REM Get the repo root (parent of scripts directory)
for %%I in ("%SCRIPT_DIR%..") do set "REPO_ROOT=%%~fI"
set "HOOKS_DIR=%SCRIPT_DIR%"

echo 📦 Installing git hooks for this repository only...

REM Check if we're in a git repository
cd /d "%REPO_ROOT%"
git rev-parse --git-dir >nul 2>&1
if errorlevel 1 (
    echo ❌ Not a git repository: %REPO_ROOT%
    exit /b 1
)

REM Make hooks executable (on Windows, just set file permissions)
for %%H in (pre-commit.bat pre-commit.ps1) do (
    if exist "%HOOKS_DIR%\%%H" (
        echo ✅ Hook ready: %%H
    )
)

REM Set core.hooksPath to use hooks from script/hooks
REM This is relative to the repository root
git config core.hooksPath script/hooks

echo ✅ Git hooks installed successfully!
