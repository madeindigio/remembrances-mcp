#!/bin/bash
set -e

echo "Building remembrances-mcp..."
cd /www/MCP/remembrances-mcp

# Build the binary
go build -o dist/remembrances-mcp ./cmd/remembrances-mcp

if [[ -f "dist/remembrances-mcp" ]]; then
    echo "✅ Build successful: dist/remembrances-mcp"
    ls -lh dist/remembrances-mcp
else
    echo "❌ Build failed: binary not found"
    exit 1
fi
