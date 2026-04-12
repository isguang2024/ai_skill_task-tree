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
echo Starting backend in debug mode...
set "TTS_ADDR=127.0.0.1:8880"
echo Working directory: %CD%
where go
go version
go run ./cmd/task-tree-service serve
if errorlevel 1 (
  echo Backend exited with errorlevel %errorlevel%.
)
echo Backend stopped. See the output above for details.
pause
endlocal
