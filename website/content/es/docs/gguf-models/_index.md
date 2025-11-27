---
title: "Modelos GGUF"
linkTitle: "Modelos GGUF"
weight: 5
description: >
  Descarga y optimiza modelos de embeddings GGUF
---

Esta sección sólo es relevante si estás usando modelos GGUF locales para generación de embeddings. Si estás usando Ollama o una API externa de embeddings (como OpenAI), no necesitas esta sección.

## Modelos Recomendados

### nomic-embed-text-v1.5 (Recomendado)

**Mejor para**: Embeddings de propósito general con excelente calidad

- **Dimensiones**: 768
- **Tamaño**: ~275MB (cuantización Q4_K_M)
- **Descarga**: [Hugging Face](https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF)

```bash
wget https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf
```

### Qwen3-Embedding-0.6B-Q8_0

**Mejor para**: Alta calidad de embeddings cuando el tamaño no es un problema

- **Dimensiones**: 1024
- **Tamaño**: ~1.2GB (cuantización Q8_0)
- **Descarga**: [Hugging Face](https://huggingface.co/Qwen/Qwen3-Embedding-0.6B-GGUF)

```bash
wget https://huggingface.co/Qwen/Qwen3-Embedding-0.6B-GGUF/resolve/main/Qwen3-Embedding-0.6B-Q8_0.gguf
```

## Niveles de Cuantización

Los modelos GGUF vienen en diferentes niveles de cuantización:

| Cuantización | Tamaño | Calidad | Velocidad | Recomendado Para |
|--------------|--------|---------|-----------|------------------|
| **Q4_K_M** | Pequeño | Buena | Rápida | ⭐ Uso general |
| **Q5_K_M** | Mediano | Mejor | Media | Alta calidad |
| **Q8_0** | Grande | Óptima | Lenta | Máxima calidad |
| **F16** | Muy grande | Perfecta | Muy lenta | Benchmarking |

**Recomendación**: Usa **Q4_K_M** para el mejor balance de tamaño, velocidad y calidad.

## Aceleración GPU

### Determinando Capas GPU

El parámetro `--gguf-gpu-layers` controla cuántas capas se descargan a la GPU:

```bash
# Solo CPU
--gguf-gpu-layers 0

# GPU parcial (recomendado para pruebas)
--gguf-gpu-layers 16

# GPU completa (mejor rendimiento)
--gguf-gpu-layers 32
```

**Encontrando el valor correcto**:
1. Comienza con `--gguf-gpu-layers 32`
2. Si obtienes errores OOM, reduce de 8 en 8
3. Monitoriza el uso de memoria GPU con `nvidia-smi` (NVIDIA) o `rocm-smi` (AMD)

### Consejos Específicos por Plataforma

#### NVIDIA (CUDA)

```bash
# Verificar disponibilidad de CUDA
nvidia-smi

# Ejecutar con GPU completa
./run-remembrances.sh \
  --gguf-model-path ./model.gguf \
  --gguf-gpu-layers 32 \
  --gguf-threads 8
```

#### AMD (ROCm)

```bash
# Verificar disponibilidad de ROCm
rocm-smi

# Ejecutar con GPU completa
./run-remembrances.sh \
  --gguf-model-path ./model.gguf \
  --gguf-gpu-layers 32 \
  --gguf-threads 8
```

#### Apple Silicon (Metal)

```bash
# Metal se detecta automáticamente
./run-remembrances.sh \
  --gguf-model-path ./model.gguf \
  --gguf-gpu-layers 32 \
  --gguf-threads 8
```

## Optimización de Rendimiento

### Número de Hilos

```bash
# Auto-detectar (recomendado)
--gguf-threads 0

# Manual (usa el número de núcleos CPU)
--gguf-threads 8
```

### Gestión de Memoria

- **Modelos pequeños** (< 100MB): Pueden ejecutarse completamente en memoria GPU
- **Modelos medianos** (100-500MB): Pueden necesitar descarga parcial a GPU
- **Modelos grandes** (> 500MB): Considera usar cuantización menor

## Guía de Selección de Modelos

Elige según tus necesidades:

| Caso de Uso | Modelo | Cuantización | Capas GPU |
|-------------|--------|--------------|-----------|
| **Producción** | nomic-embed-text-v1.5 | Q4_K_M | 32 |
| **Desarrollo** | all-MiniLM-L6-v2 | Q4_K_M | 16 |
| **Alta Calidad** | nomic-embed-text-v1.5 | Q8_0 | 32 |
| **Poca Memoria** | all-MiniLM-L6-v2 | Q4_K_M | 0 |

## Solución de Problemas

### Sin Memoria

Reduce las capas GPU:
```bash
--gguf-gpu-layers 16  # o menor
```

### Rendimiento Lento

Aumenta las capas GPU e hilos:
```bash
--gguf-gpu-layers 32
--gguf-threads 8
```

### El Modelo No Carga

Verifica la ruta del archivo y permisos:
```bash
ls -lh ./model.gguf
chmod +r ./model.gguf
```

## Ver También

- [Configuración](../configuration/) - Opciones de configuración del servidor
- [Primeros Pasos](../getting-started/) - Guía de instalación
