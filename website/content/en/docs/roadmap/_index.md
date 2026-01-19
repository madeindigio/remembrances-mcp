---
title: "Roadmap"
linkTitle: "Roadmap"
weight: 50
description: >
  What features and characteristics are planned for future versions of Remembrances
---

## OpenSource and Free Version

It will always remain free and open source under MIT license.

- [x] SurrealDB support in server and embedded mode
- [x] Embeddings support (ollama and OpenAI embeddings API)
- [x] Local GGUF models support for embedding generation
- [x] GPU acceleration support (Metal, CUDA, ROCm)
- [x] Knowledge base support (markdown files)
- [x] Tools support similar to [Mem0](https://mem0.ai/) but in a single binary without scripting language dependencies like Python or NodeJS, faster and more efficient.
- [x] Short-term immediate memory support (key-value)
- [ ] [In progress] Support for source code and software projects indexing using code-specialized embeddings, implementation of tools inspired by [Serena](https://oraios.github.io/serena/) but with faster and more efficient indexing with AST and Tree-Sitter.
- [ ] AI agent for complex data ingestion and understanding how to store it through MCP sampling
- [ ] AI agent for deep knowledge querying through MCP sampling
- [ ] Implementation of knowledge reinforcement algorithms, generalization, and selective forgetting
- [ ] Support for advanced knowledge files (PDF, DOCX, etc)
- [ ] Support for multimedia files (images, audio, video) with multimodal embeddings
- [ ] Support for time series

## Commercial Version

- [ ] Support for multiple users and work teams
- [ ] Support for advanced auditing and logging
- [ ] Support for automatic backups and restoration and migration/export between database instances (both embedded and external SurrealDB server)
- [ ] Support for enterprise integrations (LDAP, SSO, etc)
- [ ] Web administration and monitoring interface
- [ ] Knowledge visualization interface (graphs, statistics, etc) and integrated chatbots for direct querying of stored knowledge
