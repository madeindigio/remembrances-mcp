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
