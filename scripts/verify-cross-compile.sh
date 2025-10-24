#!/bin/bash

# Cross-Compilation Verification Script for Remembrances-MCP
# This script verifies that the Docker-based cross-compilation system is properly configured
#
# Usage: ./scripts/verify-cross-compile.sh [--full]
#
# Options:
#   --full    Run full verification including test build
#   --help    Show this help message

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configuration
DOCKER_IMAGE="ghcr.io/goreleaser/goreleaser-cross:v1.21"
GORELEASER_CONFIG=".goreleaser-multiplatform.yml"
GORELEASER_FAST_CONFIG=".goreleaser-fast.yml"
FULL_TEST=false

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

log_step() {
    echo -e "\n${CYAN}▶${NC} $1"
}

print_banner() {
    echo ""
    echo "=========================================="
    echo "  Cross-Compilation Verification"
    echo "  Remembrances-MCP Build System"
    echo "=========================================="
    echo ""
}

print_help() {
    cat <<EOF
Usage: $0 [OPTIONS]

Verify Docker-based cross-compilation setup for Remembrances-MCP.

Options:
    --full      Run full verification including test build
    --help      Show this help message

Examples:
    # Quick verification
    $0

    # Full verification with test build
    $0 --full

EOF
}

parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --full)
                FULL_TEST=true
                shift
                ;;
            --help|-h)
                print_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                print_help
                exit 1
                ;;
        esac
    done
}

check_docker() {
    log_step "Checking Docker installation..."

    if ! command -v docker &> /dev/null; then
        log_error "Docker not found"
        echo "  Install Docker: https://docs.docker.com/get-docker/"
        return 1
    fi
    log_success "Docker found: $(docker --version)"

    if ! docker info &> /dev/null; then
        log_error "Docker daemon not running"
        echo "  Start Docker and try again"
        return 1
    fi
    log_success "Docker daemon running"

    return 0
}

check_docker_image() {
    log_step "Checking goreleaser-cross Docker image..."

    if docker image inspect "$DOCKER_IMAGE" &> /dev/null; then
        log_success "Image already available: $DOCKER_IMAGE"

        local created=$(docker image inspect "$DOCKER_IMAGE" --format '{{.Created}}' | cut -d'T' -f1)
        log_info "Image created: $created"
        return 0
    else
        log_warning "Image not found locally: $DOCKER_IMAGE"
        log_info "Pull with: docker pull $DOCKER_IMAGE"
        return 1
    fi
}

check_goreleaser_configs() {
    log_step "Checking GoReleaser configurations..."

    local all_found=true

    if [ -f "$GORELEASER_CONFIG" ]; then
        log_success "Standard config found: $GORELEASER_CONFIG"

        # Check if it has before hooks
        if grep -q "before:" "$GORELEASER_CONFIG"; then
            log_info "  Contains before hooks (builds libraries in Docker)"
        fi
    else
        log_error "Missing: $GORELEASER_CONFIG"
        all_found=false
    fi

    if [ -f "$GORELEASER_FAST_CONFIG" ]; then
        log_success "Fast config found: $GORELEASER_FAST_CONFIG"
        log_info "  Uses pre-built libraries (faster)"
    else
        log_warning "Missing: $GORELEASER_FAST_CONFIG (optional)"
    fi

    if [ "$all_found" = false ]; then
        return 1
    fi

    return 0
}

check_build_scripts() {
    log_step "Checking build scripts..."

    local scripts=(
        "scripts/release-multiplatform.sh"
        "scripts/build-llama-parallel.sh"
        "scripts/cache-manager.sh"
        "go-llama.cpp/scripts/build-static-multi.sh"
    )

    local all_found=true

    for script in "${scripts[@]}"; do
        if [ -f "$script" ]; then
            log_success "Found: $script"

            # Check if executable
            if [ -x "$script" ]; then
                log_info "  ✓ Executable"
            else
                log_warning "  ⚠ Not executable (will be fixed automatically)"
            fi
        else
            log_error "Missing: $script"
            all_found=false
        fi
    done

    if [ "$all_found" = false ]; then
        return 1
    fi

    return 0
}

check_makefile_targets() {
    log_step "Checking Makefile targets..."

    local targets=(
        "llama-deps-all-parallel"
        "release-multi"
        "release-multi-snapshot"
        "release-multi-fast"
        "release-multi-snapshot-fast"
        "cache-info"
    )

    local all_found=true

    for target in "${targets[@]}"; do
        if grep -q "^${target}:" Makefile; then
            log_success "Target available: make $target"
        else
            log_error "Target missing: $target"
            all_found=false
        fi
    done

    if [ "$all_found" = false ]; then
        return 1
    fi

    return 0
}

check_go_environment() {
    log_step "Checking Go environment..."

    if ! command -v go &> /dev/null; then
        log_error "Go not found"
        return 1
    fi
    log_success "Go found: $(go version)"

    if [ -f "go.mod" ]; then
        log_success "go.mod found"

        local module=$(grep "^module" go.mod | awk '{print $2}')
        log_info "  Module: $module"
    else
        log_error "go.mod not found"
        return 1
    fi

    return 0
}

check_llama_cpp_submodule() {
    log_step "Checking llama.cpp submodule..."

    if [ ! -d "go-llama.cpp" ]; then
        log_error "go-llama.cpp directory not found"
        log_info "Initialize with: git submodule update --init --recursive"
        return 1
    fi
    log_success "go-llama.cpp directory found"

    if [ ! -d "go-llama.cpp/llama.cpp" ]; then
        log_error "llama.cpp submodule not initialized"
        log_info "Initialize with: git submodule update --init --recursive"
        return 1
    fi
    log_success "llama.cpp submodule initialized"

    # Check for key files
    if [ -f "go-llama.cpp/llama.cpp/CMakeLists.txt" ]; then
        log_success "llama.cpp CMakeLists.txt found"
    else
        log_warning "llama.cpp may not be fully initialized"
    fi

    return 0
}

