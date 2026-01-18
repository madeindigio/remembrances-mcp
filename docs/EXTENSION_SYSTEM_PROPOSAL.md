# Sistema de Extensiones para Remembrances-MCP

## Resumen Ejecutivo

Este documento propone un sistema de módulos/extensiones para remembrances-mcp inspirado en el modelo de Caddy + xcaddy. El objetivo es crear un ecosistema de extensiones que permita:

1. **Módulos de terceros**: Añadir funcionalidad sin modificar el core
2. **Extensiones comerciales**: Monetizar funcionalidades premium
3. **Personalización**: Modificar comportamiento por defecto
4. **Compilación personalizada**: Similar a xcaddy, compilar binarios con módulos seleccionados

---

## Análisis del Modelo Caddy/xcaddy

### Cómo Funciona Caddy

#### 1. Interfaz Module Minimalista
```go
type Module interface {
    CaddyModule() ModuleInfo
}

type ModuleInfo struct {
    ID  ModuleID           // Namespace único: "http.handlers.file_server"
    New func() Module      // Factory para crear instancias
}
```

#### 2. Registro Global en `init()`
```go
func init() {
    caddy.RegisterModule(MyModule{})
}
```

#### 3. Ciclo de Vida Bien Definido
```
New() → JSON Unmarshal → Provision(ctx) → Validate() → [Uso] → Cleanup()
```

#### 4. Interfaces Opcionales de Ciclo de Vida
- `Provisioner`: Configuración post-carga
- `Validator`: Validación de configuración
- `CleanerUpper`: Limpieza de recursos

### Cómo Funciona xcaddy

1. **Crea un módulo Go temporal** con `go mod init`
2. **Genera `main.go`** con blank imports de los plugins
3. **Ejecuta `go get`** para cada plugin
4. **Compila** con `go build`

```go
// main.go generado
package main

import (
    caddycmd "github.com/caddyserver/caddy/v2/cmd"
    _ "github.com/caddyserver/caddy/v2/modules/standard"
    _ "github.com/custom/plugin1"  // Plugin de terceros
    _ "github.com/custom/plugin2"
)

func main() {
    caddycmd.Main()
}
```

---

## Propuesta para Remembrances-MCP

### Arquitectura de Módulos

#### 1. Interfaz Base del Módulo

```go
// pkg/modules/module.go

package modules

import (
    "context"
    "github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// ModuleID identifica un módulo de forma única
// Formato: namespace.nombre (ej: "storage.postgresql", "tools.reasoning")
type ModuleID string

// ModuleInfo contiene la metadata del módulo
type ModuleInfo struct {
    ID          ModuleID
    Name        string
    Description string
    Version     string
    Author      string
    License     string  // "MIT", "Commercial", etc.
    New         func() Module
}

// Module es la interfaz base que todos los módulos deben implementar
type Module interface {
    ModuleInfo() ModuleInfo
}

// Provisioner permite configuración adicional después de cargar
type Provisioner interface {
    Provision(ctx context.Context, cfg ModuleConfig) error
}

// Validator valida la configuración del módulo
type Validator interface {
    Validate() error
}

// CleanerUpper libera recursos cuando el módulo se descarga
type CleanerUpper interface {
    Cleanup() error
}
```

#### 2. Tipos de Módulos Específicos

```go
// pkg/modules/types.go

// ToolProvider añade herramientas MCP
type ToolProvider interface {
    Module
    Tools() []ToolDefinition
}

type ToolDefinition struct {
    Tool    *protocol.Tool
    Handler func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)
}

// StorageProvider añade backends de almacenamiento
type StorageProvider interface {
    Module
    StorageType() string  // "postgresql", "redis", etc.
    NewStorage(cfg map[string]any) (storage.FullStorage, error)
}

// EmbedderProvider añade proveedores de embeddings
type EmbedderProvider interface {
    Module
    EmbedderType() string
    NewEmbedder(cfg map[string]any) (embedder.Embedder, error)
}

// Middleware intercepta y modifica requests/responses
type Middleware interface {
    Module
    Wrap(next ToolHandler) ToolHandler
}

// InstructionEnhancer modifica las instrucciones del servidor
type InstructionEnhancer interface {
    Module
    EnhanceInstructions(base string) string
}

// ConfigProvider carga configuración desde fuentes adicionales
type ConfigProvider interface {
    Module
    LoadConfig() (map[string]any, error)
}
```

#### 3. Registry Global de Módulos

```go
// pkg/modules/registry.go

package modules

import (
    "fmt"
    "sync"
)

var (
    modules   = make(map[ModuleID]ModuleInfo)
    modulesMu sync.RWMutex
)

// RegisterModule registra un módulo en el registry global
// Se llama típicamente desde init()
func RegisterModule(instance Module) {
    info := instance.ModuleInfo()
    
    if info.ID == "" {
        panic("module ID is required")
    }
    if info.New == nil {
        panic("ModuleInfo.New is required")
    }
    
    modulesMu.Lock()
    defer modulesMu.Unlock()
    
    if _, exists := modules[info.ID]; exists {
        panic(fmt.Sprintf("module already registered: %s", info.ID))
    }
    
    modules[info.ID] = info
}

// GetModule obtiene un módulo por ID
func GetModule(id ModuleID) (ModuleInfo, bool) {
    modulesMu.RLock()
    defer modulesMu.RUnlock()
    info, ok := modules[id]
    return info, ok
}

// GetModulesByNamespace obtiene todos los módulos de un namespace
func GetModulesByNamespace(namespace string) []ModuleInfo {
    modulesMu.RLock()
    defer modulesMu.RUnlock()
    
    var result []ModuleInfo
    prefix := namespace + "."
    for id, info := range modules {
        if strings.HasPrefix(string(id), prefix) || string(id) == namespace {
            result = append(result, info)
        }
    }
    return result
}

// ListModules lista todos los módulos registrados
func ListModules() []ModuleInfo {
    modulesMu.RLock()
    defer modulesMu.RUnlock()
    
    result := make([]ModuleInfo, 0, len(modules))
    for _, info := range modules {
        result = append(result, info)
    }
    return result
}
```

#### 4. Module Manager para Ciclo de Vida

