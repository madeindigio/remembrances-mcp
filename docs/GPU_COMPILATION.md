# Guía de Compilación con Soporte GPU

Esta guía te ayudará a compilar llama.cpp con soporte GPU para acelerar significativamente la inferencia de modelos GGUF.

## Índice

- [Requisitos Previos](#requisitos-previos)
- [Compilación para NVIDIA GPU (CUDA)](#compilación-para-nvidia-gpu-cuda)
- [Compilación Rápida con Script](#compilación-rápida-con-script)
- [Configuración y Uso](#configuración-y-uso)
- [Troubleshooting](#troubleshooting)
- [Rendimiento Esperado](#rendimiento-esperado)

## Requisitos Previos

### Para NVIDIA GPU (CUDA)

1. **GPU NVIDIA compatible** (Compute Capability 3.5+)
   - Verifica tu GPU: `nvidia-smi`
   - Para RTX 3060: 6GB VRAM, ~32 capas GPU recomendadas

2. **NVIDIA Drivers actualizados**
   ```bash
   nvidia-smi  # Debe mostrar tu GPU
   ```

3. **CUDA Toolkit** (11.0 o superior)
   ```bash
   nvcc --version  # Debe mostrar la versión de CUDA
   ```

   Si no tienes CUDA instalado:
   ```bash
   # Ubuntu/Debian
   wget https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64/cuda-keyring_1.0-1_all.deb
   sudo dpkg -i cuda-keyring_1.0-1_all.deb
   sudo apt-get update
   sudo apt-get install cuda
   
   # Añadir a ~/.bashrc
   export PATH=/usr/local/cuda/bin:$PATH
   export LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH
   ```

4. **CMake** (3.14 o superior)
   ```bash
   cmake --version
   # Si no está instalado:
   sudo apt-get install cmake  # Ubuntu/Debian
   ```

5. **Compilador C/C++**
   ```bash
   gcc --version
   g++ --version
   # Si no están instalados:
   sudo apt-get install build-essential
   ```

### Para AMD GPU (ROCm)

1. **GPU AMD compatible**
2. **ROCm 5.0+** instalado
3. Consulta: https://rocmdocs.amd.com/

### Para Apple Silicon (Metal)

El soporte Metal se compila automáticamente en macOS con Apple Silicon.

## Compilación para NVIDIA GPU (CUDA)

### Opción 1: Script Automático (Recomendado)

Hemos creado un script que automatiza todo el proceso:

```bash
# Ejecutar el script de compilación CUDA
./scripts/build-cuda-libs.sh
```

El script hará:
1. ✓ Verificar CUDA y GPU
2. ✓ Verificar go-llama.cpp
3. ✓ Limpiar compilaciones anteriores
4. ✓ Compilar llama.cpp con CUDA
5. ✓ Copiar librerías a `build/`

### Opción 2: Compilación Manual

Si prefieres compilar manualmente:

#### Paso 1: Navegar al directorio go-llama.cpp

```bash
cd ~/www/MCP/Remembrances/go-llama.cpp
```

#### Paso 2: Inicializar submodulos (si no lo has hecho)

```bash
git submodule update --init --recursive
```

#### Paso 3: Limpiar compilación anterior

```bash
rm -rf build
rm -f prepare *.o *.a
```

#### Paso 4: Compilar con CMake

```bash
mkdir -p build
cd build

# Detectar arquitectura CUDA (necesario para algunas versiones de CMake)
GPU_ARCH=$(nvidia-smi --query-gpu=compute_cap --format=csv,noheader | head -1 | tr -d '.')
echo "Usando arquitectura CUDA: $GPU_ARCH"

# Configurar con CUDA
cmake ../llama.cpp \
    -DGGML_CUDA=ON \
    -DLLAMA_STATIC=OFF \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_CUDA_ARCHITECTURES=${GPU_ARCH}

# Compilar (usa todos los cores disponibles)
cmake --build . --config Release -j$(nproc)
```

#### Paso 5: Copiar librerías al proyecto

```bash
# Volver al directorio del proyecto
cd ~/www/MCP/remembrances-mcp

# Copiar todas las librerías .so
find ~/www/MCP/Remembrances/go-llama.cpp/build -name "*.so*" -exec cp {} build/ \;
```

### Opción 3: Usar el Makefile del proyecto

```bash
cd ~/www/MCP/remembrances-mcp

# Limpiar compilación anterior
make clean-all

# Compilar con CUDA
make BUILD_TYPE=cuda build
```

## Verificación de la Compilación

Después de compilar, verifica que las librerías CUDA estén presentes:

```bash
ls -lh build/*.so*
```

Deberías ver algo como:
```
-rwxr-xr-x libggml.so
-rwxr-xr-x libggml-base.so
-rwxr-xr-x libggml-cpu.so
-rwxr-xr-x libggml-cuda.so      # ← Esta es importante para CUDA
-rwxr-xr-x libllama.so
-rwxr-xr-x libcommon.so
```

## Configuración y Uso

### Recompilar el Proyecto

Después de compilar las librerías CUDA, recompila el proyecto:

```bash
cd ~/www/MCP/remembrances-mcp
make clean
make BUILD_TYPE=cuda build
```

### Ejecutar con GPU

Usa el flag `--gguf-gpu-layers` para habilitar GPU:

```bash
# Ejemplo básico
./build/remembrances-mcp \
  --gguf-model-path ./models/nomic-embed-text-v1.5.Q4_K_M.gguf \
  --gguf-gpu-layers 32

# Configuración óptima para RTX 3060 (6GB VRAM)
./build/remembrances-mcp \
  --gguf-model-path ./models/nomic-embed-text-v1.5.Q4_K_M.gguf \
  --gguf-threads 8 \
  --gguf-gpu-layers 32 \
  --gguf-batch-size 512
```

### Configuración en YAML

Edita tu `config.yaml`:

```yaml
embedder:
  type: "gguf"
  gguf:
    model_path: "./models/nomic-embed-text-v1.5.Q4_K_M.gguf"
    threads: 8
    gpu_layers: 32          # Número de capas en GPU
    batch_size: 512
    use_mmap: true
    use_mlock: false
```

### Ajustar Capas GPU según VRAM

| GPU | VRAM | Capas Recomendadas | Modelo |
|-----|------|-------------------|--------|
| RTX 3060 | 6GB | 24-32 | Q4_K_M |
| RTX 3070 | 8GB | 32-48 | Q4_K_M/Q5_K_M |
| RTX 3080 | 10GB | 48-64 | Q5_K_M/Q8_0 |
| RTX 3090 | 24GB | 99 (todas) | Q8_0/F16 |
| RTX 4090 | 24GB | 99 (todas) | Q8_0/F16 |

**Regla general**: Comienza con 32 capas. Si tienes errores de memoria, reduce. Si todo va bien, incrementa gradualmente.

## Verificar que GPU está en Uso

### Durante la ejecución:

```bash
# En otra terminal, monitorea el uso de GPU
watch -n 1 nvidia-smi
```

Deberías ver:
- **GPU-Util**: >0% (indica uso de GPU)
- **Memory-Usage**: Incremento en memoria usada
- **Power**: Mayor consumo de energía

### Logs del servidor:

El servidor debería indicar:
```
Loading GGUF model: ./models/nomic-embed-text-v1.5.Q4_K_M.gguf
Using GPU layers: 32
CUDA device 0: NVIDIA GeForce RTX 3060
Model loaded successfully
```

## Troubleshooting

### Error: "libcuda.so.1: cannot open shared object file"

**Problema**: Drivers NVIDIA no encontrados.

**Solución**:
```bash
# Verifica drivers
nvidia-smi

# Si falla, reinstala drivers
sudo apt-get install nvidia-driver-535  # O versión más reciente
sudo reboot
```

### Error: "nvcc: command not found"

**Problema**: CUDA Toolkit no instalado o no en PATH.

**Solución**:
```bash
# Verifica instalación
which nvcc

# Si no existe, añade a ~/.bashrc
export PATH=/usr/local/cuda/bin:$PATH
export LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH
source ~/.bashrc
```

### Error: "CUDA error: out of memory"

**Problema**: Modelo muy grande para tu VRAM.

**Soluciones**:
1. Reduce `--gguf-gpu-layers`:
   ```bash
   ./build/remembrances-mcp --gguf-model-path ./models/model.gguf --gguf-gpu-layers 16
   ```

2. Usa un modelo más cuantizado (Q4_K_M en lugar de Q8_0)

3. Cierra otros programas que usen GPU

4. Reduce `--gguf-batch-size`:
   ```bash
   --gguf-batch-size 256
   ```

### Error: "CUDA not detected" pero tengo GPU NVIDIA

**Problema**: Librerías compiladas sin CUDA o no encontradas.

**Solución**:
```bash
# Verifica que libggml-cuda.so existe
ls -la build/libggml-cuda.so

# Si no existe, recompila con CUDA
./scripts/build-cuda-libs.sh

# Recompila el proyecto
make clean
make BUILD_TYPE=cublas build
```

### Rendimiento no mejora con GPU

**Verificaciones**:

1. **Confirma que GPU está en uso**:
   ```bash
   watch nvidia-smi  # Debe mostrar uso >0%
   ```

2. **Verifica capas GPU en logs**:
   ```bash
   ./build/remembrances-mcp --gguf-model-path model.gguf --gguf-gpu-layers 32 2>&1 | grep -i gpu
   ```

3. **Incrementa capas GPU**:
   ```bash
   --gguf-gpu-layers 48  # O más, según tu VRAM
   ```

4. **Verifica que compilación tiene CUDA**:
   ```bash
   ldd build/libllama.so | grep cuda
   # Debe mostrar: libcuda.so.1 => /path/to/libcuda.so.1
   ```

### Error: "cmake: command not found"

**Solución**:
```bash
sudo apt-get update
sudo apt-get install cmake
```

### Error: "Unsupported gpu architecture 'compute_'"

**Problema**: CMake no puede detectar la arquitectura de la GPU con `native`.

**Solución**: El script automático ya detecta la arquitectura. Si compilas manualmente, usa:
```bash
# Detectar arquitectura de tu GPU
GPU_ARCH=$(nvidia-smi --query-gpu=compute_cap --format=csv,noheader | head -1 | tr -d '.')
echo "Arquitectura: $GPU_ARCH"

# Usar en cmake
cmake ../llama.cpp -DGGML_CUDA=ON -DCMAKE_CUDA_ARCHITECTURES=${GPU_ARCH}
```

Arquitecturas comunes:
- RTX 30xx: 86
- RTX 40xx: 89
- RTX 20xx: 75
- GTX 16xx: 75

### Error: "Feature 'movmatrix' requires PTX ISA .version 7.8 or later"

**Problema**: CUDA 11.x no soporta algunas características modernas (Flash Attention) que requieren CUDA 12+.

**Solución 1 (Recomendada)**: Actualizar a CUDA 12.x
```bash
# Ubuntu/Debian
wget https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64/cuda-keyring_1.1-1_all.deb
sudo dpkg -i cuda-keyring_1.1-1_all.deb
sudo apt-get update
sudo apt-get install cuda-toolkit-12-6
```

**Solución 2**: El script automático deshabilita Flash Attention en CUDA 11.x automáticamente. Si compilas manualmente:
```bash
cmake ../llama.cpp \
    -DGGML_CUDA=ON \
    -DGGML_CUDA_FLASH_ATTN=OFF \
    -DGGML_CUDA_FA_ALL_QUANTS=OFF \
    -DCMAKE_CUDA_ARCHITECTURES=${GPU_ARCH}
```

**Nota**: Flash Attention mejora el rendimiento en modelos de lenguaje grandes, pero no es crítico para modelos de embeddings pequeños.

## Rendimiento Esperado

### CPU vs GPU (RTX 3060, modelo Q4_K_M)

| Operación | CPU (8 cores) | GPU (32 layers) | Mejora |
|-----------|---------------|-----------------|---------|
| Cargar modelo | ~5s | ~3s | 1.7x |
| Embedding (1 texto) | ~200ms | ~20ms | **10x** |
| Embedding (batch 8) | ~1500ms | ~80ms | **19x** |
| Embedding (batch 32) | ~5000ms | ~250ms | **20x** |

### GPUs más potentes

Con RTX 4090 o superior, puedes ver mejoras de **30-50x** sobre CPU.

## Modelos Recomendados por GPU

### RTX 3060 (6GB)
- ✓ nomic-embed-text-v1.5.Q4_K_M
- ✓ mxbai-embed-large-v1.Q4_K_M
- ⚠️ nomic-embed-text-v1.5.Q8_0 (límite de capas)

### RTX 3080+ (10GB+)
- ✓ Todos los modelos Q4_K_M, Q5_K_M
- ✓ Modelos Q8_0
- ✓ Algunos modelos F16 pequeños

### RTX 4090 (24GB)
- ✓ Cualquier modelo de embedding
- ✓ Modelos grandes en F16
- ✓ Múltiples modelos simultáneos

## Siguiente Paso

Después de compilar con éxito:

1. **Descarga un modelo GGUF optimizado**:
   ```bash
   mkdir -p models
   wget https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf -P models/
   ```

2. **Prueba el rendimiento**:
   ```bash
   ./scripts/test-gguf.sh models/nomic-embed-text-v1.5.Q4_K_M.gguf 8 32
   ```

3. **Configura tu servidor** con GPU habilitada en `config.yaml`

4. **Disfruta de embeddings 10-20x más rápidos!** 🚀

## Referencias

- [llama.cpp CUDA Docs](https://github.com/ggerganov/llama.cpp#cuda)
- [NVIDIA CUDA Installation](https://developer.nvidia.com/cuda-downloads)
- [Compute Capability](https://developer.nvidia.com/cuda-gpus)

## Soporte

Si encuentras problemas:

1. Revisa la sección [Troubleshooting](#troubleshooting)
2. Verifica logs: `./build/remembrances-mcp 2>&1 | tee debug.log`
3. Abre un issue en GitHub con:
   - Output de `nvidia-smi`
   - Output de `nvcc --version`
   - Output de `ls -la build/`
   - Logs completos de error
