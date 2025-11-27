---
title: "Examples"
linkTitle: "Examples"
weight: 4
description: >
  Practical examples and use cases for Remembrances
---

## Basic Usage Examples

### Read, Write and work with Knowledge Base

Add text similar to the following in your instructions for your AI agent:

```
Instructions for performing tasks:

- First search if there is any relevant information in your knowledge base.
- If the information does not exist, perform the task and save the results in the knowledge base as well as any relevant information you find for future queries.
- Use the knowledge base to answer user questions.
```

### Use semantic search

Add text similar to the following in your instructions for your AI agent:

```
Use Remembrances' hybrid search to find relevant information. Hybrid search combines literal matching and semantic search to get the best results.
```

### Store and retrieve key-value data

Add text similar to the following in your instructions for your AI agent:

```
Save the following fact in immediate memory with key "important_fact": "The capital of France is Paris".
Then, when you need to retrieve this fact, use the key "important_fact" to get the stored value.
```
```
<file_path>
remembrances-mcp/website/content/en/docs/examples/_index.md
</file_path>