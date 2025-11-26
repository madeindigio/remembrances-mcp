# Quick Start: GGUF Embeddings

Esta guía te llevará desde cero hasta tener Remembrances-MCP corriendo con embeddings GGUF locales en menos de 5 minutos.

## Resumen de la Implementación ✅

Se ha implementado completamente el soporte para modelos GGUF de embeddings usando go-llama.cpp. La implementación incluye:

- ✅ Embedder GGUF (`pkg/embedder/gguf.go`)
- ✅ Integración con factory y configuración
- ✅ CLI flags, variables de entorno y YAML config
- ✅ Makefile con compilación automática
- ✅ Script wrapper para runtime (`run-remembrances.sh`)
- ✅ Tests completos y ejemplos
- ✅ Documentación exhaustiva
- ✅ Soporte GPU (Metal/CUDA/ROCm)

## Paso 1: Compilar el Proyecto


```bash
cd ~/www/MCP/remembrances-mcp
make build
```

Esto compilará:
1. llama.cpp con todas sus dependencias
2. go-llama.cpp bindings
3. remembrances-mcp con soporte GGUF

**Salida esperada:**
```
Checking llama.cpp library...
llama.cpp library already built at ~/www/MCP/Remembrances/go-llama.cpp/build/bin/
llama.cpp library ready
Building remembrances-mcp with GGUF support...
Build complete: build/remembrances-mcp
```

## Paso 2: Descargar un Modelo GGUF

### Opción A: Descargar con wget

```bash
# Modelo recomendado: nomic-embed-text-v1.5 Q4_K_M (768 dimensiones, ~200MB)
wget https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf
```

### Opción B: Descargar con huggingface-cli

```bash
pip install huggingface-hub
huggingface-cli download nomic-ai/nomic-embed-text-v1.5-GGUF nomic-embed-text-v1.5.Q4_K_M.gguf --local-dir ./models
```

### Modelos Recomendados

| Modelo | Tamaño | Calidad | Uso |
|--------|--------|---------|-----|
| Q4_K_M | ~200MB | Buena | **Recomendado** para uso general |
| Q8_0 | ~350MB | Excelente | Mejor calidad, más lento |
| Q2_K | ~100MB | Básica | Recursos limitados |

## Paso 3: Ejecutar con GGUF

### Usando el Script Wrapper (Recomendado)

```bash
./run-remembrances.sh \
  --gguf-model-path ./nomic-embed-text-v1.5.Q4_K_M.gguf \
  --gguf-threads 8
```

### Configuración Manual de LD_LIBRARY_PATH

```bash
export LD_LIBRARY_PATH=~/www/MCP/Remembrances/go-llama.cpp/build/bin:$LD_LIBRARY_PATH
./build/remembrances-mcp \
  --gguf-model-path ./nomic-embed-text-v1.5.Q4_K_M.gguf \
  --gguf-threads 8
```

### Con Aceleración GPU

**Linux con NVIDIA (CUDA):**
```bash
# Primero recompilar con CUDA
make clean-all
make BUILD_TYPE=cublas build

# Ejecutar con GPU
./run-remembrances.sh \
  --gguf-model-path ./nomic-embed-text-v1.5.Q4_K_M.gguf \
  --gguf-threads 8 \
  --gguf-gpu-layers 32
```

**macOS con Metal:**
```bash
# Metal viene activado por defecto
./run-remembrances.sh \
  --gguf-model-path ./nomic-embed-text-v1.5.Q4_K_M.gguf \
  --gguf-threads 4 \
  --gguf-gpu-layers 99  # 99 = todas las capas a GPU
```

## Verificación

Una vez iniciado, deberías ver:

```
Loading GGUF model: ./nomic-embed-text-v1.5.Q4_K_M.gguf
Model loaded successfully
Embedding dimension: 768
Server started...
```

## Configuración via YAML (Opcional)

Crear `config.yaml`:

```yaml
# GGUF embeddings (prioridad más alta)
gguf-model-path: "./nomic-embed-text-v1.5.Q4_K_M.gguf"
gguf-threads: 8
gguf-gpu-layers: 32

# Base de datos
surrealdb-url: "ws://localhost:8000"
surrealdb-user: "root"
surrealdb-pass: "root"

# Transporte
sse: true
sse-addr: ":3000"
```

Ejecutar:
```bash
./run-remembrances.sh --config config.yaml
```

## Pruebas

### Test Rápido con Script Automatizado

```bash
./scripts/test-gguf.sh ./nomic-embed-text-v1.5.Q4_K_M.gguf 8 0
```

### Ejecutar Tests Go

```bash
GGUF_TEST_MODEL_PATH=./nomic-embed-text-v1.5.Q4_K_M.gguf \
  go test -v ./pkg/embedder -run TestGGUF
```

### Ejecutar Ejemplo Standalone

```bash
go run examples/gguf_embeddings.go \
  --model ./nomic-embed-text-v1.5.Q4_K_M.gguf \
  --text "Hello, world!"
```

### Benchmark de Rendimiento

```bash
go run examples/gguf_embeddings.go \
  --model ./nomic-embed-text-v1.5.Q4_K_M.gguf \
  --benchmark
```

## Troubleshooting

### Error: "libllama.so: cannot open shared object file"

**Solución**: Usa el wrapper script `run-remembrances.sh` en lugar de ejecutar el binario directamente.

### Error: "failed to load GGUF model"

**Posibles causas:**
1. Modelo no existe o ruta incorrecta
2. Archivo corrupto (re-descargar)
3. Memoria RAM insuficiente (usar modelo más cuantizado como Q4_K_M o Q2_K)

### Rendimiento Lento

**Optimizaciones:**
1. Aumentar threads: `--gguf-threads $(nproc)`
2. Activar GPU: `--gguf-gpu-layers 32`
3. Usar modelo más cuantizado (Q4_K_M en lugar de Q8_0)

### GPU no se está usando

**Verificar:**
```bash
# NVIDIA
nvidia-smi  # Ver uso de GPU durante inferencia

# Recompilar con CUDA
make clean-all
make BUILD_TYPE=cublas build
```

## Próximos Pasos

1. **Leer documentación completa**: `docs/GGUF_EMBEDDINGS.md`
2. **Ver implementación técnica**: `GGUF_IMPLEMENTATION_SUMMARY.md`
3. **Explorar ejemplos**: `examples/gguf_embeddings.go`
4. **Configurar para producción**: Ver `config.sample.yaml`

## Comandos Útiles

```bash
# Ver todas las opciones de compilación
make help

# Verificar entorno de compilación
make check-env

# Limpiar y recompilar
make clean-all && make build

# Ejecutar con configuración completa
./run-remembrances.sh \
  --config config.yaml \
  --sse \
  --sse-addr ":3000"

# Ver versión
./run-remembrances.sh --version
```

## Soporte

Si encuentras problemas:

1. Revisa `BUILD_INSTRUCTIONS.md` para problemas de compilación
2. Consulta `docs/GGUF_EMBEDDINGS.md` para troubleshooting detallado
3. Abre un issue en GitHub con:
   - Salida de `make check-env`
   - Mensaje de error completo
   - Sistema operativo y arquitectura

## ¡Éxito! 🎉

Si todo funciona, ahora tienes:

- ✅ Embeddings locales y privados
- ✅ Sin costos de API
- ✅ Aceleración GPU (si está disponible)
- ✅ Control total sobre tus datos
- ✅ Rendimiento optimizado

**¡Disfruta usando GGUF embeddings en Remembrances-MCP!**
