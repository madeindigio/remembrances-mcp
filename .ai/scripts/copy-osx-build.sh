#!/bin/bash

#change permissions for build using docker
sudo chown ${USER}.${USER} dist/*
echo "Copying libs OSX x64 build from mac-mini-de-digio"
scp -r mac-mini-de-digio:~/Documents/Projects/Remembrances/remembrances-mcp/dist/libs/darwin-arm64 dist/libs/

echo "Copying OSX ARM64 build from mac-mini-de-digio"
scp -r mac-mini-de-digio:~/Documents/Projects/Remembrances/remembrances-mcp/output/dist/remembrances-mcp-darwin-arm64 dist/outputs/dist/remembrances-mcp-darwin-arm64_darwin_arm64_v8.0/
