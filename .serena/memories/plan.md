# Plan Actual: Migración JSON → YAML en MCP Tools

**Branch:** `feature/resolve-user-id`  
**Fecha:** 7 diciembre 2025

## Objetivo

Convertir todas las respuestas de MCP tools de JSON a YAML y agregar sugerencias de user_id/project_id alternativos cuando búsquedas retornan 0 resultados.

## Fases del Plan

1. **Utilidad YAML** → Crear `pkg/mcp_tools/yaml_utils.go`
2. **Count por user_id** → Implementar `CountByUserID()` en storage
3. **Handlers user_id** → Agregar alternativas cuando count=0
4. **Memory tools → YAML** → Convertir vector/fact/event/graph/kb tools
5. **Code tools → YAML** → Convertir search/indexing/manipulation tools
6. **Handlers project_id** → Agregar alternativas para code tools
7. **Testing** → Validar todas las conversiones

## Detalles Completos

Ver facts guardados: `yaml_migration_phase_1` a `yaml_migration_phase_7`  
Recuperar con: `get_fact` o `last_to_remember`

## Archivos Principales

- `pkg/mcp_tools/yaml_utils.go` (nuevo)
- `internal/storage/surrealdb_stats.go`
- `pkg/mcp_tools/*_tools.go` (10+ archivos)

---

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
