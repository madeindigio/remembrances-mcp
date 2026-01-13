# Graph Entity Name Resolution - 2026-01-13

## Problema Identificado

Las herramientas de grafo (`create_relationship`, `get_entity`, `traverse_graph`) no eran intuitivas porque requerían usar IDs internos de SurrealDB en lugar de nombres de entidades.

**Escenario problemático anterior:**
1. `create_entity` devolvía un ID generado (ej: `entities:abc123`)
2. `create_relationship` requería usar esos IDs, no los nombres
3. No había forma de buscar entidades por nombre desde las tools

## Solución Implementada

Ya existía una función `resolveEntityID` en `internal/storage/surrealdb_entities.go` que:
- Primero intenta buscar por ID de SurrealDB (formato `table:id`)
- Si no encuentra, busca por nombre en la tabla `entities`
- Devuelve el ID resuelto o un error con sugerencias

Esta función ya se usaba en:
- `CreateRelationship` (línea 86)
- `GetEntity` (línea 158)

Pero faltaba en:
- `TraverseGraph` (línea 140) - **corregido**

## Cambios Realizados

### 1. Actualización de Mensajes de Error (pkg/mcp_tools/graph_tools.go)

Cambiados todos los mensajes de error para reflejar que aceptan "nombre o ID":

```go
// Antes:
"No entity found with ID '%s'"

// Después:
"No entity found with name or ID '%s'"
```

Afectó a:
- `createRelationshipHandler` (from_entity y to_entity)
- `traverseGraphHandler` (start_entity)
- `getEntityHandler` (entity_id)

### 2. Actualización de Documentación

Archivos actualizados en `pkg/mcp_tools/docs/tools/`:

#### create_relationship.txt
- Aclarado que acepta nombres o IDs para `from_entity` y `to_entity`
- Ejemplos actualizados usando nombres: `"Alice"` y `"Acme Corp"`

#### get_entity.txt
- Título cambiado de "by ID" a "by name or ID"
- Descripción ampliada explicando que acepta ambos
- Ejemplos con nombres y con IDs de SurrealDB

#### traverse_graph.txt
- Aclarado que `start_entity` acepta nombre o ID
- Ejemplos usando nombres directamente

### 3. Corrección de TraverseGraph

Actualizado `internal/storage/surrealdb_entities.go:140` para usar `resolveEntityID`:

```go
// TraverseGraph traverses the graph starting from an entity
func (s *SurrealDBStorage) TraverseGraph(ctx context.Context, startEntity, relationshipType string, depth int) ([]GraphResult, error) {
	// Resolve the start entity name to its ID
	startEntityID, err := s.resolveEntityID(ctx, startEntity)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve start entity '%s': %w", startEntity, err)
	}
	// ... resto del código
}
```

## Funcionamiento del Sistema de Resolución

La función `resolveEntityID` implementa un patrón de fallback inteligente:

1. **Detección de ID directo**: Si el string contiene `:` (formato SurrealDB), intenta usarlo como ID
2. **Búsqueda por nombre**: Si falla o no tiene `:`, busca en `entities WHERE name = $name`
3. **Extracción de ID**: Del resultado obtiene el `id` del record usando `extractRecordID`
4. **Error con contexto**: Si no encuentra, devuelve error descriptivo

## Beneficios

- **Mejor UX**: Los usuarios pueden usar nombres directamente sin preocuparse por IDs internos
- **Compatibilidad**: Sigue funcionando con IDs para casos avanzados
- **Mensajes claros**: Los errores indican que se aceptan ambos formatos
- **Consistencia**: Todas las herramientas de grafo funcionan de la misma manera

## Archivos Modificados

1. `pkg/mcp_tools/graph_tools.go` - Mensajes de error
2. `pkg/mcp_tools/docs/tools/create_relationship.txt` - Documentación
3. `pkg/mcp_tools/docs/tools/get_entity.txt` - Documentación
4. `pkg/mcp_tools/docs/tools/traverse_graph.txt` - Documentación
5. `internal/storage/surrealdb_entities.go` - Implementación de TraverseGraph

## Verificación

- ✅ Compilación exitosa sin errores
- ✅ Todos los archivos actualizados consistentemente
- ✅ Documentación alineada con implementación

## Ejemplo de Uso Mejorado

**Antes (requería IDs):**
```json
{
  "from_entity": "entities:abc123",
  "to_entity": "entities:def456",
  "relationship_type": "works_at"
}
```

**Ahora (acepta nombres):**
```json
{
  "from_entity": "Alice",
  "to_entity": "Acme Corp",
  "relationship_type": "works_at"
}
```
