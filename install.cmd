@echo off
setlocal enabledelayedexpansion

REM OrchCLI Installation Script for Windows CMD
REM Usage:
REM   curl -fsSL https://raw.githubusercontent.com/KubeOrch/cli/main/install.cmd -o install.cmd && install.cmd && del install.cmd
REM
REM Environment variables:
REM   - ORCHCLI_INSTALL_DIR: Installation directory (default: %LOCALAPPDATA%\Programs\orchcli)
REM   - ORCHCLI_VERSION: Version to install (default: latest)

set "GITHUB_REPO=KubeOrch/cli"
set "BINARY_NAME=orchcli.exe"

echo.
echo ================================================
echo      OrchCLI - KubeOrch Developer CLI
echo ================================================
echo.

REM Determine architecture
set "ARCH=amd64"
if "%PROCESSOR_ARCHITECTURE%"=="ARM64" set "ARCH=arm64"

echo [INFO] Detected architecture: %ARCH%

REM Determine version
if defined ORCHCLI_VERSION (
    set "VERSION=%ORCHCLI_VERSION%"
    echo [INFO] Using specified version: !VERSION!
) else (
    echo [INFO] Fetching latest version...
    for /f "tokens=*" %%i in ('curl -s "https://api.github.com/repos/%GITHUB_REPO%/releases/latest" ^| findstr "tag_name"') do (
        set "TAG_LINE=%%i"
    )
    REM Extract version from JSON line like:  "tag_name": "v0.0.3",
    for /f "tokens=2 delims=:" %%a in ("!TAG_LINE!") do (
        set "VERSION=%%a"
    )
    REM Clean up whitespace, quotes, and commas
    set "VERSION=!VERSION: =!"
    set "VERSION=!VERSION:"=!"
    set "VERSION=!VERSION:,=!"

    if "!VERSION!"=="" (
        echo [ERROR] Failed to fetch latest version
        exit /b 1
    )
    echo [INFO] Latest version: !VERSION!
)

REM Determine install directory
if defined ORCHCLI_INSTALL_DIR (
    set "INSTALL_DIR=%ORCHCLI_INSTALL_DIR%"
) else (
    set "INSTALL_DIR=%LOCALAPPDATA%\Programs\orchcli"
)

REM Create install directory
if not exist "!INSTALL_DIR!" (
    echo [INFO] Creating installation directory: !INSTALL_DIR!
    mkdir "!INSTALL_DIR!"
)

REM Download binary
set "DOWNLOAD_URL=https://github.com/%GITHUB_REPO%/releases/download/!VERSION!/orchcli_windows_%ARCH%.exe"
set "DEST_PATH=!INSTALL_DIR!\%BINARY_NAME%"

echo [INFO] Downloading OrchCLI from: !DOWNLOAD_URL!
curl -fsSL "!DOWNLOAD_URL!" -o "!DEST_PATH!"

if not exist "!DEST_PATH!" (
    echo [ERROR] Download failed - file not created
    exit /b 1
)

REM Validate download size
for %%F in ("!DEST_PATH!") do set "FILE_SIZE=%%~zF"
if !FILE_SIZE! LSS 1000 (
    echo [ERROR] Downloaded file is too small ^(!FILE_SIZE! bytes^) - possibly a 404 error page
    del "!DEST_PATH!" 2>nul
    exit /b 1
)

echo [INFO] Binary downloaded successfully (size: !FILE_SIZE! bytes)

REM Add to user PATH if not already present
echo !PATH! | findstr /I /C:"!INSTALL_DIR!" >nul 2>&1
if errorlevel 1 (
    echo [INFO] Adding !INSTALL_DIR! to user PATH...
    for /f "tokens=2*" %%a in ('reg query "HKCU\Environment" /v Path 2^>nul') do set "USER_PATH=%%b"
    if defined USER_PATH (
        reg add "HKCU\Environment" /v Path /t REG_EXPAND_SZ /d "!USER_PATH!;!INSTALL_DIR!" /f >nul 2>&1
    ) else (
        reg add "HKCU\Environment" /v Path /t REG_EXPAND_SZ /d "!INSTALL_DIR!" /f >nul 2>&1
    )
    set "PATH=!PATH!;!INSTALL_DIR!"
    echo [INFO] Added to PATH. Restart your terminal for changes to take effect in new sessions.
)

REM Verify installation
echo.
echo [INFO] OrchCLI installed successfully!
"!DEST_PATH!" --version 2>nul

echo.
echo ================================================
echo          Installation Complete!
echo ================================================
echo.
echo [INFO] Run 'orchcli --help' to get started

endlocal
