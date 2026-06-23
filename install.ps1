#!/usr/bin/env pwsh
# OrchCLI Installation Script for Windows PowerShell
# Usage:
#   irm https://kubeorch.dev/install.ps1 | iex
#
# Environment variables:
#   - ORCHCLI_INSTALL_DIR: Installation directory (default: %LOCALAPPDATA%\Programs\orchcli)
#   - ORCHCLI_VERSION: Version to install (default: latest)

$ErrorActionPreference = "Stop"

# Configuration
$GithubRepo = "KubeOrch/cli"
$BinaryName = "orchcli.exe"

function Write-Info { param([string]$Message) Write-Host "[INFO] $Message" -ForegroundColor Green }
function Write-Warn { param([string]$Message) Write-Host "[WARN] $Message" -ForegroundColor Yellow }
function Write-Err { param([string]$Message) Write-Host "[ERROR] $Message" -ForegroundColor Red }

function Get-Architecture {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
    switch ($arch) {
        "X64"   { return "amd64" }
        "Arm64" { return "arm64" }
        default {
            Write-Err "Unsupported architecture: $arch"
            exit 1
        }
    }
}

function Get-LatestVersion {
    if ($env:ORCHCLI_VERSION) {
        Write-Info "Using specified version: $env:ORCHCLI_VERSION"
        return $env:ORCHCLI_VERSION
    }

    Write-Info "Fetching latest version..."
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$GithubRepo/releases/latest" -Headers @{ "User-Agent" = "orchcli-installer" }
        $version = $release.tag_name
        Write-Info "Latest version: $version"
        return $version
    }
    catch {
        Write-Err "Failed to fetch latest version: $_"
        exit 1
    }
}

function Install-OrchCLI {
    Write-Host ""
    Write-Host "================================================" -ForegroundColor Cyan
    Write-Host "     OrchCLI - KubeOrch Developer CLI           " -ForegroundColor Cyan
    Write-Host "================================================" -ForegroundColor Cyan
    Write-Host ""

    $arch = Get-Architecture
    $version = Get-LatestVersion
    $binarySuffix = "windows_${arch}.exe"

    # Determine install directory
    $installDir = if ($env:ORCHCLI_INSTALL_DIR) {
        $env:ORCHCLI_INSTALL_DIR
    } else {
        Join-Path $env:LOCALAPPDATA "Programs\orchcli"
    }

    # Create install directory
    if (-not (Test-Path $installDir)) {
        Write-Info "Creating installation directory: $installDir"
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    }

    # Download binary
    $downloadUrl = "https://github.com/$GithubRepo/releases/download/$version/orchcli_$binarySuffix"
    $destPath = Join-Path $installDir $BinaryName

    Write-Info "Downloading OrchCLI from: $downloadUrl"
    try {
        $ProgressPreference = 'SilentlyContinue'
        Invoke-WebRequest -Uri $downloadUrl -OutFile $destPath -UseBasicParsing
        $ProgressPreference = 'Continue'
    }
    catch {
        Write-Err "Failed to download binary: $_"
        exit 1
    }

    # Validate download
    $fileSize = (Get-Item $destPath).Length
    if ($fileSize -lt 1000) {
        Write-Err "Downloaded file is too small ($fileSize bytes) - possibly a 404 error page"
        Write-Err "The release $version may not have binaries uploaded yet"
        Remove-Item $destPath -Force
        exit 1
    }

    Write-Info "Binary downloaded successfully (size: $fileSize bytes)"

    # Add to PATH if not already present
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$installDir*") {
        Write-Info "Adding $installDir to user PATH..."
        [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
        $env:Path = "$env:Path;$installDir"
        Write-Info "Added to PATH. Restart your terminal for changes to take effect in new sessions."
    }

    # Verify installation
    Write-Host ""
    Write-Info "OrchCLI installed successfully!"
    try {
        & $destPath --version
    }
    catch {
        Write-Warn "Installed but could not verify version. Try restarting your terminal."
    }

    Write-Host ""
    Write-Host "================================================" -ForegroundColor Green
    Write-Host "         Installation Complete!                  " -ForegroundColor Green
    Write-Host "================================================" -ForegroundColor Green
    Write-Host ""
    Write-Info "Run 'orchcli --help' to get started"
}

# Run
Install-OrchCLI
