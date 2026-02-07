#!/bin/bash
# Script para compilar una variante específica de llama.cpp
# Uso: ./scripts/build-variant-libs.sh <variant>
# Variantes: cpu, cuda, hipblas, metal, openblas

set -e

# Colores
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Verificar argumento
if [ -z "$1" ]; then
    echo -e "${RED}Error: Variante no especificada${NC}"
    echo "Uso: $0 <variant>"
    echo "Variantes disponibles: cpu, cuda, hipblas, metal, openblas"
    exit 1
fi

VARIANT=$1
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
GO_LLAMA_DIR="${GO_LLAMA_DIR:-$HOME/www/MCP/Remembrances/go-llama.cpp}"
BUILD_DIR="$PROJECT_ROOT/build/libs/$VARIANT"

echo -e "${BLUE}=== Compilando llama.cpp - Variante: $VARIANT ===${NC}"
echo ""

# Crear directorio de destino
mkdir -p "$BUILD_DIR"

# Limpiar compilación anterior
echo -e "${YELLOW}[1/4] Limpiando compilación anterior...${NC}"
cd "$GO_LLAMA_DIR"
rm -rf build
rm -f prepare *.o *.a
echo -e "${GREEN}✓ Limpieza completada${NC}"

# Preparar flags según variante
echo ""
echo -e "${YELLOW}[2/4] Configurando variante $VARIANT...${NC}"

CMAKE_FLAGS="-DLLAMA_STATIC=OFF -DCMAKE_BUILD_TYPE=Release"

case "$VARIANT" in
    cpu)
        echo "Configurando para CPU solamente..."
        # No se necesitan flags adicionales
        ;;

    cuda)
        echo "Configurando para NVIDIA CUDA..."
        # Detectar CUDA 12.6 si existe
        if [ -d "/usr/local/cuda-12.6" ]; then
            export CUDA_HOME=/usr/local/cuda-12.6
            echo "Usando CUDA 12.6"
        elif [ -d "/usr/local/cuda-12" ]; then
            export CUDA_HOME=/usr/local/cuda-12
            echo "Usando CUDA 12"
        else
            export CUDA_HOME=/usr/local/cuda
            echo "Usando CUDA en $CUDA_HOME"
        fi
        export PATH=$CUDA_HOME/bin:$PATH
        export LD_LIBRARY_PATH=$CUDA_HOME/lib64:$LD_LIBRARY_PATH
        export CUDACXX=$CUDA_HOME/bin/nvcc
        export CMAKE_CUDA_COMPILER=$CUDA_HOME/bin/nvcc

        # Detectar arquitectura GPU
        if command -v nvidia-smi &> /dev/null; then
            GPU_ARCH=$(nvidia-smi --query-gpu=compute_cap --format=csv,noheader | head -1 | tr -d '.')

            # Permite compilar fatbin con varias arquitecturas.
            # Ejemplo: GPU_ARCH_LIST="75;80;86;89" (sin prefijo sm_)
            GPU_ARCHES="${GPU_ARCH_LIST:-$GPU_ARCH}"
            if [ -n "$GPU_ARCH_LIST" ]; then
                echo "Arquitecturas GPU configuradas (fatbin): $GPU_ARCHES"
            else
                echo "Arquitectura GPU detectada: sm_$GPU_ARCH"
            fi

            CMAKE_FLAGS="$CMAKE_FLAGS -DGGML_CUDA=ON -DCMAKE_CUDA_COMPILER=$CUDA_HOME/bin/nvcc -DCMAKE_CUDA_ARCHITECTURES=$GPU_ARCHES"
        else
            echo -e "${RED}Error: nvidia-smi no encontrado${NC}"
            exit 1
        fi
        ;;

    hipblas)
        echo "Configurando para AMD ROCm..."
        ROCM_HOME="${ROCM_HOME:-/opt/rocm}"
        if [ ! -d "$ROCM_HOME" ]; then
            echo -e "${RED}Error: ROCm no encontrado en $ROCM_HOME${NC}"
            exit 1
        fi
        export CXX="$ROCM_HOME/llvm/bin/clang++"
        export CC="$ROCM_HOME/llvm/bin/clang"
        GPU_TARGETS="${GPU_TARGETS:-gfx900,gfx90a,gfx1030,gfx1100}"
        CMAKE_FLAGS="$CMAKE_FLAGS -DGGML_HIPBLAS=ON -DAMDGPU_TARGETS=$GPU_TARGETS -DGPU_TARGETS=$GPU_TARGETS"
        ;;

    metal)
        echo "Configurando para Apple Metal..."
        if [ "$(uname -s)" != "Darwin" ]; then
            echo -e "${RED}Error: Metal solo está disponible en macOS${NC}"
            exit 1
        fi
        CMAKE_FLAGS="$CMAKE_FLAGS -DGGML_METAL=ON"
        ;;

    openblas)
        echo "Configurando para OpenBLAS..."
        if ! pkg-config --exists openblas 2>/dev/null && [ ! -f "/usr/include/openblas/cblas.h" ]; then
            echo -e "${RED}Error: OpenBLAS no encontrado${NC}"
            echo "Instalación: sudo apt-get install libopenblas-dev"
            exit 1
        fi
        CMAKE_FLAGS="$CMAKE_FLAGS -DGGML_BLAS=ON -DGGML_BLAS_VENDOR=OpenBLAS"
        ;;

    *)
        echo -e "${RED}Error: Variante desconocida: $VARIANT${NC}"
        echo "Variantes válidas: cpu, cuda, hipblas, metal, openblas"
        exit 1
        ;;