```go
// pkg/modules/manager.go

package modules

import (
    "context"
    "fmt"
)

// ModuleConfig es la configuración pasada a Provision()
type ModuleConfig struct {
    Raw      map[string]any
    Storage  storage.FullStorage
    Embedder embedder.Embedder
    Logger   *slog.Logger
}

// ModuleManager gestiona el ciclo de vida de los módulos
type ModuleManager struct {
    instances map[ModuleID]Module
    config    ModuleConfig
    mu        sync.RWMutex
}

func NewModuleManager(cfg ModuleConfig) *ModuleManager {
    return &ModuleManager{
        instances: make(map[ModuleID]Module),
        config:    cfg,
    }
}

// LoadModule carga e inicializa un módulo por ID
func (mm *ModuleManager) LoadModule(ctx context.Context, id ModuleID, cfg map[string]any) (Module, error) {
    info, ok := GetModule(id)
    if !ok {
        return nil, fmt.Errorf("module not found: %s", id)
    }
    
    // Crear instancia
    instance := info.New()
    
    // Provision si implementa la interfaz
    if prov, ok := instance.(Provisioner); ok {
        modCfg := mm.config
        modCfg.Raw = cfg
        if err := prov.Provision(ctx, modCfg); err != nil {
            return nil, fmt.Errorf("provision failed for %s: %w", id, err)
        }
    }
    
    // Validate si implementa la interfaz
    if val, ok := instance.(Validator); ok {
        if err := val.Validate(); err != nil {
            // Cleanup si falla la validación
            if cu, ok := instance.(CleanerUpper); ok {
                cu.Cleanup()
            }
            return nil, fmt.Errorf("validation failed for %s: %w", id, err)
        }
    }
    
    mm.mu.Lock()
    mm.instances[id] = instance
    mm.mu.Unlock()
    
    return instance, nil
}

// UnloadModule descarga un módulo y libera sus recursos
func (mm *ModuleManager) UnloadModule(id ModuleID) error {
    mm.mu.Lock()
    instance, ok := mm.instances[id]
    if ok {
        delete(mm.instances, id)
    }
    mm.mu.Unlock()
    
    if !ok {
        return fmt.Errorf("module not loaded: %s", id)
    }
    
    if cu, ok := instance.(CleanerUpper); ok {
        return cu.Cleanup()
    }
    return nil
}

// GetToolProviders obtiene todos los módulos que proveen herramientas
func (mm *ModuleManager) GetToolProviders() []ToolProvider {
    mm.mu.RLock()
    defer mm.mu.RUnlock()
    
    var providers []ToolProvider
    for _, instance := range mm.instances {
        if tp, ok := instance.(ToolProvider); ok {
            providers = append(providers, tp)
        }
    }
    return providers
}
```

---

### Namespaces de Módulos Propuestos

| Namespace | Descripción | Ejemplos |
|-----------|-------------|----------|
| `storage.*` | Backends de almacenamiento | `storage.postgresql`, `storage.redis`, `storage.mongodb` |
| `embedder.*` | Proveedores de embeddings | `embedder.openai`, `embedder.cohere`, `embedder.local` |
| `tools.*` | Herramientas MCP adicionales | `tools.reasoning`, `tools.web_search`, `tools.calendar` |
| `middleware.*` | Interceptores de requests | `middleware.ratelimit`, `middleware.auth`, `middleware.cache` |
| `indexer.*` | Indexadores de código | `indexer.rust`, `indexer.java`, `indexer.custom` |
| `llm.*` | Integración con LLMs | `llm.anthropic`, `llm.openai`, `llm.local` |

---

### xremembrances: Builder de Binarios Personalizados

Similar a xcaddy, crear `xremembrances` para compilar binarios con módulos seleccionados:

```go
// cmd/xremembrances/main.go

package main

// Uso:
// xremembrances build \
//   --with github.com/remembrances/storage-postgresql \
//   --with github.com/remembrances/tools-reasoning@v1.2.0 \
//   --with github.com/company/commercial-module@v2.0.0

func main() {
    // Parsea argumentos
    // Crea directorio temporal
    // Genera main.go con imports
    // go get de cada módulo
    // go build
}
```

**main.go generado:**
```go
package main

import (
    remembrances "github.com/madeindigio/remembrances-mcp/cmd/remembrances-mcp"
    
    // Módulos estándar
    _ "github.com/madeindigio/remembrances-mcp/modules/standard"
    
    // Módulos de terceros
    _ "github.com/remembrances/storage-postgresql"
    _ "github.com/remembrances/tools-reasoning"
    _ "github.com/company/commercial-module"
)

func main() {
    remembrances.Main()
}
```

---

### Ejemplo de Módulo: Storage PostgreSQL

```go
// github.com/remembrances/storage-postgresql/postgresql.go

package postgresql

import (
    "context"
    "github.com/madeindigio/remembrances-mcp/pkg/modules"
    "github.com/madeindigio/remembrances-mcp/internal/storage"
)

func init() {
    modules.RegisterModule(PostgreSQLStorage{})
}

type PostgreSQLStorage struct {
    connString string
    pool       *pgxpool.Pool
}

func (PostgreSQLStorage) ModuleInfo() modules.ModuleInfo {
    return modules.ModuleInfo{
        ID:          "storage.postgresql",
        Name:        "PostgreSQL Storage",
        Description: "Use PostgreSQL as the storage backend for remembrances",
        Version:     "1.0.0",
        Author:      "Remembrances Community",
        License:     "MIT",
        New:         func() modules.Module { return new(PostgreSQLStorage) },
    }
}

func (p *PostgreSQLStorage) Provision(ctx context.Context, cfg modules.ModuleConfig) error {
    connStr, _ := cfg.Raw["connection_string"].(string)
    if connStr == "" {
        return fmt.Errorf("connection_string is required")
    }
    p.connString = connStr
    
    pool, err := pgxpool.New(ctx, connStr)
    if err != nil {
        return err
    }
    p.pool = pool
    return nil
}

func (p *PostgreSQLStorage) Validate() error {
    if p.pool == nil {
        return fmt.Errorf("database pool not initialized")
    }
    return nil
}

func (p *PostgreSQLStorage) Cleanup() error {
    if p.pool != nil {
        p.pool.Close()
    }
    return nil
}

func (p *PostgreSQLStorage) StorageType() string {
    return "postgresql"
}

func (p *PostgreSQLStorage) NewStorage(cfg map[string]any) (storage.FullStorage, error) {
    // Retorna una implementación de storage.FullStorage usando PostgreSQL
    return &pgStorage{pool: p.pool}, nil
}
```

---

### Ejemplo de Módulo: Tool de Razonamiento

```go
// github.com/remembrances/tools-reasoning/reasoning.go

package reasoning

import (
    "context"
    "github.com/madeindigio/remembrances-mcp/pkg/modules"
    "github.com/ThinkInAIXYZ/go-mcp/protocol"
)

func init() {
    modules.RegisterModule(ReasoningTools{})
}

type ReasoningTools struct {
    llmClient LLMClient
}

func (ReasoningTools) ModuleInfo() modules.ModuleInfo {
    return modules.ModuleInfo{
        ID:          "tools.reasoning",
        Name:        "Advanced Reasoning Tools",
        Description: "Adds chain-of-thought and multi-step reasoning tools",
        Version:     "1.0.0",
        Author:      "Remembrances Team",
        License:     "Commercial",
        New:         func() modules.Module { return new(ReasoningTools) },
    }
}

func (r *ReasoningTools) Tools() []modules.ToolDefinition {
    return []modules.ToolDefinition{
        {
            Tool: &protocol.Tool{
                Name:        "reason_step_by_step",
                Description: "Break down a complex problem into reasoning steps",
                InputSchema: protocol.InputSchema{
                    Type: "object",
                    Properties: map[string]protocol.PropertyDetail{
                        "problem": {Type: "string", Description: "The problem to reason about"},
                        "context": {Type: "string", Description: "Additional context"},
                    },
                    Required: []string{"problem"},
                },
            },
            Handler: r.reasonStepByStepHandler,
        },
        {
            Tool: &protocol.Tool{
                Name:        "synthesize_knowledge",
                Description: "Synthesize knowledge from multiple sources",
                InputSchema: protocol.InputSchema{
                    Type: "object",
                    Properties: map[string]protocol.PropertyDetail{
                        "query": {Type: "string", Description: "What to synthesize"},
                        "sources": {Type: "array", Description: "Source IDs to include"},
                    },
                    Required: []string{"query"},
                },
            },
            Handler: r.synthesizeKnowledgeHandler,
        },
    }
}

func (r *ReasoningTools) reasonStepByStepHandler(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
    // Implementación del razonamiento paso a paso
    // ...
}
```

