# Refactor: user_id opcional y metadata dinámica en Remembrances-MCP (oct 2025)

## Cambios principales

### 1. user_id opcional en todos los tipos de memoria
- Se añadió el campo `user_id` como opcional a todas las tablas principales: `entities`, `knowledge_base`, y todas las tablas de relaciones (`wrote`, `mentioned_in`, `related_to`, etc.), además de los ya existentes en `vector_memories` y `kv_memories`.
- El campo `user_id` permite distinguir entre memorias globales y memorias asociadas a un usuario concreto.
- Todos los métodos de almacenamiento (crear entidad, relación, documento, vector, fact) aceptan y propagan `user_id` si se proporciona. Si no, el registro es global.

### 2. Metadata dinámica en la KB y vectores
- El campo `metadata` en `knowledge_base` y `vector_memories` se almacena y recupera como `map[string]interface{}` (Go) y `object` (SurrealDB), permitiendo cualquier estructura arbitraria.
- Esto permite enriquecer los documentos y vectores con información adicional flexible (fuente, etiquetas, contexto, etc.).

### 3. Estadísticas coherentes global/usuario
- El cálculo de estadísticas (`GetStats`) ahora distingue correctamente entre datos globales y por usuario:
  - Para `key_value` y `vector_memories`, las stats son por usuario.
  - Para entidades, relaciones y documentos, se cuentan por `user_id` si se proporciona, o global si no.
- El campo `total_size_bytes` suma los tamaños de los contenidos relevantes según el ámbito (usuario/global).

## Impacto en Remembrances y la KB
- **Remembrances**: Ahora se pueden almacenar y consultar memorias (facts, vectores, entidades, relaciones) tanto globales como asociadas a un usuario, facilitando escenarios multiusuario y compartidos.
- **Knowledge Base (KB)**: Los documentos pueden tener metadata arbitraria y asociarse opcionalmente a un usuario, permitiendo búsquedas y organización más flexibles.

## Migración
- Se añadió la migración v4 para actualizar el esquema y soportar estos cambios sin perder datos previos.

## Estado
- Todos los tests automáticos pasan correctamente.
- El sistema es retrocompatible y preparado para escenarios multiusuario y metadatos enriquecidos.

---

_Refactor documentado automáticamente por GitHub Copilot, 27 oct 2025._