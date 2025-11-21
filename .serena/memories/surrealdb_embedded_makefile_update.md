# Actualización del Makefile para SurrealDB Embedded

## Fecha
2025-11-21

## Resumen
Se ha actualizado el Makefile del proyecto `remembrances-mcp` para soportar correctamente la compilación y enlace de la librería `surrealdb-embedded` con soporte para múltiples backends de almacenamiento.

## Cambios Realizados

### 1. Actualización del Target `surrealdb-embedded`
- **Archivo**: `Makefile`
- **Cambio**: Se actualizó la ruta de la librería compilada para apuntar a `surrealdb_embedded_rs/target/release/`
- **Razón**: La librería Rust se compila en el subdirectorio `target/release/` dentro del proyecto Rust

```makefile
# Antes
@if [ ! -f "$(SURREALDB_EMBEDDED_DIR)/libsurrealdb_embedded_rs.so" ]; then

# Después
@if [ ! -f "$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/release/libsurrealdb_embedded_rs.so" ] && \
   [ ! -f "$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/release/libsurrealdb_embedded_rs.dylib" ]; then
```

### 2. Corrección de CGO_LDFLAGS
- **Archivo**: `Makefile`
- **Cambio**: Se actualizó el path de enlace para apuntar al directorio correcto
- **Razón**: El linker necesita encontrar la librería en su ubicación real

```makefile
# Antes
export CGO_LDFLAGS := ... -L$(SURREALDB_EMBEDDED_DIR) -lsurrealdb_embedded_rs ...

# Después
export CGO_LDFLAGS := ... -L$(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/release -lsurrealdb_embedded_rs ...
```

### 3. Actualización del Target `build`
- **Archivo**: `Makefile`
- **Cambio**: Se mejoró la copia de la librería al directorio `build/`
- **Mejoras**:
  - Soporte para `.so` (Linux) y `.dylib` (macOS)
  - Mensajes de advertencia si la librería no se encuentra
  - Verificación post-copia con mensaje de éxito

```makefile
@echo "Copying SurrealDB embedded library..."
@cp $(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/release/libsurrealdb_embedded_rs.so $(BUILD_DIR)/ 2>/dev/null || \
 cp $(SURREALDB_EMBEDDED_DIR)/surrealdb_embedded_rs/target/release/libsurrealdb_embedded_rs.dylib $(BUILD_DIR)/ 2>/dev/null || \
 echo "⚠ Warning: SurrealDB embedded library not found"
@ls -lh $(BUILD_DIR)/libsurrealdb_embedded_rs.* 2>/dev/null && echo "✓ SurrealDB embedded library copied" || echo "⚠ SurrealDB embedded library not found in build/"
```

### 4. Actualización de Build Variants
- **Archivo**: `Makefile` (target `build-variant`)
- **Cambio**: Se aplicaron los mismos cambios al target de compilación de variantes
- **Objetivo**: Asegurar que todas las variantes (CPU, CUDA, Metal, etc.) incluyan la librería de SurrealDB

### 5. Actualización de Distribución
- **Archivo**: `Makefile` (target `dist-variant`)
- **Cambio**: Se mejoró la copia de la librería al paquete de distribución
- **Objetivo**: Incluir la librería de SurrealDB en los paquetes ZIP de distribución

## Soporte de Backends en surrealdb-embedded

La librería actualizada ahora soporta los siguientes formatos de URL:

1. **Memory (En memoria)**
   ```
   memory://
   ```

2. **RocksDB (Persistencia con RocksDB)**
   ```
   rocksdb://path/to/database
   rocksdb:path/to/database
   ```

3. **SurrealKV (Persistencia con SurrealKV)**
   ```
   surrealkv://path/to/database
   surrealkv:path/to/database
   ```

4. **File (Deprecado, usa RocksDB)**
   ```
   file://path/to/database
   ```

## Verificación de la Compilación

Después de los cambios, la compilación es exitosa:

```bash
$ make build
Building remembrances-mcp with GGUF and embedded SurrealDB support...
Copying shared libraries to build directory...
Copying SurrealDB embedded library...
✓ SurrealDB embedded library copied
Build complete: build/remembrances-mcp
```

Las librerías en el directorio `build/`:
```bash
-rwxrwxr-x 1 sevir sevir  59M libsurrealdb_embedded_rs.so
-rwxrwxr-x 1 sevir sevir 2.6M libllama.so
-rwxrwxr-x 1 sevir sevir 716K libggml-base.so
-rwxrwxr-x 1 sevir sevir  13M remembrances-mcp
```

Las dependencias dinámicas se resuelven correctamente:
```bash
$ ldd build/remembrances-mcp | grep -E "(surrealdb|llama|ggml)"
	libllama.so (0x00007b467aa00000)
	libggml-base.so (0x00007b467ad42000)
	libsurrealdb_embedded_rs.so (0x00007b4677a00000)
	libggml.so (0x00007b467a9f3000)
	libggml-cpu.so (0x00007b4677890000)
```

## Archivos Modificados

- `Makefile` - Actualizados múltiples targets:
  - `surrealdb-embedded`
  - `build`
  - `build-variant`
  - `dist-variant`
  - Variables CGO_LDFLAGS

## Próximos Pasos

1. Verificar que la aplicación puede inicializar SurrealDB con los diferentes backends
2. Probar la persistencia de datos con RocksDB y SurrealKV
3. Actualizar la documentación de usuario con ejemplos de uso de cada backend
4. Considerar añadir tests de integración para cada backend

## Notas Técnicas

- La librería `libsurrealdb_embedded_rs.so` es de ~59MB, lo cual es normal para una librería Rust que incluye SurrealDB completo
- El RPATH está configurado para buscar librerías en `$ORIGIN`, lo que permite ejecutar el binario desde su directorio sin configuración adicional
- Las librerías se copian automáticamente durante el proceso de compilación
