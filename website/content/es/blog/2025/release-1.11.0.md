---
title: "Versión 1.11.0: Seguimiento de Eventos y Embeddings Especializados para Código"
linkTitle: Versión 1.11.0
date: 2025-11-30
author: Remembrances MCP Team
description: >
  Remembrances MCP 1.11.0 introduce seguimiento de eventos temporales con búsqueda híbrida y soporte para modelos de embedding especializados en código.
tags: [release, features, events, embeddings]
---

Nos complace anunciar **Remembrances MCP 1.11.0**, que trae dos nuevas capacidades potentes: un completo **sistema de Eventos y Logs** para seguimiento temporal y **Embeddings Duales de Código** para búsqueda de código optimizada.

## 📅 Sistema de Eventos y Logs

Rastrea actividades, conversaciones, logs y hitos con el nuevo sistema de Eventos. A diferencia de las memorias regulares, los eventos están diseñados para datos ordenados en el tiempo con potentes consultas temporales.

### ¿Qué Hace Especiales a los Eventos?

**Búsqueda Híbrida:**
Los eventos combinan similitud vectorial con búsqueda de texto BM25. Busca por significado ("problemas de autenticación") y obtén resultados ordenados tanto por relevancia semántica como por coincidencia de palabras clave.

**Consultas Basadas en Tiempo:**
Encuentra eventos de las últimas 24 horas, la última semana o dentro de rangos de fechas específicos. Perfecto para rastrear qué pasó y cuándo.

**Organización por Subject:**
Categoriza eventos por tema usando patrones de subject como `conversation:tema`, `error:tipo` o `milestone:nombre`.

### Casos de Uso

**📝 Memoria de Conversaciones:**
Rastrea discusiones importantes a través de múltiples sesiones:
```
save_event({
  "user_id": "proyecto-alpha",
  "subject": "conversation:planificacion-sprint",
  "content": "El equipo acordó priorizar mejoras de autenticación. Fecha límite del MVP fijada para el 15 de marzo."
})
```

**⚠️ Seguimiento de Errores:**
Registra y busca incidentes:
```
save_event({
  "user_id": "api-service",
  "subject": "error:database",
  "content": "Pool de conexiones agotado. Aumentado tamaño del pool de 10 a 25."
})
```

**🚀 Seguimiento de Hitos:**
Marca y encuentra logros:
```
save_event({
  "user_id": "producto",
  "subject": "milestone:release",
  "content": "Versión 2.0 lanzada con modo oscuro y soporte multiidioma."
})
```

### Búsqueda Potente

Encuentra eventos con filtros combinados:
```
search_events({
  "user_id": "api-service",
  "subject": "error:database",
  "query": "timeout conexión",
  "last_days": 7,
  "limit": 20
})
```

Esto encuentra errores de base de datos de la última semana que están semánticamente relacionados con timeouts de conexión.

## 🧠 Embeddings Duales de Código

Los modelos de embedding de texto genéricos funcionan bien para lenguaje natural, pero el código tiene patrones y semántica diferentes. La versión 1.11.0 introduce soporte para **modelos de embedding especializados en código**.

### ¿Por Qué Modelos Especializados?

Los modelos de embedding específicos para código como **CodeRankEmbed** o **Jina Code Embeddings** están entrenados con código fuente y entienden:
- Sintaxis y patrones de lenguajes de programación
- Semántica y relaciones del código
- Mapeo de lenguaje natural a código

Esto se traduce en mejores resultados al buscar código semánticamente.

### Cómo Configurar

Usa un modelo dedicado para indexación de código mientras mantienes tu modelo general para texto:

**GGUF (Local):**
```yaml
# Modelo principal para texto
gguf-model-path: "./nomic-embed-text-v1.5.Q4_K_M.gguf"

# Modelo específico para código
code-gguf-model-path: "./coderankembed.Q4_K_M.gguf"
```

**Ollama:**
```yaml
ollama-model: "nomic-embed-text"
code-ollama-model: "jina/jina-embeddings-v3"
```

**OpenAI:**
```yaml
openai-model: "text-embedding-3-small"
code-openai-model: "text-embedding-3-large"
```

### Fallback Automático

Si no configuras un modelo específico para código, Remembrances usa tu modelo de embedding por defecto para todo. ¡Actualiza a tu propio ritmo!

### Modelos Recomendados

| Proveedor | Modelo | Mejor Para |
|-----------|--------|------------|
| GGUF | CodeRankEmbed | Búsqueda de código local y privada |
| Ollama | jina-embeddings-v3 | Equilibrio calidad + velocidad |
| OpenAI | text-embedding-3-large | Máxima calidad |

## Empezando

### Actualizar

Descarga desde [GitHub Releases](https://github.com/madeindigio/remembrances-mcp/releases/tag/v1.11.0) y reemplaza tu binario existente.

### Prueba los Eventos

Comienza a rastrear con:
```
save_event({
  "user_id": "mi-proyecto",
  "subject": "log:sesion",
  "content": "Comenzado trabajo en la nueva función del dashboard."
})
```

Busca después:
```
search_events({
  "user_id": "mi-proyecto",
  "query": "función dashboard",
  "last_days": 30
})
```

### Configura Embeddings de Código

Añade a tu `config.yaml`:
```yaml
code-gguf-model-path: "./coderankembed.Q4_K_M.gguf"
```

Luego re-indexa tus proyectos para beneficiarte de embeddings optimizados para código.

## Próximos Pasos

Continuamos mejorando Remembrances MCP con:
- Más capacidades de búsqueda de eventos
- Soporte adicional para modelos de embedding de código
- Mejoras de rendimiento para registro de eventos de alto volumen

¡Gracias por vuestro continuo apoyo y feedback!

---

*¿Preguntas o feedback? ¡Abre un issue en [GitHub](https://github.com/madeindigio/remembrances-mcp/issues)!*
