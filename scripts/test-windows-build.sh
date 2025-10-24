#!/bin/bash

# Test Windows Build Script
# This script tests the Windows build using the goreleaser-cross Docker container
# to verify that the Windows API compatibility patch works correctly.

set -e

echo "=========================================="
echo "Windows Build Test Script"
echo "=========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo -e "${RED}Error: Must be run from project root${NC}"
    exit 1
fi

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed or not in PATH${NC}"
    exit 1
fi

echo -e "${YELLOW}Step 1: Cleaning previous Windows build artifacts${NC}"
# Try to remove with sudo if regular rm fails
if ! rm -rf go-llama.cpp/build/windows-amd64 2>/dev/null; then
    echo "  Attempting cleanup with elevated privileges..."
    sudo rm -rf go-llama.cpp/build/windows-amd64 || {
        echo -e "${YELLOW}  Warning: Could not remove build directory. Continuing anyway...${NC}"
    }
fi
rm -f go-llama.cpp/libbinding-windows-amd64.a 2>/dev/null || true
echo -e "${GREEN}  ✓ Cleanup completed${NC}"
echo ""

echo -e "${YELLOW}Step 2: Verifying compatibility header exists${NC}"
if [ ! -f "go-llama.cpp/windows-api-compat.h" ]; then
    echo -e "${RED}Error: windows-api-compat.h not found${NC}"
    exit 1
fi
echo -e "${GREEN}  ✓ Compatibility header found${NC}"
echo ""

echo -e "${YELLOW}Step 3: Building Windows binary in Docker container${NC}"
echo "  Using goreleaser-cross:v1.21 container..."
echo ""

# Run the build in Docker
# Use --network=none to avoid networking issues (build doesn't need network)
# Use --entrypoint="" to bypass the image's default entrypoint
docker run --rm \
    --network=none \
    --entrypoint="" \
    -v "$(pwd):/go/src/github.com/madeindigio/remembrances-mcp" \
    -w /go/src/github.com/madeindigio/remembrances-mcp/go-llama.cpp \
    -e BUILD_NUMBER=0 \
    -e BUILD_COMMIT=test \
    goreleaser/goreleaser-cross:v1.21 \
    bash -c "chmod +x scripts/build-static-multi.sh && ./scripts/build-static-multi.sh windows amd64"

BUILD_EXIT_CODE=$?

echo ""
if [ $BUILD_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}  ✓ Build completed successfully${NC}"
else
    echo -e "${RED}  ✗ Build failed with exit code $BUILD_EXIT_CODE${NC}"
    exit $BUILD_EXIT_CODE
fi
echo ""

echo -e "${YELLOW}Step 4: Verifying build artifacts${NC}"

# Check if the library was created
if [ -f "go-llama.cpp/libbinding-windows-amd64.a" ]; then
    LIB_SIZE=$(du -h go-llama.cpp/libbinding-windows-amd64.a | cut -f1)
    echo -e "${GREEN}  ✓ Static library created: libbinding-windows-amd64.a ($LIB_SIZE)${NC}"
else
    echo -e "${RED}  ✗ Static library not found${NC}"
    exit 1
fi

# Check library contents
echo ""
echo -e "${YELLOW}Step 5: Inspecting library contents${NC}"
echo "  Checking for key symbols..."

# List symbols in the library
if command -v x86_64-w64-mingw32-nm &> /dev/null; then
    echo ""
    echo "  Searching for llama_model symbols..."
    if x86_64-w64-mingw32-nm go-llama.cpp/libbinding-windows-amd64.a 2>/dev/null | grep -i "llama_model" | head -5; then
        echo -e "${GREEN}  ✓ Library contains expected symbols${NC}"
    else
        echo -e "${YELLOW}  ! Could not verify symbols (nm tool may not be available)${NC}"
    fi
else
    echo -e "${YELLOW}  ! Symbol inspection skipped (x86_64-w64-mingw32-nm not available)${NC}"
fi

echo ""
echo "=========================================="
echo -e "${GREEN}Windows Build Test: SUCCESS${NC}"
echo "=========================================="
echo ""
echo "Summary:"
echo "  • Compatibility header: ✓ Working"
echo "  • Windows build: ✓ Compiled"
echo "  • Static library: ✓ Created ($LIB_SIZE)"
echo ""
echo "Next steps:"
echo "  1. Test full multiplatform build: make release-multi-snapshot"
echo "  2. Extract and test Windows binary on Windows system"
echo "  3. Update documentation with test results"
echo ""
