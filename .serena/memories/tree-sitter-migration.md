# Tree-sitter Migration to madeindigio Fork

## Summary
Successfully migrated remembrances-mcp from `github.com/smacker/go-tree-sitter` to the updated fork `github.com/madeindigio/go-tree-sitter` and added support for Markdown and Vue languages.

## Changes Made

### 1. Updated Dependencies
- **go.mod**: Changed from `github.com/smacker/go-tree-sitter v0.0.0-20240827094217-dd81d9e9be82` to `github.com/madeindigio/go-tree-sitter v0.0.0-20260112132453-61cbcfc43e18`
- The fork includes more up-to-date language grammars and additional languages

### 2. Updated All Imports
Replaced all occurrences of `github.com/smacker/go-tree-sitter` with `github.com/madeindigio/go-tree-sitter` across all files in `pkg/treesitter/`

### 3. Added New Language Support

#### Markdown Support
- Added `LanguageMarkdown` constant in `pkg/treesitter/types.go`
- Added markdown import and language info in `pkg/treesitter/languages.go`
- Created `pkg/treesitter/markdown_extractor.go` with support for:
  - Heading extraction (using SymbolTypeNamespace for sections)
  - Hierarchical document structure
  - Supports extensions: `.md`, `.markdown`

#### Vue Support
- Added `LanguageVue` constant in `pkg/treesitter/types.go`
- Added vue2 import and language info in `pkg/treesitter/languages.go`
- Created `pkg/treesitter/vue_extractor.go` with support for:
  - Script section extraction
  - Class, function, method, variable, and constant symbols
  - Vue component object structure (methods, data, computed, etc.)
  - Supports extension: `.vue`
- Note: Uses `vue2` package as that's what's available in the fork

### 4. Updated AST Walker
- Registered `NewMarkdownExtractor(config)` in `pkg/treesitter/ast_walker.go`
- Registered `NewVueExtractor(config)` in `pkg/treesitter/ast_walker.go`

### 5. PHP Support
- PHP support was already implemented in the old version
- Continues to work with the new fork
- No changes needed

## Files Modified
1. `go.mod` - Updated dependency
2. `pkg/treesitter/types.go` - Added LanguageVue constant
3. `pkg/treesitter/languages.go` - Added imports and language definitions for markdown and vue2
4. `pkg/treesitter/ast_walker.go` - Registered new extractors
5. All extractor files in `pkg/treesitter/*_extractor.go` - Updated imports via sed

## Files Created
1. `pkg/treesitter/markdown_extractor.go` - New Markdown language extractor
2. `pkg/treesitter/vue_extractor.go` - New Vue language extractor

## Available Languages in Fork
The madeindigio/go-tree-sitter fork now supports:
bash, c, cpp, csharp, css, cue, dockerfile, elixir, elm, golang, groovy, hcl, html, java, javascript, kotlin, lua, markdown, mdx, ocaml, php, protobuf, python, ruby, rust, scala, sql, svelte, swift, toml, typescript, vue2, yaml

## Build Status
✅ Project builds successfully with warnings only in lua parser (upstream issue)
✅ Binary runs correctly and shows all options
✅ All language extractors registered and available

## Implementation Details

### Markdown Extractor Strategy
- Extracts headings as namespace symbols
- Builds hierarchical structure from document sections
- Handles both ATX (`#`) and Setext heading styles
- Trims markdown markers from heading text

### Vue Extractor Strategy
- Similar to Svelte extractor (both are component-based frameworks)
- Extracts JavaScript/TypeScript code from `<script>` sections
- Supports class declarations, functions, methods, variables
- Special handling for Vue component options object (methods, computed, etc.)
- Uses vue2 grammar from the fork

## Testing
Compiled binary successfully with command:
```bash
go build -mod=mod -o /tmp/remembrances-mcp ./cmd/remembrances-mcp
```

No errors, only one warning in lua parser (null character in literal - upstream issue).
