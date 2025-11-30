# Plan: Dual Code Embeddings System

**Feature**: Specialized embedding models for code indexing  
**Status**: ✅ IMPLEMENTATION COMPLETE  
**Created**: November 30, 2025  
**Completed**: November 30, 2025  
**Branch**: `feature/dual-code-embeddings`

---

## Problem Statement

Generic text embedding models may not perform optimally for code understanding tasks. Specialized models like **CodeRankEmbed** or **Jina-code-embeddings** are trained specifically on code and can provide better semantic representation for:
- Code symbol search
- Semantic code similarity
- Code-to-code matching
- Natural language to code queries

The system should support:
- Configuring a separate embedding model for code indexing
- Fallback to default model if code model not specified
- Support for all 3 providers: GGUF (local), Ollama, OpenAI-compatible

---

## Phase Overview

| Phase | Title | Description | Status |
|-------|-------|-------------|--------|
| 1 | Configuration Extension | Add code-specific embedding model config options | ✅ Complete |
| 2 | Embedder Factory Extension | Create code-specific embedder factory function | ✅ Complete |
| 3 | Dual Embedder Integration | Integrate code embedder with indexer | ✅ Complete |
| 4 | Documentation | Update docs and sample configs | ✅ Complete |
| 5 | Testing & Validation | Test with real code embedding models | ✅ Complete |

---

## Configuration Additions

### New Configuration Options

| Option | CLI Flag | Env Var | YAML Key | Description |
|--------|----------|---------|----------|-------------|
| Code GGUF Model | `--code-gguf-model-path` | `GOMEM_CODE_GGUF_MODEL_PATH` | `code-gguf-model-path` | Path to GGUF model for code embeddings |
| Code Ollama Model | `--code-ollama-model` | `GOMEM_CODE_OLLAMA_MODEL` | `code-ollama-model` | Ollama model name for code embeddings |
| Code OpenAI Model | `--code-openai-model` | `GOMEM_CODE_OPENAI_MODEL` | `code-openai-model` | OpenAI model for code embeddings |

### Behavior

- **If code model specified**: Use it for code indexing, use default for text/facts/vectors/events
- **If code model NOT specified**: Use default model for everything (current behavior preserved)
- **Provider priority**: GGUF > Ollama > OpenAI (same as default embedder)

---

## PHASE 1: Configuration Extension

**Objective**: Add code-specific embedding model configuration to all configuration sources

### Tasks

1. Add new Config fields for code embeddings:
   - `CodeGGUFModelPath string`
   - `CodeOllamaModel string`
   - `CodeOpenAIModel string`
2. Add CLI flags:
   - `--code-gguf-model-path`
   - `--code-ollama-model`
   - `--code-openai-model`
3. Add env vars:
   - `GOMEM_CODE_GGUF_MODEL_PATH`
   - `GOMEM_CODE_OLLAMA_MODEL`
   - `GOMEM_CODE_OPENAI_MODEL`
4. Add YAML mapstructure tags:
   - `code-gguf-model-path`
   - `code-ollama-model`
   - `code-openai-model`
5. Add getter methods with fallback:
   - `GetCodeGGUFModelPath()` - returns CodeGGUFModelPath or GGUFModelPath
   - `GetCodeOllamaModel()` - returns CodeOllamaModel or OllamaModel
   - `GetCodeOpenAIModel()` - returns CodeOpenAIModel or OpenAIModel

### Files to Modify

- `internal/config/config.go` - Add fields, getters, and mapstructure tags
- `cmd/remembrances-mcp/main.go` - Add CLI flag definitions

### Notes

- Priority: GGUF > Ollama > OpenAI
- If code model not specified, use corresponding default model
- Keep same URL/key config for each provider (no need to duplicate)

---

## PHASE 2: Embedder Factory Extension

**Objective**: Extend embedder factory to create code-specific embedder instance

### Tasks

1. Add code embedder fields to `pkg/embedder/factory.go` Config struct:
   - `CodeGGUFModelPath string`
   - `CodeOllamaModel string`
   - `CodeOpenAIModel string`
2. Create `NewCodeEmbedderFromConfig()` function that creates embedder with code model config
3. Add `NewCodeEmbedderFromMainConfig()` that extracts code model config from main Config
4. Implement logic:
   - If code model specified for provider, create separate embedder
   - Else return same embedder as default
5. Each provider should check code-specific model first, fallback to default

### Files to Modify

- `pkg/embedder/factory.go` - Add code embedder factory functions

### Notes

- GGUF may need separate llama instance for code model (memory consideration)
- Ollama/OpenAI can use same client with different model param
- Consider memory implications of loading 2 GGUF models simultaneously

---

## PHASE 3: Dual Embedder Integration

**Objective**: Create dual embedder system and integrate with code indexer

### Tasks

1. Create `pkg/embedder/dual.go` with DualEmbedder struct:
   ```go
   type DualEmbedder struct {
       DefaultEmbedder Embedder
       CodeEmbedder    Embedder
   }
   ```
2. Add methods:
   - `EmbedCode(ctx, text)` - uses CodeEmbedder
   - `EmbedText(ctx, text)` - uses DefaultEmbedder
3. Update `internal/indexer/indexer.go`:
   - Accept separate code embedder parameter
   - Use code embedder for symbol embeddings