---

### Ejemplo de Módulo: Middleware de Rate Limiting

```go
// github.com/remembrances/middleware-ratelimit/ratelimit.go

package ratelimit

import (
    "context"
    "github.com/madeindigio/remembrances-mcp/pkg/modules"
    "golang.org/x/time/rate"
)

func init() {
    modules.RegisterModule(RateLimitMiddleware{})
}

type RateLimitMiddleware struct {
    limiter *rate.Limiter
    rps     float64
    burst   int
}

func (RateLimitMiddleware) ModuleInfo() modules.ModuleInfo {
    return modules.ModuleInfo{
        ID:          "middleware.ratelimit",
        Name:        "Rate Limiting Middleware",
        Description: "Limits the rate of tool invocations",
        Version:     "1.0.0",
        Author:      "Remembrances Team",
        License:     "MIT",
        New:         func() modules.Module { return new(RateLimitMiddleware) },
    }
}

func (m *RateLimitMiddleware) Provision(ctx context.Context, cfg modules.ModuleConfig) error {
    m.rps = 10.0 // default
    m.burst = 20
    
    if rps, ok := cfg.Raw["requests_per_second"].(float64); ok {
        m.rps = rps
    }
    if burst, ok := cfg.Raw["burst"].(int); ok {
        m.burst = burst
    }
    
    m.limiter = rate.NewLimiter(rate.Limit(m.rps), m.burst)
    return nil
}

func (m *RateLimitMiddleware) Wrap(next modules.ToolHandler) modules.ToolHandler {
    return func(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
        if err := m.limiter.Wait(ctx); err != nil {
            return &protocol.CallToolResult{
                IsError: true,
                Content: []protocol.Content{{Type: "text", Text: "Rate limit exceeded"}},
            }, nil
        }
        return next(ctx, req)
    }
}
```

---

### Configuración de Módulos

La configuración en YAML/JSON permitiría activar y configurar módulos:

```yaml
# config.yaml

# Módulos habilitados
modules:
  storage.postgresql:
    enabled: true
    connection_string: "postgres://user:pass@localhost/remembrances"
  
  tools.reasoning:
    enabled: true
    llm_provider: "anthropic"
    api_key: "${ANTHROPIC_API_KEY}"
  
  middleware.ratelimit:
    enabled: true
    requests_per_second: 100
    burst: 200
  
  middleware.auth:
    enabled: true
    jwt_secret: "${JWT_SECRET}"
    
# Módulos deshabilitados (core que se puede desactivar)
disable:
  - tools.code_indexing  # Desactivar si no se necesita
```

---

### Casos de Uso Comerciales

1. **Módulos Enterprise**
   - `storage.enterprise-cluster`: Multi-node storage con replicación
   - `middleware.enterprise-auth`: SAML, LDAP, SSO
   - `tools.enterprise-audit`: Auditoría completa de operaciones

2. **Módulos de IA Avanzada**
   - `tools.reasoning-advanced`: CoT, ToT, multi-agent
   - `llm.fine-tuned`: Modelos fine-tuned para dominios específicos
   - `indexer.ml-enhanced`: Indexación con ML para mejor relevancia

3. **Integraciones**
   - `tools.salesforce`: Integración con Salesforce CRM
   - `tools.jira`: Integración con Jira
   - `storage.snowflake`: Storage en Snowflake

---

### Roadmap de Implementación

#### Fase 1: Fundamentos (2-3 semanas de desarrollo)
- [ ] Crear `pkg/modules/` con interfaces base
- [ ] Implementar registry global
- [ ] Implementar ModuleManager con ciclo de vida
- [ ] Refactorizar ToolManager para usar el sistema de módulos
- [ ] Tests unitarios completos

#### Fase 2: Módulos Standard (2 semanas)
- [ ] Crear `modules/standard/` con módulos core
- [ ] Migrar herramientas existentes a módulos
- [ ] Documentación de desarrollo de módulos

#### Fase 3: xremembrances (1-2 semanas)
- [ ] Implementar builder de binarios
- [ ] Templates de main.go
- [ ] CI/CD para builds automatizados

#### Fase 4: Ecosistema (Ongoing)
- [ ] Repositorio de módulos oficiales
- [ ] Documentación para desarrolladores
- [ ] Módulos de ejemplo
- [ ] Sistema de licenciamiento para módulos comerciales

---

### Beneficios del Diseño

1. **Desacoplamiento**: Los módulos no necesitan conocerse entre sí
2. **Extensibilidad**: Agregar funcionalidad es tan simple como importar un paquete
3. **Type Safety**: Go garantiza seguridad de tipos en tiempo de compilación
4. **Zero Runtime Overhead**: Los módulos se compilan en el binario
5. **Comercialización**: Licencias por módulo permiten monetización flexible
6. **Comunidad**: Terceros pueden contribuir módulos sin acceso al core

---

## Extensiones Avanzadas: Casos de Uso Específicos

### 1. Extensiones de Storage: Multi-BBDD y Sincronización

Este es uno de los casos más complejos. El objetivo es permitir:
- Usar múltiples bases de datos simultáneamente
- Sincronizar datos entre backends
- Routing inteligente de operaciones

#### 1.1 Interfaces para Storage Extensions

```go
// pkg/modules/storage_ext.go

package modules

import (
    "context"
    "time"
    "github.com/madeindigio/remembrances-mcp/internal/storage"
)

// StorageProvider permite registrar nuevos backends de almacenamiento
type StorageProvider interface {
    Module
    StorageType() string  // "postgresql", "redis", "mongodb", etc.
    NewStorage(ctx context.Context, cfg map[string]any) (storage.FullStorage, error)
}

// StorageRouter decide a qué backend va cada operación
type StorageRouter interface {
    Module
    // Route decide qué storages usar para una operación
    Route(op StorageOperation) RoutingDecision
}

type StorageOperation struct {
    Type      OperationType // READ, WRITE, DELETE
    Method    string        // "SaveFact", "SearchSimilar", etc.
    Table     string        // "facts", "vectors", "entities", etc.
    UserID    string
    Key       string
    Data      any
}

type OperationType string
const (
    OpRead   OperationType = "READ"
    OpWrite  OperationType = "WRITE"
    OpDelete OperationType = "DELETE"
)

type RoutingDecision struct {
    Targets     []string      // IDs de storages destino
    Fallbacks   []string      // Storages de fallback si falla el primario
    Strategy    WriteStrategy // Para escrituras
}

type WriteStrategy string
const (
    WriteAll      WriteStrategy = "all"       // Escribe a todos (síncrono)
    WritePrimary  WriteStrategy = "primary"   // Solo al primario
    WriteAsync    WriteStrategy = "async"     // Primario síncrono, resto async
)

// StorageSync maneja la sincronización entre backends
type StorageSync interface {
    Module
    // OnWrite se llama después de cada escritura exitosa
    OnWrite(ctx context.Context, event WriteEvent) error
    // Sync fuerza sincronización de datos pendientes
    Sync(ctx context.Context) error
    // Strategy retorna la estrategia de sincronización
    Strategy() SyncStrategy
}

type WriteEvent struct {
    Operation   StorageOperation
    Source      string    // ID del storage donde se escribió
    Timestamp   time.Time
    Success     bool
    Error       error
}

type SyncStrategy string
const (
    SyncImmediate SyncStrategy = "immediate"  // Síncrono, bloquea hasta completar
    SyncAsync     SyncStrategy = "async"      // Asíncrono con cola
    SyncBatch     SyncStrategy = "batch"      // En lotes periódicos
    SyncManual    SyncStrategy = "manual"     // Solo cuando se llama Sync()
)
```

