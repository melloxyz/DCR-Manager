@echo off
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0tests\All.Tests.ps1"
exit /b %errorlevel%