check_disk_space() {
    log_step "Checking disk space..."

    local available_gb=$(df -BG . | tail -1 | awk '{print $4}' | sed 's/G//')
    local required_gb=15

    if [ "$available_gb" -lt "$required_gb" ]; then
        log_warning "Low disk space: ${available_gb}GB available, ${required_gb}GB recommended"
        log_info "You may experience issues during build"
        return 1
    else
        log_success "Disk space: ${available_gb}GB available"
    fi

    return 0
}

check_prebuilt_libraries() {
    log_step "Checking for pre-built libraries..."

    local platforms=("linux-amd64" "linux-arm64" "darwin-amd64" "darwin-arm64" "windows-amd64")
    local found=0
    local total=${#platforms[@]}

    for platform in "${platforms[@]}"; do
        if [ -f "go-llama.cpp/libbinding-${platform}.a" ]; then
            local size=$(du -h "go-llama.cpp/libbinding-${platform}.a" | cut -f1)
            log_success "Found: libbinding-${platform}.a ($size)"
            found=$((found + 1))
        else
            log_info "Not found: libbinding-${platform}.a"
        fi
    done

    echo ""
    if [ $found -eq 0 ]; then
        log_info "No pre-built libraries found (normal for first build)"
        log_info "Build with: make llama-deps-all-parallel"
    elif [ $found -eq $total ]; then
        log_success "All platform libraries available ($found/$total)"
        log_info "You can use fast release: make release-multi-snapshot-fast"
    else
        log_warning "Partial libraries found ($found/$total)"
        log_info "Rebuild all with: make llama-deps-all-parallel"
    fi

    return 0
}

test_docker_build() {
    log_step "Testing Docker build (minimal)..."

    log_info "Pulling goreleaser-cross image..."
    if docker pull "$DOCKER_IMAGE" 2>&1 | tail -5; then
        log_success "Docker image pulled successfully"
    else
        log_error "Failed to pull Docker image"
        return 1
    fi

    log_info "Testing Docker run..."
    if docker run --rm "$DOCKER_IMAGE" goreleaser --version &> /dev/null; then
        log_success "Docker container can run goreleaser"
    else
        log_error "Failed to run goreleaser in Docker"
        return 1
    fi

    return 0
}

show_recommendations() {
    echo ""
    echo "=========================================="
    echo "  Recommendations"
    echo "=========================================="
    echo ""

    log_info "Quick build commands:"
    echo "  make llama-deps-all-parallel      # Build libraries (6-10 min)"
    echo "  make release-multi-snapshot-fast  # Create release (3-5 min with cache)"
    echo ""

    log_info "Standard build (slower, builds in Docker):"
    echo "  make release-multi-snapshot       # Full build (30-40 min)"
    echo ""

    log_info "Cache management:"
    echo "  make cache-info                   # View cache status"
    echo "  make cache-clean                  # Clean old caches"
    echo ""

    log_info "Documentation:"
    echo "  docs/BUILD_QUICK_START.md         # Quick start guide"
    echo "  docs/BUILD_CACHE_SYSTEM.md        # Cache system docs"
    echo "  docs/CROSS_COMPILATION.md         # Cross-compilation guide"
    echo ""
}

show_summary() {
    echo ""
    echo "=========================================="
    echo "  Verification Summary"
    echo "=========================================="
    echo ""

    local checks_passed=0
    local checks_total=0

    # Count results (simplified)
    if [ -n "$RESULTS" ]; then
        checks_passed=$(echo "$RESULTS" | grep -c "PASS" || echo "0")
        checks_total=$(echo "$RESULTS" | wc -l)
    fi

    if command -v docker &> /dev/null && docker info &> /dev/null; then
        log_success "Docker: Ready"
    else
        log_error "Docker: Not ready"
    fi

    if [ -f "$GORELEASER_CONFIG" ] && [ -f "Makefile" ]; then
        log_success "Configuration: Complete"
    else
        log_error "Configuration: Incomplete"
    fi

    if [ -d "go-llama.cpp/llama.cpp" ]; then
        log_success "Submodules: Initialized"
    else
        log_error "Submodules: Not initialized"
    fi

    echo ""
}

main() {
    print_banner

    # Parse arguments
    parse_arguments "$@"

    # Run checks
    local failed=0

    check_docker || failed=$((failed + 1))
    check_docker_image || failed=$((failed + 1))
    check_goreleaser_configs || failed=$((failed + 1))
    check_build_scripts || failed=$((failed + 1))
    check_makefile_targets || failed=$((failed + 1))
    check_go_environment || failed=$((failed + 1))
    check_llama_cpp_submodule || failed=$((failed + 1))
    check_disk_space || failed=$((failed + 1))
    check_prebuilt_libraries

    # Full test if requested
    if [ "$FULL_TEST" = true ]; then
        test_docker_build || failed=$((failed + 1))
    fi

    # Show summary
    show_summary

    # Show recommendations
    if [ $failed -eq 0 ]; then
        echo ""
        log_success "All checks passed! System is ready for cross-compilation."
        show_recommendations
        exit 0
    else
        echo ""
        log_error "$failed check(s) failed. Please fix the issues above."
        echo ""
        log_info "Common fixes:"
        echo "  - Install Docker: https://docs.docker.com/get-docker/"
        echo "  - Initialize submodules: git submodule update --init --recursive"
        echo "  - Make scripts executable: chmod +x scripts/*.sh"
        echo ""
        exit 1
    fi
}

# Run main
main "$@"
