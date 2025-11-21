# Knowledge Base Timestamp Optimization

## Fecha
2025-11-21

## Problema Identificado

El servidor MCP reprocesaba todos los archivos de la knowledge base en cada reinicio, incluso cuando los archivos no habían sido modificados. Esto causaba:

1. **Reprocesamiento innecesario**: Todos los archivos `.md` se volvían a leer, chunkear y embedder en cada inicio
2. **Consumo de recursos**: CPU y memoria desperdiciados regenerando embeddings que ya existían
3. **Tiempo de inicio lento**: El `initialScan` tardaba mucho tiempo en completarse
4. **Apariencia de pérdida de datos**: Aunque los datos persistían en RocksDB, parecía que la base de datos se vaciaba porque se eliminaban y recreaban

## Análisis del Código

### Flujo Original

```
main.go:266-271
  ↓
kb.StartWatcher()
  ↓
initialScan() [línea 89-131]
  ↓
processFile() [línea 191-246] - SIEMPRE procesaba sin verificar
  ↓
SaveDocumentChunks() [línea 220+]
  - DELETE FROM knowledge_base WHERE source_file = $file_path
  - CREATE nuevos chunks con embeddings
```

### Causa Raíz

En `internal/kb/kb.go`, la función `processFile()` no verificaba si el archivo ya estaba procesado ni si había sido modificado desde la última vez. Simplemente:

1. Leía el archivo
2. Generaba embeddings
3. Eliminaba chunks anteriores
4. Guardaba nuevos chunks

## Solución Implementada

### Verificación de Timestamp de Modificación

Se implementó un sistema de verificación basado en el timestamp de modificación del archivo (`mtime`):

#### 1. Modificación en `processFile()` (`internal/kb/kb.go`)

**Cambios realizados:**

- Se obtiene el `fileInfo` con `os.Stat()` antes de leer el archivo
- Se consulta si el documento ya existe en la base de datos usando `GetDocument()`
- Se compara el timestamp del archivo con el guardado en metadata
- Solo se procesa si:
  - El documento no existe, O
  - El archivo ha sido modificado después del último procesamiento

**Código añadido:**

```go
// Get file info to check modification time
fileInfo, err := os.Stat(fullPath)
if err != nil {
    slog.Warn("failed to stat kb file", "file", rel, "error", err)
    return
}
fileModTime := fileInfo.ModTime()

// Check if document already exists and compare modification times
existing, err := w.storage.GetDocument(processingCtx, rel)
if err == nil && existing != nil {
    // Document exists, check if file has been modified since last processing
    if lastModStr, ok := existing.Metadata["last_modified"].(string); ok {
        if lastModTime, err := time.Parse(time.RFC3339, lastModStr); err == nil {
            // If file hasn't been modified since last processing, skip
            if !fileModTime.After(lastModTime) {
                slog.Debug("kb file not modified since last processing, skipping", 
                    "file", rel, 
                    "file_mtime", fileModTime.Format(time.RFC3339), 
                    "db_mtime", lastModTime.Format(time.RFC3339))
                return
            }
            slog.Debug("kb file modified, reprocessing", "file", rel)
        }
    }
}
```

#### 2. Almacenamiento del Timestamp

El timestamp de modificación se guarda en el metadata de cada chunk:

```go
metadata := map[string]interface{}{
    "source":        "watcher",
    "total_size":    contentSize,
    "last_modified": fileModTime.Format(time.RFC3339),  // NUEVO
}
```

#### 3. Mejora en `GetDocument()` (`internal/storage/surrealdb_documents.go`)

Como los documentos se guardan como chunks individuales con `source_file`, se modificó la query para buscar correctamente:

**Antes:**
```go
query := "SELECT * FROM knowledge_base WHERE file_path = $file_path"
```

**Después:**
```go
query := "SELECT * FROM knowledge_base WHERE source_file = $file_path OR file_path = $file_path ORDER BY chunk_index ASC LIMIT 1"
```

Esto permite:
- Buscar por `source_file` (para documentos chunkeados)
- Buscar por `file_path` (para documentos sin chunks o legacy)
- Ordenar por `chunk_index` para obtener el primer chunk
- Limitar a 1 resultado para obtener solo el metadata del primer chunk

## Beneficios

### ✅ Rendimiento

- **Arranque rápido**: Solo se procesan archivos nuevos o modificados
- **Menos CPU**: No se regeneran embeddings innecesariamente
- **Menos memoria**: Especialmente importante con modelos GGUF que consumen mucha memoria

### ✅ Persistencia Real

- Los datos ya NO se eliminan y recrean en cada reinicio
- Los chunks existentes se mantienen intactos si el archivo no cambió
- La base de datos RocksDB mantiene su contenido entre reinicios

### ✅ Detección de Cambios

- Los archivos modificados **SÍ se reprocesarán** automáticamente
- El timestamp se actualiza con cada procesamiento
- Compatible con el watcher de eventos en tiempo real

## Archivos Modificados

1. **`internal/kb/kb.go`**
   - Función `processFile()`: Verificación de timestamps
   - Metadata: Inclusión de `last_modified`

2. **`internal/storage/surrealdb_documents.go`**
   - Función `GetDocument()`: Query mejorada para buscar por `source_file`

## Testing

### Compilación
```bash
cd /www/MCP/remembrances-mcp
make build
```

### Verificación del Comportamiento

1. **Primer arranque**: Procesa todos los archivos, guarda timestamps
2. **Segundo arranque**: Skip archivos no modificados, logs muestran:
   ```
   kb file not modified since last processing, skipping
   ```
3. **Modificar archivo**: Touch o editar un `.md`
4. **Tercer arranque**: Solo reprocesa el archivo modificado

### Logs Esperados

**Archivo no modificado:**
```
level=DEBUG msg="kb file not modified since last processing, skipping" 
  file=example.md 
  file_mtime=2025-11-21T10:30:00Z 
  db_mtime=2025-11-21T10:30:00Z
```

**Archivo modificado:**
```
level=DEBUG msg="kb file modified, reprocessing" 
  file=example.md 
  file_mtime=2025-11-21T11:00:00Z 
  db_mtime=2025-11-21T10:30:00Z
```

## Compatibilidad

### Documentos Existentes

Los documentos ya existentes en la base de datos (antes de este cambio):
- **No tienen** el campo `last_modified` en metadata
- Serán **reprocesados una vez** en el primer arranque después del cambio
- En procesamientos subsecuentes, tendrán el timestamp y funcionarán correctamente

### Chunks vs Documentos Simples

La solución es compatible con ambos formatos:
- **Chunks**: Busca por `source_file` y obtiene metadata del primer chunk
- **Documentos simples**: Busca por `file_path` (fallback)

## Próximas Mejoras Potenciales

1. **Checksum adicional**: Además del timestamp, verificar hash del contenido
2. **Migración de datos antiguos**: Script para añadir `last_modified` a documentos legacy
3. **Estadísticas de procesamiento**: Contador de archivos skipped vs procesados
4. **Configuración**: Flag para forzar reprocesamiento completo si es necesario

## Relacionado

- `surrealdb_cbor_fix.md`: Fix de unmarshaling CBOR
- `surrealdb_schema_migration_refactoring_completed.md`: Sistema de migraciones
- `SURREALDB_EMBEDDED.md`: Documentación de SurrealDB embedded con RocksDB
