# Build & Installation Scripts

Este directorio contiene scripts para compilar, crear releases e instalar el proyecto Remembrances-MCP.

## Scripts Disponibles

### 📦 `install.sh`

Script de instalación automática para Linux y macOS. Permite instalar Remembrances-MCP con un solo comando.

**Instalación rápida:**
```bash
curl -fsSL https://raw.githubusercontent.com/madeindigio/remembrances-mcp/main/scripts/install.sh | bash
```

**¿Qué hace?**
1. Detecta automáticamente el sistema operativo (Linux/macOS)
2. Detecta la arquitectura (amd64/aarch64)
3. Descarga la release apropiada de GitHub
4. Instala el binario y las bibliotecas compartidas
5. Crea la configuración por defecto con rutas apropiadas
6. Descarga el modelo GGUF de embeddings (~260MB)
7. Configura el PATH en `.bashrc` y `.zshrc`

**Directorios de instalación:**

| OS | Binario + Libraries | Configuración |
|----|---------------------|---------------|
| Linux | `~/.local/share/remembrances/bin/` | `~/.config/remembrances/` |
| macOS | `~/Library/Application Support/remembrances/bin/` | `~/Library/Application Support/remembrances/` |

> **Nota:** Las shared libraries (`.so`, `.dylib`) se instalan en el mismo directorio que el binario. El binario está compilado para buscar las bibliotecas primero en su propio directorio.

**Variables de entorno:**
- `REMEMBRANCES_VERSION` - Versión a instalar (default: `v1.4.6`)

**Ejemplo con versión específica:**
```bash
REMEMBRANCES_VERSION=v1.4.5 curl -fsSL https://raw.githubusercontent.com/madeindigio/remembrances-mcp/main/scripts/install.sh | bash
```

**Después de la instalación:**
```bash
# Recargar shell
source ~/.bashrc  # o ~/.zshrc

# Verificar instalación
remembrances-mcp --help
```

---

### 🚀 `release-cross.sh`

Script principal para cross-compilation usando Docker y goreleaser-cross.

**Uso:**
```bash
./scripts/release-cross.sh [OPTIONS] [COMMAND]
```

**Comandos:**
- `build` - Compilar binarios sin crear release (default)
- `release` - Compilar y crear release en GitHub
- `snapshot` - Compilar snapshot release (no requiere tag)

**Opciones:**
- `-v, --version VERSION` - Versión de goreleaser-cross (default: v1.23)
- `-c, --clean` - Limpiar antes de compilar
- `--skip-libs` - Saltar compilación de shared libraries
- `--libs-only` - Solo compilar shared libraries
- `-h, --help` - Mostrar ayuda

**Ejemplos:**
```bash
# Snapshot para testing
./scripts/release-cross.sh snapshot

# Con versión específica
./scripts/release-cross.sh -v v1.22 build

# Solo libraries
./scripts/release-cross.sh --libs-only

# Release completo
export GITHUB_TOKEN="tu_token"
./scripts/release-cross.sh release
```

---

### 🔧 `build-libs-cross.sh`

Script para compilar shared libraries (llama.cpp y surrealdb-embedded) para todas las plataformas.

**Nota:** Este script debe ejecutarse dentro del container Docker goreleaser-cross.

**Uso:**
```bash
# Normalmente llamado por release-cross.sh
docker run --rm \
  -v $PWD:/go/src/github.com/madeindigio/remembrances-mcp \
  -v ~/www/MCP/Remembrances:~/www/MCP/Remembrances \
  -w /go/src/github.com/madeindigio/remembrances-mcp \
  ghcr.io/goreleaser/goreleaser-cross:v1.23 \
  bash scripts/build-libs-cross.sh
```

**Funciones:**
- `build_llama_cpp()` - Compila llama.cpp usando CMake
- `build_surrealdb_embedded()` - Compila surrealdb-embedded usando Rust/cargo
- `build_for_platform()` - Orquesta la compilación para una plataforma

**Output:**
Las bibliotecas compiladas se colocan en:
```
dist/libs/{platform}-{arch}/
  ├── libllama.so (o .dylib para macOS)
  ├── libggml.so
  ├── libggml-base.so
  ├── libcommon.so
  └── libsurrealdb_embedded_rs.so
```

---

### 🧪 `test-gguf.sh`

Script para probar la funcionalidad GGUF del proyecto.

**Uso:**
```bash
./scripts/test-gguf.sh
```

## Variables de Entorno

### Para `release-cross.sh`:

| Variable | Descripción | Default |
|----------|-------------|---------|
| `GORELEASER_CROSS_VERSION` | Versión de la imagen Docker | `v1.23` |
| `GITHUB_TOKEN` | Token para GitHub releases | - |

### Para `build-libs-cross.sh`:

| Variable | Descripción | Default |
|----------|-------------|---------|
| `PROJECT_ROOT` | Raíz del proyecto | `/go/src/github.com/madeindigio/remembrances-mcp` |
| `LLAMA_CPP_DIR` | Directorio de llama.cpp | `~/www/MCP/Remembrances/go-llama.cpp` |
| `SURREALDB_DIR` | Directorio de surrealdb-embedded | `~/www/MCP/Remembrances/surrealdb-embedded` |
| `DIST_LIBS_DIR` | Directorio de salida | `${PROJECT_ROOT}/dist/libs` |

