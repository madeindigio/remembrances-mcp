#!/bin/bash
# Download Native Library for kelindar/search
# This script automatically downloads the correct native library for your platform

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
GITHUB_REPO="kelindar/search"
DOWNLOAD_DIR="${DOWNLOAD_DIR:-.}"
VERSION="${VERSION:-latest}"

echo -e "${BLUE}Native Library Downloader for kelindar/search${NC}"
echo "=================================================="
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

# Function to print info
info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

# Detect OS and architecture
detect_platform() {
    OS="$(uname -s)"
    ARCH="$(uname -m)"

    case "$OS" in
        Linux*)
            PLATFORM_OS="linux"
            LIB_NAME="libllama_go.so"
            ;;
        Darwin*)
            PLATFORM_OS="darwin"
            LIB_NAME="libllama_go.dylib"
            ;;
        CYGWIN*|MINGW*|MSYS*)
            PLATFORM_OS="windows"
            LIB_NAME="llama_go.dll"
            ;;
        *)
            error_exit "Unsupported operating system: $OS"
            ;;
    esac

    case "$ARCH" in
        x86_64|amd64)
            PLATFORM_ARCH="amd64"
            ;;
        arm64|aarch64)
            PLATFORM_ARCH="arm64"
            ;;
        *)
            error_exit "Unsupported architecture: $ARCH"
            ;;
    esac

    info "Detected platform: $PLATFORM_OS/$PLATFORM_ARCH"
    info "Library name: $LIB_NAME"
}

