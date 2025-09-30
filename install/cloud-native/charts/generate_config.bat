@echo off
cd /d %~dp0

..\..\atdtool\bin\atdtool.exe template .\ -o ..\..  --values ..\values\default  --set global.world_id=1