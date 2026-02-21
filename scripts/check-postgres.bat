@echo off
echo ============================================
echo AwoChat - PostgreSQL Status Check
echo ============================================
echo.

REM Check if PostgreSQL service exists and is running
sc query postgresql-x64-16 | find "RUNNING" >nul
if %ERRORLEVEL% EQU 0 (
    echo [OK] PostgreSQL 16 is RUNNING
    goto :test_connection
)

sc query postgresql-x64-15 | find "RUNNING" >nul
if %ERRORLEVEL% EQU 0 (
    echo [OK] PostgreSQL 15 is RUNNING
    goto :test_connection
)

sc query postgresql-x64-14 | find "RUNNING" >nul
if %ERRORLEVEL% EQU 0 (
    echo [OK] PostgreSQL 14 is RUNNING
    goto :test_connection
)

echo [WARN] PostgreSQL service not running or not found
echo.
echo Trying to start PostgreSQL 16...
sc query postgresql-x64-16 >nul 2>nul
if %ERRORLEVEL% EQU 0 (
    echo Starting PostgreSQL 16 service...
    net start postgresql-x64-16
    if %ERRORLEVEL% EQU 0 (
        echo [OK] PostgreSQL 16 started
    ) else (
        echo [ERROR] Failed to start PostgreSQL 16
        echo Please run this script as Administrator
        goto :end
    )
) else (
    echo [ERROR] PostgreSQL service not found
    echo Please install PostgreSQL from:
    echo https://www.postgresql.org/download/windows/
    goto :end
)

:test_connection
echo.
echo ============================================
echo Testing database connection...
echo ============================================
echo.

REM Try to connect
psql -U postgres -c "SELECT 'PostgreSQL is working!' as status;" 2>nul
if %ERRORLEVEL% EQU 0 (
    echo.
    echo [OK] PostgreSQL connection successful!
) else (
    echo.
    echo [WARN] Connection test failed
    echo Make sure you can connect with: psql -U postgres
)

:end
echo.
echo ============================================
echo Next step: Run setup-db.bat to create database
echo ============================================
echo.
pause
