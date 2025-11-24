---
title: "Acerca de Remembrances MCP"
linkTitle: "Acerca de"
---

## ¿Qué es Remembrances MCP?

Remembrances MCP es un **servidor Model Context Protocol (MCP)** que proporciona capacidades de memoria a largo plazo para agentes IA. Construido con Go y potenciado por SurrealDB, ofrece una solución flexible que prioriza la privacidad para gestionar la memoria de agentes IA.

## Características Principales

### 🔒 Embeddings Locales que Priorizan la Privacidad

Genera embeddings completamente en local usando modelos GGUF. Tus datos nunca salen de tu máquina, garantizando privacidad y seguridad completas.

### ⚡ Aceleración GPU

Aprovecha la aceleración por hardware con soporte para:
- **Metal** (macOS)
- **CUDA** (GPUs NVIDIA)
- **ROCm** (GPUs AMD)

### 💾 Múltiples Capas de Almacenamiento

- **Almacén Clave-Valor**: Almacenamiento y recuperación simple de hechos
- **Vector/RAG**: Búsqueda semántica con embeddings
- **Base de Datos de Grafos**: Mapeo y recorrido de relaciones

### 📝 Gestión de Base de Conocimiento

Gestiona bases de conocimiento usando simples archivos Markdown, facilitando la organización y mantenimiento del conocimiento de tu IA.

### 🔌 Integración Flexible

Soporte para múltiples proveedores de embeddings:
- **Modelos GGUF** (local, prioriza privacidad) ⭐ Recomendado
- **Ollama** (servidor local)
- **API de OpenAI** (basado en la nube)

## ¿Por qué Remembrances MCP?

Los agentes IA tradicionales no tienen estado - olvidan todo entre conversaciones. Remembrances MCP resuelve esto proporcionando:

1. **Memoria Persistente**: Almacena hechos, conversaciones y conocimiento permanentemente
2. **Búsqueda Semántica**: Encuentra información relevante usando embeddings vectoriales
3. **Mapeo de Relaciones**: Entiende conexiones entre diferentes piezas de información
4. **Control de Privacidad**: Mantén datos sensibles en local con embeddings GGUF

## Casos de Uso

- **Asistentes IA Personales**: Recuerda preferencias de usuario y conversaciones pasadas
- **Asistentes de Investigación**: Construye y consulta bases de conocimiento desde documentos
- **Soporte al Cliente**: Mantén contexto a través de múltiples interacciones
- **Herramientas de Desarrollo**: Almacena y recupera fragmentos de código y documentación

## Stack Tecnológico

- **Lenguaje**: Go 1.20+
- **Base de Datos**: SurrealDB (embebida o externa)
- **Embeddings**: Modelos GGUF vía llama.cpp
- **Protocolo**: Model Context Protocol (MCP)

## Código Abierto

Remembrances MCP es código abierto y está disponible en [GitHub](https://github.com/madeindigio/remembrances-mcp). ¡Las contribuciones son bienvenidas!

## Desarrollado por Digio

Remembrances MCP es desarrollado y mantenido por [Digio](https://digio.es), una empresa de desarrollo de software especializada en IA y soluciones innovadoras.
