---
title: "MCP API & Tools"
linkTitle: "MCP API"
weight: 4
description: >
  Available MCP tools and API endpoints
---

## MCP Tools

Remembrances MCP provides the following tools for AI agents:

### Memory Tools

#### `remembrance_save_fact`

Save a simple key-value fact to memory.

**Arguments**:
- `key` (string): Unique identifier for the fact
- `value` (string): The fact content

**Example**:
```json
{
  "name": "remembrance_save_fact",
  "arguments": {
    "key": "user_name",
    "value": "Alice"
  }
}
```

#### `remembrance_get_fact`

Retrieve a fact by its key.

**Arguments**:
- `key` (string): The fact identifier

**Returns**: The fact value or null if not found

#### `remembrance_search`

Semantic search across all stored memories using embeddings.

**Arguments**:
- `query` (string): Search query
- `limit` (integer, optional): Maximum results (default: 10)

**Returns**: Array of relevant memories with similarity scores

### Knowledge Base Tools

#### `kb_add_document`

Add a document to the knowledge base.

**Arguments**:
- `path` (string): Document path
- `content` (string): Document content
- `metadata` (object, optional): Additional metadata

#### `kb_search`

Search the knowledge base.

**Arguments**:
- `query` (string): Search query
- `limit` (integer, optional): Maximum results

**Returns**: Relevant documents with scores

### Graph Tools

#### `graph_add_relation`

Create a relationship between entities.

**Arguments**:
- `from` (string): Source entity
- `to` (string): Target entity
- `relation` (string): Relationship type

#### `graph_query`

Query the graph database.

**Arguments**:
- `query` (string): SurrealDB query
- `params` (object, optional): Query parameters

## HTTP API

When running with `--http`, the server exposes REST endpoints:

### Health Check

```bash
GET /health
```

**Response**:
```json
{
  "status": "ok",
  "version": "1.0.0"
}
```

### List Tools

```bash
GET /mcp/tools
```

**Response**:
```json
{
  "tools": [
    {
      "name": "remembrance_save_fact",
      "description": "Save a fact to memory",
      "inputSchema": {...}
    },
    ...
  ]
}
```

### Call Tool

```bash
POST /mcp/tools/call
Content-Type: application/json

{
  "name": "remembrance_save_fact",
  "arguments": {
    "key": "test",
    "value": "example"
  }
}
```

**Response**:
```json
{
  "content": [
    {
      "type": "text",
      "text": "Fact saved successfully"
    }
  ]
}
```

## Integration Examples

### Claude Desktop

Add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "remembrances": {
      "command": "/path/to/remembrances-mcp",
      "args": [
        "--gguf-model-path",
        "/path/to/model.gguf",
        "--gguf-gpu-layers",
        "32"
      ]
    }
  }
}
```

### Python MCP Client

```python
from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client

server_params = StdioServerParameters(
    command="/path/to/remembrances-mcp",
    args=["--gguf-model-path", "/path/to/model.gguf"]
)

async with stdio_client(server_params) as (read, write):
    async with ClientSession(read, write) as session:
        await session.initialize()
        
        # Save a fact
        result = await session.call_tool(
            "remembrance_save_fact",
            arguments={"key": "test", "value": "Hello"}
        )
        
        # Search
        results = await session.call_tool(
            "remembrance_search",
            arguments={"query": "test", "limit": 5}
        )
```

### cURL Examples

```bash
# Save a fact
curl -X POST http://localhost:8080/mcp/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "remembrance_save_fact",
    "arguments": {"key": "user_name", "value": "Alice"}
  }'

# Search
curl -X POST http://localhost:8080/mcp/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "remembrance_search",
    "arguments": {"query": "Alice", "limit": 10}
  }'
```

## See Also

- [Configuration](../configuration/) - Server configuration
- [Getting Started](../getting-started/) - Installation guide
