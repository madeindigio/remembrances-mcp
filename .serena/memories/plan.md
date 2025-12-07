# Plan: Embeber bibliotecas llama.cpp con variantes GPU

**Fecha**: 7 de diciembre de 2025  
**Última actualización**: 8 de diciembre de 2025  
**Branch**: `feature/shared_surrealdb_included_inbinary`  
**Estado**: Planificación completada (revisión 2)

## Objetivo

Crear binarios autocontenidos de remembrances-mcp que incluyan las shared libraries de llama.cpp para diferentes variantes de GPU, siguiendo el patrón ya implementado para SurrealDB embedded.

## Contexto

- Sistema actual de embedded libraries documentado en facts `embed_library_phase_1` a `embed_library_phase_8`
- Implementación existente en `internal/embedded/` usa `go:embed` y `purego`
- Solo está implementada la variante CPU para `linux/amd64`
- Se requieren variantes para: CUDA (optimizado y portable), CPU-only, y Metal (macOS)
- Script `scripts/build-cuda-libs.sh` ya soporta `PORTABLE=1` para compilación Intel/AMD compatible

## Cambios en Revisión 2 (8 dic 2025)

1. **Eliminada variante darwin/amd64/cpu** - Solo soportamos Apple Silicon (M1/M2/M3) con Metal
2. **Renombrada cuda-amd a cuda-portable** - Más descriptivo: compilación AVX2 compatible Intel/AMD
3. **Nueva Fase 10** - Corregir bug crítico en task dist donde libs portable se sobrescriben
4. **Integrar script build-cuda-libs.sh** - Usar script existente en lugar de duplicar lógica en Makefile

## Variantes Objetivo (ACTUALIZADO)

| Variante | Plataforma | Arquitectura | GPU Support | PORTABLE | Tamaño Est. |
|----------|-----------|--------------|-------------|----------|-------------|
| `cpu` | Linux | amd64 | Ninguno | N/A | ~80 MB |
| `cuda` | Linux | amd64 | NVIDIA (optimizado CPU actual) | 0 | ~150 MB |
| `cuda-portable` | Linux | amd64 | NVIDIA (Intel/AMD AVX2) | 1 | ~150 MB |
| `metal` | Darwin | arm64 | Apple Metal | N/A | ~90 MB |

> **Nota**: No se incluye darwin/amd64/cpu ya que solo soportamos Apple Silicon con Metal.

## Bug Crítico Identificado

La task `dist` actual tiene un bug:
1. `xc build-llama-cpp-portable` compila con `PORTABLE=1` → libs en `build/`
2. Se copian las libs a `dist-variants/`
3. `make BUILD_TYPE=cuda build` RECOMPILA las libs SIN `PORTABLE=1`, sobrescribiendo las correctas

**Solución**: Separar compilación de libs de compilación de binario.

## Fases de Desarrollo

### Fase 1: Análisis y estructura de directorios (ACTUALIZADO)
**Fact**: `embed_llama_libs_phase_1`

- Crear estructura `internal/embedded/libs/{linux,darwin}/{amd64,arm64}/{cpu,cuda,cuda-portable,metal}/`
- Identificar bibliotecas necesarias por variante:
  - **CPU**: libggml-base, libggml, libggml-cpu, libllama, libmtmd
  - **CUDA**: + libggml-cuda
  - **Metal**: + libggml-metal
- Documentar tamaños estimados
- Crear archivo de metadatos `BUILD_INFO.json` con info de PORTABLE flag

### Fase 2: Build tags y archivos platform por variante (ACTUALIZADO)
**Fact**: `embed_llama_libs_phase_2`

- Crear `platform_linux_amd64_cpu.go` con `//go:build linux && amd64 && embedded_cpu`
- Crear `platform_linux_amd64_cuda.go` con `//go:build linux && amd64 && embedded_cuda`
- Crear `platform_linux_amd64_cuda_portable.go` con `//go:build linux && amd64 && embedded_cuda_portable`
- Crear `platform_darwin_arm64_metal.go` con `//go:build darwin && arm64 && embedded_metal`

