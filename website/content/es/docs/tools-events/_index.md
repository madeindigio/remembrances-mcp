---
title: "Herramientas de Eventos"
linkTitle: "Events Tools"
weight: 13
description: >
  Almacenamiento temporal de eventos y búsqueda híbrida
---

El sistema de eventos proporciona almacenamiento temporal para logs, conversaciones, auditoría y datos históricos con búsqueda híbrida (texto + semántica).

## Herramientas Disponibles

### save_event
Almacena un evento con timestamp automático y categorización semántica por subject.

### search_events
Consulta eventos usando búsqueda híbrida (BM25 + similitud vectorial) con filtrado temporal.

## Características Principales

- **Timestamps automáticos**: Cada evento se marca con fecha/hora de creación
- **Categorización por subject**: Organiza eventos con patrones de subject descriptivos
- **Búsqueda híbrida**: Combina búsqueda de texto (BM25) con búsqueda semántica (vectores)
- **Filtrado temporal**: Consulta eventos por fechas absolutas o rangos relativos
- **Metadatos flexibles**: Añade contexto adicional mediante metadatos JSON

## Patrones de Subject Recomendados

Usa patrones descriptivos para categorizar tus eventos:

| Patrón | Uso | Ejemplo |
|--------|-----|---------|
| `conversation:session_id` | Logs de conversación | `conversation:chat_001` |
| `log:category` | Logs de aplicación | `log:build`, `log:deploy` |
| `audit:action` | Eventos de auditoría | `audit:file_modified` |
| `milestone:name` | Hitos de proyecto | `milestone:v1.0_released` |
| `error:type` | Eventos de error | `error:runtime`, `error:validation` |

## Prompts Recomendados

### Para Guardar Eventos

**Logs de conversación:**
```
Guarda esta conversación con subject "conversation:sesion_123"
```

```
Registra este mensaje del usuario en la conversación actual
```

**Logs de aplicación:**
```
Guarda este log de build con subject "log:build" y metadata status: "success"
```

```
Registra el evento de despliegue en producción con timestamp actual
```

**Auditoría:**
```
Guarda un evento de auditoría: usuario modificó archivo config.yaml
```

```
Registra el acceso del usuario admin al sistema con subject "audit:login"
```

**Errores:**
```
Guarda este error de validación con subject "error:validation"
```

```
Registra el error de conexión a base de datos con stack trace completo
```

### Para Buscar Eventos

**Por contenido semántico:**
```
Busca eventos relacionados con errores de autenticación
```

```
Encuentra conversaciones sobre despliegue en producción
```

**Por subject:**
```
Muéstrame todos los eventos de tipo "log:build"
```

```
Recupera la conversación con session_id "chat_001"
```

**Por rango temporal:**
```
Busca errores de las últimas 24 horas
```

```
Muéstrame todos los eventos de build de los últimos 7 días
```

```
Encuentra eventos de auditoría del último mes
```

**Por fechas absolutas:**
```
Busca eventos entre 2025-01-01 y 2025-01-31
```

```
Muéstrame logs de deploy desde el 15 de enero
```

**Búsqueda combinada:**
```
Busca eventos de tipo "error:runtime" de las últimas 48 horas que mencionen "database"
```

```
Encuentra conversaciones sobre microservicios de la última semana
```

## Consultas Temporales

### Fechas Absolutas (formato RFC3339)

```json
{
  "from_date": "2025-01-01T00:00:00Z",
  "to_date": "2025-12-31T23:59:59Z"
}
```

### Rangos Relativos (mutuamente exclusivos)

Elige **uno** de estos parámetros:

- `last_hours: 24` - Últimas 24 horas
- `last_days: 7` - Últimos 7 días  
- `last_months: 3` - Últimos 3 meses

## Búsqueda Híbrida

Cuando proporcionas una query, el sistema realiza búsqueda híbrida:

1. **BM25 text matching (50% peso)** - Encuentra eventos que contienen los términos de búsqueda
2. **Similitud vectorial (50% peso)** - Encuentra eventos semánticamente relacionados

Esto asegura encontrar tanto coincidencias exactas como contenido relacionado.

## Casos de Uso Comunes

### 1. Historial de Conversación

**Guardar:**
```
Guarda cada mensaje con subject="conversation:session_123"
```

**Recuperar:**
```
Busca todos los eventos con subject "conversation:session_123"
```

### 2. Logs de Build

**Guardar:**
```
Registra evento de build con subject="log:build" y metadata {status: "success", duration: 45}
```

**Consultar:**
```
Busca builds fallidos de la última semana
Muéstrame todos los eventos de build del proyecto X
```

### 3. Auditoría de Seguridad

**Guardar:**
```
Registra acceso de usuario con subject="audit:user_login" y metadata {user: "admin", ip: "192.168.1.10"}
```

**Consultar:**
```
Busca todos los accesos del usuario admin en enero
Encuentra eventos de auditoría relacionados con modificaciones de archivos
```

### 4. Tracking de Errores

**Guardar:**
```
Guarda error con subject="error:database" y full stack trace
```

**Consultar:**
```
Busca errores de base de datos de las últimas 48 horas
Encuentra todos los errores de tipo "runtime" del último mes
```

### 5. Hitos de Proyecto

**Guardar:**
```
Registra milestone con subject="milestone:v2.0_released" y metadata {version: "2.0.0", features: [...]}
```

**Consultar:**
```
Muéstrame todos los hitos del proyecto
Busca releases de los últimos 6 meses
```

## Mejores Prácticas

### Organización de Subjects

- Usa convenciones consistentes (namespace:identifier)
- Agrupa eventos relacionados con mismo prefijo
- Incluye identificadores únicos cuando sea relevante
- Mantén subjects descriptivos pero concisos

### Metadatos Útiles

```json
{
  "user_id": "admin",
  "action": "file_edit",
  "file_path": "/config/app.yaml",
  "status": "success",
  "duration_ms": 150
}
```

- Incluye contexto relevante para filtrado futuro
- Usa tipos de datos consistentes
- Añade información que facilite debugging
- No dupliques información del subject o content

### Búsquedas Eficientes

**Específicas:**
```
Busca eventos exactos usando subject y rango temporal
```

**Exploratorias:**
```
Usa búsqueda semántica para encontrar eventos relacionados
```

**Combinadas:**
```
Combina subject + query + rango temporal para precisión máxima
```

## Integración con Flujos de Trabajo

### CI/CD Pipeline

```
# Durante el build
Guarda evento de inicio de build
Guarda resultado de tests con métricas
Guarda evento de despliegue exitoso

# Para análisis
Busca builds fallidos de la última semana
Encuentra despliegues en producción del último mes
```

### Debugging de Aplicación

```
# Durante ejecución
Guarda errores con stack trace completo
Registra warnings importantes

# Para debugging
Busca errores relacionados con módulo X
Encuentra warnings antes de que ocurriera el error
```

### Análisis de Uso

```
# Tracking de actividad
Guarda eventos de acciones de usuario
Registra métricas de rendimiento

# Para análisis
Busca patrones de uso en el último mes
Encuentra acciones del usuario específico
```

## Ver Más

Para documentación detallada de cada herramienta:

```
how_to_use("events")
how_to_use("save_event")
how_to_use("search_events")
```

También consulta la documentación de [Eventos](/es/docs/events/) para más detalles técnicos.
