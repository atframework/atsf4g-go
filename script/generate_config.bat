@echo off
cd /d %~dp0

.\atdtool\bin\atdtool.exe template .\deploy\charts -o .\  --values .\deploy\values\default,.\deploy\values\dev --set global.world_id=1