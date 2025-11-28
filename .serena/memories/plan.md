# Plan de Implementación: Sistema de Indexación de Código con Tree-sitter

## Resumen Ejecutivo

Implementar un sistema completo de indexación y búsqueda de código fuente que:
- Utilice tree-sitter para parsear múltiples lenguajes de programación
- Indexe proyectos de código en SurrealDB con embeddings semánticos
- Proporcione herramientas MCP para búsqueda y manipulación de código
- Soporte indexación asíncrona en background

## Contexto Técnico Actual

### Arquitectura Existente
- **Base de datos**: SurrealDB (embedded y externa) ya configurada
- **Embeddings**: Sistema GGUF local con GPU (nomic-embed-text-v1.5)
- **MCP Server**: Framework establecido con múltiples herramientas
- **Lenguaje**: Go 1.23.4
- **Estructura**: `internal/` (storage, memory, kb, config), `pkg/` (embedder, mcp_tools)

### Componentes Relevantes
- `pkg/embedder/`: Factory de embeddings (GGUF, Ollama, OpenAI)
- `internal/storage/`: Capa de almacenamiento SurrealDB
- `pkg/mcp_tools/`: Herramientas MCP existentes
- Sistema de migraciones en `internal/storage/migrations/`

## Lenguajes a Soportar

### Parsers Tree-sitter Confirmados
1. **PHP** - tree-sitter-php
2. **TypeScript** - tree-sitter-typescript
3. **JavaScript** - tree-sitter-javascript
4. **Golang** - tree-sitter-go
5. **Rust** - tree-sitter-rust
6. **Java** - tree-sitter-java
7. **Kotlin** - tree-sitter-kotlin
8. **Swift** - tree-sitter-swift
9. **Objective-C** - tree-sitter-objc

### Binding Go para Tree-sitter
- **Opción 1**: `github.com/tree-sitter/go-tree-sitter` (oficial, grammars separadas)
- **Opción 2**: `github.com/smacker/go-tree-sitter` (incluye grammars)
- **Recomendación**: Usar oficial + grammars individuales para control granular

---

## FASE 1: Infraestructura Base de Tree-sitter

### 1.1 Integración de Tree-sitter en Go
**Duración estimada**: Base técnica
**Archivos a crear**:
- `pkg/treesitter/parser.go` - Parser principal
- `pkg/treesitter/languages.go` - Registro de lenguajes
- `pkg/treesitter/ast_walker.go` - Navegación del AST
- `pkg/treesitter/types.go` - Tipos comunes

**Tareas**:
- [ ] Añadir dependencias tree-sitter a `go.mod`
  ```go
  github.com/tree-sitter/go-tree-sitter
  github.com/tree-sitter/tree-sitter-php
  github.com/tree-sitter/tree-sitter-typescript
  github.com/tree-sitter/tree-sitter-javascript
  github.com/tree-sitter/tree-sitter-go
  github.com/tree-sitter/tree-sitter-rust
  github.com/tree-sitter/tree-sitter-java
  github.com/tree-sitter/tree-sitter-kotlin
  github.com/tree-sitter/tree-sitter-swift
  github.com/tree-sitter/tree-sitter-objc
  ```
- [ ] Crear factory de parsers por extensión de archivo
- [ ] Implementar walker para extraer símbolos (clases, métodos, funciones)
- [ ] Definir estructura `CodeSymbol` con metadata relevante