#### 1.2 CompositeStorage: Implementación del Patrón

```go
// pkg/modules/composite_storage.go

package modules

import (
    "context"
    "fmt"
    "sync"
    
    "github.com/madeindigio/remembrances-mcp/internal/storage"
)

// CompositeStorage implementa FullStorage delegando a múltiples backends
type CompositeStorage struct {
    storages map[string]storage.FullStorage // ID -> Storage
    router   StorageRouter
    syncs    []StorageSync
    primary  string // ID del storage primario
    mu       sync.RWMutex
}

func NewCompositeStorage(primary string) *CompositeStorage {
    return &CompositeStorage{
        storages: make(map[string]storage.FullStorage),
        primary:  primary,
    }
}

func (cs *CompositeStorage) AddStorage(id string, s storage.FullStorage) {
    cs.mu.Lock()
    defer cs.mu.Unlock()
    cs.storages[id] = s
}

func (cs *CompositeStorage) SetRouter(r StorageRouter) {
    cs.router = r
}

func (cs *CompositeStorage) AddSync(s StorageSync) {
    cs.syncs = append(cs.syncs, s)
}

// Ejemplo: SaveFact con routing y sync
func (cs *CompositeStorage) SaveFact(ctx context.Context, userID, key string, value interface{}) error {
    op := StorageOperation{
        Type:   OpWrite,
        Method: "SaveFact",
        Table:  "facts",
        UserID: userID,
        Key:    key,
        Data:   value,
    }
    
    // Obtener decisión de routing
    decision := cs.route(op)
    
    // Ejecutar según estrategia
    var firstErr error
    var successTargets []string
    
    for i, targetID := range decision.Targets {
        storage, ok := cs.storages[targetID]
        if !ok {
            continue
        }
        
        err := storage.SaveFact(ctx, userID, key, value)
        if err != nil {
            if i == 0 { // Primario falló
                firstErr = err
            }
            continue
        }
        successTargets = append(successTargets, targetID)
        
        // Si es WriteAsync, solo esperamos al primario
        if decision.Strategy == WriteAsync && i == 0 {
            // Lanzar resto en goroutines
            go cs.writeToRemainingAsync(ctx, op, decision.Targets[1:])
            break
        }
    }
    
    // Notificar a los syncs
    for _, sync := range cs.syncs {
        event := WriteEvent{
            Operation: op,
            Source:    decision.Targets[0],
            Timestamp: time.Now(),
            Success:   firstErr == nil,
            Error:     firstErr,
        }
        // Async notification
        go sync.OnWrite(ctx, event)
    }
    
    return firstErr
}

func (cs *CompositeStorage) route(op StorageOperation) RoutingDecision {
    if cs.router != nil {
        return cs.router.Route(op)
    }
    // Default: solo primario
    return RoutingDecision{
        Targets:  []string{cs.primary},
        Strategy: WritePrimary,
    }
}

// GetFact con fallback
func (cs *CompositeStorage) GetFact(ctx context.Context, userID, key string) (interface{}, error) {
    op := StorageOperation{
        Type:   OpRead,
        Method: "GetFact",
        Table:  "facts",
        UserID: userID,
        Key:    key,
    }
    
    decision := cs.route(op)
    
    // Intentar targets en orden
    for _, targetID := range append(decision.Targets, decision.Fallbacks...) {
        storage, ok := cs.storages[targetID]
        if !ok {
            continue
        }
        
        result, err := storage.GetFact(ctx, userID, key)
        if err == nil {
            return result, nil
        }
        // Si no encontró, intentar siguiente
    }
    
    return nil, fmt.Errorf("fact not found in any storage")
}
```

#### 1.3 Ejemplo: Módulo de Routing Configurable

```go
// github.com/remembrances/storage-router/router.go

package router

import (
    "regexp"
    "github.com/madeindigio/remembrances-mcp/pkg/modules"
)

func init() {
    modules.RegisterModule(ConfigurableRouter{})
}

type ConfigurableRouter struct {
    rules []RoutingRule
}

type RoutingRule struct {
    Match    RuleMatch
    Targets  []string
    Fallback []string
    Strategy modules.WriteStrategy
}

type RuleMatch struct {
    Operation *modules.OperationType
    Method    *regexp.Regexp  // Regex para method name
    Table     *string
    UserID    *regexp.Regexp  // Puede routear por usuario
}

func (ConfigurableRouter) ModuleInfo() modules.ModuleInfo {
    return modules.ModuleInfo{
        ID:          "storage.router",
        Name:        "Configurable Storage Router",
        Description: "Routes storage operations based on configurable rules",
        Version:     "1.0.0",
        New:         func() modules.Module { return new(ConfigurableRouter) },
    }
}

func (r *ConfigurableRouter) Provision(ctx context.Context, cfg modules.ModuleConfig) error {
    // Parsear reglas desde configuración
    rulesRaw, _ := cfg.Raw["rules"].([]any)
    for _, ruleRaw := range rulesRaw {
        rule := parseRule(ruleRaw)
        r.rules = append(r.rules, rule)
    }
    return nil
}

func (r *ConfigurableRouter) Route(op modules.StorageOperation) modules.RoutingDecision {
    for _, rule := range r.rules {
        if r.matches(rule.Match, op) {
            return modules.RoutingDecision{
                Targets:   rule.Targets,
                Fallbacks: rule.Fallback,
                Strategy:  rule.Strategy,
            }
        }
    }
    // Default
    return modules.RoutingDecision{
        Targets:  []string{"primary"},
        Strategy: modules.WritePrimary,
    }
}

func (r *ConfigurableRouter) matches(m RuleMatch, op modules.StorageOperation) bool {
    if m.Operation != nil && *m.Operation != op.Type {
        return false
    }
    if m.Method != nil && !m.Method.MatchString(op.Method) {
        return false
    }
    if m.Table != nil && *m.Table != op.Table {
        return false
    }
    if m.UserID != nil && !m.UserID.MatchString(op.UserID) {
        return false
    }
    return true
}
```

#### 1.4 Ejemplo: Módulo de Sincronización a Elasticsearch

