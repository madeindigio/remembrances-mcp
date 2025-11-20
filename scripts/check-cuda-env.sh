#!/bin/bash
# Script para verificar el entorno CUDA y mostrar información de la GPU

# Colores
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Verificación del Entorno CUDA ===${NC}"
echo ""

# Verificar NVIDIA Driver
echo -e "${YELLOW}[1] Verificando NVIDIA Driver...${NC}"
if command -v nvidia-smi &> /dev/null; then
    echo -e "${GREEN}✓ NVIDIA Driver instalado${NC}"
    DRIVER_VERSION=$(nvidia-smi --query-gpu=driver_version --format=csv,noheader | head -1)
    echo "  Versión: $DRIVER_VERSION"

    # Info de la GPU
    GPU_NAME=$(nvidia-smi --query-gpu=name --format=csv,noheader | head -1)
    GPU_MEMORY=$(nvidia-smi --query-gpu=memory.total --format=csv,noheader | head -1)
    GPU_COMPUTE=$(nvidia-smi --query-gpu=compute_cap --format=csv,noheader | head -1)

    echo -e "${GREEN}✓ GPU detectada:${NC}"
    echo "  Nombre: $GPU_NAME"
    echo "  VRAM: $GPU_MEMORY"
    echo "  Compute Capability: $GPU_COMPUTE"

    # Calcular arquitectura CUDA
    GPU_ARCH=$(echo $GPU_COMPUTE | tr -d '.')
    echo "  Arquitectura CUDA: sm_${GPU_ARCH}"

    # Recomendar capas GPU según VRAM
    VRAM_GB=$(echo $GPU_MEMORY | grep -oP '^\d+')
    if [ "$VRAM_GB" -ge 20 ]; then
        RECOMMENDED_LAYERS="99 (todas las capas)"
    elif [ "$VRAM_GB" -ge 10 ]; then
        RECOMMENDED_LAYERS="48-64"
    elif [ "$VRAM_GB" -ge 8 ]; then
        RECOMMENDED_LAYERS="32-48"
    elif [ "$VRAM_GB" -ge 6 ]; then
        RECOMMENDED_LAYERS="24-32"
    else
        RECOMMENDED_LAYERS="16-24"
    fi
    echo "  Capas GPU recomendadas: $RECOMMENDED_LAYERS"
else
    echo -e "${RED}✗ nvidia-smi no encontrado${NC}"
    echo "  ¿Tienes los drivers NVIDIA instalados?"
    exit 1
fi

echo ""

# Verificar CUDA Toolkit
echo -e "${YELLOW}[2] Verificando CUDA Toolkit...${NC}"
if command -v nvcc &> /dev/null; then
    echo -e "${GREEN}✓ CUDA Toolkit instalado${NC}"
    CUDA_VERSION=$(nvcc --version | grep "release" | sed -n 's/.*release \([0-9.]*\).*/\1/p')
    echo "  Versión: $CUDA_VERSION"
    echo "  Ubicación nvcc: $(which nvcc)"
else
    echo -e "${RED}✗ nvcc no encontrado${NC}"
    echo "  CUDA Toolkit no está instalado o no está en PATH"
    echo "  Instalación: https://developer.nvidia.com/cuda-downloads"
fi

echo ""

# Verificar CMake
echo -e "${YELLOW}[3] Verificando CMake...${NC}"
if command -v cmake &> /dev/null; then
    CMAKE_VERSION=$(cmake --version | head -1 | grep -oP '\d+\.\d+\.\d+')
    echo -e "${GREEN}✓ CMake instalado${NC}"
    echo "  Versión: $CMAKE_VERSION"

    # CMake 3.18+ es mejor para CUDA
    CMAKE_MAJOR=$(echo $CMAKE_VERSION | cut -d. -f1)
    CMAKE_MINOR=$(echo $CMAKE_VERSION | cut -d. -f2)
    if [ "$CMAKE_MAJOR" -ge 3 ] && [ "$CMAKE_MINOR" -ge 18 ]; then
        echo -e "${GREEN}  ✓ Versión compatible con CUDA${NC}"
    else
        echo -e "${YELLOW}  ⚠ CMake 3.18+ recomendado para mejor soporte CUDA${NC}"
    fi
else
    echo -e "${RED}✗ CMake no encontrado${NC}"
    echo "  Instalación: sudo apt-get install cmake"
fi

echo ""

