---
title: "Ejemplos"
linkTitle: "Ejemplos"
weight: 5
description: >
  Ejemplos prácticos y casos de uso para Remembrances MCP
---

## Ejemplos de Uso Básico

### Almacenar y Recuperar Hechos

Almacena hechos simples clave-valor que tu agente IA puede recordar después:

```bash
# Almacenar preferencias de usuario
./remembrances-mcp --gguf-model-path ./model.gguf << 'EOF'
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"remembrance_save_fact","arguments":{"key":"zona_horaria_usuario","value":"Europe/Madrid"}}}
EOF

# Recuperar el hecho
./remembrances-mcp --gguf-model-path ./model.gguf << 'EOF'
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"remembrance_get_fact","arguments":{"key":"zona_horaria_usuario"}}}
EOF
```

### Búsqueda Semántica

Almacena información y búscala semánticamente:

```bash
# Almacenar múltiples piezas de información
./remembrances-mcp --gguf-model-path ./model.gguf << 'EOF'
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"remembrance_store","arguments":{"content":"El usuario prefiere modo oscuro en todas las aplicaciones"}}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"remembrance_store","arguments":{"content":"Reunión programada con Juan para revisión del proyecto el viernes"}}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"remembrance_store","arguments":{"content":"El lenguaje de programación favorito del usuario es Python"}}}
EOF

# Buscar memorias relevantes
./remembrances-mcp --gguf-model-path ./model.gguf << 'EOF'
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"remembrance_search","arguments":{"query":"¿Cuáles son las preferencias del usuario?","limit":5}}}
EOF
```

## Ejemplos de Base de Conocimiento

### Construyendo una Base de Conocimiento de Documentación

Crea una base de conocimiento desde la documentación de tu proyecto:

```bash
# Añadir archivos de documentación
./remembrances-mcp --gguf-model-path ./model.gguf << 'EOF'
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"kb_add_document","arguments":{"path":"docs/instalacion.md","content":"# Instalación\n\nPara instalar el proyecto, ejecuta:\n\n```bash\nmake build\n```\n\nEsto compilará todas las dependencias."}}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"kb_add_document","arguments":{"path":"docs/configuracion.md","content":"# Configuración\n\nLa aplicación puede configurarse usando variables de entorno:\n\n- `DB_PATH`: Ruta a la base de datos\n- `LOG_LEVEL`: Nivel de verbosidad de logs"}}}
EOF

# Buscar en la base de conocimiento
./remembrances-mcp --gguf-model-path ./model.gguf << 'EOF'
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"kb_search","arguments":{"query":"¿Cómo configuro la base de datos?","limit":3}}}
EOF
```

### Asistente de Investigación

Construye una base de conocimiento desde papers de investigación y notas:

```python
import asyncio
from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client

async def construir_kb_investigacion():
    server_params = StdioServerParameters(
        command="./remembrances-mcp",
        args=["--gguf-model-path", "./model.gguf"]
    )
    
    async with stdio_client(server_params) as (read, write):
        async with ClientSession(read, write) as session:
            await session.initialize()
            
            # Añadir notas de investigación
            papers = [
                {
                    "path": "papers/attention.md",
                    "content": "# Attention Is All You Need\n\nLa arquitectura Transformer se basa en mecanismos de auto-atención..."
                },
                {
                    "path": "papers/bert.md", 
                    "content": "# BERT: Pre-entrenamiento de Transformers Bidireccionales Profundos\n\nBERT usa modelado de lenguaje enmascarado..."
                }
            ]
            
            for paper in papers:
                await session.call_tool(
                    "kb_add_document",
                    arguments=paper
                )
            
            # Buscar papers relevantes
            results = await session.call_tool(
                "kb_search",
                arguments={
                    "query": "mecanismos de atención en transformers",
                    "limit": 5
                }
            )
            print(results)

asyncio.run(construir_kb_investigacion())
```

## Ejemplos de Base de Datos de Grafos

### Construyendo Relaciones entre Entidades

Crea un grafo de entidades y sus relaciones:

```bash
# Crear entidades y relaciones
./remembrances-mcp --gguf-model-path ./model.gguf << 'EOF'
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"graph_add_entity","arguments":{"name":"Alicia","type":"Persona","properties":{"rol":"Desarrolladora"}}}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"graph_add_entity","arguments":{"name":"ProyectoX","type":"Proyecto","properties":{"estado":"Activo"}}}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"graph_add_relation","arguments":{"from":"Alicia","to":"ProyectoX","relation":"trabaja_en"}}}
EOF

# Consultar relaciones
./remembrances-mcp --gguf-model-path ./model.gguf << 'EOF'
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"graph_query","arguments":{"query":"SELECT * FROM trabaja_en WHERE in.name = 'Alicia'"}}}
EOF
```

### Grafo de Conocimiento de Equipo

