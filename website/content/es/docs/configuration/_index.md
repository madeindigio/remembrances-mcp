---
title: "Configuración"
linkTitle: "Configuración"
weight: 2
description: >
  Configura Remembrances según tus necesidades
---

## Métodos de Configuración

Remembrances puede configurarse usando:

1. **Archivo de configuración YAML** (recomendado)
2. **Variables de entorno**
3. **Flags de línea de comandos**

Prioridad: Flags CLI > Variables de entorno > Archivo YAML > Valores por defecto

## Configuración YAML

Crea un archivo de configuración en:
- **Linux**: `~/.config/remembrances/config.yaml`
- **macOS**: `~/Library/Application Support/remembrances/config.yaml`

O especifica una ruta personalizada con `--config`:

```yaml
# Configuración de base de datos, sólo si usas SurrealDB embebida, sino, comenta esta línea
db-path: "./remembrances.db"

# Embeddings GGUF (recomendado por portabilidad y privacidad)
gguf-model-path: "./nomic-embed-text-v1.5.Q4_K_M.gguf"
gguf-threads: 8
gguf-gpu-layers: 32

# Alternativa: Ollama, si prefieres usar Ollama para embeddings (descomenta las siguientes líneas, tendrás que comentar las anteriores de GGUF)
# ollama-url: "http://localhost:11434"
# ollama-model: "nomic-embed-text"

# Alternativa: OpenAI (si prefieres usar OpenAI embeddings API - menos privacidad, pero más fácil de usar y no necesitas hardware potente)
# openai-key: "sk-..."
# openai-model: "text-embedding-3-large"

# Opciones de transporte
# MCP Streamable HTTP es el transporte de red recomendado para las tools de MCP.
mcp-http: false
mcp-http-addr: ":3000"
mcp-http-endpoint: "/mcp"

# API JSON HTTP (útil para integración con otros sistemas o agentes que no soporten MCP nativamente)
http: false
http-addr: ":8080"

# Base de conocimiento, ruta a la carpeta con ficheros markdown para indexar y usar como knowledge base, también se generarán ficheros markdown cada web que se requiera a Remembrances guardar información en la knowledge base
knowledge-base: "./kb"
```

## Variables de Entorno

Todas las opciones pueden configurarse mediante variables de entorno con prefijo `GOMEM_`:

```bash
# GOMEM_DB_PATH: Ruta a la base de datos embebida de SurrealDB (por defecto: ./remembrances.db)
export GOMEM_DB_PATH="./data.db"

# GOMEM_GGUF_MODEL_PATH: Ruta al archivo de modelo GGUF para embeddings (recomendado por portabilidad y privacidad)
export GOMEM_GGUF_MODEL_PATH="./model.gguf"

# GOMEM_GGUF_THREADS: Número de hilos a utilizar para el procesamiento GGUF (0 = auto-detectar)
export GOMEM_GGUF_THREADS=8

# GOMEM_GGUF_GPU_LAYERS: Número de capas GPU a descargar para acelerar el procesamiento (0 = solo CPU)
export GOMEM_GGUF_GPU_LAYERS=32

# GOMEM_OLLAMA_URL: URL del servidor Ollama para embeddings (alternativa a GGUF)
export GOMEM_OLLAMA_URL="http://localhost:11434"

# GOMEM_OLLAMA_MODEL: Modelo de Ollama a utilizar para embeddings (ej. nomic-embed-text)
export GOMEM_OLLAMA_MODEL="nomic-embed-text"

# GOMEM_OPENAI_KEY: Clave API de OpenAI para embeddings (menos privacidad, pero más fácil de usar)
export GOMEM_OPENAI_KEY="sk-..."

# GOMEM_OPENAI_MODEL: Modelo de OpenAI para embeddings (ej. text-embedding-3-large)
export GOMEM_OPENAI_MODEL="text-embedding-3-large"

# GOMEM_MCP_HTTP: Habilitar transporte MCP Streamable HTTP (por defecto: false)
export GOMEM_MCP_HTTP=false

# GOMEM_MCP_HTTP_ADDR: Dirección del servidor MCP Streamable HTTP (por defecto: :3000)
export GOMEM_MCP_HTTP_ADDR=":3000"

# GOMEM_MCP_HTTP_ENDPOINT: Path del endpoint MCP Streamable HTTP (por defecto: /mcp)
export GOMEM_MCP_HTTP_ENDPOINT="/mcp"

# GOMEM_HTTP: Habilitar API JSON HTTP para integración con otros sistemas (por defecto: false)
export GOMEM_HTTP=false

# GOMEM_HTTP_ADDR: Dirección del servidor HTTP (por defecto: :8080)
export GOMEM_HTTP_ADDR=":8080"

# GOMEM_KNOWLEDGE_BASE: Ruta a la carpeta con archivos markdown para la base de conocimiento
export GOMEM_KNOWLEDGE_BASE="./kb"

# GOMEM_SURREALDB_URL: URL para instancia remota de SurrealDB (si no se usa embebida)
export GOMEM_SURREALDB_URL=""

# GOMEM_SURREALDB_USER: Usuario para SurrealDB remoto (por defecto: root)
export GOMEM_SURREALDB_USER="root"

# GOMEM_SURREALDB_PASS: Contraseña para SurrealDB remoto (por defecto: root)
export GOMEM_SURREALDB_PASS="root"
```

