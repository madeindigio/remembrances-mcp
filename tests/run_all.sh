#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Build and run all smoke tests
echo "1) Build"
bash "$SCRIPT_DIR/build.sh"

echo "2) stdio smoke test"
bash "$SCRIPT_DIR/test_stdio.sh"

echo "3) sse smoke test"
bash "$SCRIPT_DIR/test_sse.sh"

echo "All tests passed"
