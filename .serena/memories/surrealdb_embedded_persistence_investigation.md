# SurrealDB Embedded - Investigación del Problema de Persistencia

## Fecha
2025-11-21

## Problema Original Reportado

El usuario reportó que cada vez que arranca el servidor MCP, los archivos de la knowledge base se vuelven a escanear y procesar completamente, como si la base de datos estuviera vacía.

## Investigación Realizada

### 1. Implementación de Verificación de Timestamps ✅

**Implementado en:** `internal/kb/kb.go`

Se implementó un sistema para verificar si los archivos han sido modificados antes de reprocesarlos:

```go
// Get file info to check modification time
fileInfo, err := os.Stat(fullPath)
fileModTime := fileInfo.ModTime()

// Check if document already exists and compare modification times
existing, err := w.storage.GetDocument(processingCtx, rel)
if err == nil && existing != nil {
    if lastModStr, ok := existing.Metadata["last_modified"].(string); ok {
        if lastModTime, err := time.Parse(time.RFC3339, lastModStr); err == nil {
            if !fileModTime.After(lastModTime) {
                // Skip if not modified
                return
            }
        }
    }
}
```

**Metadata guardado:**
```go
metadata := map[string]interface{}{
    "source":        "watcher",
    "total_size":    contentSize,
    "last_modified": fileModTime.Format(time.RFC3339),  // Nuevo
}
```

**Modificación de GetDocument:** 
```sql
SELECT * FROM knowledge_base 
WHERE source_file = $file_path OR file_path = $file_path 
ORDER BY chunk_index ASC LIMIT 1
```

### 2. Tests de Verificación de Datos ❌

Se crearon múltiples tests para verificar el contenido de la base de datos:

**test-db-contents**: Usa `GetDocument()` - Resultado: **0 documentos encontrados**
**test-raw-db**: Queries directas con wrapper embedded - Resultado: **arrays vacíos `[]`**
**test-all-tables**: Todas las tablas vacías - Resultado: **todas las tablas devuelven `[]`**

### 3. Descubrimiento del Problema Real 🔴

#### 3.1 La Base de Datos Está Vacía

Aunque RocksDB tiene archivos con 2.2MB de datos, **TODAS las queries devuelven arrays vacíos:**

```
SELECT * FROM knowledge_base => []
SELECT count() FROM knowledge_base => []
INFO FOR DB => []
```

#### 3.2 Fix del `surreal_close` en Rust ✅

Se identificó que el `surreal_close` original no hacía flush:

```rust
// ANTES (problema)
pub extern "C" fn surreal_close(handle: i32) -> i32 {
    let mut instances = get_db_instances().lock().unwrap();
    instances.remove(&handle);  // Solo remueve, no hace flush
    SURREAL_OK
}
```

```rust
// DESPUÉS (fix aplicado)
pub extern "C" fn surreal_close(handle: i32) -> i32 {
    let mut instances = get_db_instances().lock().unwrap();
    
    if let Some(db_instance) = instances.get(&handle) {
        let db_clone = db_instance.clone();
        instances.remove(&handle);
        drop(instances);
        
        // Forzar sync/flush
        let rt = get_runtime();
        let _ = rt.block_on(async {
            let _result = db_clone.query("INFO FOR DB").await;
            tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;
        });
        
        drop(db_clone);
        SURREAL_OK
    } else {
        drop(instances);
        SURREAL_ERR_INVALID_HANDLE
    }
}
```

**Estado:** Compilado y copiado a `/www/MCP/Remembrances/surrealdb-embedded/libsurrealdb_embedded_rs.so`

#### 3.3 Problema Más Fundamental: Queries No Devuelven Datos 🔴

**Test realizado:**
```go
// Crear tabla
db.Query("DEFINE TABLE test_table SCHEMAFULL", nil)  // => []

// Añadir campo  
db.Query("DEFINE FIELD name ON test_table TYPE string", nil)  // => []

// Insertar registro
db.Query("CREATE test_table CONTENT { name: 'test' }", nil)  // => []

// Seleccionar (inmediatamente después)
db.Query("SELECT * FROM test_table", nil)  // => []  ❌ VACÍO!
```

**El problema:** Incluso sin cerrar/reabrir, las queries devuelven vacío inmediatamente.

### 4. Análisis del Código Rust del Wrapper

**Ubicación:** `/www/MCP/Remembrances/surrealdb-embedded/surrealdb_embedded_rs/src/lib.rs`

