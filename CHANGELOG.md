# Changelog

### [v1.16.3](https://github.com-josedigio/madeindigio/remembrances-mcp/compare/v1.14.1...v1.16.3) (2025-12-11)

#### Features

* search posible project when activate_project does not found by project id
([8c8760d](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/8c8760dc79426dbfa3e18fa892c48ef262cce865))
* return toon format in mcp tools
([34b851e](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/34b851eda16747053ab51a954abea2ef5429bc40))
* add compilation options for embeded shared libraries
([a960176](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/a960176bd4cb03d8c23252479aae6d874cc4a8fc))
* include shared library for surrealdb_embedded in the binary
([58ba324](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/58ba3247a3281b1845677e604e7276e7bb1c4230))
* return alternatives of no-results returned with a specific user_id, and now
responses are in yaml
([874b90a](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/874b90a6c8bc1b5adba3ccbf7533d2305eb3b5c9))

#### Fixes

* Some fixes using surrealdb external
([a41adc0](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/a41adc0fe694def033b882b007d73e478d477b46))
* search shared libraries in binary directory, if not found use temporal
folder
([d8cb701](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/d8cb701708b9257856a270ff53130033001f5650))
* update install script
([75f3628](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/75f3628803001f2390697c60583c7043913b64c0))
* Docker compilation and upload fixed
([d68ae47](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/d68ae4793be8dd50ac3cb6d1d8a98d35de891d78))
* compilation automatic task in osx
([1481856](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/148185658014fce3c36e4eb6510d1ca937c18487))
* enable cuda support for AMD Ryzen CPU
([7f56b7c](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/7f56b7c13e6bb329046ac7f9eb7b446ea56e6369))

### [v1.14.1](https://github.com-josedigio/madeindigio/remembrances-mcp/compare/v1.14.0...v1.14.1) (2025-12-03)

#### Fixes

* problem indexing code with tree sitter and python
([c449242](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/c449242fd14a66426834aec159ae5bbc1e46dabc))

## [v1.14.0](https://github.com-josedigio/madeindigio/remembrances-mcp/compare/v1.13.1...v1.14.0) (2025-12-03)

### Fixes

* problems with timestamps in events tools
([9cb8687](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/9cb86875411e15f5989689ed8cd18c3beb8b67a2))
* add slog instead log
([811980e](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/811980e8512bf704f62f81e447bd88d1ac77010b))
* indexing code jobs completion error
([7ef87d8](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/7ef87d83cbb50fc958a0a1a02431fd345ce389fc))
* fix problems in status of code project indexation
([096a199](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/096a19942c4be82f50bc815a10717a997af503c9))

### [v1.13.1](https://github.com-josedigio/madeindigio/remembrances-mcp/compare/v1.11.0...v1.13.1) (2025-12-01)

#### Features

* refactor code splitting
([2c036d5](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/2c036d526ee1c12bef4a894e3657b91f5af38989))
* **code-monitoring:** implement automatic file monitoring system
([ddb1f9c](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/ddb1f9cd34eb8045e764974205500db32bd63fdd))

## [v1.11.0](https://github.com-josedigio/madeindigio/remembrances-mcp/compare/v1.9.0...v1.11.0) (2025-11-30)

### Features

* dual code embeddings model
([882abc8](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/882abc8a87e5ebff813e9b0c54e02f191ca2618d))
* add events or logs tools
([ff410a0](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/ff410a0a31191a713037ef4109b48411343cf7f0))

## [v1.9.0](https://github.com-josedigio/madeindigio/remembrances-mcp/compare/v1.4.6...v1.9.0) (2025-11-30)

### Features

* token reduction in tool description using how-to-use
([34aa10f](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/34aa10f0976c9d47b7b25e22365ad400999085bc))
* documentation of code indexing
([5c6258d](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/5c6258dcc8f9b70b45a6b37cbc24946b1b454d2a))
* add code manipulation tools
([a93aaa5](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/a93aaa50f02f81e35087a3cc28c8f72965987a70))
* add tools for code search, symbols and patterns
([931a4d9](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/931a4d9d45cff638045de6f375d44377c2990345))
* first mvp of code indexing
([c454e09](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/c454e09bd5a6b50bfdd64512eeb15e445d86ccbd))
* Add installation script
([6626231](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/6626231fe430a6e40dd8ab30458a3bc970b30886))
* add complete documentation and Digio-branded styling
([0bb6994](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/0bb699474e2b219c75062d893b19b224bd1d16ad))

### Fixes

* fixes list facts and add a new feature plan
([a9cf036](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/a9cf036b73b9cbaad6545c0ae0b734e8b5572427))
* correct About page desktop layout for feature blocks grid
([8401285](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/84012858e6caf61924c84707494b2f38a444723e))
* remove custom head-css.html to fix CSS loading
([9ca50d3](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/9ca50d36e1b7b3a8997cace632299cfaed14c68c))
* rename custom CSS to _styles_project.scss for Docsy import
([d1d9a9b](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/d1d9a9b378e6a13262c8c12926947dc352fb1ca2))
* configure language-specific menus in hugo.toml
([9f92422](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/9f9242275785df22c30c1c3dc9766b3c3d2cc3a2))

### [v1.4.6](https://github.com-josedigio/madeindigio/remembrances-mcp/compare/v1.0.0...v1.4.6) (2025-11-22)

#### Features

* Add short place memory
([167ea8a](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/167ea8af9afc02cf793c4692233fc6a7904db25b))
* Add support to config file in user config standard path
([6007449](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/6007449d862fdd37e1f4b7af44788cf69cce2b66))
* enable compilation of llama.cpp for all gpu variants
([e1dc8ae](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/e1dc8ae71a79f1bd505329c346bc511b5064579c))

#### Fixes

* problems with update in some tools
([f32ca0b](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/f32ca0b135f76aa0e730b5fec820d93393b37e92))
* error with surreal-embedded, and knowledge base storage
([0237129](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/02371297bfb6a069effaa6e98d80bdbd2759538c))
* update makefile for variants
([0702bd9](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/0702bd9accb708a6d3b22821f26eb74bb0470fd2))
* use chunking into database and read big files
([11d5793](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/11d5793f6f48678174407a1d5a9fc72661d64080))

## [v1.0.0](https://github.com-josedigio/madeindigio/remembrances-mcp/compare/v0.50.3...v1.0.0) (2025-11-18)

### Features

* add osx scripts
([cfd7e1a](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/cfd7e1a72b8651ad49762f73384f23f8b51452ff))
* working compilation in osx with shared libraries
([852a786](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/852a786a1dc53092e3bc0c5e40d7ecf042d31860))
* enable cross compilation for linux
([f7b8fbf](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/f7b8fbfd098afee785604e08616f421d592a3389))
* add surrealdb-embedded library
([9251916](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/92519163f45069390a2eed7874facc094c7d1036))
* Add more agents configuration and fix config
([c4740d8](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/c4740d8e1f4303e93e87713a43690be902a79e7a))

### Fixes

* Error nbatch gguf embedded error
([0573904](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/057390427cb7ceb2618c2fab5688771404a8f328))

### [v0.50.3](https://github.com-josedigio/madeindigio/remembrances-mcp/compare/v0.40.0...v0.50.3) (2025-10-27)

#### Features

* refactor surrealdb storage code
([a1af4cb](https://github.com-josedigio/madeindigio/remembrances-mcp/commit/a1af4cb419c858243b712656ad8fe7d245ea4411))

#### Fixes

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
