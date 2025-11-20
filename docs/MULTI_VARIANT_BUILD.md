# Compilación Multi-Variante de Librerías llama.cpp

Esta guía explica cómo compilar múltiples variantes de las librerías llama.cpp para diferentes GPUs y distribuirlas en formato binario.

## Índice

- [Visión General](#visión-general)
- [Estructura de Directorios](#estructura-de-directorios)
- [Compilar Variantes Individuales](#compilar-variantes-individuales)
- [Compilar Todas las Variantes](#compilar-todas-las-variantes)
- [Empaquetar para Distribución](#empaquetar-para-distribución)
- [Usar una Variante Específica](#usar-una-variante-específica)

## Visión General

El sistema de compilación multi-variante te permite:

1. **Compilar librerías para diferentes GPUs** sin interferir entre sí
2. **Organizar librerías en subdirectorios** separados por tipo
3. **Distribuir binarios** para diferentes plataformas
4. **Cambiar fácilmente** entre variantes sin recompilar

### Variantes Disponibles

| Variante | Plataforma | Descripción | Directorio |
|----------|-----------|-------------|-----------|
| `cpu` | Todas | CPU solamente (sin GPU) | `build/libs/cpu/` |
| `cuda` | Linux | NVIDIA CUDA (RTX, GTX, etc.) | `build/libs/cuda/` |
| `hipblas` | Linux | AMD ROCm (Radeon RX, MI) | `build/libs/hipblas/` |
| `metal` | macOS | Apple Metal (M1, M2, M3) | `build/libs/metal/` |
| `openblas` | Todas | OpenBLAS (CPU optimizado) | `build/libs/openblas/` |

## Estructura de Directorios

Después de compilar variantes, tendrás:

```
build/
├── libs/
│   ├── cpu/
│   │   ├── libllama.so
│   │   ├── libggml.so
│   │   ├── libggml-base.so
│   │   ├── libggml-cpu.so
│   │   └── BUILD_INFO.txt
│   ├── cuda/
│   │   ├── libllama.so
│   │   ├── libggml.so
│   │   ├── libggml-base.so
│   │   ├── libggml-cpu.so
│   │   ├── libggml-cuda.so          ← Específico CUDA
│   │   └── BUILD_INFO.txt
│   ├── hipblas/
│   │   ├── libllama.so
│   │   ├── libggml.so
│   │   ├── libggml-rocm.so          ← Específico ROCm
│   │   └── BUILD_INFO.txt
│   ├── metal/
│   │   ├── libllama.dylib
│   │   ├── libggml.dylib
│   │   ├── libggml-metal.dylib      ← Específico Metal
│   │   └── BUILD_INFO.txt
│   └── openblas/
│       ├── libllama.so
│       ├── libggml.so
│       └── BUILD_INFO.txt
└── remembrances-mcp                  ← Binario principal
```

## Compilar Variantes Individuales

### Método 1: Usando el Makefile

```bash
# Compilar variante CUDA
make build-libs-cuda

# Compilar variante CPU
make build-libs-cpu

# Compilar variante HIPBlas (ROCm)
make build-libs-hipblas

# Compilar variante Metal (macOS)
make build-libs-metal

# Compilar variante OpenBLAS
make build-libs-openblas
```

### Método 2: Usando el Script

```bash
# Compilar CUDA
./scripts/build-variant-libs.sh cuda

# Compilar CPU
./scripts/build-variant-libs.sh cpu

# Compilar HIPBlas
./scripts/build-variant-libs.sh hipblas

# Compilar Metal
./scripts/build-variant-libs.sh metal

# Compilar OpenBLAS
./scripts/build-variant-libs.sh openblas
```

### Ejemplo: Compilar CUDA

```bash
$ make build-libs-cuda

Building CUDA variant...
Building llama.cpp with cuda support...

[1/4] Limpiando compilación anterior...
✓ Limpieza completada

[2/4] Configurando variante cuda...
Usando CUDA 12.6
Arquitectura GPU detectada: sm_86
✓ Configuración lista

[3/4] Compilando llama.cpp...
Flags: -DLLAMA_STATIC=OFF -DCMAKE_BUILD_TYPE=Release -DGGML_CUDA=ON ...
[Compilación en progreso...]
✓ Compilación exitosa

[4/4] Copiando librerías a build/libs/cuda...
  Copiado: libllama.so
  Copiado: libggml.so
  Copiado: libggml-base.so
  Copiado: libggml-cpu.so
  Copiado: libggml-cuda.so
✓ 5 librerías copiadas

✓ cuda libraries built successfully
```

## Compilar Todas las Variantes

Compila todas las variantes disponibles para tu plataforma:

```bash
make build-libs-all-variants
```

### En Linux

Compilará (si están disponibles):
- ✓ CPU (siempre)
- ✓ CUDA (si tienes nvcc)
- ✓ HIPBlas (si tienes ROCm)
- ✓ OpenBLAS (si está instalado)

### En macOS

Compilará:
- ✓ CPU (siempre)
- ✓ Metal (siempre en macOS)

### Ejemplo de Salida

```bash
$ make build-libs-all-variants

Building all library variants for linux...

=== Building CPU variant ===
[Compilación CPU...]
✓ cpu libraries built successfully

=== Building CUDA variant ===
[Compilación CUDA...]
✓ cuda libraries built successfully

⚠ Skipping HIPBlas (ROCm not found)
⚠ Skipping OpenBLAS (not found)

✓ All Linux variants built successfully!

Libraries available in:
  - build/libs/cpu/BUILD_INFO.txt
  - build/libs/cuda/BUILD_INFO.txt

To use a specific variant, copy libraries from build/libs/{variant}/ to build/
```

## Empaquetar para Distribución

Empaqueta todas las variantes compiladas en archivos `.tar.gz`:

```bash
make package-libs-all
```

Esto creará:

```
dist/libs/
├── llama-cpp-cpu-linux-x86_64.tar.gz
├── llama-cpp-cuda-linux-x86_64.tar.gz
├── llama-cpp-hipblas-linux-x86_64.tar.gz
└── llama-cpp-openblas-linux-x86_64.tar.gz
```

### Desempaquetar en Otro Sistema

```bash
# En la máquina de destino
mkdir -p build
tar -xzf llama-cpp-cuda-linux-x86_64.tar.gz -C build/
mv build/cuda/*.so build/
```

## Usar una Variante Específica

### Opción 1: Copiar Manualmente

```bash
# Copiar librerías CUDA al directorio principal
cp build/libs/cuda/*.so build/

# Recompilar el proyecto
make clean
make build
```

### Opción 2: Crear Script de Cambio

Crea un script para cambiar entre variantes:

```bash
#!/bin/bash
# scripts/switch-variant.sh
VARIANT=$1

if [ -z "$VARIANT" ]; then
    echo "Uso: $0 <variant>"
    echo "Variantes: cpu, cuda, hipblas, metal, openblas"
    exit 1
fi

if [ ! -d "build/libs/$VARIANT" ]; then
    echo "Error: Variante $VARIANT no compilada"
    echo "Ejecuta: make build-libs-$VARIANT"
    exit 1
fi

echo "Cambiando a variante: $VARIANT"
cp build/libs/$VARIANT/*.{so,dylib} build/ 2>/dev/null || true
echo "✓ Librerías $VARIANT copiadas a build/"
echo "Reinicia el servidor para usar las nuevas librerías"
```

Uso:

```bash
chmod +x scripts/switch-variant.sh

# Cambiar a CUDA
./scripts/switch-variant.sh cuda

# Cambiar a CPU
./scripts/switch-variant.sh cpu
```

## Verificar Variante Activa

Verifica qué librerías estás usando actualmente:

```bash
# Ver librerías en build/
ls -lh build/*.so*

# Ver información de compilación
cat build/libs/cuda/BUILD_INFO.txt
```

Ejemplo de `BUILD_INFO.txt`:

```
Variant: cuda
Built: Thu Nov 21 01:23:45 UTC 2025
Platform: Linux
Architecture: x86_64
CMake flags: -DLLAMA_STATIC=OFF -DCMAKE_BUILD_TYPE=Release -DGGML_CUDA=ON -DCMAKE_CUDA_ARCHITECTURES=86
CUDA version: release 12.6, V12.6.85
GPU architecture: sm_86
```

## Casos de Uso

### Caso 1: Desarrollo con Múltiples GPUs

Tienes una máquina con NVIDIA y quieres probar CPU vs CUDA:

```bash
# Compilar ambas variantes
make build-libs-cpu
make build-libs-cuda

# Probar con CPU
cp build/libs/cpu/*.so build/
./build/remembrances-mcp --gguf-model-path model.gguf

# Probar con CUDA
cp build/libs/cuda/*.so build/
./build/remembrances-mcp --gguf-model-path model.gguf --gguf-gpu-layers 32
```

### Caso 2: Distribución Multi-Plataforma

Compilar para diferentes usuarios:

```bash
# En máquina con NVIDIA
make build-libs-cuda
make package-libs-all
# Genera: llama-cpp-cuda-linux-x86_64.tar.gz

# En máquina con AMD
make build-libs-hipblas
make package-libs-all
# Genera: llama-cpp-hipblas-linux-x86_64.tar.gz

# En Mac M1
make build-libs-metal
make package-libs-all
# Genera: llama-cpp-metal-darwin-arm64.tar.gz
```

### Caso 3: CI/CD Pipeline

```bash
# En tu workflow de CI/CD
make build-libs-all-variants
make package-libs-all

# Subir artefactos
# Los usuarios descargan el .tar.gz apropiado
```

## Limpieza

```bash
# Limpiar solo variantes
make clean-libs-variants

# Limpiar todo (incluyendo build principal)
make clean

# Limpiar todo incluyendo llama.cpp source
make clean-all
```

## Troubleshooting

### Error: "Variante no compilada"

```bash
# Verifica qué variantes tienes
ls -la build/libs/

# Compila la variante faltante
make build-libs-cuda
```

### Error: "nvcc not found" al compilar CUDA

```bash
# Verifica CUDA
which nvcc
# Si no existe, instala CUDA Toolkit

# O usa el script que detecta automáticamente
./scripts/build-variant-libs.sh cuda
```

### Error: Librerías mezcladas

```bash
# Limpia y reconstruye
make clean-libs-variants
make build-libs-cuda
cp build/libs/cuda/*.so build/
make clean && make build
```

## Mejores Prácticas

1. **Compila todas las variantes antes de distribuir**:
   ```bash
   make build-libs-all-variants
   make package-libs-all
   ```

2. **Incluye BUILD_INFO.txt** en tus paquetes para que usuarios sepan qué tienen

3. **Documenta requisitos** de cada variante (CUDA version, ROCm version, etc.)

4. **Usa scripts** para cambiar variantes en desarrollo en lugar de copiar manualmente

5. **Versiona tus paquetes** con la versión de llama.cpp:
   ```bash
   # Ejemplo
   llama-cpp-cuda-linux-x86_64-v0.9.4.tar.gz
   ```

## Siguientes Pasos

- Ver [GPU_COMPILATION.md](GPU_COMPILATION.md) para detalles de compilación GPU
- Ver [BUILD_INSTRUCTIONS.md](../BUILD_INSTRUCTIONS.md) para instrucciones generales
- Ver [QUICK_START_GPU.md](../QUICK_START_GPU.md) para inicio rápido
