#!/bin/bash
set -e

# Opun installer script
# Usage: curl -sSL https://raw.githubusercontent.com/rizome-dev/opun/main/install.sh | bash

REPO="rizome-dev/opun"
BINARY_NAME="opun"
INSTALL_DIR="/usr/local/bin"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
error() {
    echo -e "${RED}Error: $1${NC}" >&2
    exit 1
}

success() {
    echo -e "${GREEN}$1${NC}"
}

info() {
    echo -e "${YELLOW}$1${NC}"
}

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s)
    ARCH=$(uname -m)
    
    # Map OS to GoReleaser format
    case $OS in
        Darwin) OS_NAME="Darwin" ;;
        Linux) OS_NAME="Linux" ;;
        *) error "Unsupported operating system: $OS" ;;
    esac
    
    # Map architecture to GoReleaser format
    case $ARCH in
        x86_64) ARCH_NAME="x86_64" ;;
        aarch64|arm64) ARCH_NAME="arm64" ;;
        *) error "Unsupported architecture: $ARCH" ;;
    esac
    
    echo "${OS_NAME}_${ARCH_NAME}"
}

# Get latest release version
get_latest_version() {
    curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
}

# Download and install
install_opun() {
    PLATFORM=$(detect_platform)
    VERSION=$(get_latest_version)
    
    if [ -z "$VERSION" ]; then
        error "Could not determine latest version"
    fi
    
    info "Installing Opun ${VERSION} for ${PLATFORM}..."
    
    # Construct download URL
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}_${PLATFORM}.tar.gz"
    
    # Create temp directory
    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT
    
    # Download
    info "Downloading from ${DOWNLOAD_URL}..."
    curl -sL "$DOWNLOAD_URL" -o "$TMP_DIR/${BINARY_NAME}.tar.gz" || error "Failed to download"
    
    # Extract
    info "Extracting..."
    tar -xzf "$TMP_DIR/${BINARY_NAME}.tar.gz" -C "$TMP_DIR" || error "Failed to extract"
    
    # Install
    info "Installing to ${INSTALL_DIR}..."
    if [ -w "$INSTALL_DIR" ]; then
        mv "$TMP_DIR/${BINARY_NAME}" "$INSTALL_DIR/" || error "Failed to install"
    else
        sudo mv "$TMP_DIR/${BINARY_NAME}" "$INSTALL_DIR/" || error "Failed to install (sudo required)"
    fi
    
    # Make executable
    chmod +x "$INSTALL_DIR/${BINARY_NAME}"
    
    # Create directory structure with proper ownership
    info "Creating Opun directory structure..."
    ACTUAL_USER=$([ -n "$SUDO_USER" ] && echo "$SUDO_USER" || whoami)
    ACTUAL_HOME=$([ -n "$SUDO_USER" ] && eval echo ~$SUDO_USER || echo ~)
    
    # Create all required directories
    mkdir -p "$ACTUAL_HOME/.opun/workflows" \
             "$ACTUAL_HOME/.opun/promptgarden" \
             "$ACTUAL_HOME/.opun/sessions" \
             "$ACTUAL_HOME/.opun/mcp" \
             "$ACTUAL_HOME/.opun/workspace"
    
    # Fix ownership if running with sudo
    if [ -n "$SUDO_USER" ]; then
        chown -R "$ACTUAL_USER:$(id -gn $ACTUAL_USER)" "$ACTUAL_HOME/.opun"
    fi
    
    # Verify installation
    if command -v $BINARY_NAME >/dev/null 2>&1; then
        success "✓ Opun ${VERSION} installed successfully!"
        success "✓ Created ~/.opun directory structure with correct ownership"
        echo ""
        info "Next steps:"
        echo "  1. Run 'opun setup' to configure Opun"
        echo "  2. Run 'opun --help' to see available commands"
    else
        error "Installation failed - opun command not found"
    fi
}

# Main
main() {
    echo "Opun Installer"
    echo "=============="
    echo ""
    
    # Check for curl
    if ! command -v curl >/dev/null 2>&1; then
        error "curl is required but not installed"
    fi
    
    # Check for tar
    if ! command -v tar >/dev/null 2>&1; then
        error "tar is required but not installed"
    fi
    
    # Install
    install_opun
}

main "$@"