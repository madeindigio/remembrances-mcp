# Remembrances-MCP: Runtime library loading issues (Linux/macOS) — 2025-12-15

## Context
User reported failures when running **recent compiled remembrances binaries** built **without embedded libs (purego/go:embed)**, while shipping `.so`/`.dylib` files alongside the executable.

Expected behavior:
1) Prefer loading shared libraries from **the executable directory** (same folder as the binary).
2) Then fall back to system dynamic loader search (e.g., `LD_LIBRARY_PATH` on Linux).

Observed behavior:
- The program logged:
  - `Embedded libraries not available for this platform; falling back to system lookup`
  - Then attempted embedded SurrealDB and failed due to URL parsing:
    - `unsupported embedded SurrealDB URL: surrealkv:///...`
- Later, GGUF embedder failed:
  - `dlopen /home/.../libggml.so: libggml-cuda.so: cannot open shared object file: No such file or directory`

## Root causes
### A) Non-embedded builds incorrectly skipped binary-dir lookup
`internal/embedded/manager.go` had a guard in `tryLoadFromBinaryDir` that returned `ErrPlatformUnsupported` when `platformSupported == false`.

But `platformSupported` is **only true when compiled with embedded build tags** (`embedded && ...`).
Therefore, binaries built without embedded libs *could not* attempt loading libraries next to the executable.

Additionally, file filtering and ordering relied on `platformLibExt` which is empty in non-embedded builds, preventing correct `.so`/`.dylib` detection.

### B) Embedded SurrealDB URL parser rejected `surrealkv://`
`internal/surrealembedded/surreal.go` supported:
- `memory`, `memory://`
- `rocksdb://<path>`
- `file://<path>` (alias)
- `<path>` (plain path)

But it rejected any other scheme, including `surrealkv://...`.

### C) llama/ggml dynamic dependency issue (CUDA/Metal backends)
In a distribution layout, `libggml.so` (or `libggml.dylib`) can have a dynamic dependency on backend libs (e.g., `libggml-cuda.so` or `libggml-metal.dylib`).
If that dependency is missing from the dynamic loader search path *at dlopen time*, loading fails.

## Fixes implemented
### 1) Enable binary-dir library loading even in non-embedded builds
Files:
- `internal/embedded/manager.go`
- `internal/embedded/extractor.go`
- `internal/embedded/loader.go`
- `internal/embedded/embedded_test.go`

Key changes:
- Removed the incorrect `platformSupported` gate from binary-dir loading.
- Introduced `libraryFileExt()` runtime fallback so library extension works even when `platformLibExt == ""` (non-embedded builds).
  - Linux: `.so`
  - macOS: `.dylib`
  - Windows: `.dll`
- Updated ordering logic to use `libraryFileExt()`.
- Added/updated tests to not depend on `platformLibExt`.

Result:
- When `UseEmbeddedLibs` is enabled, the loader now:
  1) Scans executable directory for shared libs.
  2) Prepends executable directory to loader path via `AppendLibraryPath`.
  3) dlopen’s libraries in a stable order.
  4) Falls back to embedded extraction only if embedded build tags are present.

### 2) Support `surrealkv://` in embedded SurrealDB URL parser
Files:
- `internal/surrealembedded/surreal.go`
- `internal/surrealembedded/surreal_test.go`

Key changes:
- Added a parse function that accepts `surrealkv://<path>` as a supported alias.
- Parsing now happens **before** attempting to load the native library, enabling unit testing without requiring successful dlopen.

### 3) Harden llama loader against missing backend dependency
Files:
- `internal/llama/loader.go`
- `internal/llama/loader_test.go`

Key changes:
- If dlopen of an **absolute path** fails and the error message indicates a **known backend dependency** missing:
  - Linux: `libggml-cuda.so` / `libggml-cpu.so`
  - macOS: `libggml-metal.dylib`
- The loader attempts to dlopen the dependency from the **same directory** as the library and retries the original dlopen.
- If missing, returns a clearer error hinting that the full variant library set must be copied.

## macOS notes (important)
The same issue can occur on macOS with `.dylib` dependencies (e.g., `libggml-metal.dylib`) because dyld resolves transitive dependencies at load time.

Additionally:
- `DYLD_LIBRARY_PATH` can be restricted depending on SIP / restricted contexts / signed apps.
- For robust distribution on macOS, prefer using `@rpath`, `@executable_path`, or `@loader_path` in install names and setting appropriate `LC_RPATH` entries.

## Verification
- `go test ./...` passed after changes.

## Log symptom references
- Embedded libs fallback message originally seen in: `internal/storage/surrealdb.go`
- Unsupported URL error originally from: `internal/surrealembedded/surreal.go`
- GGUF embedder error example:
  - `dlopen /home/sevir/bin/libggml.so: libggml-cuda.so: cannot open shared object file: No such file or directory`

## Actionable guidance for packaging
- When shipping a non-embedded binary, always copy the **complete** set of shared libs for the selected variant (cpu/cuda/cuda-portable/metal) into the same directory as the binary.
- Ensure backend libs (`libggml-cuda.*`, `libggml-metal.*`) are included when present in dependencies.
