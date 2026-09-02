#!/bin/sh
set -e

# OrchCLI Installation Script
# Usage:
#   curl -sfL https://kubeorch.dev/install.sh | sh
#   wget -qO- https://kubeorch.dev/install.sh | sh
#
# Environment variables:
#   - ORCHCLI_INSTALL_DIR: Installation directory (default: /usr/local/bin)
#   - ORCHCLI_VERSION: Version to install (default: latest)
#   - ORCHCLI_NO_SUDO: Set to 1 to disable sudo usage

# Configuration
GITHUB_REPO="KubeOrch/cli"
BINARY_NAME="orchcli"
DEFAULT_INSTALL_DIR="/usr/local/bin"
TEMP_DIR=""

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

cleanup_temp() {
    if [ -n "${TEMP_DIR:-}" ] && [ -d "$TEMP_DIR" ]; then
        rm -f "$TEMP_DIR/$BINARY_NAME" "$TEMP_DIR/checksums.txt"
        rmdir "$TEMP_DIR" 2>/dev/null || true
    fi
}

trap cleanup_temp EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

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
        if command -v curl >/dev/null 2>&1; then
            RELEASE_JSON=$(curl -fsSL --retry 3 \
                "https://api.github.com/repos/${GITHUB_REPO}/releases/latest")
        elif command -v wget >/dev/null 2>&1; then
            RELEASE_JSON=$(wget -q -O - \
                "https://api.github.com/repos/${GITHUB_REPO}/releases/latest")
        else
            error "Neither curl nor wget found. Please install one of them."
            exit 1
        fi
        VERSION=$(printf '%s\n' "$RELEASE_JSON" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

        if [ -z "$VERSION" ]; then
            error "Failed to fetch latest version"
            exit 1
        fi
        info "Latest version: $VERSION"
    fi

    case "$VERSION" in
        v*) ;;
        *) VERSION="v$VERSION" ;;
    esac
}

download_file() {
    source_url="$1"
    destination="$2"

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL --retry 3 --output "$destination" "$source_url"
    elif command -v wget >/dev/null 2>&1; then
        wget -q -O "$destination" "$source_url"
    else
        error "Neither curl nor wget found. Please install one of them."
        exit 1
    fi
}

sha256_file() {
    target="$1"
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$target" | awk '{print $1}'
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$target" | awk '{print $1}'
    elif command -v openssl >/dev/null 2>&1; then
        openssl dgst -sha256 "$target" | awk '{print $NF}'
    else
        error "A SHA256 tool is required (sha256sum, shasum, or openssl)."
        exit 1
    fi
}

# Download and verify the binary from GitHub Releases.
download_binary() {
    BINARY_ASSET="orchcli_${BINARY_SUFFIX}"
    RELEASE_URL="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}"
    DOWNLOAD_URL="${RELEASE_URL}/${BINARY_ASSET}"
    TEMP_DIR=$(mktemp -d)
    TEMP_BINARY="$TEMP_DIR/$BINARY_NAME"
    TEMP_CHECKSUMS="$TEMP_DIR/checksums.txt"

    info "Downloading OrchCLI from: $DOWNLOAD_URL"
    download_file "${RELEASE_URL}/checksums.txt" "$TEMP_CHECKSUMS" || {
        error "Failed to download checksums.txt for $VERSION"
        exit 1
    }
    download_file "$DOWNLOAD_URL" "$TEMP_BINARY" || {
        error "Failed to download $BINARY_ASSET"
        exit 1
    }
    
    # Check if download was successful and file is valid
    if [ ! -f "$TEMP_BINARY" ]; then
        error "Download failed - file not created"
        exit 1
    fi
    
    FILE_SIZE=$(stat -f%z "$TEMP_BINARY" 2>/dev/null || stat -c%s "$TEMP_BINARY" 2>/dev/null)
    if [ "$FILE_SIZE" -lt 1000 ]; then
        error "Downloaded file is too small ($FILE_SIZE bytes) - possibly a 404 error page"
        error "The release $VERSION may not have binaries uploaded yet"
        exit 1
    fi

    EXPECTED_CHECKSUM=$(awk -v asset="$BINARY_ASSET" \
        '$2 == asset || $2 == "*" asset { print tolower($1); exit }' \
        "$TEMP_CHECKSUMS")
    if [ -z "$EXPECTED_CHECKSUM" ]; then
        error "checksums.txt does not contain $BINARY_ASSET"
        exit 1
    fi
    ACTUAL_CHECKSUM=$(sha256_file "$TEMP_BINARY" | tr '[:upper:]' '[:lower:]')
    if [ "$ACTUAL_CHECKSUM" != "$EXPECTED_CHECKSUM" ]; then
        error "Checksum mismatch for $BINARY_ASSET"
        error "Expected: $EXPECTED_CHECKSUM"
        error "Received: $ACTUAL_CHECKSUM"
        exit 1
    fi

    chmod +x "$TEMP_BINARY"
    info "Verified SHA256 checksum: $ACTUAL_CHECKSUM"
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
    
    # Clean up the checksum file and now-empty temporary directory.
    cleanup_temp
    
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
