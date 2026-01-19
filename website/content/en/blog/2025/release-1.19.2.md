---
title: "Release 1.19.2"
date: 2026-01-19
categories: [release]
tags: [release, update, code-indexing, languages]
---

Version **1.19.2** of Remembrances-MCP is now available! This release expands language support for code indexing and improves stability.

## What's New

### Expanded Language Support for Code Indexing

Remembrances-MCP now understands and indexes code in even more languages and frameworks:

- **Svelte:** Full support for Svelte components, enabling better code understanding in Svelte-based projects
- **MDX:** Index and search MDX files, perfect for documentation-as-code workflows
- **Markdown:** Enhanced support for standard Markdown files
- **Vue:** Complete Vue.js component support for better project navigation
- **Lua:** Support for Lua scripts, ideal for embedded systems and game development

These additions make Remembrances-MCP a more versatile tool for multi-language projects and modern web development stacks.

### How to Use Code Indexing with New Languages

Simply index your project with the `code_project_index` tool, and Remembrances-MCP will automatically:

1. Detect files in the newly supported languages
2. Parse and extract symbols, functions, and structure
3. Create searchable embeddings for semantic code search
4. Enable intelligent code completion and understanding

Example use cases:

- **Documentation sites:** Index your MDX-based documentation for intelligent search
- **Vue/Svelte projects:** Navigate large component libraries with ease
- **Lua scripts:** Search and understand embedded Lua configurations
- **Mixed-language codebases:** Work seamlessly across JavaScript, TypeScript, Vue, Svelte, and Markdown in the same project

### Enhanced Stability

- **Better exception handling:** Improved error handling when indexing code prevents crashes on malformed files
- **Ignored file handling:** Better management of ignored files during indexing process

## Why Update?

- **Work with modern frameworks:** Full support for Svelte and Vue makes this essential for modern web developers
- **Better documentation workflows:** MDX support enables powerful documentation-as-code capabilities
- **More robust indexing:** Enhanced error handling ensures reliable code indexing even with imperfect codebases

Download the new version here:

[Download Remembrances-MCP v1.19.2](https://github.com/madeindigio/remembrances-mcp/releases/tag/v1.19.2)

Update now and index your entire tech stack!
