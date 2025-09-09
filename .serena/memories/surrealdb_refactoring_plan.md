# SurrealDB Refactoring Plan

## Current situation
- `surrealdb.go` has 1287 lines
- `tools.go` has 1027 lines

## Logical separation for SurrealDB storage:

### Core file (`surrealdb.go`) - Keep:
- SurrealDBStorage struct
- NewSurrealDBStorage functions  
- Connect, Close, Ping methods
- imports and basic types

### Schema file (`surrealdb_schema.go`):
- defaultMtreeDim constant
- SchemaElement type
- InitializeSchema and all schema-related methods
- Methods: ensureSchemaVersionTable, getCurrentSchemaVersion, setSchemaVersion, runMigrations, applyMigration, applyMigrationV1
- Schema checking methods: checkSchemaElementExists, checkTableExists, checkFieldExists, checkIndexExists
- Helper methods: isAlreadyExistsError, extractTableName, extractFieldName, extractIndexName

### Facts file (`surrealdb_facts.go`):
- SaveFact, GetFact, UpdateFact, DeleteFact, ListFacts

### Vectors file (`surrealdb_vectors.go`):
- IndexVector, SearchSimilar, UpdateVector, DeleteVector
- parseVectorResults helper

### Graph file (`surrealdb_graph.go`):
- CreateEntity, CreateRelationship, TraverseGraph, GetEntity, DeleteEntity
- parseGraphResults helper

### Documents file (`surrealdb_documents.go`):
- SaveDocument, SearchDocuments, DeleteDocument, GetDocument
- parseDocumentResults helper

### Utils file (`surrealdb_utils.go`):
- HybridSearch, GetStats
- extractCount helper
- Type conversion helpers: getString, getFloat64, getMap, getTime, convertEmbeddingToFloat64

## MCP Tools refactoring:

### Main file (`tools.go`):
- ToolManager struct and NewToolManager
- RegisterTools and registration helper methods

### Types file (`types.go`):
- All input structs and constants

### Fact tools (`fact_tools.go`):
- Tool definitions and handlers for facts

### Vector tools (`vector_tools.go`):
- Tool definitions and handlers for vectors

### Graph tools (`graph_tools.go`):
- Tool definitions and handlers for graph operations

### Document tools (`document_tools.go`):
- Tool definitions and handlers for documents

### Misc tools (`misc_tools.go`):
- Hybrid search and stats tools
- Utility functions like stringMapToInterfaceMap