```go
// github.com/remembrances/sync-elasticsearch/elasticsearch.go

package elasticsearch

import (
    "context"
    "github.com/elastic/go-elasticsearch/v8"
    "github.com/madeindigio/remembrances-mcp/pkg/modules"
)

func init() {
    modules.RegisterModule(ElasticsearchSync{})
}

type ElasticsearchSync struct {
    client   *elasticsearch.Client
    index    string
    queue    chan modules.WriteEvent
    strategy modules.SyncStrategy
}

func (ElasticsearchSync) ModuleInfo() modules.ModuleInfo {
    return modules.ModuleInfo{
        ID:          "sync.elasticsearch",
        Name:        "Elasticsearch Sync",
        Description: "Syncs data to Elasticsearch for advanced search",
        Version:     "1.0.0",
        License:     "Commercial",
        New:         func() modules.Module { return new(ElasticsearchSync) },
    }
}

func (es *ElasticsearchSync) Provision(ctx context.Context, cfg modules.ModuleConfig) error {
    url, _ := cfg.Raw["url"].(string)
    es.index, _ = cfg.Raw["index"].(string)
    strategyStr, _ := cfg.Raw["strategy"].(string)
    
    client, err := elasticsearch.NewClient(elasticsearch.Config{
        Addresses: []string{url},
    })
    if err != nil {
        return err
    }
    es.client = client
    es.strategy = modules.SyncStrategy(strategyStr)
    es.queue = make(chan modules.WriteEvent, 10000)
    
    // Worker para procesar cola async
    if es.strategy == modules.SyncAsync {
        go es.worker(ctx)
    }
    
    return nil
}

func (es *ElasticsearchSync) OnWrite(ctx context.Context, event modules.WriteEvent) error {
    if !event.Success {
        return nil // No sincronizar fallos
    }
    
    switch es.strategy {
    case modules.SyncImmediate:
        return es.indexDocument(ctx, event)
    case modules.SyncAsync:
        select {
        case es.queue <- event:
        default:
            // Cola llena, log warning
        }
    }
    return nil
}

func (es *ElasticsearchSync) Strategy() modules.SyncStrategy {
    return es.strategy
}

func (es *ElasticsearchSync) Sync(ctx context.Context) error {
    // Forzar flush de la cola
    for len(es.queue) > 0 {
        select {
        case event := <-es.queue:
            es.indexDocument(ctx, event)
        default:
            return nil
        }
    }
    return nil
}

func (es *ElasticsearchSync) worker(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case event := <-es.queue:
            es.indexDocument(ctx, event)
        }
    }
}

func (es *ElasticsearchSync) indexDocument(ctx context.Context, event modules.WriteEvent) error {
    // Indexar en Elasticsearch
    doc := map[string]any{
        "operation": event.Operation.Method,
        "table":     event.Operation.Table,
        "user_id":   event.Operation.UserID,
        "key":       event.Operation.Key,
        "data":      event.Operation.Data,
        "timestamp": event.Timestamp,
    }
    // ... indexar documento
    return nil
}
```

#### 1.5 Configuración Multi-BBDD

```yaml
# config.yaml

modules:
  # Backend primario: SurrealDB embebido (ya existente)
  storage.surrealdb:
    enabled: true
    id: "primary"
    config:
      db_path: "./remembrances.db"
  
  # Backend secundario: PostgreSQL para facts (más rápido para KV)
  storage.postgresql:
    enabled: true
    id: "postgres"
    config:
      connection: "postgres://user:pass@localhost/remembrances"
  
  # Backend de caché: Redis
  storage.redis:
    enabled: true
    id: "cache"
    config:
      url: "redis://localhost:6379"
      ttl: 3600  # 1 hora
  
  # Router para decidir dónde va cada operación
  storage.router:
    enabled: true
    config:
      rules:
        # Facts: escribir a Redis (cache) y PostgreSQL, leer de Redis primero
        - match:
            table: "facts"
            operation: "WRITE"
          targets: ["cache", "postgres"]
          strategy: "all"
        
        - match:
            table: "facts"
            operation: "READ"
          targets: ["cache"]
          fallback: ["postgres"]
        
        # Vectors: solo SurrealDB (tiene buen soporte vectorial)
        - match:
            table: "vectors"
          targets: ["primary"]
        
        # Graph: SurrealDB (nativo para grafos)
        - match:
            table: "entities"
          targets: ["primary"]
        
        - match:
            table: "relationships"
          targets: ["primary"]
        
        # Code: primario con backup a PostgreSQL
        - match:
            method: ".*Code.*"
          targets: ["primary"]
          strategy: "async"
  
  # Sincronización a Elasticsearch para búsquedas avanzadas
  sync.elasticsearch:
    enabled: true
    config:
      url: "http://elasticsearch:9200"
      index: "remembrances"
      strategy: "async"
      sync_tables: ["vectors", "documents", "events"]
  
  # Backup a S3
  sync.s3:
    enabled: true
    config:
      bucket: "remembrances-backup"
      region: "eu-west-1"
      strategy: "batch"
      batch_interval: "1h"
```

---

### 2. Extensiones de Tool Middleware: Modificar Respuestas

El objetivo es interceptar las llamadas a tools para:
- Pre-procesar requests (validación, enriquecimiento)
- Post-procesar responses (transformación, caché, logging)
- Cortocircuitar (caché hits, rate limiting)

#### 2.1 Interfaces para Tool Middleware

```go
// pkg/modules/middleware.go

package modules

import (
    "context"
    "github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// ToolHandler es la firma de un handler de tool
type ToolHandler func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)

// ToolMiddleware intercepta llamadas a tools
type ToolMiddleware interface {
    Module
    // Wrap envuelve un handler con lógica adicional
    Wrap(next ToolHandler) ToolHandler
    // Priority determina el orden de ejecución (menor = antes)
    Priority() int
    // ToolFilter indica qué tools afecta (nil = todas)
    ToolFilter() []string
}

// ToolTransformer solo transforma la respuesta (más simple que Middleware)
type ToolTransformer interface {
    Module
    // Transform modifica el resultado de una tool
    Transform(ctx context.Context, req *protocol.CallToolRequest, res *protocol.CallToolResult) (*protocol.CallToolResult, error)
    // ToolFilter indica qué tools afecta
    ToolFilter() []string
}

// ToolValidator valida requests antes de ejecutar
type ToolValidator interface {
    Module
    // Validate retorna error si el request no es válido
    Validate(ctx context.Context, req *protocol.CallToolRequest) error
    // ToolFilter indica qué tools afecta
    ToolFilter() []string
}

// ResponseEnricher añade información adicional a las respuestas
type ResponseEnricher interface {
    Module
    // Enrich añade metadata/contexto a la respuesta
    Enrich(ctx context.Context, req *protocol.CallToolRequest, res *protocol.CallToolResult) (*protocol.CallToolResult, error)
    // ToolFilter indica qué tools afecta
    ToolFilter() []string
}
```

#### 2.2 MiddlewareChain: Ejecución en Cadena

