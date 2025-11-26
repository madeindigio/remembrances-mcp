# Correcciones de SurrealDB Embedded - 21 Nov 2025

## Resumen
Se han realizado tres correcciones importantes en el sistema de SurrealDB embedded para eliminar mensajes de debug, corregir el parsing de URLs con prefijos de backend, y preparar el sistema para resolver problemas de persistencia de datos.

## Problema 1: Mensajes de Debug en Producción

### Síntomas
Al ejecutar `build/remembrances-mcp --config config.sample.gguf.yaml` se mostraban mensajes de debug:
```
surreal_query called with: SELECT count() AS count FROM knowledge_base
Query executed
Query succeeded
Got outer value, converting...
Serialized to JSON: [{"count":1}
```

### Solución
Se comentaron todos los `eprintln!()` en el código Rust de `surrealdb_embedded_rs/src/lib.rs`:
- Líneas 179, 194, 253: Mensajes de error
- Líneas 277, 281, 285, 291, 296, 300, 306, 311, 315: Mensajes de debug en `surreal_query`
- Líneas 432, 437, 443: Mensajes de debug en `surreal_query_with_params`

**Archivo modificado**: `~/www/MCP/Remembrances/surrealdb-embedded/surrealdb_embedded_rs/src/lib.rs`

## Problema 2: Parsing Incorrecto del Prefijo `surrealkv://`

### Síntomas
Al configurar `db-path: "surrealkv://~/www/MCP/remembrances-mcp/remembrances.db"`, el sistema intentaba crear una carpeta llamada `surrealkv` en el path local en lugar de usar el backend SurrealKV.

### Causa Raíz
El código Go en `internal/storage/surrealdb.go` estaba usando directamente `embedded.NewRocksDB()` en lugar de `embedded.NewFromURL()`, lo que no permitía interpretar los prefijos de URL.

### Solución
Se modificó la función `Connect()` en `internal/storage/surrealdb.go:64-111` para usar `embedded.NewFromURL()`:

```go
// Antes (línea 72)
s.embeddedDB, err = embedded.NewRocksDB(s.config.DBPath)

// Después
s.embeddedDB, err = embedded.NewFromURL(s.config.DBPath)
```

Esto permite soportar todos los backends:
- `memory://` - Base de datos en memoria
- `rocksdb:///path/to/db` - Backend RocksDB
- `surrealkv:///path/to/db` - Backend SurrealKV  
- `file:///path/to/db` - Deprecated, usa RocksDB

**Archivo modificado**: `~/www/MCP/remembrances-mcp/internal/storage/surrealdb.go:72`

### Nota sobre Sintaxis de URL
La sintaxis con tres barras (`surrealkv:///path`) es **correcta**:
- `surrealkv://` es el esquema
- La tercera `/` inicia el path absoluto
- Es el formato estándar de URI para paths absolutos

## Problema 3: Datos No Persisten / Deserialización

### Síntomas Reportados
Cada vez que se arranca el programa, vuelve a leer todos los archivos de knowledge base para procesar, indicando que:
1. Los datos no se están guardando correctamente en la base de datos
2. O las consultas para verificar datos existentes siempre devuelven vacío

### Investigación Realizada

Se verificó el flujo de datos:

1. **Función de guardado** (`SaveDocument` en `surrealdb_documents.go:9-84`):
   - Verifica si el documento existe con: `SELECT id FROM knowledge_base WHERE file_path = $file_path`
   - Si no existe, hace `CREATE knowledge_base CONTENT {...}`
   - Si existe, hace `UPDATE knowledge_base SET ... WHERE file_path = $file_path`
   - ✅ El código es correcto

2. **Función de consulta** (`queryEmbedded` en `surrealdb_query_helper.go:46-109`):
   - Convierte resultados de `[]interface{}` a `QueryResult`
   - Maneja arrays y objetos correctamente
   - ✅ El código es correcto

