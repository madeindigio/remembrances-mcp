#!/bin/bash

echo "Copying libs OSX ARM64 build from mac-mini-de-digio"
scp -r mac-mini-de-digio:~/www/MCP/remembrances-mcp/build dist-variants/
cp config.sample* dist-variants/build/
mv dist-variants/build dist-variants/darwin_aarch64

echo "Copying OSX ARM64 build from mac-mini-de-digio"
scp -r mac-mini-de-digio:~/www/MCP/remembrances-mcp/dist/darwin-amd64 dist-variants/
