---
title: "Examples"
linkTitle: "Examples"
weight: 5
description: >
  Practical examples and use cases for Remembrances MCP
---

## Basic Usage Examples

### Storing and Retrieving Facts

Store simple key-value facts that your AI agent can recall later:

```bash
# Store user preferences
./remembrances-mcp --gguf-model-path ./model.gguf << 'EOF'
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"remembrance_save_fact","arguments":{"key":"user_timezone","value":"Europe/Madrid"}}}
EOF

# Retrieve the fact
./remembrances-mcp --gguf-model-path ./model.gguf << 'EOF'
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"remembrance_get_fact","arguments":{"key":"user_timezone"}}}
EOF
```

### Semantic Search

Store information and search it semantically:

```bash
# Store multiple pieces of information
./remembrances-mcp --gguf-model-path ./model.gguf << 'EOF'
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"remembrance_store","arguments":{"content":"The user prefers dark mode in all applications"}}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"remembrance_store","arguments":{"content":"Meeting scheduled with John for project review on Friday"}}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"remembrance_store","arguments":{"content":"The user's favorite programming language is Python"}}}
EOF

# Search for relevant memories
./remembrances-mcp --gguf-model-path ./model.gguf << 'EOF'
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"remembrance_search","arguments":{"query":"What are the user's preferences?","limit":5}}}
EOF
```

## Knowledge Base Examples

### Building a Documentation Knowledge Base

Create a knowledge base from your project documentation:

```bash
# Add documentation files
./remembrances-mcp --gguf-model-path ./model.gguf << 'EOF'
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"kb_add_document","arguments":{"path":"docs/installation.md","content":"# Installation\n\nTo install the project, run:\n\n```bash\nmake build\n```\n\nThis will compile all dependencies."}}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"kb_add_document","arguments":{"path":"docs/configuration.md","content":"# Configuration\n\nThe application can be configured using environment variables:\n\n- `DB_PATH`: Path to the database\n- `LOG_LEVEL`: Logging verbosity"}}}
EOF

# Search the knowledge base
./remembrances-mcp --gguf-model-path ./model.gguf << 'EOF'
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"kb_search","arguments":{"query":"How do I configure the database?","limit":3}}}
EOF
```

### Research Assistant

Build a knowledge base from research papers and notes:

```python
import asyncio
from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client

async def build_research_kb():
    server_params = StdioServerParameters(
        command="./remembrances-mcp",
        args=["--gguf-model-path", "./model.gguf"]
    )
    
    async with stdio_client(server_params) as (read, write):
        async with ClientSession(read, write) as session:
            await session.initialize()
            
            # Add research notes
            papers = [
                {
                    "path": "papers/attention.md",
                    "content": "# Attention Is All You Need\n\nThe Transformer architecture relies on self-attention mechanisms..."
                },
                {
                    "path": "papers/bert.md", 
                    "content": "# BERT: Pre-training of Deep Bidirectional Transformers\n\nBERT uses masked language modeling..."
                }
            ]
            
            for paper in papers:
                await session.call_tool(
                    "kb_add_document",
                    arguments=paper
                )
            
            # Search for relevant papers
            results = await session.call_tool(
                "kb_search",
                arguments={
                    "query": "transformer attention mechanisms",
                    "limit": 5
                }
            )
            print(results)

asyncio.run(build_research_kb())
```

## Graph Database Examples

### Building Entity Relationships

Create a graph of entities and their relationships:

```bash
# Create entities and relationships
./remembrances-mcp --gguf-model-path ./model.gguf << 'EOF'
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"graph_add_entity","arguments":{"name":"Alice","type":"Person","properties":{"role":"Developer"}}}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"graph_add_entity","arguments":{"name":"ProjectX","type":"Project","properties":{"status":"Active"}}}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"graph_add_relation","arguments":{"from":"Alice","to":"ProjectX","relation":"works_on"}}}
EOF

# Query relationships
./remembrances-mcp --gguf-model-path ./model.gguf << 'EOF'
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"graph_query","arguments":{"query":"SELECT * FROM works_on WHERE in.name = 'Alice'"}}}
EOF
```

### Team Knowledge Graph