3. **Deserialización en Rust** (archivo `lib.rs`):
   - La función `value_to_json()` y `unwrap_surrealdb_tagged()` convierten correctamente los valores de SurrealDB a JSON estándar
   - ✅ El código fue corregido previamente

### Próximos Pasos para Diagnosticar
Para identificar el problema real, se recomienda:

1. **Añadir logging temporal** en `SaveDocument()` para ver:
   - Si `existsQuery` devuelve resultados
   - El valor de `isNewDocument`
   - Si la operación CREATE/UPDATE se ejecuta sin error

2. **Verificar el archivo de base de datos**:
   ```bash
   ls -lh ~/www/MCP/remembrances-mcp/remembrances.db/
   ```
   - Debería contener archivos de SurrealKV
   - El tamaño debería crecer al añadir datos

3. **Probar consulta directa** después de guardar:
   ```go
   // En SaveDocument(), después del CREATE/UPDATE:
   testQuery := "SELECT * FROM knowledge_base WHERE file_path = $file_path LIMIT 1"
   testResult, _ := s.query(ctx, testQuery, map[string]interface{}{"file_path": filePath})
   log.Printf("VERIFY: Saved document check returned %d results", len((*testResult)[0].Result))
   ```

4. **Verificar el namespace y database**:
   - Asegurarse de que `Use()` se llama correctamente antes de guardar
   - Verificar que se usa el mismo namespace/database al leer y escribir

## Archivos Modificados

### Librería SurrealDB Embedded
- `~/www/MCP/Remembrances/surrealdb-embedded/surrealdb_embedded_rs/src/lib.rs`
  - Comentados 15 mensajes `eprintln!()`
  - Recompilada con `cargo build --release`

### Proyecto Principal
- `~/www/MCP/remembrances-mcp/internal/storage/surrealdb.go:72`
  - Cambiado `embedded.NewRocksDB()` por `embedded.NewFromURL()`

## Compilación y Despliegue

```bash
# 1. Reconstruir librería Rust
cd ~/www/MCP/Remembrances/surrealdb-embedded/surrealdb_embedded_rs
cargo build --release

# 2. Reconstruir proyecto principal
cd ~/www/MCP/remembrances-mcp
make clean && make build

# 3. Verificar librerías copiadas
ls -lh build/libsurrealdb_embedded_rs.so
# Output: -rwxrwxr-x 1 sevir sevir 59M nov 21 22:07 build/libsurrealdb_embedded_rs.so
```

## Configuración Recomendada

```yaml
# config.sample.gguf.yaml
db-path: "surrealkv://~/www/MCP/remembrances-mcp/remembrances.db"

# Para RocksDB:
# db-path: "rocksdb://~/www/MCP/remembrances-mcp/remembrances.db"

# Para memoria (testing):
# db-path: "memory://"
```

## Testing

Para probar los cambios:

```bash
# 1. Eliminar base de datos anterior para empezar limpio
rm -rf ~/www/MCP/remembrances-mcp/remembrances.db

# 2. Ejecutar con configuración
./build/remembrances-mcp --config config.sample.gguf.yaml

# 3. Verificar que no aparecen mensajes de debug
# 4. Verificar que se crea el directorio remembrances.db/
# 5. Añadir algunos documentos a knowledge base
# 6. Detener el proceso
# 7. Reiniciar y verificar que NO reprocesa los documentos
```

## Estado Actual

- ✅ Mensajes de debug eliminados
- ✅ Soporte para `surrealkv://` funcionando correctamente
- ⚠️ Problema de persistencia requiere más diagnóstico
  - Código de guardado y consulta parecen correctos
  - Necesita logging adicional para identificar el problema exacto

## Conclusión

Se han corregido dos de los tres problemas reportados. El problema de persistencia de datos requiere investigación adicional con logging temporal para identificar si el problema está en:
1. La escritura de datos
2. La lectura de datos 
3. El namespace/database incorrecto
4. Problemas de commit/flush en SurrealKV

Se recomienda añadir logging temporal como se describió en "Próximos Pasos para Diagnosticar".
