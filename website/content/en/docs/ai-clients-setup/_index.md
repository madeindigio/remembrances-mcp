---
title: "AI Clients Setup"
linkTitle: "AI Clients Setup"
weight: 3
description: >
  Configuration in AI clients
---

## Integration with Main AI Clients

### Claude Desktop

Add to `claude_desktop_config.json`:

The --config arguments are only necessary if you want to use a custom configuration file. By default the installation through the installation script creates the configuration file in the default path and this parameter is not necessary.
```json
{
  "mcpServers": {
    "remembrances": {
      "command": "/path/to/remembrances-mcp",
      "args": [
        "--config",
        "/path/to/config.yaml"
      ]
    }
  }
}
```

Simpler:
```json
{
  "mcpServers": {
    "remembrances": {
      "command": "/path/to/remembrances-mcp",
      "args": []
    }
  }
}
```

## Github Copilot

Add in your mcp.json
```json
{
  "servers": {
    "remembrances": {
      "command": "/path/to/remembrances-mcp",
      "args": []
    }
  }
}
```

### MCP Client in Python

```python
from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client

server_params = StdioServerParameters(
    command="/path/to/remembrances-mcp",
    args=["--config", "/path/to/config.yaml"]
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
