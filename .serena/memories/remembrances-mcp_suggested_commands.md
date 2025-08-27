Quick commands (Linux, bash) for working on remembrances-mcp

Setup & deps:
- go mod tidy
- go test ./...           # run all tests
- go vet ./...            # static checks
- go fmt ./...            # format

Build & run:
- go build -o remembrances-mcp ./cmd/remembrances-mcp
- GOMEM_OPENAI_KEY=sk-xxx GOMEM_DB_PATH=./data.db go run ./cmd/remembrances-mcp/main.go --knowledge-base ./kb --rest-api-serve
- Run with SSE transport: --sse (optionally set GOMEM_SSE_ADDR)

Run quick embedded DB dev mode:
- GOMEM_OPENAI_KEY=sk-xxx go run ./cmd/remembrances-mcp/main.go --db-path ./data.db

Inspect & debugging:
- tail -f remembrances.log
- grep -R "TODO" -n .
- go test ./... -run TestName

Git & useful shell:
- git status
- git add . && git commit -m "msg"
- git diff origin/main..HEAD
- ls -la, cd, find . -name "*.go"

Notes:
- Env var prefix: `GOMEM_` (flags -> env mapping). Examples: `GOMEM_SURREALDB_NAMESPACE`, `GOMEM_SURREALDB_DATABASE`, `GOMEM_OPENAI_KEY`.
- When you change MTREE index dimension, update embedder.Dimension() or vice versa.