**Estructura `CodeSymbol`**:
```go
type CodeSymbol struct {
    ID           string    // UUID único
    ProjectID    string    // Identificador del proyecto
    FilePath     string    // Ruta relativa en el proyecto
    Language     string    // Lenguaje de programación
    SymbolType   string    // class, method, function, interface, etc.
    Name         string    // Nombre del símbolo
    NamePath     string    // Ruta jerárquica (ej: "MyClass/myMethod")
    StartLine    int       // Línea de inicio
    EndLine      int       // Línea de fin
    StartByte    int       // Byte de inicio
    EndByte      int       // Byte de fin
    SourceCode   string    // Código fuente del símbolo
    Signature    string    // Firma (para métodos/funciones)
    DocString    string    // Documentación extraída
    Embedding    []float32 // Vector embedding del código
    ParentID     *string   // ID del símbolo padre (si existe)
    Metadata     map[string]interface{} // Metadata adicional
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

### 1.2 Extracción de Símbolos del AST
**Archivos a crear**:
- `pkg/treesitter/extractors/base.go` - Interfaz base
- `pkg/treesitter/extractors/go_extractor.go`
- `pkg/treesitter/extractors/typescript_extractor.go`
- `pkg/treesitter/extractors/php_extractor.go`
- ... (uno por lenguaje)

**Tareas**:
- [ ] Definir interfaz `SymbolExtractor`
- [ ] Implementar extractor específico por lenguaje
- [ ] Extraer información contextual (imports, comentarios, etc.)
- [ ] Manejar símbolos anidados (métodos dentro de clases)
- [ ] Normalizar nombres de símbolos entre lenguajes

**Tipos de símbolos a extraer**:
- Clases/Estructuras
- Interfaces/Traits
- Métodos/Funciones
- Variables globales/Constantes
- Enums
- Type aliases

---

## FASE 2: Esquema de Base de Datos

### 2.1 Tablas SurrealDB para Code Indexing
**Archivo**: `internal/storage/surrealdb_code_schema.go`

**Tablas a crear**:

```sql
-- Tabla de proyectos
DEFINE TABLE code_projects SCHEMAFULL;
DEFINE FIELD project_id ON code_projects TYPE string ASSERT $value != NONE;
DEFINE FIELD name ON code_projects TYPE string ASSERT $value != NONE;
DEFINE FIELD root_path ON code_projects TYPE string ASSERT $value != NONE;
DEFINE FIELD language_stats ON code_projects TYPE object;
DEFINE FIELD last_indexed_at ON code_projects TYPE datetime;
DEFINE FIELD indexing_status ON code_projects TYPE string; -- "pending", "in_progress", "completed", "failed"
DEFINE FIELD created_at ON code_projects TYPE datetime DEFAULT time::now();
DEFINE FIELD updated_at ON code_projects TYPE datetime DEFAULT time::now();
DEFINE INDEX idx_project_id ON code_projects COLUMNS project_id UNIQUE;

-- Tabla de símbolos de código
DEFINE TABLE code_symbols SCHEMAFULL;
DEFINE FIELD id ON code_symbols TYPE string ASSERT $value != NONE;
DEFINE FIELD project_id ON code_symbols TYPE string ASSERT $value != NONE;
DEFINE FIELD file_path ON code_symbols TYPE string ASSERT $value != NONE;
DEFINE FIELD language ON code_symbols TYPE string ASSERT $value != NONE;
DEFINE FIELD symbol_type ON code_symbols TYPE string ASSERT $value != NONE;
DEFINE FIELD name ON code_symbols TYPE string ASSERT $value != NONE;
DEFINE FIELD name_path ON code_symbols TYPE string ASSERT $value != NONE;
DEFINE FIELD start_line ON code_symbols TYPE int;
DEFINE FIELD end_line ON code_symbols TYPE int;
DEFINE FIELD start_byte ON code_symbols TYPE int;
DEFINE FIELD end_byte ON code_symbols TYPE int;
DEFINE FIELD source_code ON code_symbols TYPE string;
DEFINE FIELD signature ON code_symbols TYPE string;
DEFINE FIELD doc_string ON code_symbols TYPE string;
DEFINE FIELD embedding ON code_symbols TYPE array;
DEFINE FIELD parent_id ON code_symbols TYPE option<string>;
DEFINE FIELD metadata ON code_symbols TYPE object;
DEFINE FIELD created_at ON code_symbols TYPE datetime DEFAULT time::now();
DEFINE FIELD updated_at ON code_symbols TYPE datetime DEFAULT time::now();

-- Índices
DEFINE INDEX idx_code_project ON code_symbols COLUMNS project_id;
DEFINE INDEX idx_code_file ON code_symbols COLUMNS file_path;
DEFINE INDEX idx_code_name ON code_symbols COLUMNS name;
DEFINE INDEX idx_code_namepath ON code_symbols COLUMNS name_path;
DEFINE INDEX idx_code_type ON code_symbols COLUMNS symbol_type;
DEFINE INDEX idx_code_language ON code_symbols COLUMNS language;

-- Índice vectorial HNSW para búsqueda semántica
DEFINE INDEX idx_code_embedding ON code_symbols FIELDS embedding 
  HNSW DIMENSION 768 DIST COSINE TYPE F32;

-- Tabla de archivos indexados (para tracking)
DEFINE TABLE code_files SCHEMAFULL;
DEFINE FIELD project_id ON code_files TYPE string ASSERT $value != NONE;
DEFINE FIELD file_path ON code_files TYPE string ASSERT $value != NONE;
DEFINE FIELD language ON code_files TYPE string;
DEFINE FIELD file_hash ON code_files TYPE string; -- Para detectar cambios
DEFINE FIELD symbols_count ON code_files TYPE int DEFAULT 0;
DEFINE FIELD indexed_at ON code_files TYPE datetime DEFAULT time::now();
DEFINE INDEX idx_file_project ON code_files COLUMNS project_id, file_path UNIQUE;
```

### 2.2 Migration para Code Schema
**Archivo**: `internal/storage/migrations/v9_code_indexing.go`

**Tareas**:
- [ ] Crear migration que defina las tablas arriba
- [ ] Implementar rollback si es necesario
- [ ] Actualizar `migration.go` para incluir v9

---

## FASE 3: Sistema de Indexación

### 3.1 Servicio de Indexación
**Archivos a crear**:
- `internal/indexer/indexer.go` - Lógica principal
- `internal/indexer/file_scanner.go` - Escaneo de archivos
- `internal/indexer/job_manager.go` - Gestión de jobs async
- `internal/indexer/types.go`

**Componentes**:

```go
type Indexer struct {
    storage    storage.Storage
    embedder   embedder.Embedder
    parser     *treesitter.Parser
    jobManager *JobManager
}

