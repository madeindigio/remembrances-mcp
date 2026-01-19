---
title: "Help System (how_to_use)"
linkTitle: "Help System"
weight: 6
description: >
  On-demand documentation for reduced token consumption
---

Remembrances includes a built-in help system called `how_to_use` that provides on-demand documentation for all tools. This design reduces initial token consumption by approximately **85%**, making your AI interactions more efficient and cost-effective.

## Why how_to_use?

Traditionally, AI agents receive full documentation for all available tools at the start of each conversation. For Remembrances with its 37+ tools, this means loading ~15,000+ tokens of documentation before any actual work begins.

The `how_to_use` approach changes this:
- **Minimal initial context**: Each tool has a 1-2 line description
- **On-demand details**: Full documentation loaded only when needed
- **Better focus**: AI agents see relevant documentation when they need it

## Token Savings

| Metric | Traditional | With how_to_use | Savings |
|--------|-------------|-----------------|---------|
| Initial context | ~15,000 tokens | ~2,500 tokens | ~83% |
| Per conversation | Same every time | Only what's needed | Varies |

## How to Use

### Get Complete Overview

Ask your AI to call `how_to_use()` without parameters:

```
how_to_use()
```

This returns a high-level overview of all tool categories:
- Memory tools (facts, vectors, graph)
- Knowledge base tools
- Code indexing tools
- Events tools

### Get Group Documentation

For detailed information about a category of tools:

```
how_to_use("memory")
```

Available groups:
- `memory` – Facts, vectors, and graph operations
- `kb` – Knowledge base document tools
- `code` – Code indexing and search tools
- `events` – Event logging and search tools

### Get Specific Tool Documentation

For complete documentation on a single tool:

```
how_to_use("remembrance_save_fact")
```

```
how_to_use("code_semantic_search")
```

```
how_to_use("save_event")
```

This returns:
- Full description
- All parameters with types and descriptions
- Usage examples
- Related tools

## For AI Agents

When your AI agent encounters an unfamiliar tool or needs more information, it can use `how_to_use` to get exactly the documentation it needs. This pattern:

1. **Reduces context overhead**: Only loads documentation when needed
2. **Improves accuracy**: Fresh, focused documentation at the point of use
3. **Saves costs**: Fewer tokens means lower API costs

### Example Workflow

Instead of having all tool documentation loaded upfront, your AI agent can:

1. See the brief tool descriptions in its initial context
2. When it needs to use a specific tool, call `how_to_use("tool_name")`
3. Get detailed parameters and examples
4. Proceed with the actual tool call

## Best Practices

### For Users

- Let your AI discover tools naturally through `how_to_use()`
- If your AI seems confused about a tool, suggest it calls `how_to_use("tool_name")`
- Start sessions by having the AI check `how_to_use()` for an overview if it's unfamiliar with Remembrances

### For AI Agents

- Use `how_to_use()` at the start of complex tasks to understand available capabilities
- Look up specific tools before using unfamiliar functionality
- Reference group documentation when working with a category of operations
