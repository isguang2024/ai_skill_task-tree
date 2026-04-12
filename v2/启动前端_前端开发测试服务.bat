@echo off
setlocal
cd /d "%~dp0frontend" || (
  echo Failed to enter frontend directory.
  pause
  exit /b 1
)
echo Starting frontend in debug mode...
echo Open http://127.0.0.1:5174/
echo Working directory: %CD%
where npm
npm -v
npm run dev
if errorlevel 1 (
  echo Frontend exited with errorlevel %errorlevel%.
)
echo Frontend stopped. See the output above for details.
pause
endlocal
