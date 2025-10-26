#!/bin/bash

export GOMEM_SURREALDB_START_CMD="surreal start --user root --pass root surrealkv:///www/Remembrances/programming"
/www/MCP/remembrances-mcp/dist/outputs/dist/linux-amd64_linux_amd64_v1/remembrances-mcp --surrealdb-url ws://localhost:8000 --ollama-url http://localhost:11434 --ollama-model nomic-embed-text:latest --knowledge-base /www/MCP/remembrances-mcp/.serena/memories
