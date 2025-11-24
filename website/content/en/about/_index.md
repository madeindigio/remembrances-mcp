---
title: "About Remembrances MCP"
linkTitle: "About"
---

## What is Remembrances MCP?

Remembrances MCP is a **Model Context Protocol (MCP) server** that provides long-term memory capabilities to AI agents. Built with Go and powered by SurrealDB, it offers a flexible, privacy-first solution for managing AI agent memory.

## Key Features

### 🔒 Privacy-First Local Embeddings

Generate embeddings completely locally using GGUF models. Your data never leaves your machine, ensuring complete privacy and security.

### ⚡ GPU Acceleration

Take advantage of hardware acceleration with support for:
- **Metal** (macOS)
- **CUDA** (NVIDIA GPUs)
- **ROCm** (AMD GPUs)

### 💾 Multiple Storage Layers

- **Key-Value Store**: Simple fact storage and retrieval
- **Vector/RAG**: Semantic search with embeddings
- **Graph Database**: Relationship mapping and traversal

### 📝 Knowledge Base Management

Manage knowledge bases using simple Markdown files, making it easy to organize and maintain your AI's knowledge.

### 🔌 Flexible Integration

Support for multiple embedding providers:
- **GGUF Models** (local, privacy-first) ⭐ Recommended
- **Ollama** (local server)
- **OpenAI API** (cloud-based)

## Why Remembrances MCP?

Traditional AI agents are stateless - they forget everything between conversations. Remembrances MCP solves this by providing:

1. **Persistent Memory**: Store facts, conversations, and knowledge permanently
2. **Semantic Search**: Find relevant information using vector embeddings
3. **Relationship Mapping**: Understand connections between different pieces of information
4. **Privacy Control**: Keep sensitive data local with GGUF embeddings

## Use Cases

- **Personal AI Assistants**: Remember user preferences and past conversations
- **Research Assistants**: Build and query knowledge bases from documents
- **Customer Support**: Maintain context across multiple interactions
- **Development Tools**: Store and retrieve code snippets and documentation

## Technology Stack

- **Language**: Go 1.20+
- **Database**: SurrealDB (embedded or external)
- **Embeddings**: GGUF models via llama.cpp
- **Protocol**: Model Context Protocol (MCP)

## Open Source

Remembrances MCP is open source and available on [GitHub](https://github.com/madeindigio/remembrances-mcp). Contributions are welcome!

## Developed by Digio

Remembrances MCP is developed and maintained by [Digio](https://digio.es), a software development company specializing in AI and innovative solutions.
