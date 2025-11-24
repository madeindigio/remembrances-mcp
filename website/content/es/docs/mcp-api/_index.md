---
title: "API MCP y Herramientas"
linkTitle: "API MCP"
weight: 4
description: >
  Herramientas MCP y endpoints API disponibles
---

## Herramientas MCP

Remembrances MCP proporciona las siguientes herramientas para agentes IA:

### Herramientas de Memoria

#### `remembrance_save_fact`

Guarda un hecho simple clave-valor en memoria.

**Argumentos**:
- `key` (string): Identificador único para el hecho
- `value` (string): El contenido del hecho

**Ejemplo**:
```json
{
  "name": "remembrance_save_fact",
  "arguments": {
    "key": "nombre_usuario",
    "value": "Alicia"
  }
}
```

#### `remembrance_get_fact`

Recupera un hecho por su clave.

**Argumentos**:
- `key` (string): El identificador del hecho

**Retorna**: El valor del hecho o null si no se encuentra

#### `remembrance_search`

Búsqueda semántica a través de todas las memorias almacenadas usando embeddings.

**Argumentos**:
- `query` (string): Consulta de búsqueda
- `limit` (integer, opcional): Resultados máximos (por defecto: 10)

**Retorna**: Array de memorias relevantes con puntuaciones de similitud

### Herramientas de Base de Conocimiento

#### `kb_add_document`

Añade un documento a la base de conocimiento.

**Argumentos**:
- `path` (string): Ruta del documento
- `content` (string): Contenido del documento
- `metadata` (object, opcional): Metadatos adicionales

#### `kb_search`

Busca en la base de conocimiento.

**Argumentos**:
- `query` (string): Consulta de búsqueda
- `limit` (integer, opcional): Resultados máximos

**Retorna**: Documentos relevantes con puntuaciones

### Herramientas de Grafos

#### `graph_add_relation`

Crea una relación entre entidades.

**Argumentos**:
- `from` (string): Entidad origen
- `to` (string): Entidad destino
- `relation` (string): Tipo de relación

#### `graph_query`

Consulta la base de datos de grafos.

**Argumentos**:
- `query` (string): Consulta SurrealDB
- `params` (object, opcional): Parámetros de consulta

## API HTTP

Al ejecutar con `--http`, el servidor expone endpoints REST:

### Verificación de Salud

```bash
GET /health
```

**Respuesta**:
```json
{
  "status": "ok",
  "version": "1.0.0"
}
```

### Listar Herramientas

```bash
GET /mcp/tools
```

**Respuesta**:
```json
{
  "tools": [
    {
      "name": "remembrance_save_fact",
      "description": "Guardar un hecho en memoria",
      "inputSchema": {...}
    },
    ...
  ]
}
```

### Llamar Herramienta

```bash
POST /mcp/tools/call
Content-Type: application/json

{
  "name": "remembrance_save_fact",
  "arguments": {
    "key": "prueba",
    "value": "ejemplo"
  }
}
```

**Respuesta**:
```json
{
  "content": [
    {
      "type": "text",
      "text": "Hecho guardado exitosamente"
    }
  ]
}
```

## Ejemplos de Integración

### Claude Desktop

Añade a `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "remembrances": {
      "command": "/ruta/a/remembrances-mcp",
      "args": [
        "--gguf-model-path",
        "/ruta/a/model.gguf",
        "--gguf-gpu-layers",
        "32"
      ]
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
    args=["--gguf-model-path", "/ruta/a/model.gguf"]
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

### Ejemplos con cURL

```bash
# Guardar un hecho
curl -X POST http://localhost:8080/mcp/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "remembrance_save_fact",
    "arguments": {"key": "nombre_usuario", "value": "Alicia"}
  }'

# Buscar
curl -X POST http://localhost:8080/mcp/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "remembrance_search",
    "arguments": {"query": "Alicia", "limit": 10}
  }'
```

## Ver También

- [Configuración](../configuration/) - Configuración del servidor
- [Primeros Pasos](../getting-started/) - Guía de instalación
