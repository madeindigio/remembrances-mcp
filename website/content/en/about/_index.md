---
title: "About Remembrances MCP"
linkTitle: "About"
---

{{< blocks/cover title="About Remembrances MCP" image_anchor="center" height="auto" color="primary" >}}

Remembrances MCP is a **Model Context Protocol (MCP) server** that provides long-term memory capabilities to AI agents. Built with Go and powered by SurrealDB, it offers a flexible, privacy-first solution for managing AI agent memory.

{{< /blocks/cover >}}

{{% blocks/lead color="dark" %}}

## What Makes Remembrances MCP Special?

Traditional AI agents are stateless - they forget everything between conversations. Remembrances MCP solves this by providing **persistent memory**, **semantic search**, and **relationship mapping** while keeping your data private and secure.

{{% /blocks/lead %}}

{{% blocks/section color="white" %}}

## Key Features

{{% blocks/feature icon="fa-lock" title="Privacy-First Local Embeddings" %}}
Generate embeddings completely locally using GGUF models. Your data never leaves your machine, ensuring complete privacy and security.
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-bolt" title="GPU Acceleration" %}}
Take advantage of hardware acceleration with support for Metal (macOS), CUDA (NVIDIA GPUs), and ROCm (AMD GPUs).
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-database" title="Multiple Storage Layers" %}}
Key-Value Store for simple facts, Vector/RAG for semantic search, and Graph Database for relationship mapping.
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-book" title="Knowledge Base Management" %}}
Manage knowledge bases using simple Markdown files, making it easy to organize and maintain your AI's knowledge.
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-plug" title="Flexible Integration" %}}
Support for multiple embedding providers: GGUF Models (local), Ollama (local server), and OpenAI API (cloud-based).
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-shield-alt" title="Privacy Control" %}}
Keep sensitive data local with GGUF embeddings and embedded SurrealDB - no cloud dependencies required.
{{% /blocks/feature %}}

{{% /blocks/section %}}

{{% blocks/section color="primary" %}}

## Why Choose Remembrances MCP?

Remembrances MCP empowers your AI agents with powerful memory capabilities while maintaining complete control over your data.

{{% blocks/feature icon="fa-brain" title="Persistent Memory" %}}
Store facts, conversations, and knowledge permanently. Your AI remembers what matters.
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-search" title="Semantic Search" %}}
Find relevant information using vector embeddings. Smart search that understands context.
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-project-diagram" title="Relationship Mapping" %}}
Understand connections between different pieces of information using graph database capabilities.
{{% /blocks/feature %}}

{{% /blocks/section %}}

{{% blocks/section color="white" %}}

## Use Cases

<div class="row">
<div class="col-md-6">

### 🤖 Personal AI Assistants
Remember user preferences and past conversations to provide a truly personalized experience.

### 🔬 Research Assistants
Build and query knowledge bases from documents, papers, and research materials.

</div>
<div class="col-md-6">

### 💬 Customer Support
Maintain context across multiple interactions for better customer service.

### 💻 Development Tools
Store and retrieve code snippets, documentation, and technical knowledge.

</div>
</div>

{{% /blocks/section %}}

{{% blocks/section color="dark" %}}

## Technology Stack

Remembrances MCP is built with modern, proven technologies:

- **Language**: Go 1.20+ for performance and reliability
- **Database**: SurrealDB (embedded or external) for flexible data storage
- **Embeddings**: GGUF models via llama.cpp for local, privacy-first embeddings
- **Protocol**: Model Context Protocol (MCP) for seamless AI integration

{{% /blocks/section %}}

{{% blocks/section color="secondary" %}}

## Open Source & Community

Remembrances MCP is **open source** and available on [GitHub](https://github.com/madeindigio/remembrances-mcp). 

We welcome contributions from the community! Whether you want to report a bug, suggest a feature, or contribute code, we'd love to hear from you.

{{% /blocks/section %}}

{{% blocks/section color="primary" %}}

## Developed by Digio

<div style="text-align: center; padding: 2rem 0;">

Remembrances MCP is developed and maintained by [**Digio**](https://digio.es), a software development company specializing in AI and innovative solutions.

Visit us at [digio.es](https://digio.es) to learn more about our work.

</div>

{{% /blocks/section %}}
