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
set "TTS_ADDR=127.0.0.1:8879"

if not exist "task-tree-service.exe" (
  echo First run: building task-tree-service.exe ...
  go build -o task-tree-service.exe ./cmd/task-tree-service
  if errorlevel 1 (
    echo Build failed.
    pause
    exit /b 1
  )
  echo Build complete.
)

echo Starting backend...
.\task-tree-service.exe serve
echo Backend stopped. See the output above for details.
pause
endlocal
