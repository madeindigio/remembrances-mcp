#!/bin/bash
# Build script for embedded SurrealDB Rust library across multiple platforms

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUST_LIB_DIR="${SCRIPT_DIR}/vendor/surrealdb-embedded-rust"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Rust is installed
if ! command -v cargo &> /dev/null; then
    log_error "Rust/Cargo not found. Please install Rust from https://rustup.rs/"
    exit 1
fi

# Clone or update the embedded library
if [ ! -d "${RUST_LIB_DIR}" ]; then
    log_info "Cloning surrealdb-embedded-golang..."
    mkdir -p vendor
    git clone https://github.com/madeindigio/surrealdb-embedded-golang.git "${RUST_LIB_DIR}"
fi

cd "${RUST_LIB_DIR}/surrealdb_embedded_rs" || exit 1

# Determine target platform
TARGET="${1:-native}"

case "${TARGET}" in
    linux-amd64|x86_64-linux)
        log_info "Building for Linux x86_64..."
        RUST_TARGET="x86_64-unknown-linux-gnu"
        export CC=x86_64-linux-gnu-gcc
        export CXX=x86_64-linux-gnu-g++
        ;;
    linux-arm64|aarch64-linux)
        log_info "Building for Linux ARM64..."
        RUST_TARGET="aarch64-unknown-linux-gnu"
        export CC=aarch64-linux-gnu-gcc
        export CXX=aarch64-linux-gnu-g++
        ;;
    darwin-amd64|x86_64-darwin)
        log_info "Building for macOS x86_64..."
        RUST_TARGET="x86_64-apple-darwin"
        if command -v o64-clang &> /dev/null; then
            export CC=o64-clang
            export CXX=o64-clang++
        else
            log_warn "OSXCross not found, attempting native build..."
        fi
        ;;
    darwin-arm64|aarch64-darwin)
        log_info "Building for macOS ARM64..."
        RUST_TARGET="aarch64-apple-darwin"
        if command -v oa64-clang &> /dev/null; then
            export CC=oa64-clang
            export CXX=oa64-clang++
        else
            log_warn "OSXCross not found, attempting native build..."
        fi
        ;;
    native|"")
        log_info "Building for native platform..."
        RUST_TARGET=""
        ;;
    *)
        log_error "Unknown target: ${TARGET}"
        echo "Usage: $0 [linux-amd64|linux-arm64|darwin-amd64|darwin-arm64|native]"
        exit 1
        ;;
esac

# Add target if not native
if [ -n "${RUST_TARGET}" ]; then
    log_info "Adding Rust target: ${RUST_TARGET}"
    rustup target add "${RUST_TARGET}" || true

    log_info "Building Rust library for ${RUST_TARGET}..."
    cargo build --release --target "${RUST_TARGET}"

    # Copy built library to a standard location
    RUST_LIB_NAME="libsurrealdb_embedded_rs.a"
    TARGET_DIR="target/${RUST_TARGET}/release"
else
    log_info "Building Rust library for native platform..."
    cargo build --release
    TARGET_DIR="target/release"
fi

log_info "Build complete! Library location: ${TARGET_DIR}/"
log_info "To use with Go, ensure CGO_ENABLED=1 and:"
log_info "  export CGO_LDFLAGS=\"-L${RUST_LIB_DIR}/surrealdb_embedded_rs/${TARGET_DIR}\""
log_info "  export CGO_CFLAGS=\"-I${RUST_LIB_DIR}/surrealdb_embedded_rs/include\""
