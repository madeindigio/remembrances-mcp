# Changelog

## [Unreleased](https://github.com-josedigio/madeindigio/remembrances-mcp/compare/v0.40.0...HEAD) (2025-11-12)

### Features

* **GGUF Embeddings Support** - Add support for local GGUF embedding models via go-llama.cpp
  - Load GGUF models directly (nomic-embed, qwen, and other BERT-based models)
  - GPU acceleration support (Metal for macOS, CUDA for NVIDIA, ROCm for AMD)
  - Configurable via CLI flags, environment variables, and YAML config
  - New CLI flags: `--gguf-model-path`, `--gguf-threads`, `--gguf-gpu-layers`
  - Priority: GGUF > Ollama > OpenAI
  - Full documentation in `docs/GGUF_EMBEDDINGS.md`
  - Build system with Makefile for easy compilation
  - Comprehensive tests and examples
  - Privacy-first: All embeddings generated locally
  - Cost-effective: No API costs
  - Support for quantized models (Q4_K_M, Q8_0, etc.)

* refactor surrealdb storage code
([a1af4cb](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/a1af4cb419c858243b712656ad8fe7d245ea4411))

### Fixes

* update stats and user stats
([f581744](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/f581744b5ef43a0246c5cd552aa2403064e81ec7))
* upgrade stats
([c65d9c0](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/c65d9c04a5b05f6413c5f4adefbd839bc9e16b74))
* error working with dates when inserts information
([9339687](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/9339687a491fd06a7f989920280f9212bfce0d89))

## [v0.40.0](https://github.com-josedigio/madeindigio/remembrances-mcp/compare/v0.35.3...v0.40.0) (2025-10-26)

### Features

* Add configuration file and sample
([7e38ac6](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/7e38ac64df073948eea8b7cb5d250b1c7d6167f0))

### [v0.35.3](https://github.com-josedigio/madeindigio/remembrances-mcp/compare/v0.30.4...v0.35.3) (2025-10-14)

### [v0.30.4](https://github.com-josedigio/madeindigio/remembrances-mcp/compare/v0.30.3...v0.30.4) (2025-10-26)

#### Features

