@echo off
echo ============================================
echo AwoChat - Backend Setup Script (Windows)
echo ============================================
echo.

cd /d "%~dp0\.."
cd backend

echo Checking Go installation...
where go >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Go not found in PATH!
    echo Please install Go from https://go.dev/dl/
    pause
    exit /b 1
)

go version
echo.

echo Downloading Go dependencies...
go mod download
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Failed to download dependencies
    pause
    exit /b 1
)
echo [OK] Dependencies downloaded
echo.

echo Creating .env file from template...
if not exist .env (
    if exist .env.example (
        copy .env.example .env
        echo [OK] .env file created
    ) else (
        echo [WARN] .env.example not found, skipping
    )
) else (
    echo [INFO] .env already exists
)
echo.

echo Running database migrations...
go run cmd/migrate/main.go -command up
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Migration failed! Make sure PostgreSQL is running.
    pause
    exit /b 1
)
echo [OK] Migrations completed
echo.

echo Building backend...
go build ./cmd/main.go
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Build failed
    pause
    exit /b 1
)
echo [OK] Build successful - main.exe created
echo.

echo ============================================
echo Backend setup complete!
echo ============================================
echo.
echo To start the server:
echo   go run cmd/main.go
echo.
echo Or run the built executable:
echo   main.exe
echo.
pause