```python
async def construir_grafo_equipo(session):
    # Añadir miembros del equipo
    equipo = [
        {"name": "Alicia", "type": "Persona", "properties": {"rol": "Tech Lead"}},
        {"name": "Roberto", "type": "Persona", "properties": {"rol": "Desarrollador"}},
        {"name": "Carmen", "type": "Persona", "properties": {"rol": "Diseñadora"}}
    ]
    
    for miembro in equipo:
        await session.call_tool("graph_add_entity", arguments=miembro)
    
    # Añadir proyectos
    proyectos = [
        {"name": "Website", "type": "Proyecto", "properties": {"fecha_limite": "2025-12-01"}},
        {"name": "App Móvil", "type": "Proyecto", "properties": {"fecha_limite": "2025-06-15"}}
    ]
    
    for proyecto in proyectos:
        await session.call_tool("graph_add_entity", arguments=proyecto)
    
    # Crear relaciones
    relaciones = [
        {"from": "Alicia", "to": "Website", "relation": "lidera"},
        {"from": "Roberto", "to": "Website", "relation": "trabaja_en"},
        {"from": "Carmen", "to": "Website", "relation": "diseña"},
        {"from": "Alicia", "to": "App Móvil", "relation": "trabaja_en"},
        {"from": "Roberto", "to": "App Móvil", "relation": "trabaja_en"}
    ]
    
    for rel in relaciones:
        await session.call_tool("graph_add_relation", arguments=rel)
```

## Ejemplos de Integración

### Integración con Claude Desktop

Configura Claude Desktop para usar Remembrances MCP:

**macOS/Linux** (`~/.config/claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "remembrances": {
      "command": "/usr/local/bin/remembrances-mcp",
      "args": [
        "--gguf-model-path",
        "/ruta/a/nomic-embed-text-v1.5.Q4_K_M.gguf",
        "--gguf-gpu-layers",
        "32",
        "--db-path",
        "~/.local/share/remembrances/claude.db"
      ]
    }
  }
}
```

### Integración con API HTTP

Ejecuta el servidor en modo HTTP para acceso REST API:

```bash
# Iniciar servidor en modo HTTP
./remembrances-mcp \
  --gguf-model-path ./model.gguf \
  --http \
  --http-addr ":8080"
```

Luego usa cURL o cualquier cliente HTTP:

```bash
# Almacenar una memoria
curl -X POST http://localhost:8080/mcp/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "remembrance_store",
    "arguments": {
      "content": "Notas importantes de la reunión de hoy"
    }
  }'

# Buscar memorias
curl -X POST http://localhost:8080/mcp/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "remembrance_search",
    "arguments": {
      "query": "notas de reunión",
      "limit": 10
    }
  }'
```

## Caso de Uso: Asistente IA Personal

Un ejemplo completo de construir un asistente IA personal con memoria:

```python
import asyncio
from datetime import datetime
from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client

class AsistentePersonal:
    def __init__(self, session):
        self.session = session
    
    async def recordar_preferencia(self, categoria, valor):
        """Almacena una preferencia del usuario"""
        await self.session.call_tool(
            "remembrance_save_fact",
            arguments={
                "key": f"preferencia_{categoria}",
                "value": valor
            }
        )
    
    async def obtener_preferencia(self, categoria):
        """Recupera una preferencia del usuario"""
        result = await self.session.call_tool(
            "remembrance_get_fact",
            arguments={"key": f"preferencia_{categoria}"}
        )
        return result
    
    async def registrar_interaccion(self, resumen):
        """Registra una interacción para referencia futura"""
        await self.session.call_tool(
            "remembrance_store",
            arguments={
                "content": f"[{datetime.now().isoformat()}] {resumen}"
            }
        )
    
    async def recordar_contexto(self, tema, limite=5):
        """Recuerda contexto relevante sobre un tema"""
        result = await self.session.call_tool(
            "remembrance_search",
            arguments={
                "query": tema,
                "limit": limite
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
            
            asistente = AsistentePersonal(session)
            
            # Almacenar preferencias
            await asistente.recordar_preferencia("idioma", "Español")
            await asistente.recordar_preferencia("zona_horaria", "Europe/Madrid")
            
            # Registrar interacciones
            await asistente.registrar_interaccion("Usuario preguntó sobre el clima en Madrid")
            await asistente.registrar_interaccion("Usuario programó reunión para el viernes")
            
            # Recordar contexto
            contexto = await asistente.recordar_contexto("programar reuniones")
            print("Contexto relevante:", contexto)

asyncio.run(main())
```

## Ver También

- [Primeros Pasos](../getting-started/) - Guía de instalación
- [Configuración](../configuration/) - Configuración del servidor
- [API MCP](../mcp-api/) - Referencia de herramientas disponibles
- [Solución de Problemas](../troubleshooting/) - Problemas comunes