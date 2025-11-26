---
title: "Release 1.0.0: Memoria IA Verdaderamente Local"
linkTitle: Release 1.0.0
date: 2025-11-18
author: Equipo Remembrances MCP
description: >
  ¡Anunciamos Remembrances MCP 1.0.0 con soporte nativo para modelos GGUF y SurrealDB embebido - sin dependencias externas!
tags: [release, anuncio]
---

¡Estamos encantados de anunciar el lanzamiento de **Remembrances MCP 1.0.0** – un hito importante que cumple nuestra promesa de memoria IA verdaderamente local!

## Novedades

### 🧠 Soporte Nativo para Modelos GGUF

La característica principal de esta versión es el **soporte integrado para modelos de embeddings GGUF**. Ya no necesitas ejecutar Ollama ni depender de APIs externas compatibles con OpenAI para generar embeddings. Simplemente descarga un modelo GGUF de Hugging Face y apunta Remembrances MCP hacia él:

```bash
./remembrances-mcp --gguf-model-path ./nomic-embed-text-v1.5.Q4_K_M.gguf
```

Esto significa:
- **Cero dependencias externas** para la generación de embeddings
- **Privacidad completa** – tus datos nunca salen de tu máquina
- **Despliegue simplificado** – ¡un binario, un archivo de modelo, listo!

### 💾 Base de Datos SurrealDB Embebida

Junto con el soporte GGUF, hemos integrado una **base de datos SurrealDB embebida** directamente en el binario. Ya no necesitas instalar, configurar ni gestionar un servidor de base de datos separado:

```bash
./remembrances-mcp --db-path ./mis-memorias.db --gguf-model-path ./model.gguf
```

Tus memorias ahora se almacenan en un único archivo de base de datos portátil que puedes respaldar fácilmente o mover entre sistemas.

### ⚡ Aceleración GPU

Para quienes buscan el máximo rendimiento, hemos añadido soporte de aceleración GPU:
- **Metal** para macOS (Apple Silicon)
- **CUDA** para GPUs NVIDIA
- **ROCm** para GPUs AMD

Activa la aceleración GPU con un simple flag:

```bash
./remembrances-mcp --gguf-model-path ./model.gguf --gguf-gpu-layers 32
```

### 🔄 Compatibilidad Hacia Atrás

No te preocupes – ¡todas tus configuraciones existentes siguen funcionando! Remembrances MCP 1.0.0 mantiene soporte completo para:

- **APIs de embeddings compatibles con OpenAI** – Usa OpenAI, Azure OpenAI, o cualquier servicio compatible
- **Ollama** – Continúa usando tu instalación local de Ollama si lo prefieres
- **SurrealDB externo** – Conéctate a instancias de SurrealDB remotas o auto-alojadas para despliegues distribuidos

## Por Qué Esto Importa

Con la versión 1.0.0, Remembrances MCP se convierte en una **solución de memoria IA verdaderamente autocontenida**. Ya sea que estés construyendo un asistente IA personal, una aplicación enfocada en la privacidad, o simplemente quieras experimentar con memoria IA sin dependencias en la nube, ahora tienes todo lo que necesitas en un único binario.

## Cómo Empezar

1. Descarga la última versión desde [GitHub Releases](https://github.com/madeindigio/remembrances-mcp/releases/tag/v1.0.0)
2. Descarga un modelo de embeddings GGUF (recomendamos [nomic-embed-text-v1.5](https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF))
3. Ejecuta:
   ```bash
   ./remembrances-mcp --gguf-model-path ./nomic-embed-text-v1.5.Q4_K_M.gguf
   ```

Consulta nuestra [documentación](/docs/) para instrucciones detalladas de configuración y opciones.

## Gracias

¡Un enorme agradecimiento a todos los que contribuyeron a esta versión a través de comentarios, reportes de errores y solicitudes de funcionalidades. Esto es solo el comienzo – ¡tenemos planes emocionantes para el futuro de Remembrances MCP!

---

*¿Tienes preguntas o comentarios? ¡Abre un issue en [GitHub](https://github.com/madeindigio/remembrances-mcp/issues) o inicia una discusión!*