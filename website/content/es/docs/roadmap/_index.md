---
title: "Roadmap"
linkTitle: "Roadmap"
weight: 7
description: >
  Qué funcionalidades y características están planeadas para futuras versiones de Remembrances
---

## Versión OpenSource y Gratuita

Quedará siempre gratuita y de código abierto bajo licencia MIT.

- [x] Soporte de SurrealDB en modo servidor y embebido
- [x] Soporte de embeddings (ollama y OpenAI embeddings API)
- [x] Soporte de modelos GGUF locales para generación de embeddings
- [x] Soporte de aceleración GPU (Metal, CUDA, ROCm)
- [x] Soporte de knowledge base (archivos markdown)
- [x] Soporte de tools similar a [Mem0](https://mem0.ai/) pero en un único binario sin dependencias de lenguajes de scripting como Python o NodeJS, más rápido y eficiente.
- [x] Soporte de memoria inmediata a corto plazo (clave-valor)
- [ ] [En progreso] Soporte para la indexación de código fuente y proyectos de software usando embeddings especializados en código, implementación de tools inspiradas en [Serena](https://oraios.github.io/serena/) pero con indexación más rápida y eficiente con AST y Tree-Sitter.
- [ ] Agente AI de ingestión de datos complejos y comprensión de cómo guardarlo a través de sampling MCP
- [ ] Agente AI para la consulta de conocimiento profundo a través de sampling MCP
- [ ] Implementación de algoritmos de refuerzo de conocimiento, generalización, y olvido selectivo
- [ ] Soporte para ficheros de conocimiento avanzado (PDF, DOCX, etc)
- [ ] Soporte para ficheros multimedia (imágenes, audio, vídeo) con embeddings multimodales
- [ ] Soporte para series temporales

## Versión Comercial

- [ ] Soporte para múltiples usuarios y equipos de trabajo
- [ ] Soporte para auditoría y logging avanzado
- [ ] Soporte para backups automáticos t restauración y migración/exportación entre instancias de bases de datos (tanto embebidas como servidor SurrealDB externo)
- [ ] Soporte para integraciones empresariales (LDAP, SSO, etc)
- [ ] Interfaz web de administración y monitorización
- [ ] Interfaz de visualización del conocimiento almacenado (grafos, estadísticas, etc) y chatbots integrados para consulta directa del conocimiento almacenado
