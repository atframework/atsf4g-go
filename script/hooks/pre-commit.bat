@echo off
REM Git pre-commit hook wrapper for Windows
REM This calls the PowerShell version which is more reliable on Windows

powershell.exe -ExecutionPolicy Bypass -File "%~dp0pre-commit.ps1"
exit /b %ERRORLEVEL%