type IndexingJob struct {
    ID          string
    ProjectID   string
    ProjectPath string
    Status      string // "pending", "running", "completed", "failed"
    Progress    float64
    FilesTotal  int
    FilesIndexed int
    StartedAt   time.Time
    CompletedAt *time.Time
    Error       *string
}
```

**Tareas**:
- [ ] Implementar escaneo recursivo de directorios
- [ ] Detectar lenguaje por extensión de archivo
- [ ] Parsear archivo con tree-sitter
- [ ] Extraer símbolos del AST
- [ ] Generar embeddings para cada símbolo
- [ ] Almacenar en SurrealDB
- [ ] Manejar errores y logging
- [ ] Implementar sistema de resume (re-indexar solo archivos modificados)

### 3.2 Gestión de Jobs Asíncronos
**Archivo**: `internal/indexer/job_manager.go`

**Tareas**:
- [ ] Implementar cola de trabajos en memoria (o usar SurrealDB)
- [ ] Permitir solo 1 job activo por proyecto
- [ ] API para consultar estado de jobs
- [ ] Persistencia de estado de jobs en DB
- [ ] Cancelación de jobs en progreso

**Estados de Job**:
```go
const (
    JobStatusPending   = "pending"
    JobStatusRunning   = "running"
    JobStatusCompleted = "completed"
    JobStatusFailed    = "failed"
    JobStatusCancelled = "cancelled"
)
```

### 3.3 CLI para Indexación
**Archivo**: `cmd/remembrances-mcp/main.go` (extender)

**Nuevos flags**:
```bash
--index-project <project_id>    # ID del proyecto a indexar
--project-path <path>            # Path al código fuente
--index-wait                     # Esperar a que termine la indexación
```

**Ejemplo de uso**:
```bash
./remembrances-mcp \
  --index-project "my-backend" \
  --project-path "/path/to/code" \
  --index-wait
