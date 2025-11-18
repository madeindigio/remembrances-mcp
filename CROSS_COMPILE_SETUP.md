# Cross-Compilation Setup Summary

## Fecha: 2025-11-17

Este documento resume todos los cambios realizados para habilitar la compilación cruzada del proyecto `remembrances-mcp`.

## Problemas Identificados

Durante la ejecución inicial de `./scripts/release-cross.sh`, se encontraron los siguientes problemas:

### 1. Script de Compilación de Librerías
- **Problema**: El comando `bash scripts/build-libs-cross.sh` fallaba porque goreleaser-cross no reconocía el comando
- **Solución**: Añadido `--entrypoint /bin/bash` al comando docker run en `build_shared_libraries()`

### 2. Directivas Replace Duplicadas en go.mod
- **Problema**: Dos directivas `replace` apuntaban al mismo directorio, causando error de Go
```
replace github.com/madeindigio/go-llama.cpp => /www/MCP/Remembrances/go-llama.cpp
replace github.com/go-skynet/go-llama.cpp => /www/MCP/Remembrances/go-llama.cpp
```
- **Solución**: Eliminada la directiva duplicada de `go-skynet/go-llama.cpp`

### 3. Volúmenes Docker No Montados
- **Problema**: GoReleaser no podía acceder a `/www/MCP/Remembrances/` para los módulos locales
- **Solución**: Añadido montaje del volumen en `run_goreleaser()`:
```bash
-v "/www/MCP/Remembrances:/www/MCP/Remembrances"
```

### 4. Dependencia CURL en llama.cpp
- **Problema**: CMake requería CURL que no estaba disponible en el contenedor
- **Solución**: Deshabilitado CURL en flags de CMake:
```bash
-DLLAMA_CURL=OFF
```

### 5. Vendor Directory Desactualizado
- **Problema**: El directorio vendor no estaba sincronizado con go.mod
- **Solución**: Añadido `go mod vendor` a los before hooks de `.goreleaser.yml`

### 6. Rust/Cargo No Disponible
- **Problema**: El contenedor goreleaser-cross no tiene Rust instalado para compilar surrealdb-embedded
- **Solución**: Creada imagen Docker personalizada con Rust

### 7. Herramientas de macOS Faltantes
- **Problema**: `install_name_tool` no disponible para cross-compilation de macOS
- **Estado**: Pendiente - la imagen personalizada debería resolver esto con osxcross

## Archivos Creados

### 1. `docker/Dockerfile.goreleaser-custom`
Dockerfile personalizado que extiende `goreleaser-cross:v1.23` con:
- Rust 1.75.0 y rustup
- Targets de cross-compilación para Rust
- CMake y herramientas de build
- libcurl para compilar llama.cpp

### 2. `scripts/build-docker-image.sh`
Script para construir la imagen Docker personalizada con opciones para:
- Especificar tag personalizado
- Build sin cache
- Push a registry

### 3. `docs/CROSS_COMPILE.md`
Documentación completa sobre:
- Cómo usar la compilación cruzada
- Descripción de scripts
- Variables de entorno
- Troubleshooting
- Integración CI/CD

## Archivos Modificados

### 1. `scripts/release-cross.sh`
**Cambios:**
- Añadida variable `GORELEASER_CROSS_IMAGE` para usar imagen personalizada
- Actualizado `build_shared_libraries()` con `--entrypoint /bin/bash`
- Actualizado `run_goreleaser()` para montar volumen `/www/MCP/Remembrances`
- Todas las referencias a la imagen Docker ahora usan `${GORELEASER_CROSS_IMAGE}`
- Añadida tolerancia a fallos en la compilación de librerías

### 2. `scripts/build-libs-cross.sh`
**Cambios:**
- Añadido flag `-DLLAMA_CURL=OFF` en cmake_flags para todas las plataformas

### 3. `go.mod`
**Cambios:**
- Eliminada directiva `replace github.com/go-skynet/go-llama.cpp => /www/MCP/Remembrances/go-llama.cpp`

### 4. `.goreleaser.yml`
**Cambios:**
- Añadido `go mod vendor` a los before hooks

## Uso

### Opción 1: Con Imagen Personalizada (Recomendado)

```bash
# 1. Construir la imagen personalizada
./scripts/build-docker-image.sh

# 2. Ejecutar compilación cruzada
export GORELEASER_CROSS_IMAGE=remembrances-mcp-builder:latest
./scripts/release-cross.sh --clean snapshot
```

### Opción 2: Sin Librerías Compartidas (Más Rápido)

```bash
# Solo compilar binarios Go sin librerías compartidas
./scripts/release-cross.sh --skip-libs --clean snapshot
```

### Opción 3: Solo Librerías

```bash
# Solo compilar las librerías compartidas
export GORELEASER_CROSS_IMAGE=remembrances-mcp-builder:latest
./scripts/release-cross.sh --libs-only
```

## Variables de Entorno

```bash
# Especificar imagen Docker personalizada
export GORELEASER_CROSS_IMAGE=remembrances-mcp-builder:v1.23-rust

# O usar versión específica de goreleaser-cross estándar
export GORELEASER_CROSS_VERSION=v1.22

# Para releases a GitHub
export GITHUB_TOKEN=your_token_here
```

## Estructura de Salida

```
dist/
├── libs/                           # Librerías compartidas por plataforma
│   ├── linux-amd64/
│   │   ├── libllama.so
│   │   ├── libggml.so
│   │   └── libsurrealdb_embedded_rs.so
│   ├── linux-arm64/
│   ├── darwin-amd64/
│   ├── darwin-arm64/
│   ├── windows-amd64/
│   └── windows-arm64/
└── outputs/
    └── dist/                       # Archivos de release
        ├── remembrances-mcp_VERSION_linux_amd64.tar.gz
        ├── remembrances-mcp_VERSION_linux_arm64.tar.gz
        ├── remembrances-mcp_VERSION_darwin_amd64.tar.gz
        ├── remembrances-mcp_VERSION_darwin_arm64.tar.gz
        ├── remembrances-mcp_VERSION_windows_amd64.zip
        ├── remembrances-mcp_VERSION_windows_arm64.zip
        └── checksums.txt
```

## Plataformas Soportadas

- ✅ Linux AMD64
- ✅ Linux ARM64
- ⚠️  macOS AMD64 (requiere osxcross configurado)
- ⚠️  macOS ARM64 (requiere osxcross configurado)
- ⚠️  Windows AMD64 (compilación básica funciona, librerías pendientes)
- ⚠️  Windows ARM64 (soporte experimental)

## Próximos Pasos

1. **Probar imagen Docker personalizada** - Verificar que Rust y todas las herramientas funcionan
2. **Compilar librerías compartidas** - Ejecutar `--libs-only` para verificar
3. **Compilación completa** - Ejecutar snapshot completo con todas las plataformas
4. **Validar binarios** - Probar binarios en cada plataforma
5. **CI/CD** - Integrar en GitHub Actions para builds automatizados

## Referencias

- [GoReleaser Cross](https://github.com/goreleaser/goreleaser-cross)
- [Rust Linux Darwin Builder](https://github.com/joseluisq/rust-linux-darwin-builder)
- [llama.cpp](https://github.com/ggerganov/llama.cpp)
- [SurrealDB Embedded](https://surrealdb.com/)
