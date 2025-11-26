# Remembrances-MCP Testing Guide

## Overview

The project includes comprehensive test suites for all transport types (stdio, SSE, HTTP). Tests are located in the `tests/` directory.

## Test Structure

```
tests/
├── build.sh              # Build script for tests
├── run_all.sh            # Run all tests
├── test_stdio.sh         # stdio transport test
├── test_sse.sh           # SSE transport test  
├── test_http.sh          # HTTP transport test
├── README.md             # Test documentation
└── clients/              # Test clients
    ├── mcp_stdio_client.py    # stdio MCP client
    ├── mcp_sse_client.py      # SSE MCP client
    ├── mcp_sse_client_simple.py # Simple SSE client
    └── mcp_http_client.py     # HTTP API client
```

## Running Tests

### All Tests
```bash
# Run complete test suite
cd ~/www/MCP/remembrances-mcp
bash tests/run_all.sh
```

### Individual Tests

#### 1. Build Only
```bash
bash tests/build.sh
```

#### 2. stdio Transport Test
```bash
bash tests/test_stdio.sh
```

#### 3. SSE Transport Test
```bash
bash tests/test_sse.sh
```

#### 4. HTTP Transport Test  
```bash
bash tests/test_http.sh
```

## Test Requirements

### Environment Setup
Tests require these environment variables (set automatically by test scripts):
- `GOMEM_OPENAI_KEY="testkey"` (dummy key for testing)
- `GOMEM_DB_PATH="$REPO_ROOT/surreal_data"`
- `GOMEM_SURREALDB_URL="ws://localhost:8000"`
- `GOMEM_SURREALDB_USER="root"`
- `GOMEM_SURREALDB_PASS="root"`
- `GOMEM_OLLAMA_URL="http://localhost:11434"`
- `GOMEM_OLLAMA_MODEL="nomic-embed-text:latest"`
- `GOMEM_KNOWLEDGE_BASE="~/www/MCP/remembrances-mcp/.serena/memories"`
- `GOMEM_SURREALDB_START_CMD="surreal start --user root --pass root surrealkv://$REPO_ROOT/surreal_data"`

### Dependencies
- Go 1.23+
- Python 3 with `requests` library
- SurrealDB running on localhost:8000
- Built binary at `dist/remembrances-mcp`

## Test Details

### stdio Test (`test_stdio.sh`)
- Tests default stdio transport
- Uses Python MCP client (`mcp_stdio_client.py`)
- Verifies tool listing and basic operations
- **Status**: ✅ Working

### SSE Test (`test_sse.sh`)  
- Tests Server-Sent Events transport on port 4001
- Uses Python SSE client (`mcp_sse_client.py`)
- **Status**: ⚠️ Has session management issues in go-mcp library

### HTTP Test (`test_http.sh`)
- Tests custom HTTP JSON API transport on port 8081
- Tests endpoints: `/health`, `/mcp/tools`, `/mcp/tools/call`
- Uses Python HTTP client (`mcp_http_client.py`)
- Verifies CORS support
- **Status**: ✅ Working perfectly

## Test Output

Each test produces:
- Log files in `tests/` directory (e.g., `stdio_test.log`, `http_test.log`)
- Real-time console output showing test progress
- Success/failure status with detailed error messages

## Example Test Run

```bash
$ bash tests/test_http.sh
Starting server (HTTP) -> logging to ~/www/MCP/remembrances-mcp/tests/http_test.log
HTTP smoke test: server initialized successfully (PID=12345)
Port 8081 is listening
Testing health endpoint...
✓ Health endpoint working
Testing tools list endpoint...
✓ Tools list endpoint working
Testing tool call endpoint...
✓ Tool call endpoint working
Running python HTTP client...
Testing MCP HTTP API at http://localhost:8081
Server is ready!
Testing health endpoint...
✓ Health check passed
Testing tools list endpoint...
✓ Tools list returned 1 tools
Testing CORS preflight...
✓ CORS preflight succeeded
Testing tool call endpoint...
✓ Tool call succeeded
Test Results: 4/4 passed
✓ All HTTP tests passed!
HTTP smoke test completed successfully
```

## Troubleshooting

### Common Issues
1. **Binary not found**: Run `bash tests/build.sh` first
2. **Port conflicts**: Tests use different ports (stdio=default, SSE=4001, HTTP=8081)
3. **SurrealDB not running**: Ensure SurrealDB is running on localhost:8000
4. **Python dependencies**: Install `requests` library (`pip install requests`)

### Manual Testing
```bash
# Test HTTP endpoints manually
curl http://localhost:8081/health
curl http://localhost:8081/mcp/tools
curl -X POST http://localhost:8081/mcp/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name": "remembrance_save_fact", "arguments": {"key": "test", "value": "value"}}'
```