```go
// pkg/modules/middleware_chain.go

package modules

import (
    "context"
    "sort"
    "strings"
    
    "github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// MiddlewareChain gestiona la cadena de middlewares
type MiddlewareChain struct {
    middlewares  []ToolMiddleware
    transformers []ToolTransformer
    validators   []ToolValidator
    enrichers    []ResponseEnricher
}

func NewMiddlewareChain() *MiddlewareChain {
    return &MiddlewareChain{}
}

func (mc *MiddlewareChain) AddMiddleware(m ToolMiddleware) {
    mc.middlewares = append(mc.middlewares, m)
    // Ordenar por prioridad
    sort.Slice(mc.middlewares, func(i, j int) bool {
        return mc.middlewares[i].Priority() < mc.middlewares[j].Priority()
    })
}

func (mc *MiddlewareChain) AddTransformer(t ToolTransformer) {
    mc.transformers = append(mc.transformers, t)
}

func (mc *MiddlewareChain) AddValidator(v ToolValidator) {
    mc.validators = append(mc.validators, v)
}

func (mc *MiddlewareChain) AddEnricher(e ResponseEnricher) {
    mc.enrichers = append(mc.enrichers, e)
}

// Wrap envuelve un handler con toda la cadena de middleware
func (mc *MiddlewareChain) Wrap(toolName string, handler ToolHandler) ToolHandler {
    // Construir cadena de adentro hacia afuera
    wrapped := handler
    
    // 1. Aplicar middlewares (en orden inverso para que el primero sea el más externo)
    for i := len(mc.middlewares) - 1; i >= 0; i-- {
        m := mc.middlewares[i]
        if mc.matchesFilter(toolName, m.ToolFilter()) {
            wrapped = m.Wrap(wrapped)
        }
    }
    
    // 2. Añadir validación al inicio
    withValidation := func(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
        // Ejecutar validadores
        for _, v := range mc.validators {
            if mc.matchesFilter(toolName, v.ToolFilter()) {
                if err := v.Validate(ctx, req); err != nil {
                    return &protocol.CallToolResult{
                        IsError: true,
                        Content: []protocol.Content{{Type: "text", Text: err.Error()}},
                    }, nil
                }
            }
        }
        return wrapped(ctx, req)
    }
    
    // 3. Añadir transformación y enriquecimiento al final
    withPostProcess := func(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
        res, err := withValidation(ctx, req)
        if err != nil {
            return res, err
        }
        
        // Aplicar transformadores
        for _, t := range mc.transformers {
            if mc.matchesFilter(toolName, t.ToolFilter()) {
                res, err = t.Transform(ctx, req, res)
                if err != nil {
                    return res, err
                }
            }
        }
        
        // Aplicar enrichers
        for _, e := range mc.enrichers {
            if mc.matchesFilter(toolName, e.ToolFilter()) {
                res, err = e.Enrich(ctx, req, res)
                if err != nil {
                    return res, err
                }
            }
        }
        
        return res, nil
    }
    
    return withPostProcess
}

func (mc *MiddlewareChain) matchesFilter(toolName string, filter []string) bool {
    if filter == nil || len(filter) == 0 {
        return true // nil = todas las tools
    }
    for _, pattern := range filter {
        if strings.HasSuffix(pattern, "*") {
            prefix := strings.TrimSuffix(pattern, "*")
            if strings.HasPrefix(toolName, prefix) {
                return true
            }
        } else if pattern == toolName {
            return true
        }
    }
    return false
}
```

#### 2.3 Ejemplo: Middleware de Razonamiento (Enriquece respuestas con análisis)

```go
// github.com/remembrances/middleware-reasoning/reasoning.go

package reasoning

import (
    "context"
    "encoding/json"
    "fmt"
    
    "github.com/madeindigio/remembrances-mcp/pkg/modules"
    "github.com/ThinkInAIXYZ/go-mcp/protocol"
)

func init() {
    modules.RegisterModule(ReasoningEnricher{})
}

type ReasoningEnricher struct {
    llmClient  LLMClient
    apiKey     string
    modelID    string
}

func (ReasoningEnricher) ModuleInfo() modules.ModuleInfo {
    return modules.ModuleInfo{
        ID:          "middleware.reasoning",
        Name:        "Reasoning Enricher",
        Description: "Enriches search results with AI-powered analysis and synthesis",
        Version:     "1.0.0",
        License:     "Commercial",
        New:         func() modules.Module { return new(ReasoningEnricher) },
    }
}

func (r *ReasoningEnricher) Provision(ctx context.Context, cfg modules.ModuleConfig) error {
    r.apiKey, _ = cfg.Raw["api_key"].(string)
    r.modelID, _ = cfg.Raw["model"].(string)
    if r.modelID == "" {
        r.modelID = "claude-3-haiku-20240307" // Default económico
    }
    r.llmClient = NewAnthropicClient(r.apiKey)
    return nil
}

func (r *ReasoningEnricher) ToolFilter() []string {
    return []string{
        "search_vectors",
        "hybrid_search",
        "kb_search_documents",
        "search_events",
    }
}

func (r *ReasoningEnricher) Enrich(ctx context.Context, req *protocol.CallToolRequest, res *protocol.CallToolResult) (*protocol.CallToolResult, error) {
    if res.IsError {
        return res, nil // No enriquecer errores
    }
    
    // Extraer query del request
    var args map[string]any
    json.Unmarshal(req.Arguments, &args)
    query, _ := args["query"].(string)
    
    // Extraer resultados de la respuesta
    originalContent := ""
    for _, c := range res.Content {
        if c.Type == "text" {
            originalContent = c.Text
            break
        }
    }
    
    // Generar análisis con LLM
    analysis, err := r.generateAnalysis(ctx, query, originalContent)
    if err != nil {
        // Si falla el análisis, devolver resultado original
        return res, nil
    }
    
    // Crear respuesta enriquecida
    enrichedContent := fmt.Sprintf(`## Search Results
%s

## AI Analysis
%s`, originalContent, analysis)
    
    return &protocol.CallToolResult{
        Content: []protocol.Content{{
            Type: "text",
            Text: enrichedContent,
        }},
    }, nil
}

func (r *ReasoningEnricher) generateAnalysis(ctx context.Context, query, results string) (string, error) {
    prompt := fmt.Sprintf(`Analyze these search results for the query "%s" and provide:
1. Key insights found
2. Connections between results
3. Gaps or missing information
4. Suggested follow-up queries

Results:
%s`, query, results)
    
    return r.llmClient.Complete(ctx, prompt)
}
```

#### 2.4 Ejemplo: Middleware de Caché

```go
// github.com/remembrances/middleware-cache/cache.go

package cache

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "time"
    
    "github.com/madeindigio/remembrances-mcp/pkg/modules"
    "github.com/ThinkInAIXYZ/go-mcp/protocol"
)

func init() {
    modules.RegisterModule(CacheMiddleware{})
}

type CacheMiddleware struct {
    cache    Cache
    ttl      time.Duration
    tools    []string
}

func (CacheMiddleware) ModuleInfo() modules.ModuleInfo {
    return modules.ModuleInfo{
        ID:          "middleware.cache",
        Name:        "Response Cache",
        Description: "Caches tool responses to reduce latency and API calls",
        Version:     "1.0.0",
        New:         func() modules.Module { return new(CacheMiddleware) },
    }
}

func (c *CacheMiddleware) Provision(ctx context.Context, cfg modules.ModuleConfig) error {
    ttlSec, _ := cfg.Raw["ttl_seconds"].(float64)
    c.ttl = time.Duration(ttlSec) * time.Second
    if c.ttl == 0 {
        c.ttl = 5 * time.Minute
    }
    
    c.tools, _ = cfg.Raw["tools"].([]string)
    c.cache = NewInMemoryCache() // O Redis, etc.
    return nil
}

func (c *CacheMiddleware) Priority() int {
    return 10 // Ejecutar temprano para cortocircuitar si hay cache hit
}

func (c *CacheMiddleware) ToolFilter() []string {
    return c.tools
}

func (c *CacheMiddleware) Wrap(next modules.ToolHandler) modules.ToolHandler {
    return func(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
        // Generar cache key
        key := c.cacheKey(req)
        
        // Intentar obtener de caché
        if cached, ok := c.cache.Get(key); ok {
            return cached.(*protocol.CallToolResult), nil
        }
        
        // Ejecutar handler real
        res, err := next(ctx, req)
        if err != nil {
            return res, err
        }
        
        // Guardar en caché si no es error
        if !res.IsError {
            c.cache.Set(key, res, c.ttl)
        }
        
        return res, nil
    }
}

func (c *CacheMiddleware) cacheKey(req *protocol.CallToolRequest) string {
    data, _ := json.Marshal(map[string]any{
        "tool": req.Name,
        "args": req.Arguments,
    })
    hash := sha256.Sum256(data)
    return hex.EncodeToString(hash[:])
}
```