## Opciones de Configuración Clave

### Base de Datos (embebida o remota, necesitarás de un servidor SurrealDB si usas modo remoto)

- `db-path`: Ruta a SurrealDB embebida (por defecto: `./remembrances.db`)

Y para acceder a una instancia remota de SurrealDB:
- `surrealdb-url`: URL para instancia remota de SurrealDB
- `surrealdb-user`: Usuario (por defecto: `root`)
- `surrealdb-pass`: Contraseña (por defecto: `root`)

### Embeddings GGUF

- `gguf-model-path`: Ruta al archivo de modelo GGUF
- `gguf-threads`: Número de hilos (0 = auto-detectar)
- `gguf-gpu-layers`: Capas GPU a descargar (0 = solo CPU)

### Transporte

- `mcp-http`: Habilitar transporte MCP Streamable HTTP
- `mcp-http-addr`: Dirección del servidor MCP Streamable HTTP (por defecto: `:3000`)
- `mcp-http-endpoint`: Path del endpoint MCP (por defecto: `/mcp`)

## Configuraciones de Ejemplo

### GGUF Local con GPU

```yaml
gguf-model-path: "./nomic-embed-text-v1.5.Q4_K_M.gguf"
gguf-threads: 8
gguf-gpu-layers: 32
db-path: "./remembrances.db"
```

### Integración con Ollama

```yaml
ollama-url: "http://localhost:11434"
ollama-model: "nomic-embed-text"
db-path: "./remembrances.db"
```

### Modo API HTTP

```yaml
http: true
http-addr: ":8080"
```

### Modelos de Embedding Específicos para Código

Para resultados óptimos en búsqueda de código al usar la función de [Indexación de Código](/es/docs/code-indexing/), puedes configurar un modelo de embedding dedicado especializado en código:

```yaml
# Usa las mismas opciones de proveedor que el embedder principal
# GGUF (recomendado para despliegues locales y enfocados en privacidad)
code-gguf-model-path: "./coderankembed.Q4_K_M.gguf"

# O Ollama
# code-ollama-model: "jina/jina-embeddings-v3"

# O OpenAI
# code-openai-model: "text-embedding-3-large"
```

**Modelos de Embedding de Código Recomendados:**

| Proveedor | Modelo | Notas |
|-----------|--------|-------|
| GGUF | CodeRankEmbed (Q4_K_M) | Mejor para búsqueda de código local y privada |
| Ollama | jina-embeddings-v3 | Buen equilibrio entre calidad y velocidad |
| OpenAI | text-embedding-3-large | Alta calidad, basado en nube |

**Comportamiento de Fallback:** Si no se configura un modelo específico para código, Remembrances usa automáticamente tu modelo de embedding por defecto para la indexación de código.

**Variables de Entorno:**
```bash
export GOMEM_CODE_GGUF_MODEL_PATH="./coderankembed.Q4_K_M.gguf"
export GOMEM_CODE_OLLAMA_MODEL="jina/jina-embeddings-v3"
export GOMEM_CODE_OPENAI_MODEL="text-embedding-3-large"
```

### Configuración a través de parámetros CLI

Dispones de toda la configuración anterior a través de flags en la línea de comandos. Revisa la ayuda con:

```bash
remembrances-mcp --help
```
