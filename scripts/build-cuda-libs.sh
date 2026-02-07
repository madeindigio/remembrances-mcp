#!/bin/bash
# Script para compilar llama.cpp con soporte CUDA para NVIDIA GPUs
# y copiar las librerías al directorio build/

set -e

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Directorios
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
GO_LLAMA_DIR="${GO_LLAMA_DIR:-$HOME/www/MCP/Remembrances/go-llama.cpp}"
BUILD_DIR="$PROJECT_ROOT/build"

echo -e "${GREEN}=== Compilación de llama.cpp con soporte CUDA ===${NC}"
echo ""

# Verificar que existe CUDA
echo -e "${YELLOW}[1/6] Verificando CUDA...${NC}"
if ! command -v nvcc &> /dev/null; then
    echo -e "${RED}Error: nvcc no encontrado. Asegúrate de tener CUDA Toolkit instalado.${NC}"
    echo "Instalación: https://developer.nvidia.com/cuda-downloads"
    exit 1
fi

CUDA_VERSION=$(nvcc --version | grep "release" | sed -n 's/.*release \([0-9.]*\).*/\1/p')
echo -e "${GREEN}✓ CUDA Toolkit $CUDA_VERSION encontrado${NC}"

# Verificar GPU
if ! command -v nvidia-smi &> /dev/null; then
    echo -e "${YELLOW}Advertencia: nvidia-smi no encontrado. ¿Tienes drivers NVIDIA instalados?${NC}"
else
    echo -e "${GREEN}✓ GPU NVIDIA detectada:${NC}"
    nvidia-smi --query-gpu=name,driver_version,memory.total --format=csv,noheader | head -1
fi

# Verificar directorio go-llama
echo ""
echo -e "${YELLOW}[2/6] Verificando go-llama.cpp...${NC}"
if [ ! -d "$GO_LLAMA_DIR" ]; then
    echo -e "${RED}Error: go-llama.cpp no encontrado en $GO_LLAMA_DIR${NC}"
    echo "Por favor, clona el repositorio primero."
    exit 1
fi

if [ ! -d "$GO_LLAMA_DIR/llama.cpp" ]; then
    echo -e "${YELLOW}Inicializando submodulos de llama.cpp...${NC}"
    cd "$GO_LLAMA_DIR"
    git submodule update --init --recursive
fi
echo -e "${GREEN}✓ go-llama.cpp encontrado${NC}"

# Limpiar compilación anterior
echo ""
echo -e "${YELLOW}[3/6] Limpiando compilación anterior...${NC}"
cd "$GO_LLAMA_DIR"
if [ -d "build" ]; then
    echo "Eliminando directorio build anterior..."
    rm -rf build
fi
rm -f prepare
rm -f *.o *.a
echo -e "${GREEN}✓ Limpieza completada${NC}"

# Compilar con CUDA
echo ""
echo -e "${YELLOW}[4/6] Compilando llama.cpp con soporte CUDA...${NC}"
echo "Esto puede tardar varios minutos..."
echo ""

# Configurar variables de entorno para CUDA
# Buscar la versión más reciente de CUDA instalada
if command -v nvcc &> /dev/null; then
    # Detectar CUDA_HOME desde nvcc
    NVCC_PATH=$(which nvcc)
    CUDA_HOME=$(dirname $(dirname $NVCC_PATH))
    echo -e "${GREEN}✓ CUDA detectado en: $CUDA_HOME${NC}"
    
    # Verificar versión de nvcc
    NVCC_VERSION=$(nvcc --version | grep "release" | sed -n 's/.*release \([0-9.]*\).*/\1/p')
    echo "nvcc versión: $NVCC_VERSION"
else
    echo -e "${RED}Error: nvcc no encontrado. Asegúrate de tener CUDA Toolkit instalado.${NC}"
    exit 1
fi

export CUDA_HOME
export PATH=$CUDA_HOME/bin:$PATH

# Configurar LD_LIBRARY_PATH según la ubicación de las librerías CUDA
if [ -d "$CUDA_HOME/lib64" ]; then
    export LD_LIBRARY_PATH=$CUDA_HOME/lib64:$LD_LIBRARY_PATH
elif [ -d "$CUDA_HOME/lib" ]; then
    export LD_LIBRARY_PATH=$CUDA_HOME/lib:$LD_LIBRARY_PATH
fi

