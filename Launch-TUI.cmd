@echo off
if not exist "%~dp0bin\dcr-manager.exe" (
  echo Build missing. Run Build-TUI.cmd first.
  pause
  exit /b 1
)
"%~dp0bin\dcr-manager.exe" %*
if errorlevel 1 pause
