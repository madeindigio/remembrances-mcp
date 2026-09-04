# API documents

Generated artefacts. Do not edit them by hand: they are produced from the route
registrations in `internal/transport`, and a test fails the build when they
stop matching the server.

| File | Build | Contents |
|---|---|---|
| `remembrances.openapi.json` | no tags | OpenAPI 3.1 document of the core HTTP surface |
| `remembrances.postman_collection.json` | no tags | Postman v2.1.0 collection of the same operations |
| `remembrances-commercial.openapi.json` | `-tags commercial` | the core surface plus the routes the commercial modules add |
| `remembrances-commercial.postman_collection.json` | `-tags commercial` | the same, as a collection |

## Regenerate

```bash
go generate ./internal/transport/...          # core documents
go run -tags commercial ./cmd/openapi-gen -out docs/api   # commercial documents
```

## The test that keeps them honest

`cmd/openapi-gen` regenerates in memory and compares byte for byte with the
files here:

```bash
go test ./cmd/openapi-gen/...
go test -tags commercial ./cmd/openapi-gen/...
```

`internal/transport` proves the other half, that the route table is the mux:
every documented operation of this process is served, every method the document
does not claim answers 405, and a wildcard route is documented as a path
parameter.

```bash
go test ./internal/transport/...
```

## Run the collection

The collection uses three variables: `base` (the HTTP JSON API listener),
`mcpBase` (the MCP Streamable HTTP listener) and `token` (a Control-issued JWT,
needed only for authenticated module routes).

```bash
remembrances-mcp --http --http-addr :8080 --mcp-http --mcp-http-addr 3000 &
newman run docs/api/remembrances.postman_collection.json \
  --env-var base=http://localhost:8080 \
  --env-var mcpBase=http://localhost:3000
```

## What the documents do not describe

The MCP tool surface itself. The `/mcp` operations are the Streamable HTTP
binding, whose bodies are JSON-RPC messages; the tools those messages name are
described by the MCP `tools/list` response, not by OpenAPI. The `/mcp/tools`
endpoints of the JSON API are a placeholder that does not reach the MCP tool
registry, and the document says so rather than implying a second tool API.