# Verificar Compiladores
echo -e "${YELLOW}[4] Verificando Compiladores C/C++...${NC}"
if command -v gcc &> /dev/null; then
    GCC_VERSION=$(gcc --version | head -1 | grep -oP '\d+\.\d+\.\d+')
    echo -e "${GREEN}✓ GCC instalado${NC}"
    echo "  Versión: $GCC_VERSION"
else
    echo -e "${RED}✗ GCC no encontrado${NC}"
fi

if command -v g++ &> /dev/null; then
    GXX_VERSION=$(g++ --version | head -1 | grep -oP '\d+\.\d+\.\d+')
    echo -e "${GREEN}✓ G++ instalado${NC}"
    echo "  Versión: $GXX_VERSION"
else
    echo -e "${RED}✗ G++ no encontrado${NC}"
fi

echo ""

# Verificar variables de entorno CUDA
echo -e "${YELLOW}[5] Verificando Variables de Entorno...${NC}"
if [ -n "$CUDA_HOME" ]; then
    echo -e "${GREEN}✓ CUDA_HOME definido${NC}"
    echo "  CUDA_HOME=$CUDA_HOME"
else
    echo -e "${YELLOW}⚠ CUDA_HOME no definido${NC}"
    echo "  Recomendado: export CUDA_HOME=/usr/local/cuda"
fi

if [[ ":$PATH:" == *":/usr/local/cuda/bin:"* ]]; then
    echo -e "${GREEN}✓ CUDA en PATH${NC}"
else
    echo -e "${YELLOW}⚠ /usr/local/cuda/bin no está en PATH${NC}"
    echo "  Recomendado: export PATH=/usr/local/cuda/bin:\$PATH"
fi

if [[ ":$LD_LIBRARY_PATH:" == *":/usr/local/cuda/lib64:"* ]]; then
    echo -e "${GREEN}✓ CUDA libs en LD_LIBRARY_PATH${NC}"
else
    echo -e "${YELLOW}⚠ /usr/local/cuda/lib64 no está en LD_LIBRARY_PATH${NC}"
    echo "  Recomendado: export LD_LIBRARY_PATH=/usr/local/cuda/lib64:\$LD_LIBRARY_PATH"
fi

echo ""

# Verificar librerías CUDA
echo -e "${YELLOW}[6] Verificando Librerías CUDA...${NC}"
CUDA_LIB_PATHS="/usr/local/cuda/lib64 /usr/lib/x86_64-linux-gnu"
FOUND_CUDA=false
FOUND_CUBLAS=false

for path in $CUDA_LIB_PATHS; do
    if [ -f "$path/libcuda.so" ] || [ -f "$path/libcuda.so.1" ]; then
        echo -e "${GREEN}✓ libcuda.so encontrada en $path${NC}"
        FOUND_CUDA=true
    fi
    if [ -f "$path/libcublas.so" ]; then
        echo -e "${GREEN}✓ libcublas.so encontrada en $path${NC}"
        FOUND_CUBLAS=true
    fi
done

if [ "$FOUND_CUDA" = false ]; then
    echo -e "${YELLOW}⚠ libcuda.so no encontrada en las ubicaciones estándar${NC}"
fi
if [ "$FOUND_CUBLAS" = false ]; then
    echo -e "${YELLOW}⚠ libcublas.so no encontrada en las ubicaciones estándar${NC}"
fi

echo ""

# Resumen
echo -e "${BLUE}=== Resumen ===${NC}"
echo ""

if command -v nvidia-smi &> /dev/null && command -v nvcc &> /dev/null && command -v cmake &> /dev/null; then
    echo -e "${GREEN}✓ Todo está listo para compilar con CUDA!${NC}"
    echo ""
    echo "Próximos pasos:"
    echo "  1. ./scripts/build-cuda-libs.sh"
    echo "  2. make clean && make BUILD_TYPE=cuda build"
    echo "  3. ./build/remembrances-mcp --gguf-model-path model.gguf --gguf-gpu-layers $RECOMMENDED_LAYERS"
else
    echo -e "${RED}✗ Faltan componentes para compilación CUDA${NC}"
    echo ""
    echo "Componentes faltantes:"
    if ! command -v nvidia-smi &> /dev/null; then
        echo "  - NVIDIA Drivers"
    fi
    if ! command -v nvcc &> /dev/null; then
        echo "  - CUDA Toolkit"
    fi
    if ! command -v cmake &> /dev/null; then
        echo "  - CMake"
    fi
    if ! command -v gcc &> /dev/null || ! command -v g++ &> /dev/null; then
        echo "  - GCC/G++"
    fi
fi

echo ""
