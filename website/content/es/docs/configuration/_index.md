---
title: "Configuración"
linkTitle: "Configuración"
weight: 2
description: >
  Configura Remembrances MCP según tus necesidades
---

## Métodos de Configuración

Remembrances MCP puede configurarse usando:

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
# Configuración de base de datos
db-path: "./remembrances.db"

# Embeddings GGUF (recomendado)
gguf-model-path: "./nomic-embed-text-v1.5.Q4_K_M.gguf"
gguf-threads: 8
gguf-gpu-layers: 32

# Alternativa: Ollama
# ollama-url: "http://localhost:11434"
# ollama-model: "nomic-embed-text"

# Alternativa: OpenAI
# openai-key: "sk-..."
# openai-model: "text-embedding-3-large"

# Opciones de transporte
sse: false
sse-addr: ":3000"
http: false
http-addr: ":8080"

# Base de conocimiento
knowledge-base: "./kb"
```

## Variables de Entorno

Todas las opciones pueden configurarse mediante variables de entorno con prefijo `GOMEM_`:

```bash
export GOMEM_GGUF_MODEL_PATH="./model.gguf"
export GOMEM_GGUF_THREADS=8
export GOMEM_GGUF_GPU_LAYERS=32
export GOMEM_DB_PATH="./data.db"
```

## Opciones de Configuración Clave

### Base de Datos

- `db-path`: Ruta a SurrealDB embebida (por defecto: `./remembrances.db`)
- `surrealdb-url`: URL para instancia remota de SurrealDB
- `surrealdb-user`: Usuario (por defecto: `root`)
- `surrealdb-pass`: Contraseña (por defecto: `root`)

### Embeddings GGUF

- `gguf-model-path`: Ruta al archivo de modelo GGUF
- `gguf-threads`: Número de hilos (0 = auto-detectar)
- `gguf-gpu-layers`: Capas GPU a descargar (0 = solo CPU)

### Transporte

- `sse`: Habilitar transporte SSE
- `sse-addr`: Dirección del servidor SSE (por defecto: `:3000`)
- `http`: Habilitar API JSON HTTP
- `http-addr`: Dirección del servidor HTTP (por defecto: `:8080`)

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
gguf-model-path: "./model.gguf"
gguf-gpu-layers: 32
```

## Ver También

- [Modelos GGUF](../gguf-models/) - Selección y optimización de modelos
- [API MCP](../mcp-api/) - Herramientas y endpoints disponibles
