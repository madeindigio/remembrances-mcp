#!/usr/bin/env bash
set -euo pipefail

# Simple smoke test that starts the server in stdio mode (default), waits for
# initialization log line, then sends SIGINT to shut it down.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
#BINARY="$REPO_ROOT/dist/remembrances-mcp"
BINARY="$REPO_ROOT/build/remembrances-mcp"
LOG="$SCRIPT_DIR/stdio_test.log"

# Minimal env to satisfy config validation (no external network calls are made
# by this smoke test). Adjust as needed for your environment.
export GOMEM_OPENAI_KEY="testkey"
export GOMEM_DB_PATH="$REPO_ROOT/surreal_data"
export GOMEM_LOG="$SCRIPT_DIR/stdio_test.log"
#export GOMEM_SURREALDB_URL="ws://localhost:8000"
# Unset GOMEM_SURREALDB_URL to ensure embedded mode
unset GOMEM_SURREALDB_URL 2>/dev/null || true

export GOMEM_SURREALDB_USER="root"
export GOMEM_SURREALDB_PASS="root"
export GOMEM_SURREALDB_NAMESPACE="test"
export GOMEM_SURREALDB_DATABASE="test"
# For ollama
#export GOMEM_OLLAMA_URL="http://localhost:11434"
#export GOMEM_OLLAMA_MODEL="nomic-embed-text:latest"
# For local embeddings
export GGUF_MODEL_PATH="/www/Remembrances/nomic-embed-text-v1.5.Q4_K_M.gguf"

export GOMEM_KNOWLEDGE_BASE="/www/MCP/remembrances-mcp/.serena/memories"
#export GOMEM_SURREALDB_START_CMD="surreal start --user root --pass root surrealkv://$REPO_ROOT/surreal_data"

if [[ ! -x "$BINARY" ]]; then
  echo "Binary not found or not executable at $BINARY. Run tests/build.sh first." >&2
  exit 2
fi

echo "Starting server (stdio) -> logging to $LOG"
"$BINARY" >"$LOG" 2>&1 &
PID=$!


cleanup() {
  echo "Cleaning up..."
  if kill -0 "$PID" 2>/dev/null; then
    kill -INT "$PID" || true
    wait "$PID" || true
  fi
}
trap cleanup EXIT

# Wait for initialization message (timeout)
TIMEOUT=20
SLEPT=0
INTERVAL=1
FOUND=0
while [[ $SLEPT -lt $TIMEOUT ]]; do
  if grep -q "Remembrances-MCP server initialized successfully" "$LOG" 2>/dev/null; then
    FOUND=1
    break
  fi
  sleep $INTERVAL
  SLEPT=$((SLEPT + INTERVAL))
done

if [[ $FOUND -eq 1 ]]; then
  echo "Smoke test: server initialized successfully (PID=$PID)"
else
  echo "Smoke test failed: server did not initialize within ${TIMEOUT}s" >&2
  echo "Last 200 lines of log:" >&2
  tail -n 200 "$LOG" >&2 || true
  exit 3
fi


echo "Running python client"
python3 tests/clients/mcp_stdio_client.py --binary $BINARY

# Gracefully stop the server
if kill -0 "$PID" 2>/dev/null; then
  echo "Stopping server (PID=$PID)"
  kill -INT "$PID" || true
  wait "$PID" || true
else
  echo "Server already stopped (PID=$PID)"
fi

echo "stdio smoke test completed successfully"
