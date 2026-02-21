@echo off
echo ============================================
echo AwoChat - Full Development Setup (Windows)
echo ============================================
echo.
echo This script will run all setup steps:
echo   1. Database setup
echo   2. Backend setup
echo   3. Frontend setup
echo.
echo Make sure PostgreSQL is running!
echo.
pause

cd /d "%~dp0"

echo.
echo ============================================
echo Step 1: Database Setup
echo ============================================
echo.
call setup-db.bat

echo.
echo ============================================
echo Step 2: Backend Setup
echo ============================================
echo.
call setup-backend.bat

echo.
echo ============================================
echo Step 3: Frontend Setup
echo ============================================
echo.
call setup-frontend.bat

echo.
echo ============================================
echo ALL SETUP COMPLETE!
echo ============================================
echo.
echo To start development:
echo.
echo Terminal 1 (Backend):
echo   cd d:\Projects\AwoChat\backend
echo   go run cmd/main.go
echo.
echo Terminal 2 (Frontend):
echo   cd d:\Projects\AwoChat\frontend
echo   npm run dev
echo.
echo Then open: http://localhost:3000
echo.
pause
