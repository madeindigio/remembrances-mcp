# Guía de Distribución de Binarios

Esta guía te ayuda a crear paquetes de distribución de Remembrances-MCP con librerías optimizadas para diferentes GPUs.

## Resumen Rápido

```bash
# 1. Compilar todas las variantes disponibles
make build-libs-all-variants

# 2. Empaquetar para distribución
make package-libs-all

# 3. Los paquetes están en dist/libs/
ls -lh dist/libs/
```

## Estructura de Paquetes

Cada paquete incluye:

```
llama-cpp-{variant}-{platform}-{arch}.tar.gz
├── libllama.so (o .dylib)
├── libggml.so
├── libggml-base.so
├── libggml-{variant}.so  ← Librería específica de GPU
└── BUILD_INFO.txt        ← Metadata de compilación
```

## Compilar para Distribución

### En Linux (Máquina con NVIDIA GPU)

```bash
# Compilar variantes Linux
make build-libs-cpu        # Para usuarios sin GPU
make build-libs-cuda       # Para usuarios con NVIDIA
make build-libs-openblas   # Para CPUs optimizados (opcional)

# Empaquetar
make package-libs-all
```

Genera:
- `llama-cpp-cpu-linux-x86_64.tar.gz` (5-10 MB)
- `llama-cpp-cuda-linux-x86_64.tar.gz` (50-100 MB)
- `llama-cpp-openblas-linux-x86_64.tar.gz` (5-10 MB)

### En Linux (Máquina con AMD GPU)

```bash
# Compilar variantes AMD
make build-libs-cpu
make build-libs-hipblas    # Para usuarios con AMD Radeon

# Empaquetar
make package-libs-all
```

Genera:
- `llama-cpp-cpu-linux-x86_64.tar.gz`
- `llama-cpp-hipblas-linux-x86_64.tar.gz` (50-100 MB)

### En macOS (Apple Silicon)

```bash
# Compilar variantes macOS
make build-libs-cpu
make build-libs-metal      # Para M1/M2/M3

# Empaquetar
make package-libs-all
```

Genera:
- `llama-cpp-cpu-darwin-arm64.tar.gz`
- `llama-cpp-metal-darwin-arm64.tar.gz`

## Instrucciones para Usuarios Finales

### Instalación con GPU NVIDIA

```bash
# 1. Descargar binario principal y librerías CUDA
wget https://github.com/tu-repo/releases/download/v1.0/remembrances-mcp-linux-amd64
wget https://github.com/tu-repo/releases/download/v1.0/llama-cpp-cuda-linux-x86_64.tar.gz

# 2. Extraer librerías
mkdir -p build
tar -xzf llama-cpp-cuda-linux-x86_64.tar.gz -C build/
mv build/cuda/*.so build/

# 3. Hacer ejecutable
chmod +x remembrances-mcp-linux-amd64

# 4. Ejecutar con GPU
./remembrances-mcp-linux-amd64 \
  --gguf-model-path model.gguf \
  --gguf-gpu-layers 32
```

### Instalación sin GPU (CPU solamente)

```bash
# 1. Descargar binario principal y librerías CPU
wget https://github.com/tu-repo/releases/download/v1.0/remembrances-mcp-linux-amd64
wget https://github.com/tu-repo/releases/download/v1.0/llama-cpp-cpu-linux-x86_64.tar.gz

# 2. Extraer librerías
mkdir -p build
tar -xzf llama-cpp-cpu-linux-x86_64.tar.gz -C build/
mv build/cpu/*.so build/

# 3. Hacer ejecutable
chmod +x remembrances-mcp-linux-amd64

# 4. Ejecutar
./remembrances-mcp-linux-amd64 \
  --gguf-model-path model.gguf
```

## README para Releases de GitHub

Incluye esto en tus releases:

```markdown
## Descargas

### Binario Principal

- [remembrances-mcp-linux-amd64](link) - Linux x86_64
- [remembrances-mcp-darwin-arm64](link) - macOS Apple Silicon
- [remembrances-mcp-darwin-amd64](link) - macOS Intel

### Librerías GPU (Descarga según tu hardware)

#### NVIDIA GPU (CUDA)
- [llama-cpp-cuda-linux-x86_64.tar.gz](link) - **Requerido para GPUs NVIDIA**
- Requiere: NVIDIA Driver 520+ y CUDA Runtime 11.5+
- GPUs soportadas: RTX 20/30/40 series, GTX 16 series, Tesla, etc.

#### AMD GPU (ROCm)
- [llama-cpp-hipblas-linux-x86_64.tar.gz](link) - **Requerido para GPUs AMD**
- Requiere: ROCm 5.0+
- GPUs soportadas: Radeon RX 6000/7000, MI100/MI200

#### Apple Silicon (Metal)
- [llama-cpp-metal-darwin-arm64.tar.gz](link) - **Requerido para M1/M2/M3**
- Incluido automáticamente en macOS

#### CPU Solamente
- [llama-cpp-cpu-linux-x86_64.tar.gz](link) - Sin aceleración GPU
- [llama-cpp-cpu-darwin-arm64.tar.gz](link) - macOS sin GPU

#### CPU Optimizado (OpenBLAS)
- [llama-cpp-openblas-linux-x86_64.tar.gz](link) - CPU con BLAS optimizado

## Instalación Rápida

### Con GPU NVIDIA
\`\`\`bash
# Linux
wget [binario] [cuda-libs]
tar -xzf llama-cpp-cuda-*.tar.gz
mv cuda/*.so .
chmod +x remembrances-mcp-*
./remembrances-mcp-* --gguf-model-path model.gguf --gguf-gpu-layers 32
\`\`\`

### Sin GPU
\`\`\`bash
# Linux
wget [binario] [cpu-libs]
tar -xzf llama-cpp-cpu-*.tar.gz
mv cpu/*.so .
chmod +x remembrances-mcp-*
./remembrances-mcp-* --gguf-model-path model.gguf
\`\`\`

## Verificación

Verifica que las librerías correctas están cargadas:

\`\`\`bash
# Ver librerías enlazadas
ldd ./remembrances-mcp-linux-amd64

# Ver información de compilación
cat BUILD_INFO.txt
\`\`\`

## Requisitos del Sistema

| Variante | Requisitos Mínimos |
|----------|-------------------|
| CPU | Cualquier CPU x86_64 |
| CUDA | NVIDIA GPU + Driver 520+ + CUDA Runtime 11.5+ |
| HIPBlas | AMD GPU + ROCm 5.0+ |
| Metal | macOS + Apple Silicon (M1/M2/M3) |
| OpenBLAS | libopenblas instalado |
```