> **Eliminado**: No se necesita `platform_darwin_amd64_cpu.go`

### Fase 3: Actualizar loader.go para variantes GPU
**Fact**: `embed_llama_libs_phase_3`

- Actualizar `orderedNames()` para orden correcto de carga según variante
- Añadir detección de bibliotecas CUDA
- Implementar fallback inteligente
- Añadir logging detallado
- Crear función `GetLoadedVariant()`

### Fase 4: Modificar Makefile para build de variantes embebidas (ACTUALIZADO)
**Fact**: `embed_llama_libs_phase_4`

- Añadir variable `PORTABLE` al Makefile (default: 0) junto a `BUILD_TYPE`
- Modificar `build-libs-variant` para usar `PORTABLE` cuando se especifique
- Crear target `build-libs-cuda-portable` que use `scripts/build-cuda-libs.sh` con `PORTABLE=1`
- Crear targets `prepare-embedded-libs-{cpu,cuda,cuda-portable,metal}`
- Crear targets `build-embedded-{cpu,cuda,cuda-portable,metal}` con tags apropiados
- **Crear target `build-binary-only`** que NO recompile bibliotecas
- Usar script `build-cuda-libs.sh` existente en lugar de duplicar lógica CMake

### Fase 5: Gestión de dependencias CUDA
**Fact**: `embed_llama_libs_phase_5`

- Investigar qué bibliotecas CUDA pueden embeberse vs requieren instalación
- Crear script de verificación de dependencias CUDA
- Implementar mensajes de error claros
- Documentar requisitos mínimos de driver NVIDIA

### Fase 6: Implementar selección automática de variante
**Fact**: `embed_llama_libs_phase_6`

- Crear `internal/embedded/detector.go`
- Detección de GPU en Linux y macOS
- Flag `--gpu-variant={auto,cuda,cpu,metal}`
- Fallback automático si hardware no coincide

### Fase 7: Optimización de tamaño y compresión
**Fact**: `embed_llama_libs_phase_7`

- Investigar uso de UPX para compresión
- Strip de símbolos de debug
- Variante "minimal"
- `-ldflags="-s -w"` para reducir binario Go

### Fase 8: Testing y CI/CD
**Fact**: `embed_llama_libs_phase_8`

- Tests unitarios para extractor con cada variante
- Test de integración para carga de libs
- Test de rendimiento embedded vs external
- Actualizar GitHub Actions con matrix de builds:
  - `linux-cpu`
  - `linux-cuda`
  - `linux-cuda-portable`
  - `darwin-metal`

### Fase 9: Documentación y distribución
**Fact**: `embed_llama_libs_phase_9`

- Actualizar README
- Documentar diferencias entre variantes (especialmente cuda vs cuda-portable)
- Guía de troubleshooting
- Actualizar docker images
- Script de instalación inteligente

### Fase 10: Corregir task dist y flujo de distribución (NUEVA)
**Fact**: `embed_llama_libs_phase_10`

**Problema**: Task `dist` sobrescribe libs portable al recompilar.

**Tareas**:
1. Separar compilación de libs de compilación de binario
2. Crear target `build-binary-only` que NO recompile bibliotecas
3. Guardar libs compiladas en directorios separados (`build/libs/cuda-portable/`)
4. Actualizar task `dist` en README.md con nuevo flujo:
   ```bash
   # Flujo correcto para cuda-portable:
   PORTABLE=1 ./scripts/build-cuda-libs.sh
   cp ./build/*.so ./build/libs/cuda-portable/
   make build-binary-only BUILD_TYPE=cuda
   ```
5. Verificar que libs no se sobrescriben entre variantes

## Referencias

- Facts previas de SurrealDB: `embed_library_phase_1` a `embed_library_phase_8`
- Facts de llama.cpp: `embed_llama_libs_phase_1` a `embed_llama_libs_phase_10`
- Código existente: `internal/embedded/`
- Script de compilación: `scripts/build-cuda-libs.sh`
- Documentación: `docs/GGUF_EMBEDDINGS.md`, `docs/GPU_COMPILATION.md`
