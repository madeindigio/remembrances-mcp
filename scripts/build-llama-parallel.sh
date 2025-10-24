#!/bin/bash

# Parallel build script for llama.cpp static libraries
# This script builds llama.cpp static libraries for all platforms in parallel
# to significantly reduce total build time
#
# Usage: ./scripts/build-llama-parallel.sh [--clean] [--verify]
#
# Options:
#   --clean   Clean all previous builds before starting
#   --verify  Verify libraries after build
#   --help    Show this help message
#
# Requirements:
#   - GNU Parallel (install: apt-get install parallel or brew install parallel)
#   - OR: xargs with -P flag support (built-in on most systems)
#
# Performance:
#   Sequential: ~30 minutes
#   Parallel:   ~6-8 minutes (on 4+ core systems)

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
PLATFORMS=("linux-amd64" "linux-arm64" "darwin-amd64" "darwin-arm64" "windows-amd64")
LLAMA_DIR="./go-llama.cpp"
BUILD_SCRIPT="$LLAMA_DIR/scripts/build-static-multi.sh"
MAX_JOBS=5  # Number of parallel jobs

# Flags
CLEAN_BUILD=false
VERIFY_BUILD=false
USE_GNU_PARALLEL=false

# Functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

log_platform() {
    local platform=$1
    local message=$2
    echo -e "${CYAN}[$platform]${NC} $message"
}

print_banner() {
    echo ""
    echo "=========================================="
    echo "  Llama.cpp Parallel Build System"
    echo "=========================================="
    echo ""
}

print_help() {
    cat <<EOF
Usage: $0 [OPTIONS]

Build llama.cpp static libraries for all platforms in parallel.

Options:
    --clean      Clean all previous builds before starting
    --verify     Verify libraries after build
    --help       Show this help message

Supported Platforms:
    - linux-amd64   (Linux x86_64)
    - linux-arm64   (Linux ARM64)
    - darwin-amd64  (macOS Intel)
    - darwin-arm64  (macOS Apple Silicon)
    - windows-amd64 (Windows x86_64)

Environment Variables:
    MAX_PARALLEL_JOBS  Override number of parallel jobs (default: 5)

Examples:
    # Standard parallel build
    $0

    # Clean build with verification
    $0 --clean --verify

    # Use 3 parallel jobs
    MAX_PARALLEL_JOBS=3 $0

EOF
}

parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --clean)
                CLEAN_BUILD=true
                shift
                ;;
            --verify)
                VERIFY_BUILD=true
                shift
                ;;
            --help|-h)
                print_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
    done
}

check_requirements() {
    log_info "Checking requirements..."

    # Check if llama.cpp directory exists
    if [ ! -d "$LLAMA_DIR" ]; then
        log_error "Llama.cpp directory not found: $LLAMA_DIR"
        exit 1
    fi

    # Check if build script exists
    if [ ! -f "$BUILD_SCRIPT" ]; then
        log_error "Build script not found: $BUILD_SCRIPT"
        exit 1
    fi

    # Make build script executable
    chmod +x "$BUILD_SCRIPT"

    # Check for parallel execution capability
    if command -v parallel &> /dev/null; then
        USE_GNU_PARALLEL=true
        log_success "GNU Parallel found (optimal)"
    elif echo | xargs -P 2 -I {} echo {} &> /dev/null; then
        USE_GNU_PARALLEL=false
        log_success "xargs with -P support found"
    else
        log_error "No parallel execution tool found"
        log_info "Install GNU Parallel: apt-get install parallel or brew install parallel"
        exit 1
    fi

    # Check available cores
    if command -v nproc &> /dev/null; then
        AVAILABLE_CORES=$(nproc)
    elif command -v sysctl &> /dev/null; then
        AVAILABLE_CORES=$(sysctl -n hw.ncpu)
    else
        AVAILABLE_CORES=4
    fi

    log_info "Available CPU cores: $AVAILABLE_CORES"

    # Override MAX_JOBS if environment variable is set
    if [ -n "$MAX_PARALLEL_JOBS" ]; then
        MAX_JOBS=$MAX_PARALLEL_JOBS
    fi

    # Adjust jobs if more than available cores
    if [ $MAX_JOBS -gt $AVAILABLE_CORES ]; then
        log_warning "Reducing parallel jobs from $MAX_JOBS to $AVAILABLE_CORES (available cores)"
        MAX_JOBS=$AVAILABLE_CORES
    fi

    log_success "Will use $MAX_JOBS parallel jobs"
}

clean_builds() {
    if [ "$CLEAN_BUILD" = true ]; then
        log_info "Cleaning previous builds..."

        cd "$LLAMA_DIR"

        # Clean build directories
        rm -rf build/

        # Clean static libraries
        rm -f libbinding*.a

        # Clean object files
        find llama.cpp -name "*.o" -delete 2>/dev/null || true

        cd - > /dev/null

        log_success "Cleaned previous builds"
    fi
}

