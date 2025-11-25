#!/bin/bash
# Script to build release binaries using goreleaser-cross Docker image
# This handles CGO cross-compilation for multiple platforms

set -e

# Configuration
GORELEASER_CROSS_VERSION="${GORELEASER_CROSS_VERSION:-v1.23}"
GORELEASER_CROSS_IMAGE="${GORELEASER_CROSS_IMAGE:-ghcr.io/goreleaser/goreleaser-cross:${GORELEASER_CROSS_VERSION}}"
PROJECT_NAME="remembrances-mcp"
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "${SCRIPT_DIR}")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

print_usage() {
    cat << EOF
Usage: $0 [OPTIONS] [COMMAND]

Cross-compile ${PROJECT_NAME} using goreleaser-cross Docker image.

COMMANDS:
    build       Build binaries without releasing (default)
    release     Build and create a GitHub release
    snapshot    Build snapshot release (no tag required)

OPTIONS:
    -v, --version VERSION   Specify goreleaser-cross version (default: ${GORELEASER_CROSS_VERSION})
    -c, --clean             Clean before building
    --skip-libs             Skip building shared libraries
    --libs-only             Only build shared libraries, don't run goreleaser
    -h, --help              Show this help message

EXAMPLES:
    # Build snapshot (for testing)
    $0 snapshot

    # Build with specific version
    $0 -v v1.22 build

    # Build only shared libraries
    $0 --libs-only

    # Build and release
    $0 release

ENVIRONMENT VARIABLES:
    GORELEASER_CROSS_VERSION    Docker image version (default: v1.23)
    GORELEASER_CROSS_IMAGE      Full Docker image name (default: ghcr.io/goreleaser/goreleaser-cross:VERSION)
    GITHUB_TOKEN                GitHub token for releases

NOTES:
    - Requires Docker to be installed and running
    - For releases, you need a valid git tag and GITHUB_TOKEN
    - Shared libraries are built first, then included in archives
    - Output is in dist/outputs/dist/

EOF
}

check_requirements() {
    log_step "Checking requirements..."

    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi

    # Check if Docker is running
    if ! docker info &> /dev/null; then
        log_error "Docker is not running"
        exit 1
    fi

    log_info "Docker is available"

    # Check git
    if ! command -v git &> /dev/null; then
        log_error "Git is not installed"
        exit 1
    fi

    log_info "Git is available"
}

build_shared_libraries() {
    log_step "Building shared libraries for all platforms..."

    # Run the build-libs-cross.sh script inside the Docker container
    docker run --rm \
        -v "${PROJECT_ROOT}:/go/src/github.com/madeindigio/remembrances-mcp" \
        -v "~/www/MCP/Remembrances:~/www/MCP/Remembrances" \
        -w /go/src/github.com/madeindigio/remembrances-mcp \
        -e PROJECT_ROOT=/go/src/github.com/madeindigio/remembrances-mcp \
        -e LLAMA_CPP_DIR=~/www/MCP/Remembrances/go-llama.cpp \
        -e SURREALDB_DIR=~/www/MCP/Remembrances/surrealdb-embedded \
        --entrypoint /bin/bash \
        "${GORELEASER_CROSS_IMAGE}" \
        scripts/build-libs-cross.sh

    log_info "Shared libraries built successfully"
}

run_goreleaser() {
    local command=$1
    local extra_args="${@:2}"

    log_step "Running goreleaser ${command}..."

    # Prepare environment variables
    local env_vars=""
    if [ -n "${GITHUB_TOKEN}" ]; then
        env_vars="-e GITHUB_TOKEN=${GITHUB_TOKEN}"
    fi

    # Run goreleaser in Docker
    docker run --rm --privileged \
        -v "${PROJECT_ROOT}:/go/src/github.com/madeindigio/remembrances-mcp" \
        -v "~/www/MCP/Remembrances:~/www/MCP/Remembrances" \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -w /go/src/github.com/madeindigio/remembrances-mcp \
        ${env_vars} \
        "${GORELEASER_CROSS_IMAGE}" \
        ${command} ${extra_args}

    log_info "Goreleaser ${command} completed"
}

main() {
    local command="build"
    local clean_flag=""
    local skip_libs=false
    local libs_only=false

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                print_usage
                exit 0
                ;;
            -v|--version)
                GORELEASER_CROSS_VERSION="$2"
                shift 2
                ;;
            -c|--clean)
                clean_flag="--clean"
                shift
                ;;
            --skip-libs)
                skip_libs=true
                shift
                ;;
            --libs-only)
                libs_only=true
                shift
                ;;
            build|release|snapshot)
                command="$1"
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                print_usage
                exit 1
                ;;
        esac
    done

    # Change to project root
    cd "${PROJECT_ROOT}"

    log_info "=========================================="
    log_info "GoReleaser Cross-Compilation Build"
    log_info "=========================================="
    log_info "Project: ${PROJECT_NAME}"
    log_info "Root: ${PROJECT_ROOT}"
    log_info "Command: ${command}"
    log_info "Version: ${GORELEASER_CROSS_VERSION}"
    log_info "=========================================="
    echo ""

    # Check requirements
    check_requirements

    # Build shared libraries unless skipped
    if [ "${skip_libs}" = false ]; then
        build_shared_libraries || log_warn "Shared library build failed, continuing anyway..."
        echo ""
    else
        log_warn "Skipping shared library build"
    fi

    # Exit if libs-only
    if [ "${libs_only}" = true ]; then
        log_info "Libraries built. Exiting (--libs-only flag set)"
        exit 0
    fi

    # Run goreleaser
    case "${command}" in
        build)
            run_goreleaser "build" "--snapshot ${clean_flag}"
            ;;
        snapshot)
            run_goreleaser "release" "--snapshot ${clean_flag}"
            ;;
        release)
            if [ -z "${GITHUB_TOKEN}" ]; then
                log_warn "GITHUB_TOKEN not set. Release will not be published to GitHub."
                log_warn "Set GITHUB_TOKEN environment variable to publish releases."
            fi
            run_goreleaser "release" "${clean_flag}"
            ;;
    esac

    echo ""
    log_info "=========================================="
    log_info "Build Complete!"
    log_info "=========================================="
    log_info "Output directory: ${PROJECT_ROOT}/dist/outputs/dist/"

    if [ -d "${PROJECT_ROOT}/dist/outputs/dist" ]; then
        log_info "Built artifacts:"
        ls -lh "${PROJECT_ROOT}/dist/outputs/dist/" | grep -v "^total" || true
    fi
}

# Run main
main "$@"
