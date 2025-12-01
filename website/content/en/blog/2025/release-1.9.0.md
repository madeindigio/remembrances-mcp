---
title: "Release 1.9.0: Code Indexing & Smart Token Savings"
linkTitle: Release 1.9.0
date: 2025-11-29
author: Remembrances MCP Team
description: >
  Remembrances MCP 1.9.0 introduces powerful code indexing with tree-sitter and an intelligent help system that reduces token consumption by 85%.
tags: [release, features, code-indexing]
---

We're excited to announce **Remembrances MCP 1.9.0**, a feature-packed release that brings two major capabilities: a powerful **Code Indexing System** for semantic code search and a new **how_to_use help system** that dramatically reduces token consumption.

## 🔍 Code Indexing System

The headline feature of this release is the **Code Indexing System** – a complete solution for AI agents to understand, search, and navigate codebases using semantic search.

### What Can You Do?

**Search Code by Meaning:**
Ask for "user authentication and password validation" and find relevant login functions, password checkers, and security modules – even if they don't contain those exact words.

**Navigate Large Codebases:**
Get instant overviews of file structures, find all implementations of an interface, track down references to a function, and understand call hierarchies.

**Manipulate Code Intelligently:**
Retrieve symbol implementations, replace function bodies, and insert new code at specific locations with full context awareness.

### 14+ Languages Supported

We've integrated **tree-sitter** for accurate AST parsing across a wide range of languages:

- **Go, Rust, C/C++** – Systems programming
- **TypeScript, JavaScript** – Web development  
- **Python, Ruby, PHP** – Scripting languages
- **Java, C#, Kotlin, Scala** – Enterprise languages
- **Swift** – Mobile development
- And more!

### How It Works

1. **Index your project:**
   ```
   code_index_project({
     "project_path": "/path/to/project",
     "project_name": "My App"
   })
   ```

2. **Search semantically:**
   ```
   code_semantic_search({
     "project_id": "my-app",
     "query": "database connection pooling"
   })
   ```

3. **Find and navigate symbols:**
   ```
   code_find_symbol({
     "project_id": "my-app",
     "name_path_pattern": "DatabasePool/getConnection"
   })
   ```

The indexer extracts all meaningful symbols – classes, functions, methods, interfaces – and creates vector embeddings for semantic similarity search. Changes are tracked and re-indexed automatically.

## 💡 Intelligent Help System (how_to_use)

With 37+ tools available, loading full documentation at the start of every conversation consumed ~15,000+ tokens before any actual work began. That's expensive and inefficient.

### The Solution: On-Demand Documentation

The new `how_to_use` tool provides documentation exactly when you need it:

| Before | After | Savings |
|--------|-------|---------|
| ~15,000 tokens upfront | ~2,500 tokens | **~85% reduction** |

### How It Works

Each tool now has a minimal 1-2 line description. When your AI agent needs more information:

```
how_to_use("code_semantic_search")
```

This loads only the documentation for that specific tool – full parameter descriptions, examples, and related tools.

You can also get category overviews:
```
how_to_use("code")      # All code indexing tools
how_to_use("memory")    # All memory tools
how_to_use("kb")        # All knowledge base tools
```

### Why This Matters

- **Lower costs:** Fewer tokens per conversation means lower API bills
- **Faster responses:** Less context to process means quicker initial responses
- **Better focus:** AI agents see relevant documentation when they need it

## Getting Started

### Upgrade

Download the latest release from [GitHub Releases](https://github.com/madeindigio/remembrances-mcp/releases/tag/v1.9.0) and replace your existing binary.

### Try Code Indexing

1. Start Remembrances MCP
2. Ask your AI to index a project:
   ```
   "Index my project at /path/to/project"
   ```
3. Search your code:
   ```
   "Find code related to user authentication"
   ```

### Explore the Help System

Ask your AI to run:
```
how_to_use()
```

To see an overview of all available capabilities.

## What's Next

We're continuing to enhance Remembrances MCP with:
- More language support for code indexing
- Advanced code analysis features
- Performance optimizations for large codebases

Thank you to everyone who provided feedback and feature requests. Your input shapes the future of Remembrances MCP!

---

*Found an issue? Have a feature request? Open an issue on [GitHub](https://github.com/madeindigio/remembrances-mcp/issues)!*