build_platform() {
    local platform=$1
    local os=$(echo $platform | cut -d'-' -f1)
    local arch=$(echo $platform | cut -d'-' -f2)

    local start_time=$(date +%s)

    log_platform "$platform" "Starting build..."

    # Create log file for this platform
    local log_file="build-${platform}.log"

    # Run build script
    if cd "$LLAMA_DIR" && ./scripts/build-static-multi.sh "$os" "$arch" > "../$log_file" 2>&1; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log_platform "$platform" "${GREEN}Completed${NC} in ${duration}s"
        cd - > /dev/null
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log_platform "$platform" "${RED}Failed${NC} after ${duration}s (see $log_file)"
        cd - > /dev/null
        return 1
    fi
}

export -f build_platform
export -f log_platform
export LLAMA_DIR BUILD_SCRIPT GREEN RED NC

build_all_parallel_gnu() {
    log_info "Building with GNU Parallel..."

    # Use GNU Parallel for optimal performance
    printf "%s\n" "${PLATFORMS[@]}" | \
        parallel --will-cite --jobs "$MAX_JOBS" --progress build_platform {}

    return ${PIPESTATUS[1]}
}

build_all_parallel_xargs() {
    log_info "Building with xargs..."

    # Use xargs as fallback
    printf "%s\n" "${PLATFORMS[@]}" | \
        xargs -P "$MAX_JOBS" -I {} bash -c 'build_platform "$@"' _ {}

    return $?
}

build_all_parallel() {
    local start_time=$(date +%s)

    echo ""
    log_info "Building llama.cpp for ${#PLATFORMS[@]} platforms in parallel..."
    echo ""

    local result=0

    if [ "$USE_GNU_PARALLEL" = true ]; then
        build_all_parallel_gnu
        result=$?
    else
        build_all_parallel_xargs
        result=$?
    fi

    local end_time=$(date +%s)
    local total_duration=$((end_time - start_time))
    local minutes=$((total_duration / 60))
    local seconds=$((total_duration % 60))

    echo ""
    log_info "Total build time: ${minutes}m ${seconds}s"

    return $result
}

verify_libraries() {
    if [ "$VERIFY_BUILD" = false ]; then
        return 0
    fi

    log_info "Verifying built libraries..."

    cd "$LLAMA_DIR"

    local failed=0

    for platform in "${PLATFORMS[@]}"; do
        local lib_file="libbinding-${platform}.a"

        if [ ! -f "$lib_file" ]; then
            log_error "Missing library: $lib_file"
            failed=$((failed + 1))
            continue
        fi

        # Check file size (should be > 1MB)
        local size=$(stat -f%z "$lib_file" 2>/dev/null || stat -c%s "$lib_file" 2>/dev/null)

        if [ -z "$size" ] || [ "$size" -lt 1048576 ]; then
            log_error "Library too small (probably corrupted): $lib_file"
            failed=$((failed + 1))
            continue
        fi

        # Check if it's a valid archive
        if ! ar t "$lib_file" > /dev/null 2>&1; then
            log_error "Invalid archive: $lib_file"
            failed=$((failed + 1))
            continue
        fi

        local size_mb=$((size / 1048576))
        log_success "$lib_file (${size_mb}MB)"
    done

    cd - > /dev/null

    if [ $failed -gt 0 ]; then
        log_error "$failed platform(s) failed verification"
        return 1
    fi

    log_success "All libraries verified successfully"
    return 0
}

show_results() {
    echo ""
    echo "=========================================="
    echo "  Build Results"
    echo "=========================================="
    echo ""

    cd "$LLAMA_DIR"

    log_info "Generated libraries:"
    for platform in "${PLATFORMS[@]}"; do
        local lib_file="libbinding-${platform}.a"
        if [ -f "$lib_file" ]; then
            local size=$(du -h "$lib_file" | cut -f1)
            echo "  ${GREEN}✓${NC} $lib_file ($size)"
        else
            echo "  ${RED}✗${NC} $lib_file (MISSING)"
        fi
    done

    cd - > /dev/null

    echo ""

    # Show logs if any builds failed
    if ls build-*.log > /dev/null 2>&1; then
        log_info "Build logs available:"
        for log in build-*.log; do
            echo "  - $log"
        done
    fi
}

cleanup_logs() {
    # Keep logs for debugging, but mention they can be removed
    if ls build-*.log > /dev/null 2>&1; then
        echo ""
        log_info "To clean build logs: rm build-*.log"
    fi
}

main() {
    print_banner

    # Parse command line arguments
    parse_arguments "$@"

    # Pre-flight checks
    check_requirements

    # Clean if requested
    clean_builds

    # Build all platforms in parallel
    if ! build_all_parallel; then
        log_error "One or more platforms failed to build"
        show_results
        exit 1
    fi

    # Verify libraries
    if ! verify_libraries; then
        log_error "Library verification failed"
        exit 1
    fi

    # Show results
    show_results

    # Cleanup reminder
    cleanup_logs

    echo ""
    log_success "All platforms built successfully!"
    echo ""
    log_info "Next steps:"
    echo "  - Run tests: make test"
    echo "  - Build Go binaries: make build"
    echo "  - Create release: make release-multi-snapshot"
}

# Run main function
main "$@"
