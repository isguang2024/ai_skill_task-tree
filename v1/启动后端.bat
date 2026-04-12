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
go run ./cmd/task-tree-service serve
echo Backend stopped. See the output above for details.
pause
endlocal