esac

echo -e "${GREEN}✓ Configuración lista${NC}"

# Compilar con CMake
echo ""
echo -e "${YELLOW}[3/4] Compilando llama.cpp...${NC}"
echo "Flags: $CMAKE_FLAGS"
echo ""

mkdir -p build
cd build

cmake ../llama.cpp $CMAKE_FLAGS
cmake --build . --config Release -j$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)

echo -e "${GREEN}✓ Compilación exitosa${NC}"

# Copiar librerías
echo ""
echo -e "${YELLOW}[4/4] Copiando librerías a $BUILD_DIR...${NC}"

# Copiar todas las .so/.dylib
COPIED=0
for dir in bin src common ggml ggml/src .; do
    if [ -d "$dir" ]; then
        for lib in $(find "$dir" -maxdepth 1 -type f \( -name "*.so*" -o -name "*.dylib" \) 2>/dev/null); do
            cp -f "$lib" "$BUILD_DIR/"
            echo "  Copiado: $(basename $lib)"
            COPIED=$((COPIED + 1))
        done
    fi
done

# Compilar y copiar la shim (libllama_shim.*) junto al resto de librerías.
echo ""
echo -e "${YELLOW}Compilando libllama_shim...${NC}"

SHIM_SRC="$PROJECT_ROOT/internal/llama_shim/llama_shim.c"
SHIM_INC="$PROJECT_ROOT/internal/llama_shim"

if [ -f "$SHIM_SRC" ]; then
    case "$(uname -s)" in
        Darwin)
            cc_bin="${CC:-clang}"
            "$cc_bin" -dynamiclib -O3 \
                -I"$SHIM_INC" \
                -L"$BUILD_DIR" -lllama \
                -Wl,-rpath,@loader_path \
                -Wl,-install_name,@rpath/libllama_shim.dylib \
                -o "$BUILD_DIR/libllama_shim.dylib" \
                "$SHIM_SRC" -lm
            echo "  Copiado: libllama_shim.dylib"
            ;;
        *)
            cc_bin="${CC:-gcc}"
            "$cc_bin" -shared -fPIC -O3 \
                -I"$SHIM_INC" \
                -L"$BUILD_DIR" -lllama \
                -Wl,-rpath,'$ORIGIN' \
                -o "$BUILD_DIR/libllama_shim.so" \
                "$SHIM_SRC" -lm
            echo "  Copiado: libllama_shim.so"
            ;;
    esac
else
    echo -e "${YELLOW}Advertencia: fuente de shim no encontrado en $SHIM_SRC (omitido)${NC}"
fi

# Crear archivo de información
cat > "$BUILD_DIR/BUILD_INFO.txt" << EOF
Variant: $VARIANT
Built: $(date)
Platform: $(uname -s)
Architecture: $(uname -m)
CMake flags: $CMAKE_FLAGS
EOF

if [ "$VARIANT" = "cuda" ]; then
    echo "CUDA version: $($CUDA_HOME/bin/nvcc --version | grep release)" >> "$BUILD_DIR/BUILD_INFO.txt"
    echo "GPU architectures: $GPU_ARCHES" >> "$BUILD_DIR/BUILD_INFO.txt"
fi

echo ""
echo -e "${GREEN}✓ $COPIED librerías copiadas${NC}"
ls -lh "$BUILD_DIR"/*.so* 2>/dev/null || ls -lh "$BUILD_DIR"/*.dylib 2>/dev/null || true

echo ""
echo -e "${BLUE}=== Compilación completada ===${NC}"
echo ""
echo "Librerías disponibles en: $BUILD_DIR"
echo "Información de compilación: $BUILD_DIR/BUILD_INFO.txt"
echo ""
echo "Para usar esta variante:"
echo "  cp $BUILD_DIR/*.so* build/  # Copiar librerías al directorio build principal"
echo "  make clean && make build    # Recompilar proyecto"