#### 2.5 Configuración de Middlewares

```yaml
# config.yaml

modules:
  # Caché para reducir llamadas repetidas
  middleware.cache:
    enabled: true
    config:
      ttl_seconds: 300
      tools:
        - "search_*"
        - "get_*"
        - "kb_search_*"
  
  # Rate limiting
  middleware.ratelimit:
    enabled: true
    config:
      requests_per_second: 50
      burst: 100
      # Limitar más las tools costosas
      tool_limits:
        "code_index_project": { rps: 1, burst: 2 }
        "hybrid_search": { rps: 10, burst: 20 }
  
  # Enriquecimiento con razonamiento
  middleware.reasoning:
    enabled: true
    config:
      api_key: "${ANTHROPIC_API_KEY}"
      model: "claude-3-haiku-20240307"
      # Solo para búsquedas
      tools:
        - "search_vectors"
        - "hybrid_search"
  
  # Logging/Auditoría
  middleware.audit:
    enabled: true
    config:
      log_requests: true
      log_responses: true
      log_to: "elasticsearch"  # o "file", "stdout"
      elasticsearch_url: "http://elasticsearch:9200"
  
  # Transformación de formato
  middleware.format:
    enabled: true
    config:
      # Convertir respuestas YAML a JSON para ciertas tools
      yaml_to_json:
        - "code_find_symbol"
        - "code_get_*"
```

---

### 3. Extensiones HTTP: Nuevos Endpoints

El objetivo es permitir módulos que expongan APIs REST adicionales para:
- Webhooks de servicios externos
- Admin dashboards
- APIs personalizadas
- Métricas/monitoring

#### 3.1 Interfaces para HTTP Extensions

```go
// pkg/modules/http_ext.go

package modules

import (
    "net/http"
)

// HTTPEndpointProvider permite registrar rutas HTTP adicionales
type HTTPEndpointProvider interface {
    Module
    // Routes retorna las rutas que este módulo expone
    Routes() []HTTPRoute
    // BasePath retorna el prefijo para todas las rutas (ej: "/admin")
    BasePath() string
}

// HTTPRoute define una ruta HTTP
type HTTPRoute struct {
    Method      string           // GET, POST, PUT, DELETE, etc.
    Path        string           // Relativo a BasePath
    Handler     http.HandlerFunc
    Middlewares []HTTPMiddleware // Middlewares específicos de esta ruta
    Description string           // Para documentación
}

// HTTPMiddleware es middleware HTTP estándar
type HTTPMiddleware func(http.Handler) http.Handler

// HTTPAuthProvider permite módulos de autenticación para HTTP
type HTTPAuthProvider interface {
    Module
    // Authenticate verifica credenciales y retorna user info
    Authenticate(r *http.Request) (*AuthInfo, error)
    // Middleware retorna un middleware HTTP para proteger rutas
    Middleware() HTTPMiddleware
}

type AuthInfo struct {
    UserID   string
    Roles    []string
    Metadata map[string]any
}

// WebhookHandler procesa webhooks entrantes
type WebhookHandler interface {
    Module
    // HandleWebhook procesa un webhook
    HandleWebhook(ctx context.Context, source string, payload []byte) error
    // Validate verifica la firma/autenticidad del webhook
    Validate(r *http.Request) error
    // Sources retorna los tipos de webhook que maneja
    Sources() []string // "github", "gitlab", "slack", etc.
}
```

#### 3.2 HTTPRouter: Integración con Módulos

```go
// pkg/modules/http_router.go

package modules

import (
    "net/http"
    "path"
)

// HTTPRouter gestiona las rutas HTTP de todos los módulos
type HTTPRouter struct {
    mux       *http.ServeMux
    providers []HTTPEndpointProvider
    auth      HTTPAuthProvider
}

func NewHTTPRouter() *HTTPRouter {
    return &HTTPRouter{
        mux: http.NewServeMux(),
    }
}

func (hr *HTTPRouter) SetAuthProvider(auth HTTPAuthProvider) {
    hr.auth = auth
}

func (hr *HTTPRouter) AddProvider(provider HTTPEndpointProvider) {
    hr.providers = append(hr.providers, provider)
    hr.registerRoutes(provider)
}

func (hr *HTTPRouter) registerRoutes(provider HTTPEndpointProvider) {
    basePath := provider.BasePath()
    
    for _, route := range provider.Routes() {
        fullPath := path.Join(basePath, route.Path)
        
        // Construir handler con middlewares
        handler := http.Handler(route.Handler)
        
        // Aplicar middlewares de la ruta (en orden inverso)
        for i := len(route.Middlewares) - 1; i >= 0; i-- {
            handler = route.Middlewares[i](handler)
        }
        
        // Wrapper para verificar método
        methodHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if r.Method != route.Method && route.Method != "*" {
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
                return
            }
            handler.ServeHTTP(w, r)
        })
        
        hr.mux.Handle(fullPath, methodHandler)
    }
}

func (hr *HTTPRouter) Handler() http.Handler {
    return hr.mux
}
```

#### 3.3 Ejemplo: Módulo de Webhooks GitHub

```go
// github.com/remembrances/http-webhooks/github.go

package webhooks

import (
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "io"
    "net/http"
    
    "github.com/madeindigio/remembrances-mcp/pkg/modules"
    "github.com/madeindigio/remembrances-mcp/internal/storage"
)

func init() {
    modules.RegisterModule(GitHubWebhooks{})
}

type GitHubWebhooks struct {
    secret   string
    storage  storage.FullStorage
    embedder embedder.Embedder
}

func (GitHubWebhooks) ModuleInfo() modules.ModuleInfo {
    return modules.ModuleInfo{
        ID:          "http.webhooks.github",
        Name:        "GitHub Webhooks",
        Description: "Receives GitHub webhooks and stores events in remembrances",
        Version:     "1.0.0",
        New:         func() modules.Module { return new(GitHubWebhooks) },
    }
}

func (g *GitHubWebhooks) Provision(ctx context.Context, cfg modules.ModuleConfig) error {
    g.secret, _ = cfg.Raw["secret"].(string)
    g.storage = cfg.Storage
    g.embedder = cfg.Embedder
    return nil
}

func (g *GitHubWebhooks) BasePath() string {
    return "/webhooks"
}

func (g *GitHubWebhooks) Routes() []modules.HTTPRoute {
    return []modules.HTTPRoute{
        {
            Method:      "POST",
            Path:        "/github",
            Handler:     g.handleGitHubWebhook,
            Description: "Receive GitHub webhook events",
        },
        {
            Method:      "GET",
            Path:        "/github/status",
            Handler:     g.handleStatus,
            Description: "Check webhook status",
        },
    }
}

func (g *GitHubWebhooks) handleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
    // Verificar firma
    signature := r.Header.Get("X-Hub-Signature-256")
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Failed to read body", http.StatusBadRequest)
        return
    }
    
    if !g.verifySignature(body, signature) {
        http.Error(w, "Invalid signature", http.StatusUnauthorized)
        return
    }
    
    // Parsear evento
    eventType := r.Header.Get("X-GitHub-Event")
    var payload map[string]any
    json.Unmarshal(body, &payload)
    
    // Guardar como evento en remembrances
    ctx := r.Context()
    content := g.formatEvent(eventType, payload)
    embedding, _ := g.embedder.Embed(ctx, content)
    
    g.storage.SaveEvent(ctx, "github", eventType, content, embedding, map[string]any{
        "source":     "github_webhook",
        "event_type": eventType,
        "repo":       payload["repository"],
    })
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}

func (g *GitHubWebhooks) verifySignature(body []byte, signature string) bool {
    if g.secret == "" {
        return true // No secret configured, skip verification
    }
    
    mac := hmac.New(sha256.New, []byte(g.secret))
    mac.Write(body)
    expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(signature))
}

func (g *GitHubWebhooks) formatEvent(eventType string, payload map[string]any) string {
    // Formatear evento como texto legible
    // ...
    return ""
}

func (g *GitHubWebhooks) handleStatus(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]any{
        "status": "healthy",
        "module": "github_webhooks",
    })
}
```

