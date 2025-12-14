# Remembrances-MCP — Plan to replace CGO llama.cpp binding with purego (GGUF)

## Goal
Remove the CGO-based llama.cpp integration (currently `local-go-llama` with `import "C"`) and replace it with a **purego**-based dynamic loader. The desired outcome is:

- No DT_NEEDED dependency on `libllama.so` at process start.
- Ability to ship/extract embedded `.so` files to a temporary directory and load them at runtime.
- Preserve GGUF embeddings functionality used by `pkg/embedder/gguf.go`.

## Key constraint
The current CGO code uses custom glue (`binding.h` / `binding.cpp`) that is compiled into the Go binary. If we remove CGO, we cannot call that glue unless it already exists as a shared library. Therefore the purego implementation must call **llama.cpp’s exported C API directly from `libllama.so`**, without changing how the shared libraries are built.

## Code touchpoints identified (what will change where)

### GGUF embedder (direct llama usage)
- `pkg/embedder/gguf.go`
  - Symbols observed via code index: `GGUFEmbedder`, `NewGGUFEmbedder`, `(*GGUFEmbedder).embedSingle`, `(*GGUFEmbedder).Dimension`.
  - Current coupling: `GGUFEmbedder.model` is `*llama.LLama` (from the go-llama.cpp binding), and `NewGGUFEmbedder` calls `llama.New(...)`.
  - Planned change:
    - Replace the `*llama.LLama` field with a new purego-backed handle type (e.g. `*llamapure.Model` / `*llama.Model`).
    - Replace `llama.New(...)` and `model.Embeddings(...)` with calls into the new purego wrapper.

### Configuration & selection logic
- `internal/config/config.go`
  - Holds GGUF-related getters/fields (e.g., `CodeGGUFModelPath`, `GetGGUFThreads`, etc.).
  - Planned change: no functional change required, but add/clarify logic to ensure the GGUF path triggers the purego loader initialization at first use (or during startup if preferred).

- `pkg/embedder/factory.go`
  - Symbols observed via code index: `NewEmbedderFromConfig`, `NewEmbedderFromEnv`, `NewEmbedderFromMainConfig`, `NewCodeEmbedderFromMainConfig`.
  - Planned change:
    - Keep the same priority logic (GGUF > Ollama > OpenAI), but ensure that when GGUF is selected, the purego-based llama package is used (not CGO).

### Existing embedded shared library support (already purego)
- `internal/embedded/*`
  - Already provides:
    - extraction of embedded `.so`/`.dylib` files to disk
    - path adjustment (`LD_LIBRARY_PATH`/`DYLD_LIBRARY_PATH`)
    - `purego.Dlopen(... RTLD_GLOBAL)` in a deterministic order (ggml → llama → surreal)
  - Planned change:
    - Reuse this package to load ggml/llama libraries *before* resolving symbols and before the first GGUF embedding call.

### SurrealDB embedded loader (related, but separate from GGUF)
- `internal/storage/surrealdb.go`
  - Uses `internal/embedded` to load embedded libraries when `UseEmbeddedLibs` is enabled.
  - Planned change:
    - Leave as-is for SurrealDB.
    - Optionally refactor so both SurrealDB + llama share a single “ensure embedded libs loaded” path (to avoid double work), but keep it idempotent.

### Legacy CGO llama binding (to be deprecated/removed)
- `local-go-llama/*`
  - Code index clearly contains `local-go-llama/options.go` (options surface used by GGUF embedder).
  - Note: the index does not currently surface symbols for `local-go-llama/llama.go`, but it exists in the repo and is the CGO entry point (`#cgo ...` + `import "C"`).
  - Planned change:
    - Introduce a new purego-based llama wrapper package.
    - During transition, keep `local-go-llama` behind a build tag (optional), then remove once purego is stable.

### Build/module dependencies
- `go.mod` and dependency graph
  - Planned change:
    - Remove the `go-llama.cpp` CGO module dependency from the default build once the purego wrapper is complete.
    - Ensure `purego` remains (already in use under `internal/embedded`).

## Phased migration

### Phase 1 — Discovery & constraints validation
Reference: fact `purego-plan/phase-1-discovery-and-constraints`

- Inventory the exact llama features used by remembrances today (embeddings path, required options).
- Verify exported symbols in shipped `libllama.so`/`libggml*.so` and detect API version differences.
- Produce an “API surface report” mapping current Go calls to llama C API calls.

### Phase 2 — New purego llama package skeleton
Reference: fact `purego-plan/phase-2-new-purego-llama-package-skeleton`

- Add a new Go package implementing symbol loading via purego (no `import "C"`).
- Integrate with existing `internal/embedded` extraction+`dlopen(RTLD_GLOBAL)` loader so deps resolve.
- Compile cleanly with CGO disabled.

### Phase 3 — Implement minimal embeddings path
Reference: fact `purego-plan/phase-3-implement-minimal-embeddings-path`

- Implement model load + tokenize + batch decode/eval + extract embeddings via llama.cpp C API.
- Update `pkg/embedder/gguf.go` to use the new wrapper.
- Add tests that enforce 768-dimension vectors (project MTREE constraint).

### Phase 4 — Options parity & performance hardening
Reference: fact `purego-plan/phase-4-options-parity-and-performance`

- Match current option behavior (threads, mmap/mlock, f16 memory, batch sizes, gpu layers).
- Improve error reporting and logging.
- Validate stability and concurrency behavior.

### Phase 5 — Cutover strategy, build tags, deprecation
Reference: fact `purego-plan/phase-5-cutover-build-tags-and-deprecation`

- Introduce build tags to allow a safe transition (purego default, optional legacy CGO fallback temporarily).
- Decouple embedder from a concrete llama implementation by using an internal interface.
- Deprecate and eventually remove the CGO binding.

### Phase 6 — Testing, CI, and release validation
Reference: fact `purego-plan/phase-6-testing-ci-and-release-validation`

- Add unit tests (mocked) and optional integration tests (real model) guarded by environment.
- Ensure CI runs with CGO disabled and without system libs installed.
- Validate release variants: binary runs and GGUF embeddings work after embedded extraction.

## Notes / risk management
- ABI mismatches can crash the process: prioritize strict signature/type mapping and version probes.
- llama.cpp C API changes over time: add compatibility shims selected by symbol probing.
- Embedded loader should be invoked before resolving symbols / first model usage.
