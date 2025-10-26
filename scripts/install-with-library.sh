#!/bin/bash
# Remembrances-MCP Installation Script with Native Library Support
# This script installs both the Go binary and the native kelindar/search library

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
INSTALL_DIR="${INSTALL_DIR:-/usr/local}"
BIN_DIR="${INSTALL_DIR}/bin"
LIB_DIR="${INSTALL_DIR}/lib"
DOWNLOAD_URL_BASE="https://github.com/kelindar/search/releases/latest/download"

# Detect OS and architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

echo -e "${GREEN}Remembrances-MCP Installation Script${NC}"
echo "========================================"
echo ""

# Function to print error and exit
error_exit() {
    echo -e "${RED}ERROR: $1${NC}" >&2
    exit 1
}

# Function to print warning
warn() {
    echo -e "${YELLOW}WARNING: $1${NC}"
}

# Function to print success
success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# Check if running with sudo/root when needed
check_permissions() {
    if [ "$INSTALL_DIR" = "/usr/local" ] || [ "$INSTALL_DIR" = "/usr" ]; then
        if [ "$EUID" -ne 0 ]; then
            error_exit "This script requires root privileges. Please run with sudo."
        fi
    fi
}

# Determine library name based on OS
get_library_name() {
    case "$OS" in
        Linux*)
            echo "libllama_go.so"
            ;;
        Darwin*)
            echo "libllama_go.dylib"
            ;;
        CYGWIN*|MINGW*|MSYS*)
            echo "llama_go.dll"
            ;;
        *)
            error_exit "Unsupported operating system: $OS"
            ;;
    esac
}

# Check if file exists in current directory
check_file_exists() {
    local file="$1"
    if [ ! -f "$file" ]; then
        return 1
    fi
    return 0
}

# Install binary
install_binary() {
    local binary_name="remembrances-mcp"

    echo "Installing binary..."

    if ! check_file_exists "$binary_name"; then
        error_exit "Binary '$binary_name' not found in current directory"
    fi

    # Create bin directory if it doesn't exist
    mkdir -p "$BIN_DIR"

    # Copy and set permissions
    cp "$binary_name" "$BIN_DIR/"
    chmod +x "$BIN_DIR/$binary_name"

    success "Binary installed to $BIN_DIR/$binary_name"
}

# Install native library
install_library() {
    local lib_name="$(get_library_name)"
    local found=false

    echo "Installing native library..."

    # Check current directory
    if check_file_exists "$lib_name"; then
        found=true
    # Check dist directory (if building from source)
    elif check_file_exists "dist/$lib_name"; then
        lib_name="dist/$lib_name"
        found=true
    fi

    if [ "$found" = false ]; then
        warn "Native library '$lib_name' not found"
        echo "The library can be obtained from:"
        echo "  1. Build from source: https://github.com/kelindar/search"
        echo "  2. Download pre-built: $DOWNLOAD_URL_BASE/$lib_name"
        echo ""
        read -p "Do you want to continue without the library? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            error_exit "Installation cancelled"
        fi
        warn "Skipping library installation. You'll need to install it manually."
        return
    fi

    # Create lib directory if it doesn't exist
    mkdir -p "$LIB_DIR"

    # Copy library
    cp "$lib_name" "$LIB_DIR/$(basename $lib_name)"
    chmod 755 "$LIB_DIR/$(basename $lib_name)"

    # Update library cache on Linux
    if [ "$OS" = "Linux" ]; then
        if command -v ldconfig &> /dev/null; then
            ldconfig
            success "Library cache updated"
        fi
    fi

    success "Native library installed to $LIB_DIR/$(basename $lib_name)"
}

# Verify installation
verify_installation() {
    echo ""
    echo "Verifying installation..."

    # Check binary
    if [ -x "$BIN_DIR/remembrances-mcp" ]; then
        success "Binary is executable"
    else
        error_exit "Binary verification failed"
    fi

    # Check library
    local lib_name="$(get_library_name)"
    if [ -f "$LIB_DIR/$lib_name" ]; then
        success "Native library is present"

        # On Linux, check if library is in cache
        if [ "$OS" = "Linux" ] && command -v ldconfig &> /dev/null; then
            if ldconfig -p | grep -q "llama_go"; then
                success "Library is in system cache"
            else
                warn "Library not found in system cache. You may need to run 'sudo ldconfig'"
            fi
        fi
    else
        warn "Native library not found. Application may not work correctly."
    fi

    # Try to run version command
    echo ""
    echo "Testing installation..."
    if "$BIN_DIR/remembrances-mcp" --version 2>/dev/null; then
        success "Application runs successfully"
    else
        warn "Could not run application. Check library installation."
    fi
}

# Print post-installation instructions
print_instructions() {
    echo ""
    echo "========================================"
    echo -e "${GREEN}Installation Complete!${NC}"
    echo "========================================"
    echo ""
    echo "Binary location: $BIN_DIR/remembrances-mcp"
    echo "Library location: $LIB_DIR/$(get_library_name)"
    echo ""
    echo "Next steps:"
    echo "  1. Download a BERT-based GGUF model (e.g., nomic-embed-text-v1.5)"
    echo "  2. Set environment variables:"
    echo "     export GOMEM_SEARCH_MODEL_PATH=/path/to/model.gguf"
    echo "     export GOMEM_DB_PATH=/path/to/database"
    echo "  3. Run: remembrances-mcp --help"
    echo ""

    if [ "$INSTALL_DIR" = "/usr/local" ]; then
        echo "Note: $BIN_DIR should be in your PATH"
    else
        echo "Note: You may need to add $BIN_DIR to your PATH:"
        echo "  export PATH=\"$BIN_DIR:\$PATH\""
    fi
    echo ""
}

# Main installation flow
main() {
    echo "Installation target: $INSTALL_DIR"
    echo "Operating system: $OS"
    echo "Architecture: $ARCH"
    echo ""

    # Check permissions
    check_permissions

    # Install components
    install_binary
    install_library

    # Verify
    verify_installation

    # Instructions
    print_instructions
}

# Handle command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --prefix)
            INSTALL_DIR="$2"
            BIN_DIR="${INSTALL_DIR}/bin"
            LIB_DIR="${INSTALL_DIR}/lib"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --prefix DIR    Install to DIR instead of /usr/local"
            echo "  --help, -h      Show this help message"
            echo ""
            echo "Environment variables:"
            echo "  INSTALL_DIR     Installation prefix (default: /usr/local)"
            echo ""
            echo "Example:"
            echo "  sudo $0"
            echo "  sudo $0 --prefix /opt/remembrances-mcp"
            exit 0
            ;;
        *)
            error_exit "Unknown option: $1. Use --help for usage information."
            ;;
    esac
done

# Run installation
main
