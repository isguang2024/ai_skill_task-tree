@echo off
setlocal
cd /d "%~dp0frontend" || (
echo Failed to enter frontend directory.
  pause
  exit /b 1
)
echo Starting frontend in debug mode...
echo Open http://127.0.0.1:5173/
npm run dev
echo Frontend stopped. See the output above for details.
pause
endlocal