# Get latest release version from GitHub
get_latest_version() {
    if [ "$VERSION" = "latest" ]; then
        info "Fetching latest release information..."

        if command -v curl &> /dev/null; then
            LATEST_TAG=$(curl -s "https://api.github.com/repos/$GITHUB_REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        elif command -v wget &> /dev/null; then
            LATEST_TAG=$(wget -qO- "https://api.github.com/repos/$GITHUB_REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        else
            error_exit "Neither curl nor wget found. Please install one of them."
        fi

        if [ -z "$LATEST_TAG" ]; then
            error_exit "Could not determine latest version"
        fi

        VERSION="$LATEST_TAG"
        success "Latest version: $VERSION"
    else
        info "Using specified version: $VERSION"
    fi
}

# Construct download URL
construct_url() {
    # Try different naming conventions that might be used
    # Pattern 1: libllama_go-{os}-{arch}.{ext}
    # Pattern 2: libllama_go.{ext} (generic)
    # Pattern 3: {os}-{arch}/libllama_go.{ext}

    if [ "$VERSION" = "latest" ]; then
        BASE_URL="https://github.com/$GITHUB_REPO/releases/latest/download"
    else
        BASE_URL="https://github.com/$GITHUB_REPO/releases/download/$VERSION"
    fi

    # Try platform-specific naming first
    DOWNLOAD_URL="$BASE_URL/${LIB_NAME%.*}-${PLATFORM_OS}-${PLATFORM_ARCH}.${LIB_NAME##*.}"

    info "Download URL: $DOWNLOAD_URL"
}

# Check if file already exists
check_existing() {
    if [ -f "$DOWNLOAD_DIR/$LIB_NAME" ]; then
        warn "Library already exists: $DOWNLOAD_DIR/$LIB_NAME"
        read -p "Do you want to overwrite it? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            info "Download cancelled"
            exit 0
        fi
    fi
}

# Download the library
download_library() {
    info "Downloading native library..."

    TMP_FILE="$DOWNLOAD_DIR/${LIB_NAME}.tmp"

    if command -v curl &> /dev/null; then
        if curl -L -f -o "$TMP_FILE" "$DOWNLOAD_URL" 2>/dev/null; then
            mv "$TMP_FILE" "$DOWNLOAD_DIR/$LIB_NAME"
            chmod 755 "$DOWNLOAD_DIR/$LIB_NAME"
            success "Library downloaded successfully"
            return 0
        fi
    elif command -v wget &> /dev/null; then
        if wget -q -O "$TMP_FILE" "$DOWNLOAD_URL" 2>/dev/null; then
            mv "$TMP_FILE" "$DOWNLOAD_DIR/$LIB_NAME"
            chmod 755 "$DOWNLOAD_DIR/$LIB_NAME"
            success "Library downloaded successfully"
            return 0
        fi
    fi

    # Clean up tmp file if download failed
    rm -f "$TMP_FILE"

    # Try generic URL (without platform suffix)
    info "Platform-specific build not found, trying generic build..."
    DOWNLOAD_URL="$BASE_URL/$LIB_NAME"

    if command -v curl &> /dev/null; then
        if curl -L -f -o "$TMP_FILE" "$DOWNLOAD_URL" 2>/dev/null; then
            mv "$TMP_FILE" "$DOWNLOAD_DIR/$LIB_NAME"
            chmod 755 "$DOWNLOAD_DIR/$LIB_NAME"
            success "Library downloaded successfully"
            return 0
        fi
    elif command -v wget &> /dev/null; then
        if wget -q -O "$TMP_FILE" "$DOWNLOAD_URL" 2>/dev/null; then
            mv "$TMP_FILE" "$DOWNLOAD_DIR/$LIB_NAME"
            chmod 755 "$DOWNLOAD_DIR/$LIB_NAME"
            success "Library downloaded successfully"
            return 0
        fi
    fi

    rm -f "$TMP_FILE"
    error_exit "Failed to download library. Please check:\n  1. Release exists: https://github.com/$GITHUB_REPO/releases\n  2. Platform-specific build is available for $PLATFORM_OS/$PLATFORM_ARCH"
}

# Verify downloaded library
verify_library() {
    if [ ! -f "$DOWNLOAD_DIR/$LIB_NAME" ]; then
        error_exit "Library file not found after download"
    fi

    FILE_SIZE=$(stat -f%z "$DOWNLOAD_DIR/$LIB_NAME" 2>/dev/null || stat -c%s "$DOWNLOAD_DIR/$LIB_NAME" 2>/dev/null || echo "unknown")

    if [ "$FILE_SIZE" = "unknown" ] || [ "$FILE_SIZE" -lt 1000 ]; then
        error_exit "Downloaded file appears to be invalid (size: $FILE_SIZE bytes)"
    fi

    success "Library verified (size: $FILE_SIZE bytes)"

    # Check file type
    if command -v file &> /dev/null; then
        FILE_TYPE=$(file "$DOWNLOAD_DIR/$LIB_NAME")
        info "File type: $FILE_TYPE"
    fi
}

# Print post-download instructions
print_instructions() {
    echo ""
    echo "=================================================="
    echo -e "${GREEN}Download Complete!${NC}"
    echo "=================================================="
    echo ""
    echo "Library location: $DOWNLOAD_DIR/$LIB_NAME"
    echo ""
    echo "Installation options:"
    echo ""
    echo "1. System-wide installation (recommended):"
    case "$PLATFORM_OS" in
        linux)
            echo "   sudo cp $LIB_NAME /usr/local/lib/"
            echo "   sudo ldconfig"
            ;;
        darwin)
            echo "   sudo cp $LIB_NAME /usr/local/lib/"
            ;;
        windows)
            echo "   Copy $LIB_NAME to C:\\Windows\\System32\\"
            echo "   Or add its directory to PATH"
            ;;
    esac
    echo ""
    echo "2. Local directory (for testing):"
    case "$PLATFORM_OS" in
        linux)
            echo "   export LD_LIBRARY_PATH=.:$LD_LIBRARY_PATH"
            ;;
        darwin)
            echo "   export DYLD_LIBRARY_PATH=.:$DYLD_LIBRARY_PATH"
            ;;
        windows)
            echo "   Set PATH=%CD%;%PATH%"
            ;;
    esac
    echo ""
    echo "3. Use installation script:"
    echo "   sudo ./scripts/install-with-library.sh"
    echo ""
}

# Main function
main() {
    detect_platform
    get_latest_version
    construct_url
    check_existing
    download_library
    verify_library
    print_instructions
}

# Handle command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --dir|-d)
            DOWNLOAD_DIR="$2"
            shift 2
            ;;
        --version|-v)
            VERSION="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Download native library for kelindar/search"
            echo ""
            echo "Options:"
            echo "  --dir, -d DIR       Download to DIR (default: current directory)"
            echo "  --version, -v VER   Download specific version (default: latest)"
            echo "  --help, -h          Show this help message"
            echo ""
            echo "Environment variables:"
            echo "  DOWNLOAD_DIR        Download directory"
            echo "  VERSION             Version to download"
            echo ""
            echo "Examples:"
            echo "  $0                          # Download latest to current directory"
            echo "  $0 --dir /tmp               # Download to /tmp"
            echo "  $0 --version v0.4.0         # Download specific version"
            echo ""
            exit 0
            ;;
        *)
            error_exit "Unknown option: $1. Use --help for usage information."
            ;;
    esac
done

# Run main function
main
