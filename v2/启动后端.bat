@echo off
setlocal
cd /d "%~dp0" || (
  echo Failed to enter repository root.
  pause
  exit /b 1
)
cd /d "%~dp0backend" || (
  echo Failed to enter backend directory.
  pause
  exit /b 1
)
set "TTS_ADDR=127.0.0.1:8880"
set "BACKEND_EXE=%CD%\task-tree-service.exe"
echo Working directory: %CD%
if exist "%BACKEND_EXE%" (
  echo Starting backend from prebuilt executable...
  "%BACKEND_EXE%" serve
) else (
  echo Prebuilt executable not found. Falling back to go run debug mode...
  go run ./cmd/task-tree-service serve
)
if errorlevel 1 (
  echo Backend exited with errorlevel %errorlevel%.
)
echo Backend stopped. See the output above for details.
pause
endlocal
