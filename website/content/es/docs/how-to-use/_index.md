---
title: "Sistema de Ayuda (how_to_use)"
linkTitle: "Sistema de Ayuda"
weight: 6
description: >
  Documentación bajo demanda para reducir el consumo de tokens
---

Remembrances incluye un sistema de ayuda integrado llamado `how_to_use` que proporciona documentación bajo demanda para todas las herramientas. Este diseño reduce el consumo inicial de tokens en aproximadamente un **85%**, haciendo tus interacciones de IA más eficientes y económicas.

## ¿Por qué how_to_use?

Tradicionalmente, los agentes de IA reciben la documentación completa de todas las herramientas disponibles al inicio de cada conversación. Para Remembrances con sus más de 37 herramientas, esto significa cargar ~15,000+ tokens de documentación antes de que comience cualquier trabajo real.

El enfoque `how_to_use` cambia esto:
- **Contexto inicial mínimo**: Cada herramienta tiene una descripción de 1-2 líneas
- **Detalles bajo demanda**: La documentación completa se carga solo cuando es necesaria
- **Mejor enfoque**: Los agentes de IA ven la documentación relevante cuando la necesitan

## Ahorro de Tokens

| Métrica | Tradicional | Con how_to_use | Ahorro |
|---------|-------------|----------------|--------|
| Contexto inicial | ~15,000 tokens | ~2,500 tokens | ~83% |
| Por conversación | Igual cada vez | Solo lo necesario | Variable |

## Cómo Usar

### Obtener Vista General Completa

Pide a tu IA que llame a `how_to_use()` sin parámetros:

```
how_to_use()
```

Esto devuelve una vista general de alto nivel de todas las categorías de herramientas:
- Herramientas de memoria (facts, vectors, graph)
- Herramientas de base de conocimiento
- Herramientas de indexación de código
- Herramientas de eventos

### Obtener Documentación de un Grupo

Para información detallada sobre una categoría de herramientas:

```
how_to_use("memory")
```

Grupos disponibles:
- `memory` – Operaciones de facts, vectors y graph
- `kb` – Herramientas de documentos de base de conocimiento
- `code` – Herramientas de indexación y búsqueda de código
- `events` – Herramientas de registro y búsqueda de eventos

### Obtener Documentación de una Herramienta Específica

Para documentación completa de una herramienta individual:

```
how_to_use("remembrance_save_fact")
```

```
how_to_use("code_semantic_search")
```

```
how_to_use("save_event")
```

Esto devuelve:
- Descripción completa
- Todos los parámetros con tipos y descripciones
- Ejemplos de uso
- Herramientas relacionadas

## Para Agentes de IA

Cuando tu agente de IA encuentra una herramienta desconocida o necesita más información, puede usar `how_to_use` para obtener exactamente la documentación que necesita. Este patrón:

1. **Reduce la sobrecarga de contexto**: Solo carga documentación cuando es necesario
2. **Mejora la precisión**: Documentación fresca y enfocada en el punto de uso
3. **Ahorra costes**: Menos tokens significa menores costes de API

### Ejemplo de Flujo de Trabajo

En lugar de tener toda la documentación de herramientas cargada por adelantado, tu agente de IA puede:

1. Ver las descripciones breves de herramientas en su contexto inicial
2. Cuando necesita usar una herramienta específica, llamar a `how_to_use("nombre_herramienta")`
3. Obtener parámetros detallados y ejemplos
4. Proceder con la llamada real a la herramienta

## Mejores Prácticas

### Para Usuarios

- Deja que tu IA descubra herramientas naturalmente a través de `how_to_use()`
- Si tu IA parece confundida sobre una herramienta, sugiérele que llame a `how_to_use("nombre_herramienta")`
- Comienza las sesiones haciendo que la IA consulte `how_to_use()` para una vista general si no está familiarizada con Remembrances

### Para Agentes de IA

- Usa `how_to_use()` al inicio de tareas complejas para entender las capacidades disponibles
- Busca herramientas específicas antes de usar funcionalidad desconocida
- Consulta la documentación del grupo cuando trabajes con una categoría de operaciones
