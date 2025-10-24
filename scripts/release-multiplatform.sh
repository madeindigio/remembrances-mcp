#!/bin/bash

# Multi-platform release script for Remembrances-MCP
# This script automates the entire release process using goreleaser-cross Docker image
#
# Usage:
#   ./scripts/release-multiplatform.sh [snapshot|release]
#
# Examples:
#   ./scripts/release-multiplatform.sh snapshot    # Build snapshot without releasing
#   ./scripts/release-multiplatform.sh release     # Build and release to GitHub
#   ./scripts/release-multiplatform.sh             # Default: snapshot

set -e

# Check for non-interactive mode
SKIP_CONFIRM="${SKIP_CONFIRM:-false}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DOCKER_IMAGE="ghcr.io/goreleaser/goreleaser-cross:v1.21"
GORELEASER_CONFIG=".goreleaser-multiplatform.yml"
GORELEASER_FAST_CONFIG=".goreleaser-fast.yml"
MODE="${1:-snapshot}"
USE_PREBUILT_LIBS="${2:-false}"

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

print_banner() {
    echo ""
    echo "=========================================="
    echo "  Remembrances-MCP Multi-Platform Build"
    echo "=========================================="
    echo ""
}

check_requirements() {
    log_info "Checking requirements..."

    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        log_info "Install Docker: https://docs.docker.com/get-docker/"
        exit 1
    fi

    # Check Docker daemon
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        log_info "Start Docker and try again"
        exit 1
    fi

    # Check if goreleaser config exists
    if [ ! -f "$GORELEASER_CONFIG" ]; then
        log_error "GoReleaser config not found: $GORELEASER_CONFIG"
        exit 1
    fi

    # Check git repository
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        log_error "Not a git repository"
        exit 1
    fi

    log_success "All requirements satisfied"
}

check_disk_space() {
    log_info "Checking disk space..."

    AVAILABLE_GB=$(df -BG . | tail -1 | awk '{print $4}' | sed 's/G//')
    REQUIRED_GB=10

    if [ "$AVAILABLE_GB" -lt "$REQUIRED_GB" ]; then
        log_warning "Low disk space: ${AVAILABLE_GB}GB available, ${REQUIRED_GB}GB recommended"
        read -p "Continue anyway? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    else
        log_success "Disk space: ${AVAILABLE_GB}GB available"
    fi
}

pull_docker_image() {
    log_info "Pulling Docker image: $DOCKER_IMAGE"

    if docker pull "$DOCKER_IMAGE"; then
        log_success "Docker image ready"
    else
        log_error "Failed to pull Docker image"
        exit 1
    fi
}

validate_mode() {
    case "$MODE" in
        snapshot|release)
            log_success "Mode: $MODE"
            ;;
        *)
            log_error "Invalid mode: $MODE"
            echo "Usage: $0 [snapshot|release] [prebuilt]"
            exit 1
            ;;
    esac
}

check_prebuilt_libs() {
    if [ "$USE_PREBUILT_LIBS" = "prebuilt" ]; then
        log_info "Checking for pre-built libraries..."

        local missing=0
        for platform in linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64; do
            if [ ! -f "go-llama.cpp/libbinding-${platform}.a" ]; then
                log_warning "Missing library: libbinding-${platform}.a"
                missing=$((missing + 1))
            fi
        done

        if [ $missing -gt 0 ]; then
            log_error "$missing platform libraries are missing"
            log_info "Run: make llama-deps-all-parallel"
            exit 1
        fi

        log_success "All pre-built libraries found"
        GORELEASER_CONFIG="$GORELEASER_FAST_CONFIG"
        log_info "Using fast config (skips library builds)"
        return 0
    fi

    log_info "Using standard config (builds libraries in Docker)"
    return 0
}

check_github_token() {
    if [ "$MODE" = "release" ]; then
        if [ -z "$GITHUB_TOKEN" ]; then
            log_error "GITHUB_TOKEN environment variable is required for release mode"
            log_info "Export your GitHub token: export GITHUB_TOKEN=ghp_xxxxx"
            exit 1
        fi
        log_success "GitHub token found"
    fi
}

show_git_info() {
    log_info "Git information:"
    echo "  Branch: $(git rev-parse --abbrev-ref HEAD)"
    echo "  Commit: $(git rev-parse --short HEAD)"

    if git describe --tags --exact-match 2>/dev/null; then
        echo "  Tag: $(git describe --tags --exact-match)"
    else
        echo "  Tag: (none - will use commit hash)"
    fi

    # Check for uncommitted changes
    if [ -n "$(git status --porcelain)" ]; then
        log_warning "Working directory has uncommitted changes"
        if [ "$MODE" = "release" ]; then
            log_error "Cannot create release with uncommitted changes"
            exit 1
        fi
    fi
}