## Flujo de Trabajo

### 1. Development Build (Snapshot)

Para desarrollo y testing:

```bash
# Opción 1: Script directo
./scripts/release-cross.sh snapshot

# Opción 2: Make
make build-cross
```

**Output:** 
- Binarios en `dist/outputs/dist/`
- No crea release en GitHub
- No requiere git tag

### 2. Solo Compilar Libraries

Para compilar solo las bibliotecas compartidas:

```bash
# Opción 1: Script directo
./scripts/release-cross.sh --libs-only

# Opción 2: Make
make build-libs-cross
```

**Output:**
- Libraries en `dist/libs/{platform}-{arch}/`
- No compila binarios Go

### 3. Production Release

Para crear un release oficial:

```bash
# 1. Crear y pushear tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# 2. Compilar y release
export GITHUB_TOKEN="ghp_xxxxxxxxxxxxx"

# Opción 1: Script directo
./scripts/release-cross.sh release

# Opción 2: Make
make release-cross
```

**Output:**
- Binarios en `dist/outputs/dist/`
- Release creado en GitHub
- Archives subidos como release assets
- Checksums generados

## Plataformas Compiladas

El sistema compila para las siguientes plataformas:

| OS | Arquitectura | Compilador C | Compilador C++ |
|----|--------------|--------------|----------------|
| Linux | amd64 | x86_64-linux-gnu-gcc | x86_64-linux-gnu-g++ |
| Linux | arm64 | aarch64-linux-gnu-gcc | aarch64-linux-gnu-g++ |
| macOS | amd64 | o64-clang | o64-clang++ |
| macOS | arm64 | oa64-clang | oa64-clang++ |
| Windows | amd64 | x86_64-w64-mingw32-gcc | x86_64-w64-mingw32-g++ |
| Windows | arm64 | aarch64-w64-mingw32-gcc | aarch64-w64-mingw32-g++ |

## Estructura de Output

```
dist/
├── libs/                                    # Shared libraries
│   ├── linux-amd64/
│   │   ├── libllama.so
│   │   ├── libggml.so
│   │   ├── libggml-base.so
│   │   ├── libcommon.so
│   │   └── libsurrealdb_embedded_rs.so
│   ├── linux-arm64/
│   │   └── ...
│   ├── darwin-amd64/
│   │   ├── libllama.dylib
│   │   └── ...
│   ├── darwin-arm64/
│   │   └── ...
│   ├── windows-amd64/
│   │   └── ... (*.dll)
│   └── windows-arm64/
│       └── ... (*.dll)
└── outputs/
    └── dist/                                # Distribuciones finales
        ├── remembrances-mcp_v1.0.0_linux_amd64.tar.gz
        ├── remembrances-mcp_v1.0.0_linux_arm64.tar.gz
        ├── remembrances-mcp_v1.0.0_darwin_amd64.tar.gz
        ├── remembrances-mcp_v1.0.0_darwin_arm64.tar.gz
        ├── remembrances-mcp_v1.0.0_windows_amd64.zip
        ├── remembrances-mcp_v1.0.0_windows_arm64.zip
        └── checksums.txt
```

## Troubleshooting

### Error: "Docker not found"

```bash
# Instalar Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Verificar instalación
docker --version
```

### Error: "Permission denied"

```bash
# Hacer scripts ejecutables
chmod +x scripts/*.sh
```

### Error: "Failed to build llama.cpp"

```bash
# Verificar que el submodulo existe
ls -la ~/www/MCP/Remembrances/go-llama.cpp/llama.cpp

# Inicializar submodulos si es necesario
cd ~/www/MCP/Remembrances/go-llama.cpp
git submodule update --init --recursive
```

### Error: "Rust target not found"

```bash
# Dentro del container Docker, añadir targets
rustup target add x86_64-unknown-linux-gnu
rustup target add aarch64-unknown-linux-gnu
rustup target add x86_64-apple-darwin
rustup target add aarch64-apple-darwin
```

### Libraries no incluidas en archive

```bash
# Verificar que las libraries se compilaron
ls -R dist/libs/

# Verificar logs del post-hook
# El post-hook copia las libraries al directorio del binario
```

## Mantenimiento

### Actualizar versión de goreleaser-cross

```bash
# Opción 1: Variable de entorno
export GORELEASER_CROSS_VERSION=v1.24
./scripts/release-cross.sh snapshot

# Opción 2: Flag
./scripts/release-cross.sh -v v1.24 snapshot
```

### Limpiar builds anteriores

```bash
# Limpiar todo
rm -rf dist/

# O usar make
make clean
```

### Debug de compilación

```bash
# Habilitar modo verbose añadiendo set -x en el script
# O capturar logs completos
./scripts/release-cross.sh snapshot 2>&1 | tee build.log
```

## Referencias

- [GoReleaser Documentation](https://goreleaser.com/)
- [goreleaser-cross](https://github.com/goreleaser/goreleaser-cross)
- [Documentación completa](../docs/CROSS_COMPILE.md)
- [Resumen de cambios](../CROSS_COMPILE_SETUP.md)