```

**Tareas**:
- [ ] Añadir flags a config
- [ ] Implementar comando de indexación
- [ ] Modo síncrono vs asíncrono
- [ ] Reportar progreso a stdout

---

## FASE 4: Herramientas MCP para Indexación

### 4.1 Tool: `code_index_project`
**Archivo**: `pkg/mcp_tools/code_indexing_tools.go`

**Signature**:
```go
type IndexProjectArgs struct {
    ProjectID   string `json:"project_id" jsonschema:"required,description=Unique identifier for the project"`
    ProjectPath string `json:"project_path" jsonschema:"required,description=Absolute path to the project root"`
    ProjectName string `json:"project_name,omitempty" jsonschema:"description=Human-readable project name"`
    ForceReindex bool  `json:"force_reindex,omitempty" jsonschema:"description=Force re-indexing even if already indexed"`
}
```

**Comportamiento**:
- Lanza indexación en background
- Si ya existe job activo para el proyecto, retorna estado actual
- No bloquea la respuesta MCP
- Retorna ID del job y estado inicial

**Tareas**:
- [ ] Implementar tool handler
- [ ] Validar parámetros (project_id, path)
- [ ] Verificar si ya hay job activo
- [ ] Lanzar goroutine de indexación
- [ ] Retornar estado inmediatamente

### 4.2 Tool: `code_index_status`
**Signature**:
```go
type IndexStatusArgs struct {
    ProjectID string `json:"project_id" jsonschema:"required,description=Project identifier"`
}
```

**Comportamiento**:
- Consulta estado del job de indexación
- Retorna progreso, archivos procesados, errores
- Indica si está completado, en progreso, o fallido

**Respuesta**:
```json
{
  "status": "running",
  "progress": 45.5,
  "files_total": 1200,
  "files_indexed": 546,
  "started_at": "2025-11-28T10:00:00Z",
  "completed_at": null,
  "error": null
}
```

**Tareas**:
- [ ] Implementar consulta a job manager
- [ ] Formatear respuesta detallada
- [ ] Manejar proyecto no encontrado

---

## FASE 5: Herramientas MCP de Búsqueda (estilo Serena)

### 5.1 Tool: `code_get_symbols_overview`
**Inspirado en**: `GetSymbolsOverviewTool` de Serena

**Signature**:
```go
type GetSymbolsOverviewArgs struct {
    ProjectID    string `json:"project_id" jsonschema:"required"`
    RelativePath string `json:"relative_path" jsonschema:"required,description=Relative path to the file"`
    MaxResults   int    `json:"max_results,omitempty" jsonschema:"description=Limit number of symbols returned"`
}
```

**Comportamiento**:
- Lista símbolos top-level de un archivo
- Retorna nombre, tipo, líneas, firma
- No incluye cuerpo del código (solo metadata)

**Respuesta**:
```json
{
  "file_path": "src/services/user.ts",
  "language": "typescript",
  "symbols": [
    {
      "name": "UserService",
      "type": "class",
      "name_path": "/UserService",
      "start_line": 10,
      "end_line": 150,
      "signature": "class UserService"
    },
    {
      "name": "createUser",
      "type": "method",
      "name_path": "/UserService/createUser",
      "start_line": 25,
      "end_line": 45,
      "signature": "async createUser(data: CreateUserDTO): Promise<User>"
    }
  ]
}
```

**Tareas**:
- [ ] Query a `code_symbols` filtrado por proyecto y archivo
- [ ] Ordenar por línea de inicio
- [ ] Formatear respuesta JSON
- [ ] Limitar resultados si `max_results` está presente

### 5.2 Tool: `code_find_symbol`
**Inspirado en**: `FindSymbolTool` de Serena

**Signature**:
```go
type FindSymbolArgs struct {
    ProjectID        string   `json:"project_id" jsonschema:"required"`
    NamePathPattern  string   `json:"name_path_pattern" jsonschema:"required,description=Symbol name or path pattern"`
    RelativePath     string   `json:"relative_path,omitempty" jsonschema:"description=Restrict search to file/dir"`
    Depth            int      `json:"depth,omitempty" jsonschema:"description=Include children up to depth level"`
    IncludeBody      bool     `json:"include_body,omitempty" jsonschema:"description=Include source code in results"`
    IncludeKinds     []string `json:"include_kinds,omitempty" jsonschema:"description=Filter by symbol types"`
    ExcludeKinds     []string `json:"exclude_kinds,omitempty" jsonschema:"description=Exclude symbol types"`
    SubstringMatch   bool     `json:"substring_matching,omitempty" jsonschema:"description=Enable partial name matching"`
}
```

**Comportamiento**:
- Búsqueda flexible por nombre o name_path
- Soporta wildcards o substring matching
- Puede incluir descendientes (depth > 0)
- Filtra por tipo de símbolo

**Lógica de búsqueda**:
- `/ClassName/methodName` → Búsqueda exacta de path
- `ClassName/methodName` → Búsqueda por sufijo
- `methodName` → Búsqueda por nombre simple
- Con `substring_matching=true` → `get` matchea `getUserData`

**Tareas**:
- [ ] Implementar lógica de pattern matching
- [ ] Query a SurrealDB con filtros
- [ ] Cargar hijos recursivamente si `depth > 0`
- [ ] Incluir `source_code` solo si `include_body=true`
- [ ] Aplicar filtros de tipo

### 5.3 Tool: `code_search_symbols_semantic`
**Nueva herramienta de búsqueda semántica**

**Signature**:
```go
type SearchSymbolsSemanticArgs struct {
    ProjectID  string   `json:"project_id" jsonschema:"required"`
    Query      string   `json:"query" jsonschema:"required,description=Natural language query"`
    Limit      int      `json:"limit,omitempty" jsonschema:"description=Max results to return"`
    Languages  []string `json:"languages,omitempty" jsonschema:"description=Filter by languages"`
    SymbolTypes []string `json:"symbol_types,omitempty" jsonschema:"description=Filter by symbol types"`
}
```

**Comportamiento**:
- Genera embedding del query
- Búsqueda vectorial en `code_symbols` usando índice HNSW
- Retorna símbolos ordenados por similitud

**Query SurrealDB**:
```sql
SELECT *, vector::similarity::cosine(embedding, $query_embedding) AS score
FROM code_symbols
WHERE project_id = $project_id
  AND language IN $languages
  AND symbol_type IN $symbol_types
