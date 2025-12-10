# Plan de Implementación: Añadir project_id a las Tools de Memoria

## Resumen Ejecutivo

Actualmente las tools de memoria (facts, vectors, events, etc.) utilizan únicamente `user_id` para identificar los datos. Este plan añade un nuevo parámetro `project_id` que permitirá:

1. **Separación por proyectos**: Aislar datos entre diferentes proyectos/aplicaciones
2. **Compatibilidad hacia atrás**: El parámetro es opcional con valor por defecto "default"
3. **Mejores sugerencias**: Al no encontrar un elemento, sugerir alternativas dentro del mismo proyecto

## Tools Afectadas

| Tool | Archivo | Cambio |
|------|---------|--------|
| save_fact | fact_tools.go | Añadir project_id |
| get_fact | fact_tools.go | Añadir project_id |
| list_facts | fact_tools.go | Añadir project_id |
| delete_fact | fact_tools.go | Añadir project_id |
| add_vector | vector_tools.go | Añadir project_id |
| search_vectors | vector_tools.go | Añadir project_id |
| update_vector | vector_tools.go | Añadir project_id |
| delete_vector | vector_tools.go | Añadir project_id |
| save_event | event_tools.go | Añadir project_id |
| search_events | event_tools.go | Añadir project_id |
| to_remember | remember_tools.go | Añadir project_id |
| last_to_remember | remember_tools.go | Añadir project_id |
| hybrid_search | misc_tools.go | Añadir project_id |
| get_stats | misc_tools.go | Añadir project_id |

## Tablas de Base de Datos Afectadas

- `kv_memories` (facts)
- `vector_memories` (vectors)  
- `events`
- `entities` (graph)
- `user_stats`

## Fases de Implementación

### Fase 1: Migración de Base de Datos (v13)
**Archivo**: `internal/storage/migrations/v13_project_id.go`

- Añadir campo `project_id TYPE string DEFAULT "default"` a todas las tablas
- Crear índices compuestos: `(project_id, user_id, key)` para kv_memories, etc.
- Actualizar registros existentes con `project_id = "default"`
- Actualizar `targetVersion` en `surrealdb_schema.go` a 13

### Fase 2: Actualizar Types
**Archivo**: `pkg/mcp_tools/types.go`

- Añadir campo `ProjectID string json:"project_id,omitempty"` a todos los Input structs
- El campo es opcional con valor por defecto "default"

### Fase 3: Actualizar Interfaz Storage
**Archivo**: `internal/storage/storage.go`

- Modificar firmas de métodos para incluir `projectID` como primer parámetro
- Añadir nuevo método `ListProjectIDs(ctx, table) ([]string, error)`
- Actualizar structs VectorResult, Entity, Event para incluir ProjectID

### Fase 4: Implementar Storage SurrealDB
**Archivos**:
- `internal/storage/surrealdb_facts.go`
- `internal/storage/surrealdb_vectors.go`
- `internal/storage/surrealdb_events.go`
- `internal/storage/surrealdb_hybrid.go`
- `internal/storage/surrealdb_stats.go`
- `internal/storage/surrealdb_alternatives.go`

- Actualizar todas las queries para filtrar por project_id
- Aplicar valor por defecto "default" si project_id está vacío
- Implementar ListProjectIDs

### Fase 5: Actualizar Handlers de Tools
**Archivos**:
- `pkg/mcp_tools/fact_tools.go`
- `pkg/mcp_tools/vector_tools.go`
- `pkg/mcp_tools/event_tools.go`
- `pkg/mcp_tools/remember_tools.go`
- `pkg/mcp_tools/misc_tools.go`

- Extraer project_id del input
- Aplicar valor por defecto si no se especifica
- Pasar project_id a las llamadas del storage

### Fase 6: Actualizar Sistema de Alternativas
**Archivo**: `pkg/mcp_tools/alternatives.go`

- Modificar FindUserAlternatives para filtrar por project_id
- Añadir FindProjectAlternatives para sugerir proyectos alternativos
- Mejorar mensajes de error con sugerencias

### Fase 7: Actualizar Documentación
**Archivo**: `pkg/mcp_tools/help_tool.go`

- Documentar el nuevo parámetro project_id en todas las tools
- Explicar el comportamiento por defecto
- Añadir ejemplos de uso

### Fase 8: Testing
- Tests de migración
- Tests de compatibilidad hacia atrás
- Tests de aislamiento por proyecto
- Tests de sugerencias

## Notas Importantes

1. **Las tools de código (code_*) NO se modifican** - ya tienen su propio project_id para indexación
2. **Compatibilidad total hacia atrás** - si no se especifica project_id, se usa "default"
3. **Índice único actualizado** - El índice de kv_memories pasa de `(user_id, key)` a `(project_id, user_id, key)`

## Archivos de Referencia

Las fases detalladas están guardadas como facts:
- `plan_project_id_fase1` a `plan_project_id_fase8`

Usar: `get_fact(user_id="remembrances-mcp", key="plan_project_id_faseN")`