# Detectar arquitectura CUDA de la GPU
echo "Detectando arquitectura CUDA de la GPU..."
GPU_ARCH=$(nvidia-smi --query-gpu=compute_cap --format=csv,noheader | head -1 | tr -d '.')
if [ -z "$GPU_ARCH" ]; then
    echo -e "${YELLOW}Advertencia: No se pudo detectar arquitectura GPU. Usando 86 (RTX 30xx) por defecto.${NC}"
    GPU_ARCH="86"
fi

# Permite compilar fatbin con varias arquitecturas.
# Ejemplo: GPU_ARCH_LIST="75;80;86;89" (sin prefijo sm_)
GPU_ARCHES="${GPU_ARCH_LIST:-$GPU_ARCH}"
if [ -n "$GPU_ARCH_LIST" ]; then
    echo -e "${GREEN}✓ Arquitecturas CUDA configuradas (fatbin): ${GPU_ARCHES}${NC}"
else
    echo -e "${GREEN}✓ Arquitectura CUDA detectada: sm_${GPU_ARCH}${NC}"
fi

# Compilar usando CMake con CUDA habilitado
mkdir -p build
cd build

echo "Ejecutando CMake con CUDA habilitado..."

# Detectar versión de CUDA del nvcc configurado
CUDA_VERSION_MAJOR=$(nvcc --version | grep "release" | sed -n 's/.*release \([0-9]*\)\..*/\1/p')
if [ -z "$CUDA_VERSION_MAJOR" ]; then
    echo -e "${YELLOW}Advertencia: No se pudo detectar versión de CUDA. Asumiendo CUDA 12${NC}"
    CUDA_VERSION_MAJOR=12
fi
echo "Versión CUDA detectada: $CUDA_VERSION_MAJOR.x"

# IMPORTANTE: Forzar que CMake use el nvcc correcto
export CUDACXX=$(which nvcc)
export CMAKE_CUDA_COMPILER=$(which nvcc)
echo "Forzando uso de: ${CMAKE_CUDA_COMPILER}"

# Si CUDA < 12, deshabilitar características avanzadas que requieren PTX 7.8+
CMAKE_CUDA_FLAGS=""
if [ "$CUDA_VERSION_MAJOR" -lt 12 ]; then
    echo -e "${YELLOW}Advertencia: CUDA $CUDA_VERSION_MAJOR detectado. Deshabilitando Flash Attention (requiere CUDA 12+)${NC}"
    # Deshabilitar Flash Attention que requiere PTX 7.8+ (CUDA 12+)
    CMAKE_CUDA_FLAGS="-DGGML_CUDA_FLASH_ATTN=OFF -DGGML_CUDA_FA_ALL_QUANTS=OFF"
fi

# Opciones de portabilidad CPU (Intel/AMD)
# PORTABLE=1 : Compilación genérica compatible con Intel y AMD (AVX2, sin AVX-512)
# PORTABLE=0 : Optimizado para la CPU actual (-march=native)
PORTABLE="${PORTABLE:-0}"

CMAKE_CPU_FLAGS=""
if [ "$PORTABLE" = "1" ]; then
    echo -e "${YELLOW}Modo PORTABLE: Compilando para compatibilidad Intel/AMD (AVX2, sin AVX-512)${NC}"
    # Deshabilitar optimizaciones nativas y AVX-512 para máxima compatibilidad
    CMAKE_CPU_FLAGS="-DGGML_NATIVE=OFF -DGGML_AVX512=OFF -DGGML_AVX512_VBMI=OFF -DGGML_AVX512_VNNI=OFF"
    # Usar x86-64-v3 (AVX2 + FMA) que funciona en Intel Haswell+ y AMD Zen+
    export CFLAGS="-march=x86-64-v3 -mtune=generic"
    export CXXFLAGS="-march=x86-64-v3 -mtune=generic"
else
    echo -e "${GREEN}Modo NATIVO: Optimizando para CPU actual${NC}"
fi

cmake ../llama.cpp \
    -DCMAKE_CUDA_COMPILER=$(which nvcc) \
    -DGGML_CUDA=ON \
    -DLLAMA_STATIC=OFF \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_CUDA_ARCHITECTURES=${GPU_ARCHES} \
    ${CMAKE_CUDA_FLAGS} \
    ${CMAKE_CPU_FLAGS}

