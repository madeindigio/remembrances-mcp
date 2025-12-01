---
title: "Eventos y Logs"
linkTitle: "Eventos y Logs"
weight: 6
description: >
  Almacena y busca eventos temporales con búsqueda semántica y filtros de tiempo
---

La función de Eventos proporciona almacenamiento temporal para que los agentes de IA rastreen actividades, conversaciones, logs y hitos. A diferencia de las memorias regulares, los eventos están diseñados para datos ordenados en el tiempo que pueden buscarse tanto semánticamente como por rangos de tiempo.

## ¿Qué son los Eventos?

Los eventos son registros con marca de tiempo que combinan:
- **Búsqueda semántica**: Encuentra eventos por significado usando embeddings vectoriales
- **Búsqueda de texto**: Búsqueda BM25 de texto completo para coincidencia de palabras clave
- **Filtros de tiempo**: Consulta por rangos de fechas o tiempo relativo (últimas 24 horas, última semana, etc.)
- **Categorización por tema**: Organiza eventos por tema o tipo

Esto hace que los Eventos sean perfectos para:
- 📝 **Logs de conversación**: Rastrea temas de discusión y decisiones
- 🔍 **Auditoría**: Registra acciones y cambios a lo largo del tiempo
- 🚀 **Hitos**: Marca logros importantes y progreso
- ⚠️ **Seguimiento de errores**: Registra y busca incidentes y problemas
- 📊 **Monitoreo de actividad**: Rastrea patrones y comportamientos

## Cómo Usar Eventos

### Guardar Eventos

Usa `save_event` para almacenar un nuevo evento:

```
save_event({
  "user_id": "proyecto-alpha",
  "subject": "conversation:planificacion-sprint",
  "content": "Discutimos nuevas funciones para Q1: mejoras de autenticación, rediseño del dashboard y limitación de API. El equipo acordó priorizar la autenticación primero."
})
```

**Parámetros:**
- `user_id` (requerido): Identifica a quién o qué pertenece el evento
- `subject` (requerido): Categoría o tema del evento
- `content` (requerido): El contenido del evento (se genera embedding para búsqueda semántica)
- `metadata` (opcional): Datos clave-valor adicionales

**Patrones de Subject:**

Recomendamos usar un patrón de prefijo para los subjects:
- `conversation:tema` – Logs de discusión
- `log:categoria` – Logs generales
- `audit:accion` – Entradas de auditoría
- `milestone:nombre` – Marcadores de logros
- `error:tipo` – Logs de errores/incidentes
- `task:proyecto` – Seguimiento de tareas

### Buscar Eventos

Usa `search_events` para encontrar eventos relevantes:

```
search_events({
  "user_id": "proyecto-alpha",
  "query": "mejoras de seguridad autenticación"
})
```

Esto realiza una **búsqueda híbrida** combinando:
1. Similitud vectorial (significado semántico)
2. Coincidencia de texto BM25 (relevancia de palabras clave)

#### Filtrar por Subject

```
search_events({
  "user_id": "proyecto-alpha",
  "subject": "conversation:planificacion-sprint"
})
```

#### Filtros Basados en Tiempo

**Tiempo Relativo:**

```
search_events({
  "user_id": "proyecto-alpha",
  "last_hours": 24
})
```

```
search_events({
  "user_id": "proyecto-alpha",
  "last_days": 7
})
```

```
search_events({
  "user_id": "proyecto-alpha",
  "last_months": 3
})
```

**Rango de Fechas:**

```
search_events({
  "user_id": "proyecto-alpha",
  "from_date": "2025-01-01T00:00:00Z",
  "to_date": "2025-01-31T23:59:59Z"
})
```

#### Combinar Filtros

Puedes combinar subject, query y filtros de tiempo:

```
search_events({
  "user_id": "proyecto-alpha",
  "subject": "error:api",
  "query": "timeout conexión fallida",
  "last_days": 7,
  "limit": 20
})
```

## Casos de Uso

### Memoria de Conversaciones

Rastrea discusiones importantes a través de múltiples sesiones:

```
save_event({
  "user_id": "usuario-123",
  "subject": "conversation:revision-proyecto",
  "content": "El usuario expresó preocupaciones sobre el cronograma de despliegue. Acordamos reuniones semanales y reducción del alcance del MVP.",
  "metadata": {"priority": "high", "followup": "true"}
})
```

Después, recuerda lo que se discutió:

```
search_events({
  "user_id": "usuario-123",
  "subject": "conversation:revision-proyecto",
  "query": "preocupaciones cronograma despliegue"
})
```

### Logs de Desarrollo

Rastrea actividades y decisiones de desarrollo:

```
save_event({
  "user_id": "miproyecto",
  "subject": "log:desarrollo",
  "content": "Refactorizado módulo de autenticación para usar tokens JWT. Eliminada autenticación basada en sesiones. Añadida rotación de refresh tokens."
})
```

### Seguimiento de Errores

Registra y busca problemas:

```
save_event({
  "user_id": "api-service",
  "subject": "error:database",
  "content": "Pool de conexiones agotado. 50 consultas pendientes. Aumentado tamaño del pool de 10 a 25.",
  "metadata": {"severity": "high", "resolved": "true"}
})
```

Encuentra problemas similares:

```
search_events({
  "user_id": "api-service",
  "subject": "error:database",
  "query": "pool conexiones rendimiento"
})
```

### Seguimiento de Hitos

Marca y encuentra logros importantes:

```
save_event({
  "user_id": "lanzamiento-producto",
  "subject": "milestone:release",
  "content": "Versión 2.0 lanzada a producción. Nuevas funciones: modo oscuro, soporte multiidioma, búsqueda mejorada."
})
```

## Mejores Prácticas

### Organización de Subjects

- Usa patrones de subject consistentes en tus eventos
- Mantén los subjects cortos pero descriptivos
- Usa prefijos para agrupar eventos relacionados

### Calidad del Contenido

- Escribe contenido descriptivo y buscable
- Incluye palabras clave relevantes para mejores resultados de búsqueda
- Añade contexto que ayude con la coincidencia semántica

### Uso de Metadata

- Usa metadata para datos estructurados (severidad, estado, etiquetas)
- Mantén la metadata simple – las consultas complejas usan subject y content
- Útil para filtrado o propósitos de visualización

### Estrategias de Consulta

- Comienza con consultas amplias, luego reduce el alcance
- Usa filtros de subject cuando conozcas la categoría
- Combina filtros de tiempo con búsqueda semántica para eventos recientes relevantes
