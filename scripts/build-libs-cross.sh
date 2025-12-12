#!/bin/bash
# Script to cross-compile shared libraries for different platforms
# This script should be run inside the goreleaser-cross Docker container

set -e

PROJECT_ROOT="${PROJECT_ROOT:-/go/src/github.com/madeindigio/remembrances-mcp}"
LLAMA_CPP_DIR="${LLAMA_CPP_DIR:-$HOME/www/MCP/Remembrances/go-llama.cpp}"
SURREALDB_DIR="${SURREALDB_DIR:-$HOME/www/MCP/Remembrances/surrealdb-embedded}"
DIST_LIBS_DIR="${PROJECT_ROOT}/dist/libs"

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

# Function to build llama.cpp for a specific platform
build_llama_cpp() {
    local platform=$1
    local arch=$2
    local cc=$3
    local cxx=$4

    log_info "Building llama.cpp for ${platform}-${arch}..."

    local output_dir="${DIST_LIBS_DIR}/${platform}-${arch}"
    mkdir -p "${output_dir}"

    # Check if llama.cpp source exists
    if [ ! -d "${LLAMA_CPP_DIR}/llama.cpp" ]; then
        log_error "llama.cpp source not found at ${LLAMA_CPP_DIR}/llama.cpp"
        log_error "Please run: cd ${LLAMA_CPP_DIR} && git submodule update --init --recursive"
        return 1
    fi

    # Create a build directory for this platform
    local build_dir="${LLAMA_CPP_DIR}/build-${platform}-${arch}"
    mkdir -p "${build_dir}"

    # Configure CMake for cross-compilation
    cd "${build_dir}"

    # Platform-specific CMake flags
    local cmake_flags="-DLLAMA_STATIC=OFF -DBUILD_SHARED_LIBS=ON -DLLAMA_CURL=OFF"

    case "${platform}" in
        darwin)
            cmake_flags="${cmake_flags} -DCMAKE_SYSTEM_NAME=Darwin"
            if [ "${arch}" = "arm64" ]; then
                cmake_flags="${cmake_flags} -DCMAKE_OSX_ARCHITECTURES=arm64"
                cmake_flags="${cmake_flags} -DLLAMA_METAL=ON"
            else
                cmake_flags="${cmake_flags} -DCMAKE_OSX_ARCHITECTURES=x86_64"
            fi
            ;;
        linux)
            cmake_flags="${cmake_flags} -DCMAKE_SYSTEM_NAME=Linux"
            if [ "${arch}" = "arm64" ]; then
                cmake_flags="${cmake_flags} -DCMAKE_SYSTEM_PROCESSOR=aarch64"
            else
                cmake_flags="${cmake_flags} -DCMAKE_SYSTEM_PROCESSOR=x86_64"
            fi
            ;;
        windows)
            cmake_flags="${cmake_flags} -DCMAKE_SYSTEM_NAME=Windows"
            if [ "${arch}" = "arm64" ]; then
                cmake_flags="${cmake_flags} -DCMAKE_SYSTEM_PROCESSOR=ARM64"
            else
                cmake_flags="${cmake_flags} -DCMAKE_SYSTEM_PROCESSOR=AMD64"
            fi
            # Windows-specific flags for DLL
            cmake_flags="${cmake_flags} -DCMAKE_SHARED_LIBRARY_PREFIX=''"
            ;;
    esac

    # Run CMake
    CC="${cc}" CXX="${cxx}" cmake "${LLAMA_CPP_DIR}/llama.cpp" \
        ${cmake_flags} \
        -DCMAKE_C_COMPILER="${cc}" \
        -DCMAKE_CXX_COMPILER="${cxx}" \
        -DCMAKE_BUILD_TYPE=Release

    # Build
    cmake --build . --config Release -j$(nproc)

    # Copy libraries to output directory
    log_info "Copying llama.cpp libraries to ${output_dir}..."

    case "${platform}" in
        darwin)
            find . -name "*.dylib" -exec cp {} "${output_dir}/" \;
            ;;
        windows)
            find . -name "*.dll" -exec cp {} "${output_dir}/" \;
            ;;
        *)
            find . -name "*.so*" -exec cp {} "${output_dir}/" \;
            ;;
    esac

    log_info "llama.cpp build complete for ${platform}-${arch}"
}

build_llama_shim() {
    local platform=$1
    local arch=$2
    local cc=$3

    local output_dir="${DIST_LIBS_DIR}/${platform}-${arch}"
    local shim_src="${PROJECT_ROOT}/internal/llama_shim/llama_shim.c"
    local shim_inc="${PROJECT_ROOT}/internal/llama_shim"

    log_info "Building libllama_shim for ${platform}-${arch}..."

    if [ ! -f "${shim_src}" ]; then
        log_error "llama shim source not found at ${shim_src}"
        return 1
    fi

    case "${platform}" in
        linux)
            "${cc}" -shared -fPIC -O3 \
                -I"${shim_inc}" \
                -L"${output_dir}" -lllama \
                -Wl,-rpath,'$ORIGIN' \
                -o "${output_dir}/libllama_shim.so" \
                "${shim_src}" -lm
            ;;
        darwin)
            "${cc}" -dynamiclib -O3 \
                -I"${shim_inc}" \
                -L"${output_dir}" -lllama \
                -Wl,-rpath,@loader_path \
                -Wl,-install_name,@rpath/libllama_shim.dylib \
                -o "${output_dir}/libllama_shim.dylib" \
                "${shim_src}" -lm
            ;;
        windows)
            log_warn "Skipping libllama_shim for Windows (not currently supported)"
            ;;
        *)
            log_warn "Unknown platform ${platform}; skipping libllama_shim"
            ;;
    esac

    log_info "libllama_shim build complete for ${platform}-${arch}"
}

