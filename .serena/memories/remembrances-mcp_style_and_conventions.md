Code style & conventions (project notes)

Language & tools:
- Go modules; standard Go project layout. Use `go fmt` / `gofmt` and `go vet`.
- Tests: `go test ./...`; keep tests small and table-driven when appropriate.

Naming & structure:
- Packages are lowercase, clear names: `internal/config`, `pkg/embedder`, `pkg/mcp_tools`.
- Structs use exported names where needed (ToolManager, Embedder) and JSON tags for input structs.
- Error handling: return errors up the stack, handlers convert to fmt.Errorf and return to MCP runtime.

Logging:
- Uses `log/slog` configured in `internal/config.SetupLogging` to write to stdout and optionally a file (multi-writer). Use structured logging where possible.

MCP tool patterns:
- Tool factory: protocol.NewTool("tool_name", "desc", InputStruct{})
- Handler signature: func(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error)
- Arguments: json.Unmarshal(request.RawArguments, &input)
- Return: protocol.NewCallToolResult([]protocol.Content{...}, false)

Embedder contract:
- `EmbedDocuments(ctx, []string) ([][]float32, error)`
- `EmbedQuery(ctx, string) ([]float32, error)`
- `Dimension() int` — must match SurrealDB MTREE index dimension (schema uses 768 by default).

Config conventions:
- CLI flags are defined with pflag and bound to viper. Environment variables use prefix `GOMEM_` and dashes map to underscores.

Tests & CI tips:
- When editing storage/schema, prefer using embedded SurrealDB with `GOMEM_DB_PATH` and run tests locally.
- Mock embedder in tests to avoid external API calls.

