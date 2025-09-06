#!/bin/sh
set -e

# OrchCLI Installation Script
# Usage:
#   curl -sfL https://raw.githubusercontent.com/KubeOrch/cli/main/install.sh | sh
#   wget -qO- https://raw.githubusercontent.com/KubeOrch/cli/main/install.sh | sh
#
# Environment variables:
#   - ORCHCLI_INSTALL_DIR: Installation directory (default: /usr/local/bin)
#   - ORCHCLI_VERSION: Version to install (default: latest)
#   - ORCHCLI_NO_SUDO: Set to 1 to disable sudo usage

# Configuration
GITHUB_REPO="KubeOrch/cli"
BINARY_NAME="orchcli"
DEFAULT_INSTALL_DIR="/usr/local/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
info() {
    printf "${GREEN}[INFO]${NC} %s\n" "$1"
}

error() {
    printf "${RED}[ERROR]${NC} %s\n" "$1" >&2
}

warning() {
    printf "${YELLOW}[WARN]${NC} %s\n" "$1"
}

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case "$OS" in
        linux)
            PLATFORM="linux"
            ;;
        darwin)
            PLATFORM="darwin"
            ;;
        mingw*|msys*|cygwin*)
            PLATFORM="windows"
            ;;
        *)
            error "Unsupported operating system: $OS"
            exit 1
            ;;
    esac
    
    case "$ARCH" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        *)
            error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    
    if [ "$PLATFORM" = "windows" ]; then
        BINARY_SUFFIX="${PLATFORM}_${ARCH}.exe"
        BINARY_NAME="${BINARY_NAME}.exe"
    else
        BINARY_SUFFIX="${PLATFORM}_${ARCH}"
    fi
    
    info "Detected platform: $PLATFORM/$ARCH"
}

# Get the latest version from GitHub
get_latest_version() {
    if [ -n "$ORCHCLI_VERSION" ]; then
        VERSION="$ORCHCLI_VERSION"
        info "Using specified version: $VERSION"
    else
        info "Fetching latest version..."
        VERSION=$(curl -s "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        
        if [ -z "$VERSION" ]; then
            error "Failed to fetch latest version"
            exit 1
        fi
        info "Latest version: $VERSION"
    fi
}

# Download binary from GitHub releases
download_binary() {
    DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/orchcli_${BINARY_SUFFIX}"
    TEMP_DIR=$(mktemp -d)
    TEMP_BINARY="$TEMP_DIR/$BINARY_NAME"
    
    info "Downloading OrchCLI from: $DOWNLOAD_URL"
    
    if command -v curl >/dev/null 2>&1; then
        curl -L -o "$TEMP_BINARY" "$DOWNLOAD_URL" || {
            error "Failed to download binary"
            rm -rf "$TEMP_DIR"
            exit 1
        }
    elif command -v wget >/dev/null 2>&1; then
        wget -q -O "$TEMP_BINARY" "$DOWNLOAD_URL" || {
            error "Failed to download binary"
            rm -rf "$TEMP_DIR"
            exit 1
        }
    else
        error "Neither curl nor wget found. Please install one of them."
        rm -rf "$TEMP_DIR"
        exit 1
    fi
    
    # Check if download was successful and file is valid
    if [ ! -f "$TEMP_BINARY" ]; then
        error "Download failed - file not created"
        rm -rf "$TEMP_DIR"
        exit 1
    fi
    
    FILE_SIZE=$(stat -f%z "$TEMP_BINARY" 2>/dev/null || stat -c%s "$TEMP_BINARY" 2>/dev/null)
    if [ "$FILE_SIZE" -lt 1000 ]; then
        error "Downloaded file is too small ($FILE_SIZE bytes) - possibly a 404 error page"
        error "The release $VERSION may not have binaries uploaded yet"
        rm -rf "$TEMP_DIR"
        exit 1
    fi
    
    chmod +x "$TEMP_BINARY"
    info "Binary downloaded successfully (size: $FILE_SIZE bytes)"
}

# Install binary to system
install_binary() {
    INSTALL_DIR="${ORCHCLI_INSTALL_DIR:-$DEFAULT_INSTALL_DIR}"
    
    # Check if we need sudo
    if [ -w "$INSTALL_DIR" ] || [ "$ORCHCLI_NO_SUDO" = "1" ]; then
        SUDO=""
    else
        if command -v sudo >/dev/null 2>&1; then
            SUDO="sudo"
            info "Installing to $INSTALL_DIR (requires sudo)"
        else
            error "Cannot write to $INSTALL_DIR and sudo is not available"
            error "Try running as root or set ORCHCLI_INSTALL_DIR to a writable location"
            rm -rf "$TEMP_DIR"
            exit 1
        fi
    fi
    
    # Create install directory if it doesn't exist
    if [ ! -d "$INSTALL_DIR" ]; then
        info "Creating installation directory: $INSTALL_DIR"
        $SUDO mkdir -p "$INSTALL_DIR"
    fi
    
    # Move binary to installation directory
    info "Installing OrchCLI to $INSTALL_DIR"
    $SUDO mv "$TEMP_BINARY" "$INSTALL_DIR/$BINARY_NAME"
    $SUDO chmod +x "$INSTALL_DIR/$BINARY_NAME"
    
    # Clean up
    rm -rf "$TEMP_DIR"
    
    # Verify installation
    if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
        info "✅ OrchCLI installed successfully!"
        
        # Check if install dir is in PATH
        if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
            warning "$INSTALL_DIR is not in your PATH"
            warning "Add it to your PATH by running:"
            warning "  export PATH=\"\$PATH:$INSTALL_DIR\""
            warning "Or add this line to your shell profile (~/.bashrc, ~/.zshrc, etc.)"
        else
            # Test the installation
            if command -v orchcli >/dev/null 2>&1; then
                printf "\n"
                info "Installation verified. Version information:"
                orchcli --version
                printf "\n"
                info "Run 'orchcli --help' to get started"
            fi
        fi
    else
        error "Installation failed"
        exit 1
    fi
}

# Uninstall function
uninstall() {
    INSTALL_DIR="${ORCHCLI_INSTALL_DIR:-$DEFAULT_INSTALL_DIR}"
    BINARY_PATH="$INSTALL_DIR/$BINARY_NAME"
    
    if [ -f "$BINARY_PATH" ]; then
        info "Uninstalling OrchCLI from $BINARY_PATH"
        if [ -w "$INSTALL_DIR" ] || [ "$ORCHCLI_NO_SUDO" = "1" ]; then
            rm -f "$BINARY_PATH"
        else
            sudo rm -f "$BINARY_PATH"
        fi
        info "✅ OrchCLI uninstalled successfully"
    else
        warning "OrchCLI not found at $BINARY_PATH"
    fi
    exit 0
}

# Main installation flow
main() {
    printf "${BLUE}%s${NC}\n" "================================================"
    printf "${BLUE}%s${NC}\n" "     OrchCLI - KubeOrch Developer CLI     "
    printf "${BLUE}%s${NC}\n" "================================================"
    printf "\n"
    
    # Check for uninstall flag
    if [ "$1" = "--uninstall" ] || [ "$1" = "-u" ]; then
        uninstall
    fi
    
    # Detect platform
    detect_platform
    
    # Get version
    get_latest_version
    
    # Download binary
    download_binary
    
    # Install binary
    install_binary
    
    printf "${GREEN}%s${NC}\n" "================================================"
    printf "${GREEN}%s${NC}\n" "         Installation Complete! 🎉             "
    printf "${GREEN}%s${NC}\n" "================================================"
}

# Run main function
main "$@"