```python
async def build_team_graph(session):
    # Add team members
    team = [
        {"name": "Alice", "type": "Person", "properties": {"role": "Tech Lead"}},
        {"name": "Bob", "type": "Person", "properties": {"role": "Developer"}},
        {"name": "Carol", "type": "Person", "properties": {"role": "Designer"}}
    ]
    
    for member in team:
        await session.call_tool("graph_add_entity", arguments=member)
    
    # Add projects
    projects = [
        {"name": "Website", "type": "Project", "properties": {"deadline": "2025-12-01"}},
        {"name": "Mobile App", "type": "Project", "properties": {"deadline": "2025-06-15"}}
    ]
    
    for project in projects:
        await session.call_tool("graph_add_entity", arguments=project)
    
    # Create relationships
    relationships = [
        {"from": "Alice", "to": "Website", "relation": "leads"},
        {"from": "Bob", "to": "Website", "relation": "works_on"},
        {"from": "Carol", "to": "Website", "relation": "designs"},
        {"from": "Alice", "to": "Mobile App", "relation": "works_on"},
        {"from": "Bob", "to": "Mobile App", "relation": "works_on"}
    ]
    
    for rel in relationships:
        await session.call_tool("graph_add_relation", arguments=rel)
```

## Integration Examples

### Claude Desktop Integration

Configure Claude Desktop to use Remembrances MCP:

**macOS/Linux** (`~/.config/claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "remembrances": {
      "command": "/usr/local/bin/remembrances-mcp",
      "args": [
        "--gguf-model-path",
        "/path/to/nomic-embed-text-v1.5.Q4_K_M.gguf",
        "--gguf-gpu-layers",
        "32",
        "--db-path",
        "~/.local/share/remembrances/claude.db"
      ]
    }
  }
}
```

### HTTP API Integration

Run the server in HTTP mode for REST API access:

```bash
# Start server in HTTP mode
./remembrances-mcp \
  --gguf-model-path ./model.gguf \
  --http \
  --http-addr ":8080"
```

Then use cURL or any HTTP client:

```bash
# Store a memory
curl -X POST http://localhost:8080/mcp/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "remembrance_store",
    "arguments": {
      "content": "Important meeting notes from today"
    }
  }'

# Search memories
curl -X POST http://localhost:8080/mcp/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "remembrance_search",
    "arguments": {
      "query": "meeting notes",
      "limit": 10
    }
  }'
```

## Use Case: Personal AI Assistant

A complete example of building a personal AI assistant with memory:

```python
import asyncio
from datetime import datetime
from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client

class PersonalAssistant:
    def __init__(self, session):
        self.session = session
    
    async def remember_preference(self, category, value):
        """Store a user preference"""
        await self.session.call_tool(
            "remembrance_save_fact",
            arguments={
                "key": f"preference_{category}",
                "value": value
            }
        )
    
    async def get_preference(self, category):
        """Retrieve a user preference"""
        result = await self.session.call_tool(
            "remembrance_get_fact",
            arguments={"key": f"preference_{category}"}
        )
        return result
    
    async def log_interaction(self, summary):
        """Log an interaction for future reference"""
        await self.session.call_tool(
            "remembrance_store",
            arguments={
                "content": f"[{datetime.now().isoformat()}] {summary}"
            }
        )
    
    async def recall_context(self, topic, limit=5):
        """Recall relevant context about a topic"""
        result = await self.session.call_tool(
            "remembrance_search",
            arguments={
                "query": topic,
                "limit": limit
            }
        )
        return result

async def main():
    server_params = StdioServerParameters(
        command="./remembrances-mcp",
        args=["--gguf-model-path", "./model.gguf"]
    )
    
    async with stdio_client(server_params) as (read, write):
        async with ClientSession(read, write) as session:
            await session.initialize()
            
            assistant = PersonalAssistant(session)
            
            # Store preferences
            await assistant.remember_preference("language", "English")
            await assistant.remember_preference("timezone", "UTC")
            
            # Log interactions
            await assistant.log_interaction("User asked about weather in Madrid")
            await assistant.log_interaction("User scheduled meeting for Friday")
            
            # Recall context
            context = await assistant.recall_context("schedule meetings")
            print("Relevant context:", context)

asyncio.run(main())
```

## See Also

- [Getting Started](../getting-started/) - Installation guide
- [Configuration](../configuration/) - Server configuration
- [MCP API](../mcp-api/) - Available tools reference
- [Troubleshooting](../troubleshooting/) - Common issues