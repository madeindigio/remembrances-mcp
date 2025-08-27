# Tests folder

This `tests/` directory contains small smoke-test scripts and a build helper for the `remembrances-mcp` project.

Files:

- `build.sh` - Builds the project using `xc build` and writes the binary to `./dist/remembrances-mcp` as requested.
- `test_stdio.sh` - Starts the server in default stdio transport, waits for a successful initialization message in the log, then gracefully shuts it down.
- `test_sse.sh` - Starts the server with the `--sse` flag on port `:4001`, waits for initialization, then shuts it down.
- `run_all.sh` - Runs `build.sh` then both smoke tests in sequence.

Notes:
- The project's configuration validation requires either an Ollama model or an OpenAI key. The smoke tests set a placeholder `GOMEM_OPENAI_KEY` environment variable to satisfy validation without making external API calls. Adjust environment variables in the scripts if you want to test real integrations.
- These tests perform only process-level smoke checks (looking for known log lines). They do not exercise the MCP protocol or actual embedder calls.
- Make scripts executable before running: `chmod +x tests/*.sh`.

Usage example:

```bash
# from the repository root
chmod +x tests/*.sh
./tests/run_all.sh
```

## MCP clients

Two Python example clients are provided under `tests/clients`:

- `mcp_stdio_client.py` -- starts the binary and communicates over stdio using JSON-RPC (tools/list, tools/call examples).
- `mcp_sse_client.py` -- starts the binary with `--sse` (or connects to a running SSE server) and uses the SSE + POST endpoints to call tools.

Both scripts are small examples to exercise the tool handlers. They expect the binary built at `./dist/remembrances-mcp` (use `tests/build.sh`).

Requirements:

- Python 3
- `requests` package (for the SSE client): `pip install requests`

Examples:

```bash
# Build the binary
chmod +x tests/build.sh && ./tests/build.sh

# Run the stdio client (will start the binary)
python3 tests/clients/mcp_stdio_client.py --binary ./dist/remembrances-mcp

# Run the SSE client (will start the binary)
python3 tests/clients/mcp_sse_client.py --binary ./dist/remembrances-mcp --start
```
