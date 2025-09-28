@echo off
cd /d %~dp0

for /d %%i in (*) do (
    if "%%i" neq "libapp" (
        if "%%i" neq "app" (
            echo "%%i"
            helm dependency update %%i
        )
    )
)