#### 3.4 Ejemplo: Módulo Admin API

```go
// github.com/remembrances/http-admin/admin.go

package admin

import (
    "context"
    "encoding/json"
    "net/http"
    
    "github.com/madeindigio/remembrances-mcp/pkg/modules"
)

func init() {
    modules.RegisterModule(AdminAPI{})
}

type AdminAPI struct {
    moduleManager *modules.ModuleManager
    storage       storage.FullStorage
    requireAuth   bool
    authProvider  modules.HTTPAuthProvider
}

func (AdminAPI) ModuleInfo() modules.ModuleInfo {
    return modules.ModuleInfo{
        ID:          "http.admin",
        Name:        "Admin API",
        Description: "Administrative API for managing remembrances",
        Version:     "1.0.0",
        License:     "Commercial",
        New:         func() modules.Module { return new(AdminAPI) },
    }
}

func (a *AdminAPI) BasePath() string {
    return "/admin"
}

func (a *AdminAPI) Routes() []modules.HTTPRoute {
    var middlewares []modules.HTTPMiddleware
    if a.requireAuth && a.authProvider != nil {
        middlewares = append(middlewares, a.authProvider.Middleware())
    }
    
    return []modules.HTTPRoute{
        {
            Method:      "GET",
            Path:        "/modules",
            Handler:     a.listModules,
            Middlewares: middlewares,
            Description: "List all loaded modules",
        },
        {
            Method:      "GET",
            Path:        "/stats",
            Handler:     a.getStats,
            Middlewares: middlewares,
            Description: "Get system statistics",
        },
        {
            Method:      "POST",
            Path:        "/modules/{id}/reload",
            Handler:     a.reloadModule,
            Middlewares: middlewares,
            Description: "Reload a specific module",
        },
        {
            Method:      "GET",
            Path:        "/health",
            Handler:     a.healthCheck,
            Description: "Health check endpoint",
        },
    }
}

func (a *AdminAPI) listModules(w http.ResponseWriter, r *http.Request) {
    mods := modules.ListModules()
    
    result := make([]map[string]any, 0, len(mods))
    for _, m := range mods {
        result = append(result, map[string]any{
            "id":          m.ID,
            "name":        m.Name,
            "description": m.Description,
            "version":     m.Version,
            "author":      m.Author,
            "license":     m.License,
        })
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}

func (a *AdminAPI) getStats(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    stats, _ := a.storage.GetStats(ctx, "")
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(stats)
}

func (a *AdminAPI) reloadModule(w http.ResponseWriter, r *http.Request) {
    // Implementar recarga de módulo
    w.WriteHeader(http.StatusNotImplemented)
}

func (a *AdminAPI) healthCheck(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}
```

#### 3.5 Configuración de HTTP Extensions

```yaml
# config.yaml

modules:
  # Webhooks de GitHub
  http.webhooks.github:
    enabled: true
    config:
      secret: "${GITHUB_WEBHOOK_SECRET}"
  
  # Webhooks de Slack
  http.webhooks.slack:
    enabled: true
    config:
      signing_secret: "${SLACK_SIGNING_SECRET}"
      bot_token: "${SLACK_BOT_TOKEN}"
  
  # Admin API
  http.admin:
    enabled: true
    config:
      require_auth: true
      allowed_roles: ["admin", "operator"]
  
  # Autenticación JWT
  http.auth.jwt:
    enabled: true
    config:
      secret: "${JWT_SECRET}"
      issuer: "remembrances"
      expiry: "24h"
  
  # Métricas Prometheus
  http.metrics:
    enabled: true
    config:
      path: "/metrics"
      include_go_metrics: true
  
  # GraphQL endpoint
  http.graphql:
    enabled: true
    config:
      path: "/graphql"
      playground: true  # Habilitar playground en desarrollo
      introspection: true
```

---

## Resumen de Arquitectura Final

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          remembrances-mcp                                │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                      Module Manager                                 │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                │ │
│  │  │  Storage    │  │    Tool     │  │    HTTP     │                │ │
│  │  │  Registry   │  │  Middleware │  │  Endpoints  │                │ │
│  │  │             │  │   Chain     │  │   Router    │                │ │
│  │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘                │ │
│  └─────────┼────────────────┼────────────────┼───────────────────────┘ │
│            │                │                │                          │
│            ▼                ▼                ▼                          │
│  ┌─────────────────┐ ┌─────────────┐ ┌─────────────────┐              │
│  │ CompositeStorage│ │  Wrapped    │ │   HTTP Server   │              │
│  │                 │ │  Handlers   │ │                 │              │
│  │ ┌─────┐ ┌─────┐│ │             │ │ /mcp/*          │              │
│  │ │Surr │ │Post ││ │ validate -> │ │ /admin/*        │              │
│  │ │ealDB│ │greSQL││ │ cache ->   │ │ /webhooks/*     │              │
│  │ └─────┘ └─────┘│ │ handler ->  │ │ /metrics        │              │
│  │ ┌─────┐ ┌─────┐│ │ enrich ->   │ │ /graphql        │              │
│  │ │Redis│ │Elast││ │ transform   │ │                 │              │
│  │ │Cache│ │icsrc││ │             │ │                 │              │
│  │ └─────┘ └─────┘│ │             │ │                 │              │
│  └─────────────────┘ └─────────────┘ └─────────────────┘              │
│            │                │                │                          │
│            ▼                ▼                ▼                          │
│  ┌─────────────────────────────────────────────────────────────────┐  │
│  │                         Storage Sync                             │  │
│  │   Elasticsearch │ S3 Backup │ Analytics │ Audit Log              │  │
│  └─────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Conclusión

Este diseño, inspirado en Caddy, proporciona una base sólida para un ecosistema de extensiones que puede crecer orgánicamente mientras mantiene la simplicidad y performance de Go. La separación clara entre el core y los módulos permite tanto desarrollo open source como comercialización de funcionalidades premium.
