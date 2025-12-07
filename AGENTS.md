# Remembrances‑MCP — Project Instructions

This project is a fully operational Go-based MCP (Model Context Protocol) server implementing a **x-layers memory system** (key-value facts, vector/RAG semantic memories, graph entities/relationships, code, events) backed by SurrealDB and exposing functionality as MCP tools.

## Features

- MCP server support, stdio and http api
- API REST endpoints for memory management
- Integration with embeddings models using LangChain (OpenAI and ollama implemented)
- Plain text knowledge base and vector database support (Surrealdb implemented)
- Knowledge graph and memory graph support
- gguf models support for local embeddings generation and surrealdb embedded database support and external surrealdb database
- Support for llama.cpp with gpu optimizations

## Deveplopment

To build the project, execute the following command:

```bash
make BUILD_TYPE=cuda build 
```

For testing and running the MCP server locally, use:

```bash
xc build-and-copy
```
and request to user to start the MCP server manually.

## Language for documenting, comments, and code implementation

Always use English for all code, comments, and documentation although the user may write in other languages.

## 🛠️ Work Methodology

### Essential First Steps

1. **Activate project monitoring**: Use `code_activate_project_watch` to track changes in this project
2. **Read remembrances**: Use the tools of kb and hybrid_search and read relevant ones for context
3. **Research in codebase**: Use `code_hybrid_search` for current project implementation details
4. **Check the plan**: Review using `last_to_remember` for current tasks

### Development Workflow

- **Planning**: If you are unsure about if you are following a plan, check with `last_to_remember` to see the current plan and confirm with the user if needed, before proceeding. If you don't have a plan, create one, split into phases, save them each fase using `save_fact` and save a summary of the plan using `to_remember`.
- **Implementation**: Write code in small, testable increments. After each increment, run tests to ensure functionality.
- **Code Style**: Follow Go best practices and project-specific coding standards
- **Testing**: Create tests in `tests/` folder (Python files, not in root)

### External Research

- Use web search (google/brave/perplexity) for additional information when needed
- Use Context7 for API documentation and library usage patterns

## 🚨 Important Constraints & Warnings

1. **MTREE Dimension**: Embedder.Dimension() must return 768 to match schema indexes
2. **Config Validation**: Either OllamaModel or OpenAIKey required - intentional but blocks local dev without setup
3. **SurrealDB Syntax**: Some queries use custom parameterization syntax specific to SurrealDB Go client
4. **Tool Testing**: Mock storage and embedder in tests to avoid external API calls
5. **Schema Changes**: Be careful with migrations - embedded SurrealDB data persistence matters
