@echo off
echo ============================================
echo AwoChat - Starting Development Servers
echo ============================================
echo.
echo This will start both backend and frontend.
echo Press Ctrl+C in each window to stop.
echo.
pause

REM Start backend in new window
start "AwoChat Backend" cmd /k "cd /d %~dp0\backend && go run cmd/main.go"

REM Wait a moment for backend to start
timeout /t 3 /nobreak >nul

REM Start frontend in new window
start "AwoChat Frontend" cmd /k "cd /d %~dp0\frontend && npm run dev"

echo.
echo ============================================
echo Servers starting...
echo ============================================
echo.
echo Backend: http://localhost:8080
echo Frontend: http://localhost:3000
echo.
echo Check the terminal windows for output.
echo Close them to stop the servers.
echo.
