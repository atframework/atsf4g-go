@echo off
cd /d %~dp0

if exist ".\atdtool\bin\atdtool.exe" (
  .\atdtool\bin\atdtool.exe template .\deploy\charts -o .\  --values .\deploy\values\default,.\deploy\values\dev --set global.world_id=1
) else (
  .\atdtool\atdtool.exe template .\deploy\charts -o .\  --values .\deploy\values\default,.\deploy\values\dev --set global.world_id=1
)