echo ""
echo "Compilando con $(nproc) cores..."
cmake --build . --config Release -j$(nproc)

echo -e "${GREEN}✓ Compilación exitosa${NC}"

# Verificar librerías compiladas
echo ""
echo -e "${YELLOW}[5/6] Verificando librerías compiladas...${NC}"
LIBS_FOUND=0

echo "Buscando librerías en build/..."
find . -type f \( -name "*.so" -o -name "*.dylib" \) -print

if find . -type f -name "libllama.so" | grep -q .; then
    echo -e "${GREEN}✓ libllama.so encontrada${NC}"
    LIBS_FOUND=$((LIBS_FOUND + 1))
fi

if find . -type f -name "libggml*.so" | grep -q .; then
    echo -e "${GREEN}✓ librerías libggml*.so encontradas${NC}"
    LIBS_FOUND=$((LIBS_FOUND + 1))
fi

if [ $LIBS_FOUND -eq 0 ]; then
    echo -e "${RED}Error: No se encontraron librerías compiladas${NC}"
    exit 1
fi

# Copiar librerías al directorio build del proyecto
echo ""
echo -e "${YELLOW}[6/6] Copiando librerías al directorio build...${NC}"
mkdir -p "$BUILD_DIR"

# Copiar todas las librerías .so desde build/
echo "Copiando librerías .so..."
COPIED=0

# Buscar y copiar desde diferentes ubicaciones
for dir in bin src common ggml .; do
    if [ -d "$dir" ]; then
        for lib in $(find "$dir" -maxdepth 1 -type f -name "*.so*" 2>/dev/null); do
            echo "  Copiando $(basename $lib)..."
            cp -f "$lib" "$BUILD_DIR/"
            COPIED=$((COPIED + 1))
        done
    fi
done

if [ $COPIED -eq 0 ]; then
    echo -e "${RED}Error: No se copiaron librerías${NC}"
    exit 1
fi

echo -e "${GREEN}✓ $COPIED librerías copiadas a $BUILD_DIR/${NC}"

# Compilar la shim (libllama_shim.so) junto a las librerías copiadas.
echo ""
echo -e "${YELLOW}Compilando libllama_shim...${NC}"

SHIM_SRC="$PROJECT_ROOT/internal/llama_shim/llama_shim.c"
SHIM_INC="$PROJECT_ROOT/internal/llama_shim"

if [ -f "$SHIM_SRC" ]; then
    cc_bin="${CC:-gcc}"
    "$cc_bin" -shared -fPIC -O3 \
        -I"$SHIM_INC" \
        -L"$BUILD_DIR" -lllama \
        -Wl,-rpath,'$ORIGIN' \
        -o "$BUILD_DIR/libllama_shim.so" \
        "$SHIM_SRC" -lm
    echo -e "${GREEN}✓ libllama_shim.so creada en $BUILD_DIR/${NC}"
else
    echo -e "${YELLOW}Advertencia: fuente de shim no encontrado en $SHIM_SRC (omitido)${NC}"
fi

# Mostrar resumen
echo ""
echo -e "${GREEN}=== Compilación completada exitosamente ===${NC}"
echo ""
echo "Librerías disponibles en $BUILD_DIR/:"
ls -lh "$BUILD_DIR"/*.so* 2>/dev/null | awk '{print "  " $9 " (" $5 ")"}'

echo ""
echo -e "${GREEN}Siguiente paso:${NC}"
echo "1. Recompila tu proyecto:"
echo "   cd $PROJECT_ROOT"
echo "   make clean"
echo "   make BUILD_TYPE=cublas build"
echo ""
echo "2. Al ejecutar, usa --gguf-gpu-layers para habilitar GPU:"
echo "   ./build/remembrances-mcp --gguf-model-path ./models/tu-modelo.gguf --gguf-gpu-layers 32"
echo ""
echo -e "${YELLOW}Nota: El número de capas GPU depende de tu VRAM. RTX 3060 (6GB) puede usar ~32 capas.${NC}"
echo ""
echo -e "${GREEN}Opciones de compilación:${NC}"
echo "  PORTABLE=0 (default) - Optimizado para tu CPU actual (más rápido, menos compatible)"
echo "  PORTABLE=1           - Compatible con Intel y AMD (AVX2, sin AVX-512)"
echo ""
echo "Para compilar portable (Intel/AMD compatible):"
echo "  PORTABLE=1 $0"
