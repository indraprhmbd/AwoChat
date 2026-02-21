@echo off
echo ============================================
echo AwoChat - Database Setup Script (Windows)
echo ============================================
echo.

REM Check if psql is available
where psql >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: psql not found in PATH!
    echo Please install PostgreSQL and add it to your PATH.
    echo Default location: C:\Program Files\PostgreSQL\XX\bin
    echo.
    echo Or run: scripts\check-postgres.bat
    pause
    exit /b 1
)

echo Enter PostgreSQL postgres user password:
set /p PGPASSWORD=

echo.
echo Creating database and user...
echo.

REM Create database
set PGPASSWORD=%PGPASSWORD%
psql -U postgres -c "CREATE DATABASE awochat;" 2>nul
if %ERRORLEVEL% EQU 0 (
    echo [OK] Database 'awochat' created
) else (
    echo [INFO] Database 'awochat' may already exist
)

REM Create user
psql -U postgres -c "CREATE USER awochat WITH PASSWORD 'awochat';" 2>nul
if %ERRORLEVEL% EQU 0 (
    echo [OK] User 'awochat' created
) else (
    echo [INFO] User 'awochat' may already exist
)

REM Grant database privileges
psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE awochat TO awochat;"
echo [OK] Database privileges granted

REM Grant schema permissions (fixes permission denied error)
echo.
echo Granting schema permissions...
psql -U postgres -d awochat -c "GRANT ALL ON SCHEMA public TO awochat;"
psql -U postgres -d awochat -c "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO awochat;"
psql -U postgres -d awochat -c "GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO awochat;"
psql -U postgres -d awochat -c "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO awochat;"
psql -U postgres -d awochat -c "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO awochat;"
echo [OK] Schema permissions granted

echo.
echo ============================================
echo Database setup complete!
echo ============================================
echo.
echo Next steps:
echo 1. cd d:\Projects\AwoChat\backend
echo 2. go run cmd/migrate/main.go -command up
echo.
pause
