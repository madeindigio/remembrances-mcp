#!/usr/bin/env bash
set -euo pipefail

# Smoke test for HTTP JSON API transport. Starts the server with --http and a custom
# address, waits for initialization log lines, then tests the HTTP endpoints.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BINARY="$REPO_ROOT/dist/remembrances-mcp"
LOG="$SCRIPT_DIR/http_test.log"

export GOMEM_OPENAI_KEY="testkey"
export GOMEM_HTTP="true"
export GOMEM_HTTP_ADDR=":8081"
export GOMEM_DB_PATH="$REPO_ROOT/surreal_data"
export GOMEM_LOG="$LOG"
export GOMEM_SURREALDB_URL="ws://localhost:8000"
export GOMEM_SURREALDB_USER="root"
export GOMEM_SURREALDB_PASS="root"
export GOMEM_OLLAMA_URL="http://localhost:11434"
export GOMEM_OLLAMA_MODEL="nomic-embed-text:latest"
export GOMEM_KNOWLEDGE_BASE="/www/MCP/remembrances-mcp/.serena/memories"
export GOMEM_SURREALDB_START_CMD="surreal start --user root --pass root surrealkv://$REPO_ROOT/surreal_data"

if [[ ! -x "$BINARY" ]]; then
  echo "Binary not found or not executable at $BINARY. Run tests/build.sh first." >&2
  exit 2
fi

# Start server with --http flag
echo "Starting server (HTTP) -> logging to $LOG"
"$BINARY" --http >"$LOG" 2>&1 &
PID=$!

cleanup() {
  echo "Cleaning up HTTP test..."
  if kill -0 "$PID" 2>/dev/null; then
    kill -INT "$PID" || true
    wait "$PID" || true
  fi
}
trap cleanup EXIT

# Wait for the HTTP-enabled message
TIMEOUT=20
SLEPT=0
INTERVAL=1
FOUND=0
while [[ $SLEPT -lt $TIMEOUT ]]; do
  if grep -q "HTTP transport enabled" "$LOG" 2>/dev/null && grep -q "Remembrances-MCP server initialized successfully" "$LOG" 2>/dev/null; then
    FOUND=1
    break
  fi
  sleep $INTERVAL
  SLEPT=$((SLEPT + INTERVAL))
done

if [[ $FOUND -eq 1 ]]; then
  echo "HTTP smoke test: server initialized successfully (PID=$PID)"
else
  echo "HTTP smoke test failed: server did not initialize within ${TIMEOUT}s" >&2
  echo "Last 200 lines of log:" >&2
  tail -n 200 "$LOG" >&2 || true
  exit 3
fi

# Check if the port is listening
if command -v ss >/dev/null 2>&1; then
  # derive port from GOMEM_HTTP_ADDR (formats like :8081 or 0.0.0.0:8081)
  ADDR=${GOMEM_HTTP_ADDR:-":8081"}
  PORT=$(echo "$ADDR" | awk -F: '{print $NF}')
  if ss -ltn sport = :${PORT} | grep -q LISTEN; then
    echo "Port ${PORT} is listening"
  else
    echo "Warning: port ${PORT} not observed as LISTEN." >&2
    exit 4
  fi
fi

# Test the HTTP endpoints using curl
BASE_URL="http://localhost:${PORT}"

echo "Testing health endpoint..."
HEALTH_RESPONSE=$(curl -s "${BASE_URL}/health" || echo "ERROR")
if echo "$HEALTH_RESPONSE" | grep -q '"status":"ok"'; then
  echo "✓ Health endpoint working"
else
  echo "✗ Health endpoint failed: $HEALTH_RESPONSE" >&2
  exit 5
fi

echo "Testing tools list endpoint..."
TOOLS_RESPONSE=$(curl -s "${BASE_URL}/mcp/tools" || echo "ERROR")
if echo "$TOOLS_RESPONSE" | grep -q '"tools"'; then
  echo "✓ Tools list endpoint working"
else
  echo "✗ Tools list endpoint failed: $TOOLS_RESPONSE" >&2
  exit 6
fi

echo "Testing tool call endpoint..."
CALL_PAYLOAD='{"name": "test_tool", "arguments": {}}'
CALL_RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" -d "$CALL_PAYLOAD" "${BASE_URL}/mcp/tools/call" || echo "ERROR")
if echo "$CALL_RESPONSE" | grep -q '"content"'; then
  echo "✓ Tool call endpoint working"
else
  echo "✗ Tool call endpoint failed: $CALL_RESPONSE" >&2
  exit 7
fi

# Optionally run the Python HTTP client
echo "Running python HTTP client..."
python3 tests/clients/mcp_http_client.py --base-url "$BASE_URL"

# Shut down
echo "Shutting down server..."
if kill -0 "$PID" 2>/dev/null; then
  kill -INT "$PID" || true
  wait "$PID" || true
else
  echo "Server already stopped (PID=$PID)"
fi

echo "HTTP smoke test completed successfully"
