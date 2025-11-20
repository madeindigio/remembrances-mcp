# Guía Rápida: Compilar con GPU NVIDIA

Esta es una guía rápida para compilar llama.cpp con soporte CUDA y acelerar tus embeddings.

## TL;DR - Pasos Rápidos

```bash
# 1. Ejecutar el script de compilación CUDA
./scripts/build-cuda-libs.sh

# 2. Recompilar el proyecto
make clean
make BUILD_TYPE=cuda build

# 3. Ejecutar con GPU (32 capas en GPU)
./build/remembrances-mcp \
  --gguf-model-path ./models/tu-modelo.gguf \
  --gguf-gpu-layers 32
```

## Requisitos Previos

Verifica que tienes todo instalado con nuestro script de verificación:

```bash
./scripts/check-cuda-env.sh
```

Este script verificará:
- ✓ NVIDIA Drivers y GPU
- ✓ CUDA Toolkit
- ✓ CMake y compiladores
- ✓ Variables de entorno
- ✓ Librerías CUDA

Si falta algo, consulta [docs/GPU_COMPILATION.md](docs/GPU_COMPILATION.md) para instalación completa.

## Proceso de Compilación

### Paso 1: Ejecutar Script de Compilación

El script automático hace todo por ti:

```bash
./scripts/build-cuda-libs.sh
```

Esto:
- ✓ Verifica CUDA y GPU
- ✓ Limpia compilaciones anteriores
- ✓ Compila llama.cpp con CUDA
- ✓ Copia librerías a `build/`

**Tiempo estimado**: 5-10 minutos en la primera compilación.

### Paso 2: Recompilar Proyecto

Después de tener las librerías CUDA:

```bash
cd /www/MCP/remembrances-mcp
make clean
make BUILD_TYPE=cuda build
```

### Paso 3: Verificar Librerías

Confirma que las librerías CUDA están presentes:

```bash
ls -lh build/*.so*
```

Busca: `libggml-cuda.so` (esta es la clave)

## Uso con GPU

### Comando Básico

```bash
./build/remembrances-mcp \
  --gguf-model-path ./models/nomic-embed-text-v1.5.Q4_K_M.gguf \
  --gguf-gpu-layers 32
```

### Configuración Óptima para RTX 3060

```bash
./build/remembrances-mcp \
  --gguf-model-path ./models/nomic-embed-text-v1.5.Q4_K_M.gguf \
  --gguf-threads 8 \
  --gguf-gpu-layers 32 \
  --gguf-batch-size 512
```

### En config.yaml

```yaml
embedder:
  type: "gguf"
  gguf:
    model_path: "./models/nomic-embed-text-v1.5.Q4_K_M.gguf"
    threads: 8
    gpu_layers: 32          # ← Aquí activas GPU
    batch_size: 512
    use_mmap: true
```

## Verificar que GPU Funciona

### Monitorear GPU

```bash
# En otra terminal
watch -n 1 nvidia-smi
```

Deberías ver:
- **GPU-Util** > 0%
- **Memory-Usage** incrementa
- **Power** mayor consumo

### Logs del Servidor

Busca en los logs:

```
Using GPU layers: 32
CUDA device 0: NVIDIA GeForce RTX 3060
```

## Ajustar Capas GPU

Según tu VRAM disponible:

| VRAM | Modelo Q4_K_M | Modelo Q8_0 |
|------|---------------|-------------|
| 6GB (RTX 3060) | 24-32 capas | 16-24 capas |
| 8GB (RTX 3070) | 32-48 capas | 24-32 capas |
| 10GB+ (RTX 3080) | 48-99 capas | 32-48 capas |

**Tip**: Empieza con 32, si funciona bien, incrementa gradualmente.

## Problemas Comunes

### "Out of memory"

```bash
# Reduce capas GPU
--gguf-gpu-layers 16

# O usa modelo más pequeño
# Q4_K_M en lugar de Q8_0
```

### GPU no se usa (0% utilization)

```bash
# Verifica que libggml-cuda.so existe
ls build/libggml-cuda.so

# Si no existe, recompila
./scripts/build-cuda-libs.sh
make clean
make BUILD_TYPE=cuda build
```

### "libcuda.so.1: not found"

```bash
# Reinstala drivers NVIDIA
sudo apt-get install nvidia-driver-535
sudo reboot
```

## Rendimiento Esperado

Con RTX 3060 y modelo Q4_K_M:

- **CPU (8 cores)**: ~200ms por embedding
- **GPU (32 layers)**: ~20ms por embedding
- **Mejora**: **10x más rápido** ⚡

## Siguiente Paso

1. **Descarga un modelo**:
   ```bash
   mkdir -p models
   wget https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf -P models/
   ```

2. **Prueba el rendimiento**:
   ```bash
   ./scripts/test-gguf.sh models/nomic-embed-text-v1.5.Q4_K_M.gguf 8 32
   ```

3. **¡Disfruta embeddings rápidos!** 🚀

## Más Información

Para documentación completa: [docs/GPU_COMPILATION.md](docs/GPU_COMPILATION.md)

## Tu Configuración

Según `nvidia-smi`, tienes:
- **GPU**: NVIDIA GeForce RTX 3060
- **VRAM**: 6GB
- **CUDA**: 13.0 (driver) / 11.5 y 12.6 (toolkit)
- **Capas recomendadas**: 32

Configuración sugerida:
```bash
./build/remembrances-mcp \
  --gguf-model-path ./models/nomic-embed-text-v1.5.Q4_K_M.gguf \
  --gguf-threads 8 \
  --gguf-gpu-layers 32 \
  --gguf-batch-size 512
```
