#!/bin/bash
# Script to build custom goreleaser-cross Docker image with Rust support
# This image will be used for cross-compiling remembrances-mcp

set -e

# Configuration
IMAGE_NAME="remembrances-mcp-builder"
IMAGE_TAG="v1.23-rust"
DOCKERFILE_PATH="docker/Dockerfile.goreleaser-custom"
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
Usage: $0 [OPTIONS]

Build custom Docker image for cross-compilation with Rust support.

OPTIONS:
    -t, --tag TAG       Image tag (default: ${IMAGE_TAG})
    -n, --name NAME     Image name (default: ${IMAGE_NAME})
    --no-cache          Build without using cache
    --push              Push image to registry after building
    -h, --help          Show this help message

EXAMPLES:
    # Build image with default settings
    $0

    # Build with custom tag
    $0 --tag latest

    # Build without cache
    $0 --no-cache

EOF
}

main() {
    local no_cache_flag=""
    local push_flag=false

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                print_usage
                exit 0
                ;;
            -t|--tag)
                IMAGE_TAG="$2"
                shift 2
                ;;
            -n|--name)
                IMAGE_NAME="$2"
                shift 2
                ;;
            --no-cache)
                no_cache_flag="--no-cache"
                shift
                ;;
            --push)
                push_flag=true
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
    log_info "Building Custom GoReleaser Cross Image"
    log_info "=========================================="
    log_info "Image: ${IMAGE_NAME}:${IMAGE_TAG}"
    log_info "Dockerfile: ${DOCKERFILE_PATH}"
    log_info "Project Root: ${PROJECT_ROOT}"
    log_info "=========================================="
    echo ""

    # Check if Dockerfile exists
    if [ ! -f "${DOCKERFILE_PATH}" ]; then
        log_error "Dockerfile not found at ${DOCKERFILE_PATH}"
        exit 1
    fi

    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi

    if ! docker info &> /dev/null; then
        log_error "Docker is not running"
        exit 1
    fi

    log_info "Docker is available"

    # Build the image
    log_step "Building Docker image..."

    docker build \
        ${no_cache_flag} \
        -t "${IMAGE_NAME}:${IMAGE_TAG}" \
        -f "${DOCKERFILE_PATH}" \
        .

    log_info "Docker image built successfully: ${IMAGE_NAME}:${IMAGE_TAG}"

    # Tag as latest if not already
    if [ "${IMAGE_TAG}" != "latest" ]; then
        log_step "Tagging as latest..."
        docker tag "${IMAGE_NAME}:${IMAGE_TAG}" "${IMAGE_NAME}:latest"
    fi

    # Push if requested
    if [ "${push_flag}" = true ]; then
        log_step "Pushing image to registry..."
        docker push "${IMAGE_NAME}:${IMAGE_TAG}"
        if [ "${IMAGE_TAG}" != "latest" ]; then
            docker push "${IMAGE_NAME}:latest"
        fi
        log_info "Image pushed successfully"
    fi

    echo ""
    log_info "=========================================="
    log_info "Build Complete!"
    log_info "=========================================="
    log_info "Image: ${IMAGE_NAME}:${IMAGE_TAG}"
    log_info ""
    log_info "To use this image with release-cross.sh:"
    log_info "  export GORELEASER_CROSS_IMAGE=${IMAGE_NAME}:${IMAGE_TAG}"
    log_info "  ./scripts/release-cross.sh snapshot"
    log_info ""
    log_info "Or update the release-cross.sh script to use this image by default."
    log_info "=========================================="
}

# Run main
main "$@"