ORDER BY score DESC
LIMIT $limit
```

**Tareas**:
- [ ] Generar embedding del query con embedder
- [ ] Ejecutar búsqueda vectorial
- [ ] Formatear resultados con score de similitud
- [ ] Incluir contexto (archivo, líneas)

### 5.4 Tool: `code_search_pattern`
**Búsqueda por contenido de código (regex/substring)**

**Signature**:
```go
type SearchPatternArgs struct {
    ProjectID    string   `json:"project_id" jsonschema:"required"`
    Pattern      string   `json:"pattern" jsonschema:"required,description=Regex or substring to search"`
    IsRegex      bool     `json:"is_regex,omitempty" jsonschema:"description=Treat pattern as regex"`
    Languages    []string `json:"languages,omitempty"`
    SymbolTypes  []string `json:"symbol_types,omitempty"`
    CaseSensitive bool    `json:"case_sensitive,omitempty"`
}
```

**Comportamiento**:
- Búsqueda por contenido en `source_code`
- Soporta regex o substring simple
- Retorna matches con contexto

**Tareas**:
- [ ] Query a DB filtrando por pattern
- [ ] Aplicar filtros de lenguaje/tipo
- [ ] Retornar líneas coincidentes
- [ ] Implementar case-insensitive si aplica

### 5.5 Tool: `code_find_references`
**Inspirado en**: `FindReferencingSymbolsTool` de Serena

**Signature**:
```go
type FindReferencesArgs struct {
    ProjectID    string `json:"project_id" jsonschema:"required"`
    SymbolID     string `json:"symbol_id,omitempty" jsonschema:"description=ID of the symbol to find references for"`
    SymbolName   string `json:"symbol_name,omitempty" jsonschema:"description=Name of the symbol (alternative)"`
    IncludeKinds []string `json:"include_kinds,omitempty"`
}
```

**Comportamiento**:
- Busca referencias del símbolo en todo el proyecto
- Analiza `source_code` de otros símbolos
- Retorna ubicaciones donde se usa el símbolo

**Nota**: Esta es una búsqueda simple basada en nombre. Para referencias exactas, se requeriría un LSP (Language Server Protocol), que está fuera del scope inicial.

**Tareas**:
- [ ] Buscar símbolo target
- [ ] Buscar en `source_code` de otros símbolos
- [ ] Retornar matches con contexto
- [ ] Filtrar por tipos si especificado

---

## FASE 6: Herramientas MCP de Manipulación de Código

### 6.1 Tool: `code_replace_symbol`
**Inspirado en**: `ReplaceSymbolBodyTool` de Serena

**Signature**:
```go
type ReplaceSymbolArgs struct {
    ProjectID    string `json:"project_id" jsonschema:"required"`
    SymbolID     string `json:"symbol_id,omitempty" jsonschema:"description=ID of symbol to replace"`
    NamePath     string `json:"name_path,omitempty" jsonschema:"description=Name path of symbol (alternative)"`
    RelativePath string `json:"relative_path,omitempty" jsonschema:"description=File path (if using name_path)"`
    NewBody      string `json:"new_body" jsonschema:"required,description=New source code for the symbol"`
}
```

**Comportamiento**:
- Reemplaza el código del símbolo en el archivo físico
- Actualiza la BD con nuevo código
- Re-genera embedding del nuevo código

**Flujo**:
1. Localizar símbolo (por ID o name_path)
2. Leer archivo original
3. Reemplazar sección específica (usando start_byte, end_byte)
4. Escribir archivo modificado
5. Re-parsear símbolo modificado
6. Actualizar en BD con nuevo embedding

**Tareas**:
- [ ] Implementar lectura/escritura de archivos
- [ ] Reemplazo preciso por byte offsets
- [ ] Re-parsing del símbolo modificado
- [ ] Actualización en BD
- [ ] Manejo de errores (archivo bloqueado, permisos, etc.)

### 6.2 Tool: `code_insert_after_symbol`
**Inspirado en**: `InsertAfterSymbolTool` de Serena

**Signature**:
```go
type InsertAfterSymbolArgs struct {
    ProjectID    string `json:"project_id" jsonschema:"required"`
    SymbolID     string `json:"symbol_id,omitempty"`
    NamePath     string `json:"name_path,omitempty"`
    RelativePath string `json:"relative_path,omitempty"`
    Body         string `json:"body" jsonschema:"required,description=Code to insert"`
}
```

**Comportamiento**:
- Inserta código después del final del símbolo
- Útil para añadir métodos a clases, funciones a módulos

**Tareas**:
- [ ] Localizar símbolo
- [ ] Insertar en posición `end_byte + 1`
- [ ] Re-parsear archivo completo
- [ ] Actualizar todos los símbolos afectados en BD

### 6.3 Tool: `code_insert_before_symbol`
**Signature**: Similar a `code_insert_after_symbol`

**Comportamiento**:
- Inserta código antes del inicio del símbolo
- Útil para imports, comentarios, etc.

### 6.4 Tool: `code_delete_symbol`
**Signature**:
```go
type DeleteSymbolArgs struct {
    ProjectID    string `json:"project_id" jsonschema:"required"`
    SymbolID     string `json:"symbol_id,omitempty"`
    NamePath     string `json:"name_path,omitempty"`
    RelativePath string `json:"relative_path,omitempty"`
}
```

**Comportamiento**:
- Elimina símbolo del archivo físico
- Elimina de la BD
- Re-parsea archivo

**Tareas**:
- [ ] Localizar símbolo
- [ ] Eliminar sección del archivo (start_byte a end_byte)
- [ ] Eliminar de BD
- [ ] Re-parsear y actualizar símbolos afectados

---

## FASE 7: Herramientas de Navegación y Utilidades

### 7.1 Tool: `code_list_projects`
**Signature**:
```go
type ListProjectsArgs struct {
    // Sin argumentos
}
```

**Comportamiento**:
- Lista todos los proyectos indexados
- Retorna metadata (nombre, lenguajes, fecha de indexación)

### 7.2 Tool: `code_get_project_stats`
**Signature**:
```go
type GetProjectStatsArgs struct {
    ProjectID string `json:"project_id" jsonschema:"required"`
}
```

**Respuesta**:
```json
{
  "project_id": "my-backend",
  "name": "My Backend API",
  "total_files": 1200,
  "total_symbols": 15000,
  "language_breakdown": {
    "typescript": 8000,
    "javascript": 3000,
    "go": 4000
  },
  "symbol_type_breakdown": {
    "class": 500,
    "method": 6000,
    "function": 8000,
    "interface": 500
  },
  "last_indexed_at": "2025-11-28T10:30:00Z",
  "indexing_status": "completed"
}
```

### 7.3 Tool: `code_get_file_symbols`
**Listar símbolos de un archivo específico con jerarquía**

**Signature**:
```go
type GetFileSymbolsArgs struct {
    ProjectID    string `json:"project_id" jsonschema:"required"`
    RelativePath string `json:"relative_path" jsonschema:"required"`
    IncludeBody  bool   `json:"include_body,omitempty"`
}
```

**Comportamiento**:
- Similar a `code_get_symbols_overview` pero con jerarquía completa
- Retorna árbol de símbolos padres/hijos

---

## FASE 8: Optimizaciones y Mejoras

### 8.1 Chunking Inteligente
**Problema**: Símbolos muy grandes (clases con muchos métodos)

**Solución**:
- Chunking del código fuente similar a `pkg/embedder/chunking.go`
- Embeddings separados para símbolo completo + chunks
- Tabla adicional `code_chunks`

**Tareas**:
- [ ] Implementar chunking para símbolos > 1500 caracteres
- [ ] Almacenar chunks con referencia al símbolo padre
- [ ] Búsqueda híbrida (símbolos completos + chunks)

### 8.2 Incremental Indexing
**Problema**: Re-indexar proyecto completo es costoso

**Solución**:
- Detectar archivos modificados (hash SHA-256)
- Re-indexar solo archivos cambiados
- Tabla `code_files` para tracking

**Tareas**:
- [ ] Calcular hash de archivos durante indexación
- [ ] Comparar hashes en re-indexación
- [ ] Eliminar símbolos obsoletos
- [ ] Añadir símbolos nuevos/modificados

### 8.3 Búsqueda Híbrida Avanzada
**Combinar búsqueda vectorial + filtros estructurales**

**Ejemplo**:
- Query semántica: "autenticación de usuarios"
- Filtros: lenguaje=typescript, tipo=method
- Scope: archivos en `src/auth/`

**Tareas**:
- [ ] Implementar `code_hybrid_search` tool
- [ ] Combinar cosine similarity + filtros SQL
- [ ] Ranking ponderado (similitud + exactitud de filtros)

### 8.4 Caching y Performance
**Optimizaciones**:
- Cache de parsers tree-sitter por lenguaje
- Batch inserts a SurrealDB (100 símbolos a la vez)
- Pool de workers para indexación paralela

**Tareas**:
- [ ] Implementar cache de parsers
- [ ] Batch inserts en `internal/storage/surrealdb_code.go`
- [ ] Worker pool con `sync.WaitGroup`

---

## FASE 9: Testing y Documentación

### 9.1 Tests Unitarios
**Archivos**:
- `pkg/treesitter/parser_test.go`
- `pkg/treesitter/extractors/*_test.go`
- `internal/indexer/indexer_test.go`
- `pkg/mcp_tools/code_*_tools_test.go`

**Casos de prueba**:
- [ ] Parsing de cada lenguaje soportado
- [ ] Extracción correcta de símbolos
- [ ] Búsqueda por name_path
- [ ] Búsqueda semántica
- [ ] Reemplazo de código
- [ ] Jobs asíncronos

### 9.2 Tests de Integración
**Escenarios**:
- [ ] Indexar proyecto de prueba (pequeño repo)
- [ ] Búsqueda end-to-end
- [ ] Modificación de código y re-indexación
- [ ] Consulta de estado de jobs

### 9.3 Documentación
**Archivos a crear**:
- `docs/CODE_INDEXING.md` - Guía completa
- `docs/CODE_INDEXING_API.md` - Referencia de herramientas MCP
- `docs/TREE_SITTER_LANGUAGES.md` - Lenguajes soportados
- `examples/code_indexing_example.go` - Ejemplo programático

**Contenido**:
- [ ] Guía de inicio rápido
- [ ] Ejemplos de uso por herramienta
- [ ] Configuración recomendada
- [ ] Troubleshooting

---

## FASE 10: Integración y Despliegue

### 10.1 Actualización del README
**Tareas**:
- [ ] Añadir sección "Code Indexing"
- [ ] Ejemplos de CLI para indexación
- [ ] Ejemplos de herramientas MCP
- [ ] Requisitos adicionales (tree-sitter grammars)

### 10.2 Configuración YAML
**Actualizar `config.sample.yaml`**:
```yaml
# ========== Code Indexing Configuration ==========
# Enable code indexing features (default: false)
code-indexing-enabled: true

# Default indexing options
indexing:
  # Number of parallel workers for indexing (default: 4)
  workers: 4
  
  # Batch size for database inserts (default: 100)
  batch-size: 100
  
  # Supported file extensions per language
  extensions:
    go: [".go"]
    typescript: [".ts", ".tsx"]
    javascript: [".js", ".jsx"]
    php: [".php"]
    rust: [".rs"]
    java: [".java"]
    kotlin: [".kt", ".kts"]
    swift: [".swift"]
    objc: [".m", ".mm", ".h"]
```

### 10.3 Makefile
**Añadir comandos**:
```makefile
# Install tree-sitter grammars
install-grammars:
	go get github.com/tree-sitter/tree-sitter-php@latest
	go get github.com/tree-sitter/tree-sitter-typescript@latest
	# ... resto de grammars

# Build with code indexing support
build-code-indexing:
	go build -tags code_indexing -o build/remembrances-mcp ./cmd/remembrances-mcp
```

---

## FASE 11: Features Avanzadas (Futuro)

### 11.1 Code Understanding Tools
- `code_explain_symbol` - Explicar qué hace un símbolo
- `code_generate_tests` - Generar tests para símbolo
- `code_suggest_refactoring` - Sugerir mejoras

### 11.2 Cross-Project Search
- Buscar símbolos en múltiples proyectos
- Encontrar patrones comunes
- Análisis de dependencias entre proyectos

### 11.3 LSP Integration
- Integración con Language Servers
- Referencias exactas (no solo basadas en nombre)
- Autocompletado basado en contexto

### 11.4 Code Metrics
- Complejidad ciclomática
- Líneas de código por símbolo
- Métricas de duplicación

---

## Dependencias y Requisitos

### Dependencias Go (añadir a go.mod)
```
github.com/tree-sitter/go-tree-sitter v0.23.4
github.com/tree-sitter/tree-sitter-php v0.23.10
github.com/tree-sitter/tree-sitter-typescript v0.23.2
github.com/tree-sitter/tree-sitter-javascript v0.23.0
github.com/tree-sitter/tree-sitter-go v0.23.4
github.com/tree-sitter/tree-sitter-rust v0.23.2
github.com/tree-sitter/tree-sitter-java v0.23.4
github.com/tree-sitter/tree-sitter-kotlin v0.3.9
github.com/tree-sitter/tree-sitter-swift v0.6.0
github.com/tree-sitter/tree-sitter-objc v0.1.0
```

### Requisitos de Sistema
- SurrealDB con soporte vectorial (ya disponible)
- Modelo GGUF de embeddings (ya configurado: nomic-embed-text-v1.5)
- Espacio en disco para índices (dependiente del tamaño de proyectos)

### Compatibilidad
- Go 1.23.4+
- SurrealDB 2.0+
- GPU recomendada para embeddings rápidos (ya configurado CUDA)

---

## Estimación de Esfuerzo

### Por Fase (orden de magnitud)
- **FASE 1**: Infraestructura Tree-sitter - **Alta complejidad**
- **FASE 2**: Schema BD - **Media complejidad**
- **FASE 3**: Sistema Indexación - **Alta complejidad**
- **FASE 4**: MCP Tools Indexación - **Media complejidad**
- **FASE 5**: MCP Tools Búsqueda - **Alta complejidad**
- **FASE 6**: MCP Tools Manipulación - **Alta complejidad**
- **FASE 7**: Navegación - **Baja complejidad**
- **FASE 8**: Optimizaciones - **Media complejidad**
- **FASE 9**: Testing - **Media complejidad**
- **FASE 10**: Integración - **Baja complejidad**

### Priorización Recomendada
1. **MVP Core**: FASE 1, 2, 3, 4 → Sistema básico de indexación
2. **MVP Search**: FASE 5 → Búsqueda básica
3. **Full Feature**: FASE 6, 7 → Manipulación y navegación
4. **Production Ready**: FASE 8, 9, 10 → Optimización y despliegue

---

## Riesgos y Mitigaciones

### Riesgo 1: Performance en proyectos grandes
**Mitigación**: 
- Indexación incremental
- Workers paralelos
- Batch processing

### Riesgo 2: Exactitud de parsers tree-sitter
**Mitigación**:
- Testing exhaustivo por lenguaje
- Manejo de errores de parsing
- Logging detallado

### Riesgo 3: Consistencia código-BD
**Mitigación**:
- Hashes de archivos para detectar cambios
- Re-indexación automática
- Validación de integridad

### Riesgo 4: Embeddings de símbolos grandes
**Mitigación**:
- Chunking inteligente
- Embeddings por secciones
- Agregación de resultados

---

## Referencias Técnicas

### Tree-sitter
- [Tree-sitter GitHub](https://github.com/tree-sitter/tree-sitter)
- [Go Tree-sitter Bindings](https://github.com/tree-sitter/go-tree-sitter)
- [Tree-sitter Documentation](https://tree-sitter.github.io/tree-sitter/)

### SurrealDB
- [Vector Search Guide](https://surrealdb.com/docs/surrealdb/reference-guide/vector-search)
- [HNSW Indexing](https://surrealdb.com/docs/surrealdb/models/vector)

### Serena MCP
- [Serena GitHub](https://github.com/oraios/serena)
- [Symbol Tools Reference](https://github.com/oraios/serena/tree/main/src/serena/tools)

### Code Parsing
- [AST Parsing with Tree-sitter](https://www.dropstone.io/blog/ast-parsing-tree-sitter-40-languages)
- [cAST: Code Chunking](https://arxiv.org/html/2506.15655v1)

---

## Notas de Implementación

### Convenciones de Nombres
- Tablas: `code_*` (ej: `code_symbols`, `code_projects`)
- Tipos Go: `Code*` (ej: `CodeSymbol`, `IndexingJob`)
- MCP Tools: `code_*` (ej: `code_find_symbol`, `code_replace_symbol`)

### Estructura de Archivos
```
remembrances-mcp/
├── pkg/
│   ├── treesitter/          # Nuevo package
│   │   ├── parser.go
│   │   ├── languages.go
│   │   ├── ast_walker.go
│   │   ├── types.go
│   │   └── extractors/      # Extractors por lenguaje
│   │       ├── base.go
│   │       ├── go_extractor.go
│   │       ├── typescript_extractor.go
│   │       └── ...
│   └── mcp_tools/
│       ├── code_indexing_tools.go     # Nuevo
│       ├── code_search_tools.go       # Nuevo
│       └── code_manipulation_tools.go # Nuevo
├── internal/
│   ├── indexer/             # Nuevo package
│   │   ├── indexer.go
│   │   ├── file_scanner.go
│   │   ├── job_manager.go
│   │   └── types.go
│   └── storage/
│       ├── surrealdb_code_schema.go   # Nuevo
│       ├── surrealdb_code_symbols.go  # Nuevo
│       ├── surrealdb_code_projects.go # Nuevo
│       └── migrations/
│           └── v9_code_indexing.go    # Nuevo
└── docs/
    ├── CODE_INDEXING.md               # Nuevo
    └── CODE_INDEXING_API.md           # Nuevo
```

### Logging y Monitoreo
- Log de cada archivo parseado
- Métricas de indexación (archivos/segundo)
- Errores de parsing por archivo
- Tiempo de generación de embeddings

---

## Conclusión

Este plan proporciona una hoja de ruta completa para implementar un sistema robusto de indexación y búsqueda de código fuente en Remembrances-MCP. El sistema será:

✅ **Extensible**: Fácil añadir nuevos lenguajes  
✅ **Escalable**: Indexación asíncrona y paralela  
✅ **Potente**: Búsqueda semántica + estructural  
✅ **Compatible**: Herramientas MCP estilo Serena  
✅ **Mantenible**: Código bien estructurado y testeado  

El MVP (FASE 1-4) proporciona funcionalidad básica de indexación y consulta. Las fases posteriores añaden búsqueda avanzada, manipulación de código y optimizaciones de producción.

---

**Siguiente paso**: Revisión y aprobación del plan antes de comenzar implementación.
