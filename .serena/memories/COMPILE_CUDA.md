# Guía Rápida: Compilar con GPU NVIDIA (Actualizado)

## Tu Sistema

Según la verificación:
- **GPU**: NVIDIA GeForce RTX 3060 Laptop (6GB VRAM)
- **Arquitectura**: sm_86 (Compute Capability 8.6)
- **CUDA instalado**: 11.5 y 12.6
- **Recomendación**: Usar CUDA 12.6 para mejor rendimiento

## Pasos Rápidos

### 1. Limpiar compilación anterior (si existe)

```bash
cd ~/www/MCP/Remembrances/go-llama.cpp
rm -rf build
rm -f prepare *.o *.a
```

### 2. Compilar con CUDA 12.6

El script automáticamente detectará y usará CUDA 12.6:

```bash
cd ~/www/MCP/remembrances-mcp
./scripts/build-cuda-libs.sh
```

El script hará:
- ✓ Detectar y usar CUDA 12.6 (en lugar de 11.5)
- ✓ Detectar arquitectura sm_86 de tu GPU
- ✓ Compilar llama.cpp con soporte CUDA completo
- ✓ Copiar librerías a `build/`

**Tiempo estimado**: 5-10 minutos

### 3. Recompilar el proyecto

```bash
make clean
make BUILD_TYPE=cuda build
```

### 4. Verificar librerías

```bash
ls -lh build/*.so*
```

Busca especialmente: `libggml-cuda.so`

### 5. Ejecutar con GPU

```bash
./build/remembrances-mcp \
  --gguf-model-path ./models/nomic-embed-text-v1.5.Q4_K_M.gguf \
  --gguf-gpu-layers 32 \
  --gguf-threads 8
```

## Configuración Óptima para tu RTX 3060

### Para modelos de embeddings (como nomic-embed)

```bash
./build/remembrances-mcp \
  --gguf-model-path ./models/nomic-embed-text-v1.5.Q4_K_M.gguf \
  --gguf-gpu-layers 32 \
  --gguf-threads 8 \
  --gguf-batch-size 512
```

### En config.yaml

```yaml
embedder:
  type: "gguf"
  gguf:
    model_path: "./models/nomic-embed-text-v1.5.Q4_K_M.gguf"
    threads: 8
    gpu_layers: 32
    batch_size: 512
    use_mmap: true
```

## Verificar que GPU funciona

Mientras el servidor ejecuta, en otra terminal:

```bash
watch -n 1 nvidia-smi
```

Deberías ver:
- GPU-Util > 0%
- Memory-Usage aumenta
- Power aumenta

## Si hay errores

### "Feature 'movmatrix' requires PTX ISA .version 7.8"

El script ya maneja esto automáticamente usando CUDA 12.6. Si persiste:

```bash
# Verificar que se está usando CUDA 12.6
which nvcc
# Debe mostrar: /usr/local/cuda-12.6/bin/nvcc

# Si muestra /usr/bin/nvcc (11.5), el script lo corregirá automáticamente
```

### "Out of memory"

```bash
# Reduce capas GPU
--gguf-gpu-layers 16

# O batch size
--gguf-batch-size 256
```

## Rendimiento Esperado

Con CUDA 12.6 y 32 capas GPU:

| Operación | CPU | GPU | Mejora |
|-----------|-----|-----|--------|
| Embedding (1 texto) | ~200ms | ~15-20ms | **10x** |
| Embedding (batch 8) | ~1500ms | ~60-80ms | **20x** |
| Embedding (batch 32) | ~5000ms | ~200-250ms | **20-25x** |

## Scripts Útiles

```bash
# Verificar entorno CUDA
./scripts/check-cuda-env.sh

# Compilar librerías CUDA
./scripts/build-cuda-libs.sh

# Compilar proyecto completo
make BUILD_TYPE=cuda build

# Ver ayuda del Makefile
make help
```

## Próximo Paso

¡Ejecuta la compilación!

```bash
./scripts/build-cuda-libs.sh
```