# Function to build surrealdb-embedded for a specific platform
build_surrealdb_embedded() {
    local platform=$1
    local arch=$2

    log_info "Building surrealdb-embedded for ${platform}-${arch}..."

    local output_dir="${DIST_LIBS_DIR}/${platform}-${arch}"
    mkdir -p "${output_dir}"

    # Check if surrealdb-embedded source exists
    if [ ! -d "${SURREALDB_DIR}/surrealdb_embedded_rs" ]; then
        log_error "surrealdb-embedded source not found at ${SURREALDB_DIR}"
        return 1
    fi

    cd "${SURREALDB_DIR}/surrealdb_embedded_rs"

    # Set Rust target based on platform and arch
    local rust_target=""
    case "${platform}-${arch}" in
        linux-amd64)
            rust_target="x86_64-unknown-linux-gnu"
            ;;
        linux-arm64)
            rust_target="aarch64-unknown-linux-gnu"
            ;;
        darwin-amd64)
            rust_target="x86_64-apple-darwin"
            ;;
        darwin-arm64)
            rust_target="aarch64-apple-darwin"
            ;;
        windows-amd64)
            rust_target="x86_64-pc-windows-gnu"
            ;;
        windows-arm64)
            rust_target="aarch64-pc-windows-gnu"
            ;;
        *)
            log_error "Unsupported platform: ${platform}-${arch}"
            return 1
            ;;
    esac

    # Add Rust target if not already installed
    rustup target add "${rust_target}" 2>/dev/null || true

    # Build with cargo
    log_info "Building Rust library for target: ${rust_target}..."
    cargo build --release --target "${rust_target}"

    # Copy library to output directory
    case "${platform}" in
        darwin)
            cp "target/${rust_target}/release/libsurrealdb_embedded_rs.dylib" "${output_dir}/" 2>/dev/null || \
            cp "target/${rust_target}/release/libsurrealdb_embedded_rs.so" "${output_dir}/" || \
            log_warn "Could not find surrealdb library for ${platform}-${arch}"
            ;;
        windows)
            cp "target/${rust_target}/release/surrealdb_embedded_rs.dll" "${output_dir}/" 2>/dev/null || \
            cp "target/${rust_target}/release/libsurrealdb_embedded_rs.dll" "${output_dir}/" || \
            log_warn "Could not find surrealdb library for ${platform}-${arch}"
            ;;
        *)
            cp "target/${rust_target}/release/libsurrealdb_embedded_rs.so" "${output_dir}/" || \
            log_warn "Could not find surrealdb library for ${platform}-${arch}"
            ;;
    esac

    log_info "surrealdb-embedded build complete for ${platform}-${arch}"
}

# Main build function
build_for_platform() {
    local platform=$1
    local arch=$2
    local cc=$3
    local cxx=$4

    log_info "=========================================="
    log_info "Building libraries for ${platform}-${arch}"
    log_info "=========================================="

    # Build llama.cpp
    build_llama_cpp "${platform}" "${arch}" "${cc}" "${cxx}" || log_error "Failed to build llama.cpp"

    # Build llama shim (depends on libllama)
    build_llama_shim "${platform}" "${arch}" "${cc}" || log_error "Failed to build libllama_shim"

    # Build surrealdb-embedded
    build_surrealdb_embedded "${platform}" "${arch}" || log_error "Failed to build surrealdb-embedded"

    log_info "Build complete for ${platform}-${arch}"
    log_info ""
}

# Main execution
main() {
    log_info "Starting cross-compilation of shared libraries"
    log_info "Project root: ${PROJECT_ROOT}"
    log_info "Output directory: ${DIST_LIBS_DIR}"

    # Create output directory
    mkdir -p "${DIST_LIBS_DIR}"

    # Build for each platform
    # Note: These compilers are available in the goreleaser-cross Docker image

    # Linux amd64
    build_for_platform "linux" "amd64" "x86_64-linux-gnu-gcc" "x86_64-linux-gnu-g++"

    # Linux arm64
    build_for_platform "linux" "arm64" "aarch64-linux-gnu-gcc" "aarch64-linux-gnu-g++"

    # macOS amd64
    build_for_platform "darwin" "amd64" "o64-clang" "o64-clang++"

    # macOS arm64
    build_for_platform "darwin" "arm64" "oa64-clang" "oa64-clang++"

    # Windows amd64
    build_for_platform "windows" "amd64" "x86_64-w64-mingw32-gcc" "x86_64-w64-mingw32-g++"

 
    log_info "=========================================="
    log_info "All libraries built successfully!"
    log_info "=========================================="
    log_info "Libraries are located in: ${DIST_LIBS_DIR}"
    ls -lR "${DIST_LIBS_DIR}"
}

# Run main function
main "$@"
