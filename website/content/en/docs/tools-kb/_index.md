---
title: "Knowledge Base Tools"
linkTitle: "KB Tools"
weight: 10
description: >
  Document storage and semantic search capabilities
---

The Knowledge Base (KB) system allows you to store documents with automatic semantic search, ideal for maintaining documentation, notes, and any content you need to retrieve by meaning.

## Available Tools

### kb_add_document
Add a document with automatic embedding generation for semantic search.

### kb_search_documents
Search documents by semantic similarity to a natural language query.

### kb_get_document
Retrieve a specific document by its file path.

### kb_delete_document
Remove a document from the knowledge base.

## Typical Workflow

1. **Add documents**: Use `kb_add_document` with content and file path
2. **Search**: Use `kb_search_documents` with natural language queries
3. **Retrieve**: Use `kb_get_document` for full content when needed
4. **Maintain**: Use `kb_delete_document` to remove outdated docs

## Features

- **Automatic embedding generation**: The system automatically converts your documents into vectors for semantic search
- **File path as identifier**: Each document is identified by its path (e.g., `guides/authentication.md`)
- **Optional metadata**: Add metadata for organization and filtering
- **Markdown file synchronization**: If you configure a knowledge base path, documents sync with `.md` files
- **Automatic chunking**: Large documents are automatically split into manageable chunks

## Recommended Prompts

### For Adding Documents

```
Add this document to the knowledge base at path "architecture/microservices.md":
[document content]
```

```
Save the following style guide at "team/style-guide.md" with metadata 
category: "guidelines", team: "frontend"
```

### For Searching Documents

```
Search for documentation about implementing JWT authentication
```

```
Find information related to production deployment and CI/CD configuration
```

```
What documents do we have about testing best practices?
```

### For Managing Documents

```
Show me the full content of the document at "api/endpoints.md"
```

```
Delete the obsolete document at "legacy/old-api.md"
```

## Best Practices

### Path Organization

- Use descriptive hierarchical paths: `projects/name/subsection.md`
- Group related documents in logical folders
- Keep names consistent and easy to remember

### Document Content

- Write clear and descriptive titles
- Include relevant context in the document
- Use markdown for consistent formatting
- Add relevant metadata to facilitate filtering

### Effective Search

- Formulate queries in natural language describing what you're looking for
- Results include relevance scores for ranking
- Search is semantic: finds documents by meaning, not just exact words
- Combine multiple searches to refine results

## Common Use Cases

### Project Documentation

Keep all technical project documentation accessible:

```
Add the architecture documentation
Search for information about the payments module
Update the deployment guide
```

### Team Knowledge Base

Centralize shared team knowledge:

```
Save important design decisions
Search for documents about code conventions
Retrieve the onboarding guide
```

### Notes and References

Store research notes and references:

```
Add these notes about performance optimization
Search for information I saved about GraphQL
```

## Filesystem Integration

If you configure `--knowledge-base` or `GOMEM_KNOWLEDGE_BASE` with a directory path:

- Documents are saved as `.md` files in the filesystem
- You can edit files directly and they'll sync
- Ideal for Git integration and version control
- Documents remain accessible even outside Remembrances

## See More

For detailed documentation of each tool, use the built-in help system:

```
how_to_use("kb_add_document")
how_to_use("kb_search_documents")
how_to_use("kb_get_document")
how_to_use("kb_delete_document")
```
