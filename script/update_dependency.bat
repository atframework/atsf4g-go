@echo off
cd /d %~dp0

for /d %%i in (.\deploy\charts\*) do (
    if "%%i" neq "libapp" (
        if "%%i" neq "app" (
            echo "%%i"
            helm dependency update %%i
        )
    )
)