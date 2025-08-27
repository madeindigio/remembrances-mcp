#!/usr/bin/env bash
set -euo pipefail

# Smoke test for SSE transport. Starts the server with --sse and a custom
# address, waits for initialization log lines, then shuts it down.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BINARY="$REPO_ROOT/dist/remembrances-mcp"
LOG="$SCRIPT_DIR/sse_test.log"

export GOMEM_OPENAI_KEY="testkey"
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

# Start server with --sse flag
"$BINARY" --sse >"$LOG" 2>&1 &
PID=$!

cleanup() {
  echo "Cleaning up SSE test..."
  if kill -0 "$PID" 2>/dev/null; then
    kill -INT "$PID" || true
    wait "$PID" || true
  fi
}
trap cleanup EXIT

# Wait for the SSE-enabled message
TIMEOUT=20
SLEPT=0
INTERVAL=1
FOUND=0
while [[ $SLEPT -lt $TIMEOUT ]]; do
  if grep -q "SSE transport enabled" "$LOG" 2>/dev/null && grep -q "Remembrances-MCP server initialized successfully" "$LOG" 2>/dev/null; then
    FOUND=1
    break
  fi
  sleep $INTERVAL
  SLEPT=$((SLEPT + INTERVAL))
done

if [[ $FOUND -eq 1 ]]; then
  echo "SSE smoke test: server initialized successfully (PID=$PID)"
else
  echo "SSE smoke test failed: server did not initialize within ${TIMEOUT}s" >&2
  echo "Last 200 lines of log:" >&2
  tail -n 200 "$LOG" >&2 || true
  exit 3
fi

# Optionally check the port is listening (best-effort)
if command -v ss >/dev/null 2>&1; then
  if ss -ltn sport = :4001 | grep -q LISTEN; then
    echo "Port 4001 is listening"
  else
    echo "Warning: port 4001 not observed as LISTEN. The transport may not bind yet or uses a different mechanism." >&2
  fi
fi

# Shut down
kill -INT "$PID"
wait "$PID" || true

echo "sse smoke test completed successfully"