4. Update `internal/indexer/job_manager.go`:
   - Pass code embedder to indexer constructor
5. Update `cmd/remembrances-mcp/main.go`:
   - Create both default and code embedders
   - Pass code embedder to indexer/job manager

### Files to Create/Modify

- `pkg/embedder/dual.go` (new) - DualEmbedder struct
- `internal/indexer/indexer.go` - Accept code embedder
- `internal/indexer/job_manager.go` - Pass code embedder to indexer
- `cmd/remembrances-mcp/main.go` - Create and wire both embedders

### Notes

- ToolManager keeps using default embedder for facts/vectors/events
- Only code indexer uses code embedder
- Simplest approach: pass separate codeEmbedder to indexer constructor

---

## PHASE 4: Documentation & Sample Config

**Objective**: Update documentation and sample configuration files

### Tasks

1. Update `config.sample.yaml`:
   - Add `code-gguf-model-path` with example
   - Add `code-ollama-model` with example
   - Add `code-openai-model` with example
   - Add comments explaining code embeddings
2. Update `config.sample.gguf.yaml`:
   - Add code model configuration section
3. Update `README.md`:
   - Add "Code Embeddings Configuration" section
   - Explain when to use specialized code models
   - Document fallback behavior
4. Add comments explaining:
   - When to use specialized code embedding models
   - Recommended models (CodeRankEmbed, Jina-code-embeddings)
   - Memory considerations for dual GGUF models

### Files to Modify

- `config.sample.yaml`
- `config.sample.gguf.yaml`
- `README.md`

### Recommended Models

| Provider | Recommended Code Model | Notes |
|----------|----------------------|-------|
| GGUF | CodeRankEmbed | Local, fast, code-specialized |
| Ollama | jina/jina-embeddings-v2-code | Code-optimized |
| OpenAI | text-embedding-3-large | Works for both text and code |

---

## PHASE 5: Testing & Validation

**Objective**: Test the dual embedder system with real models

### Tasks

1. Test with `coderankembed.Q4_K_M.gguf` at `/www/Remembrances/coderankembed.Q4_K_M.gguf`
2. Verify code indexing uses code embedder when configured
3. Verify regular embeddings (facts, vectors, events) use default embedder
4. Test fallback: when code model not configured, should use default model
5. Test all 3 providers with code models:
   - GGUF: coderankembed.Q4_K_M.gguf
   - Ollama: jina-embeddings-v2-code (if available)
   - OpenAI: text-embedding-3-large
6. Build and verify compilation

### Test Cases

```bash
# Test 1: Code model configured - should use separate embedder
GOMEM_CODE_GGUF_MODEL_PATH=/www/Remembrances/coderankembed.Q4_K_M.gguf \
./build/remembrances-mcp

# Test 2: No code model - should use default embedder for everything
./build/remembrances-mcp

# Test 3: Verify embedding dimensions match MTREE index (768)
# Check logs for embedding dimension output
```

### Validation Criteria

- [ ] Code indexing uses code embedder when configured
- [ ] Regular embeddings use default embedder
- [ ] Fallback works when code model not specified
- [ ] Embedding dimension compatibility (768 for MTREE indexes)
- [ ] No memory leaks with dual GGUF models
- [ ] Build successful

### Notes

- Test model path: `/www/Remembrances/coderankembed.Q4_K_M.gguf`
- Verify embedding dimension compatibility (768 for MTREE indexes)
- May need different dimension handling if code model has different output size

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        main.go                                   │
├─────────────────────────────────────────────────────────────────┤
│  Config.Load()                                                   │
│      ↓                                                           │
│  ┌─────────────────────┐    ┌─────────────────────┐             │
│  │  Default Embedder   │    │   Code Embedder     │             │
│  │  (nomic-embed-text) │    │  (coderankembed)    │             │
│  └──────────┬──────────┘    └──────────┬──────────┘             │
│             │                          │                         │
│             ▼                          ▼                         │
│  ┌─────────────────────┐    ┌─────────────────────┐             │
│  │    ToolManager      │    │      Indexer        │             │
│  │  (facts, vectors,   │    │  (code symbols,     │             │
│  │   events, kb docs)  │    │   code chunks)      │             │
│  └─────────────────────┘    └─────────────────────┘             │
└─────────────────────────────────────────────────────────────────┘
```

---

## Success Criteria

1. ✅ Code-specific embedding model can be configured for each provider
2. ✅ Code indexer uses code embedder when configured
3. ✅ Regular operations (facts, vectors, events) use default embedder
4. ✅ Fallback to default model when code model not specified
5. ✅ Configuration works via CLI, env vars, and YAML
6. ✅ Documentation updated with examples
7. ✅ All tests pass
8. ✅ Build successful

---

## Related Facts

- `dual_code_embeddings_phase_1` - Configuration extension phase
- `dual_code_embeddings_phase_2` - Embedder factory extension phase
- `dual_code_embeddings_phase_3` - Dual embedder integration phase
- `dual_code_embeddings_phase_4` - Documentation phase
- `dual_code_embeddings_phase_5` - Testing phase

---

## Test Resources

- **GGUF Code Model**: `/www/Remembrances/coderankembed.Q4_K_M.gguf`
- **Default Text Model**: `nomic-embed-text:latest` (Ollama)