run_goreleaser() {
    log_info "Starting multi-platform build..."
    echo ""

    if [ "$USE_PREBUILT_LIBS" = "prebuilt" ]; then
        log_info "Using pre-built libraries (fast mode)"
    else
        log_info "Building libraries in Docker (standard mode)"
    fi
    echo ""

    GORELEASER_ARGS="release --config $GORELEASER_CONFIG --clean"

    if [ "$MODE" = "snapshot" ]; then
        GORELEASER_ARGS="$GORELEASER_ARGS --snapshot"
    fi

    # Always skip signing for now
    GORELEASER_ARGS="$GORELEASER_ARGS --skip=sign"

    # Build Docker command
    # Note: We run as root in container but use git config workaround
    DOCKER_CMD="docker run --rm --network=host"
    DOCKER_CMD="$DOCKER_CMD -v $(pwd):/go/src/github.com/madeindigio/remembrances-mcp"
    DOCKER_CMD="$DOCKER_CMD -w /go/src/github.com/madeindigio/remembrances-mcp"

    # Add git safe directory configuration
    DOCKER_CMD="$DOCKER_CMD -e GIT_CONFIG_GLOBAL=/tmp/.gitconfig"

    if [ "$MODE" = "release" ]; then
        DOCKER_CMD="$DOCKER_CMD -e GITHUB_TOKEN=$GITHUB_TOKEN"
    fi

    DOCKER_CMD="$DOCKER_CMD $DOCKER_IMAGE $GORELEASER_ARGS"

    log_info "Running command:"
    echo "  $DOCKER_CMD"
    echo ""

    # Run goreleaser
    if eval "$DOCKER_CMD"; then
        return 0
    else
        return 1
    fi
}

fix_permissions() {
    log_info "Fixing file permissions..."

    # Fix permissions on build artifacts if they exist
    if [ -d "go-llama.cpp/build" ]; then
        chmod -R u+w go-llama.cpp/build 2>/dev/null || true
    fi

    if [ -d "dist" ]; then
        chmod -R u+w dist 2>/dev/null || true
    fi

    log_success "Permissions fixed"
}

show_results() {
    echo ""
    log_success "Build completed successfully!"
    echo ""

    # Fix permissions before showing results
    fix_permissions

    # Show generated artifacts
    if [ -d "dist/outputs/dist" ]; then
        log_info "Generated artifacts:"
        ls -lh dist/outputs/dist/*.tar.gz dist/outputs/dist/*.zip 2>/dev/null | while read -r line; do
            filename=$(echo "$line" | awk '{print $9}')
            size=$(echo "$line" | awk '{print $5}')
            echo "  ${filename##*/} (${size})"
        done

        # Show checksums
        if [ -f "dist/outputs/dist/checksums.txt" ]; then
            echo ""
            log_info "Checksums available: dist/outputs/dist/checksums.txt"
        fi
    fi

    echo ""
    log_info "Build artifacts location: dist/outputs/dist/"

    if [ "$MODE" = "release" ]; then
        echo ""
        log_success "Release published to GitHub!"
        log_info "Check: https://github.com/madeindigio/remembrances-mcp/releases"
    fi
}

cleanup_on_error() {
    log_error "Build failed!"
    log_info "Check the output above for errors"
    exit 1
}

# Main execution
main() {
    print_banner

    # Validate inputs
    validate_mode

    # Check for pre-built libraries if requested
    check_prebuilt_libs

    # Pre-flight checks
    check_requirements
    check_github_token
    check_disk_space
    show_git_info

    echo ""
    log_info "This will build binaries for:"
    echo "  - Linux (amd64, arm64)"
    echo "  - macOS (amd64, arm64)"
    echo "  - Windows (amd64)"
    echo ""
    if [ "$USE_PREBUILT_LIBS" = "prebuilt" ]; then
        log_info "Build time: ~3-5 minutes (using pre-built libraries)"
    else
        log_warning "Build time: ~30-40 minutes (building libraries in Docker)"
    fi
    echo ""

    # Confirm before proceeding (skip if SKIP_CONFIRM is set)
    if [ "$SKIP_CONFIRM" != "true" ]; then
        if [ "$MODE" = "release" ]; then
            log_warning "This will create a PUBLIC release on GitHub!"
            read -p "Continue? (y/N) " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                log_info "Aborted by user"
                exit 0
            fi
        else
            read -p "Continue? (Y/n) " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Nn]$ ]]; then
                log_info "Aborted by user"
                exit 0
            fi
        fi
    else
        log_info "Skipping confirmation (SKIP_CONFIRM=true)"
    fi

    echo ""

    # Pull Docker image
    pull_docker_image

    echo ""

    # Run the build
    if run_goreleaser; then
        show_results
    else
        cleanup_on_error
    fi
}

# Trap errors
trap cleanup_on_error ERR

# Run main
main
