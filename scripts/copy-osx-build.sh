#!/bin/bash

echo "Copying OSX ARM64 build from mac-mini-de-digio"
scp -r mac-mini-de-digio:~/www/MCP/remembrances-mcp/dist/remembrances-mcp-darwin*.zip dist-variants/
