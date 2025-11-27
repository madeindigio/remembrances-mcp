---
title: "Configuración de clientes AI"
linkTitle: "Clientes AI"
weight: 3
description: >
  Configuración en los clientes AI
---

## Integración con los principales clientes AI

### Claude Desktop

Añade a `claude_desktop_config.json`:

Los argumentos --config sólo es necesario si quieres usar un archivo de configuración personalizado. Por defecto la instalación a través del script de instalación crea el archivo de configuración en la ruta por defecto y no es necesario este parámetro.
```json
{
  "mcpServers": {
    "remembrances": {
      "command": "/ruta/a/remembrances-mcp",
      "args": [
        "--config",
        "/ruta/a/config.yaml"
      ]
    }
  }
}
```

Más simple:
```json
{
  "mcpServers": {
    "remembrances": {
      "command": "/ruta/a/remembrances-mcp",
      "args": []
    }
  }
}
```

## Github Copilot

Añade en tu mcp.json
```json
{
  "servers": {
    "remembrances": {
      "command": "/ruta/a/remembrances-mcp",
      "args": []
    }
  }
}
```

### Cliente MCP en Python

```python
from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client

server_params = StdioServerParameters(
    command="/ruta/a/remembrances-mcp",
    args=["--config", "/ruta/a/config.yaml"]
)

async with stdio_client(server_params) as (read, write):
    async with ClientSession(read, write) as session:
        await session.initialize()
        
        # Guardar un hecho
        result = await session.call_tool(
            "remembrance_save_fact",
            arguments={"key": "prueba", "value": "Hola"}
        )
        
        # Buscar
        results = await session.call_tool(
            "remembrance_search",
            arguments={"query": "prueba", "limit": 5}
        )
```
