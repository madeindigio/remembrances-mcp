---
title: "Primeros Pasos"
linkTitle: "Primeros Pasos"
weight: 1
description: >
  Instala y ejecuta Remembrances MCP en minutos
---

## Requisitos Previos

- Go 1.20 o posterior
- Git
- (Opcional) CUDA/ROCm para aceleración GPU

## Instalación

### 1. Clonar el Repositorio

```bash
git clone https://github.com/madeindigio/remembrances-mcp.git
cd remembrances-mcp
```

### 2. Compilar el Proyecto

```bash
make build
```

Esto:
- Instalará las dependencias de Go
- Compilará llama.cpp con soporte GPU (si está disponible)
- Construirá el binario `remembrances-mcp`

### 3. Descargar un Modelo GGUF

Descarga el modelo recomendado nomic-embed-text-v1.5:

```bash
wget https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf
```

Otros modelos recomendados:
- **nomic-embed-text-v1.5** (768 dimensiones) - Mejor balance
- **all-MiniLM-L6-v2** (384 dimensiones) - Más rápido, más pequeño

### 4. Ejecutar el Servidor

```bash
./run-remembrances.sh \
  --gguf-model-path ./nomic-embed-text-v1.5.Q4_K_M.gguf \
  --gguf-threads 8 \
  --gguf-gpu-layers 32
```

El servidor se iniciará en modo stdio, listo para aceptar conexiones MCP.

## Prueba Rápida

Prueba el servidor con un hecho simple:

```bash
# En otra terminal, usa el cliente MCP
echo '{"method":"tools/call","params":{"name":"remembrance_save_fact","arguments":{"key":"test","value":"Hola Mundo"}}}' | ./remembrances-mcp --gguf-model-path ./nomic-embed-text-v1.5.Q4_K_M.gguf
```

## Próximos Pasos

- [Configura el servidor](../configuration/) según tus necesidades
- [Aprende sobre modelos GGUF](../gguf-models/) y optimización
- [Explora la API MCP](../mcp-api/) y herramientas disponibles
