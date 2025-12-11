#!/bin/bash

echo "Refreshing repo on mac-mini-de-digio"
essh mac-mini-de-digio 'cd ~/www/MCP/remembrances-mcp && git fetch --tags && git pull origin main'

echo "Building libs OSX ARM64 build from mac-mini-de-digio"
essh mac-mini-de-digio 'cd ~/www/MCP/remembrances-mcp && xc build-libs-osx'

echo "Building OSX ARM64 build from mac-mini-de-digio"
essh mac-mini-de-digio 'cd ~/www/MCP/remembrances-mcp && xc build-osx'
