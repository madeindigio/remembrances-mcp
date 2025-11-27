---
title: "GGUF Models"
linkTitle: "GGUF Models"
weight: 5
description: >
  Download and optimize GGUF embedding models
---

This section is only relevant if you're using local GGUF models for embedding generation. If you're using Ollama or an external embedding API (like OpenAI), you don't need this section.

## Recommended Models

### nomic-embed-text-v1.5 (Recommended)

**Best for**: General-purpose embeddings with excellent quality

- **Dimensions**: 768
- **Size**: ~275MB (Q4_K_M quantization)
- **Download**: [Hugging Face](https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF)

```bash
wget https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf
```

### Qwen3-Embedding-0.6B-Q8_0

**Best for**: High-quality embeddings when size is not an issue

- **Dimensions**: 1024
- **Size**: ~1.2GB (Q8_0 quantization)
- **Download**: [Hugging Face](https://huggingface.co/Qwen/Qwen3-Embedding-0.6B-GGUF)

```bash
wget https://huggingface.co/Qwen/Qwen3-Embedding-0.6B-GGUF/resolve/main/Qwen3-Embedding-0.6B-Q8_0.gguf
```

## Quantization Levels

GGUF models come in different quantization levels:

| Quantization | Size | Quality | Speed | Recommended For |
|--------------|------|---------|-------|-----------------|
| **Q4_K_M** | Small | Good | Fast | ⭐ General use |
| **Q5_K_M** | Medium | Better | Medium | High quality |
| **Q8_0** | Large | Optimal | Slow | Maximum quality |
| **F16** | Very large | Perfect | Very slow | Benchmarking |

**Recommendation**: Use **Q4_K_M** for the best balance of size, speed, and quality.

## GPU Acceleration

### Determining GPU Layers

The `--gguf-gpu-layers` parameter controls how many layers are offloaded to GPU:

```bash
# CPU only
--gguf-gpu-layers 0

# Partial GPU (recommended for testing)
--gguf-gpu-layers 16

# Full GPU (best performance)
--gguf-gpu-layers 32
```

**Finding the right value**:
1. Start with `--gguf-gpu-layers 32`
2. If you get OOM errors, reduce by 8s
3. Monitor GPU memory usage with `nvidia-smi` (NVIDIA) or `rocm-smi` (AMD)

### Platform-Specific Tips

#### NVIDIA (CUDA)

```bash
# Check CUDA availability
nvidia-smi

# Run with full GPU
./run-remembrances.sh \
  --gguf-model-path ./model.gguf \
  --gguf-gpu-layers 32 \
  --gguf-threads 8
```

#### AMD (ROCm)

```bash
# Check ROCm availability
rocm-smi

# Run with full GPU
./run-remembrances.sh \
  --gguf-model-path ./model.gguf \
  --gguf-gpu-layers 32 \
  --gguf-threads 8
```

#### Apple Silicon (Metal)

```bash
# Metal is detected automatically
./run-remembrances.sh \
  --gguf-model-path ./model.gguf \
  --gguf-gpu-layers 32 \
  --gguf-threads 8
```

## Performance Optimization

### Thread Count

```bash
# Auto-detect (recommended)
--gguf-threads 0

# Manual (use your CPU core count)
--gguf-threads 8
```

### Memory Management

- **Small models** (< 100MB): Can run entirely in GPU memory
- **Medium models** (100-500MB): May need partial GPU offloading
- **Large models** (> 500MB): Consider using lower quantization

## Model Selection Guide

Choose based on your needs:

| Use Case | Model | Quantization | GPU Layers |
|----------|-------|--------------|------------|
| **Production** | nomic-embed-text-v1.5 | Q4_K_M | 32 |
| **Development** | all-MiniLM-L6-v2 | Q4_K_M | 16 |
| **High Quality** | nomic-embed-text-v1.5 | Q8_0 | 32 |
| **Low Memory** | all-MiniLM-L6-v2 | Q4_K_M | 0 |

## Troubleshooting

### Out of Memory

Reduce GPU layers:
```bash
--gguf-gpu-layers 16  # or lower
```

### Slow Performance

Increase GPU layers and threads:
```bash
--gguf-gpu-layers 32
--gguf-threads 8
```

### Model Won't Load

Check file path and permissions:
```bash
ls -lh ./model.gguf
chmod +r ./model.gguf
```

## See Also

- [Configuration](../configuration/) - Server configuration options
- [Getting Started](../getting-started/) - Installation guide
```
<file_path>
remembrances-mcp/website/content/en/docs/gguf-models/_index.md
</file_path>

<edit_description>
Translate GGUF models documentation from Spanish to English
</edit_description>