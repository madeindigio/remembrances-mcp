
# Plan: Integración de Bibliotecas Embebidas con go:embed + purego

**Fecha:** 7 diciembre 2025

## Objetivo

Integrar `libsurrealdb_embedded_rs.so/.dylib` y bibliotecas llama.cpp directamente en el binario usando `go:embed` y `purego`, eliminando la necesidad de shared libraries externas.

## Fases del Plan

1. **Análisis dependencias** → Identificar todas las .so/.dylib necesarias por plataforma
2. **Extracción go:embed** → Crear `internal/embedded/extractor.go` con embed.FS
3. **Carga purego** → Implementar `internal/embedded/loader.go` con purego
4. **Modificar storage** → Actualizar SurrealDB storage para usar libs embebidas
5. **Build process** → Actualizar Makefile con targets `build-embedded`
6. **Multi-arch** → Gestionar linux/darwin × amd64/arm64 con build tags
7. **Docker/dist** → Crear distribución de binario autocontenido
8. **Testing** → Validar extracción, carga y funcionalidad completa

## Detalles Completos

Ver facts guardados: `embed_library_phase_1` a `embed_library_phase_8`  
Recuperar con: `get_fact` o `last_to_remember`

## Archivos Nuevos

- `internal/embedded/extractor.go`
- `internal/embedded/loader.go`
- `internal/embedded/libs/{linux,darwin}/{amd64,arm64}/*.so/*.dylib`

## Tecnologías

- **go:embed** para incluir binarios en compilación
- **purego** (ebitengine/purego) para carga dinámica sin CGo
- **build tags** para optimizar por plataforma

## Impacto

- ✅ Binario autocontenido sin dependencias externas
- ✅ No requiere LD_LIBRARY_PATH
- ⚠️ Aumento ~50-100MB en tamaño del binario
