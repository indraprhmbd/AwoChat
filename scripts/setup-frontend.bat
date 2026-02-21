@echo off
echo ============================================
echo AwoChat - Frontend Setup Script (Windows)
echo ============================================
echo.

cd /d "%~dp0\.."
cd frontend

echo Checking Node.js installation...
where node >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Node.js not found in PATH!
    echo Please install Node.js from https://nodejs.org/
    pause
    exit /b 1
)

node --version
npm --version
echo.

echo Installing npm dependencies...
call npm install
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: npm install failed
    pause
    exit /b 1
)
echo [OK] Dependencies installed
echo.

echo Building frontend...
call npm run build
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Build failed
    pause
    exit /b 1
)
echo [OK] Build successful - dist/ folder created
echo.

echo ============================================
echo Frontend setup complete!
echo ============================================
echo.
echo To start the dev server:
echo   npm run dev
echo.
pause
