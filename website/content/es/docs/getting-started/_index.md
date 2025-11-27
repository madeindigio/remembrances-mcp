---
title: "Primeros Pasos"
linkTitle: "Primeros Pasos"
weight: 1
description: >
  Instala y ejecuta Remembrances MCP en minutos
---

## Requisitos Previos

- Linux, MacOSX o Windows (con WSL), alternativamente usa Docker en Windows si no tienes Windows Subsystem for Linux.
- Recomendable gener GPU Nvidia con drivers configurados en Linux, o Mac con chip M1..M4 para aceleración gráfica. En el caso de Windows, usa Docker con soporte aceleración gráfica de Nvidia.

Aunque será posible compilando la aplicación, no hay binarios disponibles por el momento para Windows nativo y Linux con soporte para GPU AMD (ROCm). Necesitarás compilar el proyecto manualmente en estos casos.

## Instalación

### En Linux o MacOSX (Windows con WSL)

```bash
curl -fsSL https://raw.githubusercontent.com/madeindigio/remembrances-mcp/main/scripts/install.sh | bash
```

Esto instalará el binario para tu sistema operativo, optimizado para CPU (MacOSX, con soporte aceleración gráfica M1..M4, y Linux para GPU NVidia -CUDA-)

### Compilar el Proyecto (si tienes una GPU AMD o quieres soporte GPU personalizado)

Esto sólo es necesario si no estás usando el script de instalación anterior, o si tienes una GPU AMD (ROCm) y quieres soporte para ella.

```bash
make surrealdb-embedded
make build-libs-hipblas
make BUILD_TYPE=hipblas build
```

Esto:
- Se compilará la librería SurrealDB embebida
- Compilará llama.cpp para soporte GPU AMD (ROCm)
- Construirá el binario `remembrances-mcp` con soporte para GPU AMD (ROCm)

### Descargar un Modelo GGUF

Este paso sólo es necesario si no tienes ya un modelo GGUF descargado (el script de instalación descarga el modelo recomendado)

Descarga el modelo recomendado nomic-embed-text-v1.5:

```bash
wget https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf
```

Otros modelos recomendados:
- **nomic-embed-text-v1.5** (768 dimensiones) - Mejor balance
- **nomic-embed-text-v2-moe** (768 dimensiones) - Más rápido, mejor calidad
- **Qwen3-Embedding-0.6B-Q8_0** (1024 dimensiones) - Alta calidad, mayor uso de memoria

Búscalos en HuggingFace: https://huggingface.co/nomic-ai y en https://huggingface.co/Qwen

### Alternativas para uso de modelos de embeddings

Si te gusta usar ollama y quieres configurar diferentes modelos de embeddings puedes alternativamente usar ollama. Remembrances soporta paso de parámetros de múltiples formas, ver el apartado de Configuración para más detalles.

Alternativamente si no tienes GPU soportada (Intel i7 Ultra o procesadores sin una GPU dedicada) puedes usar modelos de embeddings en la nube como los de OpenAI o cualquiera compatible con la API de OpenAI (Azure OpenAI, OpenRouter, etc). Ver Configuración para más detalles. La generación de embeddings en la nube puede tener costes asociados pero no requiere hardware específico y no impacta en el rendimiento de la aplicación.
