---
title: "Release 1.0.0: True Local-First AI Memory"
linkTitle: Release 1.0.0
date: 2025-11-18
author: Remembrances MCP Team
description: >
  Announcing Remembrances MCP 1.0.0 with native GGUF model support and embedded SurrealDB - no external dependencies required!
tags: [release, announcement]
---

We're thrilled to announce the release of **Remembrances MCP 1.0.0** – a major milestone that delivers on our promise of truly local-first AI memory!

## What's New

### 🧠 Native GGUF Model Support

The headline feature of this release is **built-in support for GGUF embedding models**. You no longer need to run Ollama or rely on external OpenAI-compatible APIs to generate embeddings. Simply download a GGUF model from Hugging Face and point Remembrances MCP to it:

```bash
./remembrances-mcp --gguf-model-path ./nomic-embed-text-v1.5.Q4_K_M.gguf
```

This means:
- **Zero external dependencies** for embedding generation
- **Complete privacy** – your data never leaves your machine
- **Simplified deployment** – one binary, one model file, done!

### 💾 Embedded SurrealDB Database

Alongside GGUF support, we've integrated an **embedded SurrealDB database** directly into the binary. You no longer need to install, configure, or manage a separate database server:

```bash
./remembrances-mcp --db-path ./my-memories.db --gguf-model-path ./model.gguf
```

Your memories are now stored in a single, portable database file that you can easily backup or move between systems.

### ⚡ GPU Acceleration

For those who want maximum performance, we've added GPU acceleration support:
- **Metal** for macOS (Apple Silicon)
- **CUDA** for NVIDIA GPUs
- **ROCm** for AMD GPUs

Enable GPU acceleration with a simple flag:

```bash
./remembrances-mcp --gguf-model-path ./model.gguf --gguf-gpu-layers 32
```

### 🔄 Backward Compatibility

Don't worry – all your existing setups continue to work! Remembrances MCP 1.0.0 maintains full support for:

- **OpenAI-compatible embedding APIs** – Use OpenAI, Azure OpenAI, or any compatible service
- **Ollama** – Continue using your local Ollama installation if preferred
- **External SurrealDB** – Connect to remote or self-hosted SurrealDB instances for distributed deployments

## Why This Matters

With version 1.0.0, Remembrances MCP becomes a truly **self-contained AI memory solution**. Whether you're building a personal AI assistant, a privacy-focused application, or simply want to experiment with AI memory without cloud dependencies, you now have everything you need in a single binary.

## Getting Started

1. Download the latest release from [GitHub Releases](https://github.com/madeindigio/remembrances-mcp/releases/tag/v1.0.0)
2. Download a GGUF embedding model (we recommend [nomic-embed-text-v1.5](https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF))
3. Run:
   ```bash
   ./remembrances-mcp --gguf-model-path ./nomic-embed-text-v1.5.Q4_K_M.gguf
   ```

Check out our [documentation](/docs/) for detailed setup instructions and configuration options.

## Thank You

A huge thank you to everyone who contributed to this release through feedback, bug reports, and feature requests. This is just the beginning – we have exciting plans for the future of Remembrances MCP!

---

*Have questions or feedback? Open an issue on [GitHub](https://github.com/madeindigio/remembrances-mcp/issues) or start a discussion!*