## Script de Instalación Automática

Crea un instalador para usuarios:

```bash
#!/bin/bash
# install.sh - Instalador automático de Remembrances-MCP

set -e

echo "=== Instalador de Remembrances-MCP ==="
echo ""

# Detectar GPU
GPU_TYPE="cpu"

if command -v nvidia-smi &> /dev/null; then
    echo "✓ GPU NVIDIA detectada"
    GPU_TYPE="cuda"
elif [ -d "/opt/rocm" ]; then
    echo "✓ GPU AMD/ROCm detectada"
    GPU_TYPE="hipblas"
elif [ "$(uname -s)" = "Darwin" ] && [ "$(uname -m)" = "arm64" ]; then
    echo "✓ Apple Silicon detectado"
    GPU_TYPE="metal"
else
    echo "ℹ No se detectó GPU, usando CPU"
    GPU_TYPE="cpu"
fi

# Detectar plataforma
PLATFORM=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# URLs de descarga (actualiza con tus releases)
BASE_URL="https://github.com/tu-repo/releases/download/v1.0.0"
BINARY_URL="$BASE_URL/remembrances-mcp-$PLATFORM-$ARCH"
LIBS_URL="$BASE_URL/llama-cpp-$GPU_TYPE-$PLATFORM-$ARCH.tar.gz"

echo ""
echo "Configuración detectada:"
echo "  Plataforma: $PLATFORM"
echo "  Arquitectura: $ARCH"
echo "  GPU: $GPU_TYPE"
echo ""

# Descargar
echo "Descargando binario..."
wget -q --show-progress "$BINARY_URL" -O remembrances-mcp
chmod +x remembrances-mcp

echo "Descargando librerías $GPU_TYPE..."
wget -q --show-progress "$LIBS_URL"

# Extraer
echo "Extrayendo librerías..."
tar -xzf "llama-cpp-$GPU_TYPE-$PLATFORM-$ARCH.tar.gz"
mv "$GPU_TYPE"/*.so . 2>/dev/null || mv "$GPU_TYPE"/*.dylib . 2>/dev/null || true
rm -rf "$GPU_TYPE"
rm "llama-cpp-$GPU_TYPE-$PLATFORM-$ARCH.tar.gz"

echo ""
echo "✓ Instalación completada!"
echo ""
echo "Próximos pasos:"
echo "  1. Descarga un modelo GGUF"
echo "  2. Ejecuta: ./remembrances-mcp --gguf-model-path modelo.gguf"

if [ "$GPU_TYPE" != "cpu" ]; then
    echo "  3. Para usar GPU: --gguf-gpu-layers 32"
fi

echo ""
```

## Automatización con GitHub Actions

Ejemplo de workflow para compilar y publicar releases:

```yaml
name: Build and Release

on:
  push:
    tags:
      - 'v*'

jobs:
  build-linux-cuda:
    runs-on: ubuntu-latest-gpu-nvidia
    steps:
      - uses: actions/checkout@v3
      
      - name: Build CUDA variant
        run: |
          make build-libs-cuda
          make package-libs-all
      
      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: llama-cpp-cuda
          path: dist/libs/llama-cpp-cuda-*.tar.gz

  build-macos-metal:
    runs-on: macos-14  # M1 runner
    steps:
      - uses: actions/checkout@v3
      
      - name: Build Metal variant
        run: |
          make build-libs-metal
          make package-libs-all
      
      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: llama-cpp-metal
          path: dist/libs/llama-cpp-metal-*.tar.gz
```

## Checksums

Genera checksums para verificación:

```bash
# Generar SHA256 checksums
cd dist/libs
sha256sum *.tar.gz > SHA256SUMS
```

Usuarios pueden verificar:

```bash
sha256sum -c SHA256SUMS
```

## Tamaños Esperados

| Paquete | Tamaño Aproximado |
|---------|------------------|
| CPU | 5-10 MB |
| CUDA | 50-100 MB |
| HIPBlas | 50-100 MB |
| Metal | 10-20 MB |
| OpenBLAS | 8-15 MB |

## Siguiente Paso

Compila tus variantes y crea tu primer release:

```bash
# 1. Compilar todas las variantes disponibles
make build-libs-all-variants

# 2. Empaquetar
make package-libs-all

# 3. Revisar paquetes
ls -lh dist/libs/

# 4. Crear release en GitHub
# Sube los archivos de dist/libs/ como assets
```