* Add configuration file and sample
([7e38ac6](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/7e38ac64df073948eea8b7cb5d250b1c7d6167f0))
* Update tools descriptions for using user_id as the project name, know the
knowledge base embeddings is update then files in knowledge base path are
created or modified
([d0a25a3](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/d0a25a38e890c856447e09f655db822752b4469b))
* change kb_search_documents response to YAML format
([d61cc5d](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/d61cc5d8b35d7cce4c30129993af9256b3d5f10d))
* update AI config mcp tools and remove embeddings response from knowledge
base response
([b0bb978](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/b0bb9787b738cfebea232082886bfaf1e6c0d459))
* try to add docker image (not working by now
([0d59abf](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/0d59abf7978fa3c3ccb2fd26fb4a13ae87e3cea7))

#### Fixes

* fix timestamps of documents in knowledge base
([28b729e](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/28b729eed15e936fdb4571f115b0d9ff0034719d))

### [v0.30.3](https://github.com-josedigio/madeindigio/remembrances-mcp/compare/v0.30.2...v0.30.3) (2025-09-22)

### [v0.30.2](https://github.com-josedigio/madeindigio/remembrances-mcp/compare/v0.30.1...v0.30.2) (2025-09-22)

#### Fixes

* update description of knowledgebase for not using subfolder
([84f3c08](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/84f3c0831cfc76314b9f46d7a00d55a3d5199960))

### [v0.30.1](https://github.com-josedigio/madeindigio/remembrances-mcp/compare/v0.30.0...v0.30.1) (2025-09-12)

#### Features

* add db migration refactor with separated files
([0b8c4c1](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/0b8c4c119ebae6da23ca88e5971710ea69ef68f4))

## [v0.30.0](https://github.com-josedigio/madeindigio/remembrances-mcp/compare/v0.29.13...v0.30.0) (2025-09-12)

### Features

* add tests
([9e1a37c](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/9e1a37cc2fc4ca2c06e2db0361b4363478ddf531))

### Fixes

* recovering functionality to save knowledge base into fisical md files
([27ae5b8](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/27ae5b86d934e15ae18aa4558c94ad5b0ef49fe0))

### [v0.29.13](https://github.com-josedigio/madeindigio/remembrances-mcp/compare/v0.29.12...v0.29.13) (2025-09-12)

#### Fixes

* update migration for statistics, search and vector search is working
([e222c98](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/e222c98c67a30e421832ea21f219ef69fb1d96ca))

### [v0.29.12](https://github.com-josedigio/madeindigio/remembrances-mcp/compare/v0.29.11...v0.29.12) (2025-09-09)

#### Fixes

* fixed create_relationship
([1dd1e7c](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/1dd1e7c9397a00504e56dc0c493523c7809a9206))

### [v0.29.11](https://github.com-josedigio/madeindigio/remembrances-mcp/compare/v0.28.10...v0.29.11) (2025-09-09)

#### Fixes

* Fixes in get and list remembrances tools
([143b712](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/143b7127eff3083846db5062338c889ba25ac0e1))
* Fixes in some tools for creation remembrances
([3b77a5a](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/3b77a5a9c629fd2d6dadb5db72682fe5bca17976))
* **storage:** normalize embeddings to MTREE dim and convert to []float64; add
unit tests
([27dd261](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/27dd261e1e2b818f46a72ddcc8e18fc554a04c39))
* handle the warning about revision of surrealdb
([8ca77ea](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/8ca77ea823c83082d542c6113dd82cd7b867cc73))
* Working version cli argument and starting to test the mcp server at all
([0852947](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/08529475071d6a3dae79e8b75b213c481ac543c6))

### v0.28.10 (2025-08-28)

#### Features

* Add goreleaser and update version automatically
([36fedd1](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/36fedd18afa08759a9ee2995755164e4ed9bcbb3))
* streamable http server option, add more useful tests and memories about this
new transport and how to run the tests
([3d15f49](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/3d15f491bfaad0e756121ec6b21a4a02ec13df3d))
* improve tests
([a20fd55](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/a20fd551b5a833f70db12977bcb01d4691775e76))
* add sse port flag
([0ce0b7d](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/0ce0b7d5a5b054ed6bf2a265cf62ef14ce6b1cf1))
* Add control of surrealdb external process when finish the program
([739aefb](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/739aefb4508f1faa6a5ac267be4dc76657ed42d4))
* Add mock tests
([de34f38](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/de34f38a06f4ad0a4504417eaba848268a0b871a))
* Add a start external surreal db in the system
([7d7c3c3](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/7d7c3c301a8d1b114999a38d9270dd6d78808fed))
* Add more descriptive tools, improve the ai documentation using serena and
copilot agent instructions, add MCP configuration for copilot
([7306f8a](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/7306f8a741d7f5110ce5ae92b8bcfe7e4f42aae2))
* add log file
([65c6153](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/65c6153a719f43855c059ac0ae2b6456d93f2252))
* add mcp sdk initial
([b298024](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/b29802449582220d24cb9dd613e8fa616cbc0636))
* initial project
([1266a97](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/1266a975e51501b6f57eb2402bd325cede94b595))

#### Fixes

* working stdio tests
([99b3a24](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/99b3a24ebec482db5415565fe255edb95198e0c7))
* problem registering tools
([145e09f](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/145e09fadc2dbdc303d47a66c94d6be52868827a))
* check schema if exists
([9fa7b95](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/9fa7b95c56ffa7b4fac146658dd6b32b46dd61c1))
* problem connecting database, golang doesn't included embedded db. Work in
progress
([0ea8f8e](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/0ea8f8ede66e1dacb5a7f3730ff65d3d2c7d08fb))
