---
title: "Getting Started"
linkTitle: "Getting Started"
weight: 1
description: >
  Install and run Remembrances MCP in minutes
---

## Prerequisites

- Linux, MacOSX or Windows (with WSL), alternatively use Docker on Windows if you don't have Windows Subsystem for Linux.
- Recommended: Nvidia GPU with configured drivers on Linux, or Mac with M1..M4 chip for graphics acceleration. In the case of Windows, use Docker with Nvidia graphics acceleration support.

Although it will be possible by compiling the application, there are no binaries available yet for native Windows and Linux with AMD GPU support (ROCm). You will need to compile the project manually in these cases.

## Installation

### On Linux or MacOSX (Windows with WSL)

```bash
curl -fsSL https://raw.githubusercontent.com/madeindigio/remembrances-mcp/main/scripts/install.sh | bash
```

This will install the binary for your operating system, optimized for CPU (MacOSX with M1..M4 graphics acceleration support, and Linux for Nvidia GPU -CUDA-)

### Compile the Project (if you have an AMD GPU or want custom GPU support)

This is only necessary if you're not using the installation script above, or if you have an AMD GPU (ROCm) and want support for it.

```bash
make surrealdb-embedded
make build-libs-hipblas
make BUILD_TYPE=hipblas build
```

This will:
- Compile the embedded SurrealDB library
- Compile llama.cpp for AMD GPU support (ROCm)
- Build the `remembrances-mcp` binary with AMD GPU support (ROCm)

### Download a GGUF Model

This step is only necessary if you don't already have a downloaded GGUF model (the installation script downloads the recommended model)

Download the recommended nomic-embed-text-v1.5 model:

```bash
wget https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf
```

Other recommended models:
- **nomic-embed-text-v1.5** (768 dimensions) - Best balance
- **nomic-embed-text-v2-moe** (768 dimensions) - Faster, better quality
- **Qwen3-Embedding-0.6B-Q8_0** (1024 dimensions) - High quality, higher memory usage

Search for them on HuggingFace: https://huggingface.co/nomic-ai and https://huggingface.co/Qwen

### Alternatives for embedding model usage

If you like using Ollama and want to configure different embedding models, you can alternatively use Ollama. Remembrances supports parameter passing in multiple ways, see the Configuration section for more details.

Alternatively, if you don't have supported GPU (Intel i7 Ultra or processors without a dedicated GPU), you can use cloud embedding models like OpenAI's or any compatible with the OpenAI API (Azure OpenAI, OpenRouter, etc.). See Configuration for more details. Generating embeddings in the cloud may have associated costs but doesn't require specific hardware and doesn't impact application performance.
```
<file_path>
remembrances-mcp/website/content/en/docs/getting-started/_index.md
</file_path>