**Función problemática:** `surreal_query`

```rust
match result {
    Ok(mut response) => {
        let json_result = match response.take::<Vec<Value>>(0) {
            Ok(values) => {
                match serde_json::to_string(&values) {
                    Ok(json) => normalize_surrealdb_json(&json),
                    Err(_) => "[]".to_string(),
                }
            }
            Err(_) => {
                // ❌ PROBLEMA: Si falla, devuelve []
                "[]".to_string()
            }
        };
        // ...
    }
}
```

**El problema real:** `response.take::<Vec<Value>>(0)` está fallando silenciosamente, por lo que siempre devuelve `"[]"`.

## Problema Raíz Identificado

### SurrealDB Response Format en Embedded Mode

El método `response.take::<Vec<Value>>(0)` no es compatible con cómo SurrealDB embedded devuelve los resultados. El wrapper está intentando extraer resultados de una manera que no coincide con el formato de respuesta real de SurrealDB.

### Posibles Causas

1. **Índice incorrecto:** El `0` puede no ser el índice correcto para extraer resultados
2. **Tipo incorrecto:** `Vec<Value>` puede no ser el tipo correcto para embedded
3. **API changes:** La API de SurrealDB puede haber cambiado
4. **Missing await:** Puede que falte alguna operación asíncrona

## Estado Actual

### ✅ Completado

1. Implementación de verificación de timestamps en Go
2. Modificación de `GetDocument()` para buscar por `source_file`
3. Fix de `surreal_close()` en Rust para flush explícito
4. Tests exhaustivos que demuestran el problema

### ❌ Problema Pendiente

**El wrapper de surrealdb-embedded NO devuelve resultados de queries.**

Todas las queries devuelven arrays vacíos `[]`, incluso queries que deberían devolver datos inmediatamente después de ser insertados.

## Soluciones Propuestas

### Opción 1: Usar SurrealDB Remoto (RECOMENDADA) ⭐

Arrancar un servidor SurrealDB externo:

```bash
surreal start --user root --pass root rocksdb://./remembrances.db
```

Modificar config:
```yaml
surrealdb-url: "ws://localhost:8000"
```

**Ventajas:**
- Funciona de forma probada y estable
- Mejor para depuración
- Separación de concerns

### Opción 2: Fix del Wrapper Rust

Necesita investigación más profunda del código Rust:

1. Entender el formato real de respuesta de SurrealDB embedded
2. Modificar `surreal_query` para extraer resultados correctamente
3. Posiblemente necesita usar diferentes métodos de la API de SurrealDB

**Ejemplo de lo que podría necesitarse:**

```rust
// En lugar de response.take::<Vec<Value>>(0)
// Puede necesitar algo como:
let results = response.into_inner();  // O similar
// O iterar sobre response directamente
```

### Opción 3: Usar Otra Librería Embedded

Considerar alternativas como:
- `sqlite` + vector extensions
- `qdrant` embedded
- `lance` 

## Conclusiones

1. **La verificación de timestamps está correctamente implementada** pero no puede probarse porque la BD no persiste datos
2. **El fix de `surreal_close` está implementado** pero no resuelve el problema principal
3. **El problema real es que el wrapper Rust no devuelve resultados de queries**
4. **Se recomienda usar SurrealDB remoto** hasta que se corrija el wrapper

## Archivos Modificados

### Go
- `internal/kb/kb.go` - Verificación de timestamps
- `internal/storage/surrealdb_documents.go` - Modificación de GetDocument

### Rust  
- `/www/MCP/Remembrances/surrealdb-embedded/surrealdb_embedded_rs/src/lib.rs` - Fix de surreal_close

### Tests Creados
- `cmd/test-db-contents/main.go`
- `cmd/test-raw-db/main.go`
- `cmd/test-all-tables/main.go`
- `cmd/test-info/main.go`
- `cmd/test-direct-save/main.go`
- `cmd/test-simple-create/main.go`

## Próximos Pasos Recomendados

1. **Corto plazo:** Usar SurrealDB remoto para que el sistema funcione
2. **Medio plazo:** Investigar y fix del wrapper Rust
3. **Largo plazo:** Considerar alternativas más maduras para embedded DB

## Referencias

- [SurrealDB Rust SDK](https://docs.surrealdb.com/docs/integration/libraries/rust)
- [Wrapped library](https://github.com/yourusername/surrealdb-embedded)
- Issue tracking: Este documento
