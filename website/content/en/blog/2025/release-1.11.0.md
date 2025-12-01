---
title: "Release 1.11.0: Events Tracking & Specialized Code Embeddings"
linkTitle: Release 1.11.0
date: 2025-11-30
author: Remembrances MCP Team
description: >
  Remembrances MCP 1.11.0 introduces temporal event tracking with hybrid search and support for specialized code embedding models.
tags: [release, features, events, embeddings]
---

We're pleased to announce **Remembrances MCP 1.11.0**, bringing two powerful new capabilities: a comprehensive **Events & Logs system** for temporal tracking and **Dual Code Embeddings** for optimized code search.

## 📅 Events & Logs System

Track activities, conversations, logs, and milestones with the new Events system. Unlike regular memories, events are designed for time-ordered data with powerful temporal queries.

### What Makes Events Special?

**Hybrid Search:**
Events combine vector similarity with BM25 text search. Search by meaning ("authentication issues") and get results ranked by both semantic relevance and keyword matching.

**Time-Based Queries:**
Find events from the last 24 hours, last week, or within specific date ranges. Perfect for tracking what happened when.

**Subject Organization:**
Categorize events by topic using subject patterns like `conversation:topic`, `error:type`, or `milestone:name`.

### Use Cases

**📝 Conversation Memory:**
Track important discussions across multiple sessions:
```
save_event({
  "user_id": "project-alpha",
  "subject": "conversation:sprint-planning",
  "content": "Team agreed to prioritize auth improvements. MVP deadline set for March 15."
})
```

**⚠️ Error Tracking:**
Log and search through incidents:
```
save_event({
  "user_id": "api-service",
  "subject": "error:database",
  "content": "Connection pool exhausted. Increased pool size from 10 to 25."
})
```

**🚀 Milestone Tracking:**
Mark and find achievements:
```
save_event({
  "user_id": "product",
  "subject": "milestone:release",
  "content": "Version 2.0 launched with dark mode and multi-language support."
})
```

### Powerful Search

Find events with combined filters:
```
search_events({
  "user_id": "api-service",
  "subject": "error:database",
  "query": "connection timeout",
  "last_days": 7,
  "limit": 20
})
```

This finds database errors from the last week that are semantically related to connection timeouts.

## 🧠 Dual Code Embeddings

Generic text embedding models work well for natural language, but code has different patterns and semantics. Version 1.11.0 introduces support for **specialized code embedding models**.

### Why Specialized Models?

Code-specific embedding models like **CodeRankEmbed** or **Jina Code Embeddings** are trained on source code and understand:
- Programming language syntax and patterns
- Code semantics and relationships
- Natural language to code mapping

This translates to better results when searching code semantically.

### How to Configure

Use a dedicated model for code indexing while keeping your general model for text:

**GGUF (Local):**
```yaml
# Main model for text
gguf-model-path: "./nomic-embed-text-v1.5.Q4_K_M.gguf"

# Code-specific model
code-gguf-model-path: "./coderankembed.Q4_K_M.gguf"
```

**Ollama:**
```yaml
ollama-model: "nomic-embed-text"
code-ollama-model: "jina/jina-embeddings-v3"
```

**OpenAI:**
```yaml
openai-model: "text-embedding-3-small"
code-openai-model: "text-embedding-3-large"
```

### Automatic Fallback

If you don't configure a code-specific model, Remembrances uses your default embedding model for everything. Upgrade at your own pace!

### Recommended Models

| Provider | Model | Best For |
|----------|-------|----------|
| GGUF | CodeRankEmbed | Local, private code search |
| Ollama | jina-embeddings-v3 | Quality + speed balance |
| OpenAI | text-embedding-3-large | Maximum quality |

## Getting Started

### Upgrade

Download from [GitHub Releases](https://github.com/madeindigio/remembrances-mcp/releases/tag/v1.11.0) and replace your existing binary.

### Try Events

Start tracking with:
```
save_event({
  "user_id": "my-project",
  "subject": "log:session",
  "content": "Started working on the new dashboard feature."
})
```

Search later:
```
search_events({
  "user_id": "my-project",
  "query": "dashboard feature",
  "last_days": 30
})
```

### Configure Code Embeddings

Add to your `config.yaml`:
```yaml
code-gguf-model-path: "./coderankembed.Q4_K_M.gguf"
```

Then re-index your projects to benefit from code-optimized embeddings.

## What's Next

We continue to enhance Remembrances MCP with:
- More event search capabilities
- Additional code embedding model support
- Performance improvements for high-volume event logging

Thank you for your continued support and feedback!

---

*Questions or feedback? Open an issue on [GitHub](https://github.com/madeindigio/remembrances-mcp/issues)!*
