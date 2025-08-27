#!/usr/bin/env bash
set -euo pipefail

# Build script using xc as requested. Produces ./dist/remembrances-mcp
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"
BINARY_PATH="$DIST_DIR/remembrances-mcp"

mkdir -p "$DIST_DIR"

echo "Running: xc build"

# Use xc to build the binary into the required location
xc build

if [[ -f "$BINARY_PATH" ]]; then
  echo "Build succeeded: $BINARY_PATH"
  ls -lh "$BINARY_PATH"
else
  echo "Build failed: binary not found at $BINARY_PATH" >&2
  exit 1
fi
