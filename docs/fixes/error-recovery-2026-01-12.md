# Error Recovery & Resilient Processing - 2026-01-12

## Problema Original

El sistema era **frágil** y terminaba abruptamente ante cualquier error:

- ❌ Un archivo problemático detenía toda la indexación
- ❌ Un embedding fallido crasheaba el programa (SIGABRT)
- ❌ Panics no recuperados terminaban el proceso
- ❌ No había visibilidad de qué archivos/símbolos fallaron
- ❌ Pérdida total del trabajo realizado hasta el momento del error

## Filosofía de la Solución

**"Fail gracefully, log extensively, continue processing"**

El sistema ahora:
1. ✅ **Recupera de panics** en puntos críticos
2. ✅ **Continúa procesando** aunque fallen elementos individuales
3. ✅ **Registra errores detalladamente** para debugging
4. ✅ **Reporta estadísticas** de éxito/fallo al finalizar
5. ✅ **Solo falla si TODO falla** (no por errores parciales)

---

## Arquitectura de Recuperación

### Capas de Protección

```
┌─────────────────────────────────────────┐
│  Capa 1: File Processing                │
│  - Panic recovery por archivo           │
│  - Continue con siguiente archivo        │
│  - Solo falla si TODOS los archivos     │
└──────────────┬──────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────┐
│  Capa 2: Symbol Embeddings              │
│  - Panic recovery por batch              │
│  - Continue con siguiente batch          │
│  - Guarda símbolos sin embeddings        │
└──────────────┬──────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────┐
│  Capa 3: Chunk Embeddings               │
│  - Panic recovery por batch              │
│  - Continue con siguiente batch          │
│  - Solo falla si TODOS los chunks        │
└──────────────┬──────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────┐
│  Capa 4: Individual Embedding           │
│  - Panic recovery en embedSingle()       │
│  - Validación exhaustiva pre-C call     │
│  - Logging detallado del error           │
└─────────────────────────────────────────┘
```

---

## Implementación Detallada

### 1. Recuperación de Panics en embedSingle()

**Ubicación**: `pkg/embedder/gguf.go`

```go
func (g *GGUFEmbedder) embedSingle(ctx context.Context, text string) (embeddings []float32, err error) {
    // CRITICAL: Recover from panics to prevent program termination
    defer func() {
        if r := recover() {
            stack := debug.Stack()
            slog.Error("CRITICAL: Panic recovered in embedSingle",
                "panic", r,
                "text_length", len(text),
                "stack", string(stack))
            err = fmt.Errorf("panic recovered during embedding: %v", r)
            embeddings = nil
        }
    }()
    
    // ... resto del código ...
}
```

**Beneficios**:
- Captura panics de llamadas C (llama.cpp)
- Convierte panic en error manejable
- Incluye stack trace completo para debugging
- Permite continuar con siguiente embedding

---

### 2. Continuación en EmbedDocuments()

**Ubicación**: `pkg/embedder/gguf.go`

```go
func (g *GGUFEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
    result := make([][]float32, len(texts))
    var lastError error
    failedCount := 0
    
    for i, text := range texts {
        embedding, err := g.embedSingle(ctx, text)
        if err != nil {
            // Log error but CONTINUE processing other texts
            slog.Error("Failed to embed document, continuing with next",
                "index", i,
                "error", err,
                "text_length", len(text))
            lastError = err
            failedCount++
            result[i] = nil  // Mark as failed
            continue         // CRITICAL: Continue loop
        }
        
        result[i] = embedding
    }
    
    // Only fail if ALL embeddings failed
    if failedCount == len(texts) {
        return nil, fmt.Errorf("all %d embeddings failed", failedCount)
    }
    
    // Log partial failures
    if failedCount > 0 {
        slog.Warn("Some embeddings failed but continuing",
            "failed_count", failedCount,
            "success_rate", fmt.Sprintf("%.1f%%", ...))
    }
    
    return result, nil  // Return partial results
}
```

**Comportamiento**:
- ❌ Falla individual → Log warning + continue
- ⚠️ Algunas fallan → Log warning + retornar parciales
- ❌ Todas fallan → Return error

---

### 3. Manejo de Embeddings Nulos en Indexer

**Ubicación**: `internal/indexer/indexer_embeddings.go`

```go
func (idx *Indexer) generateEmbeddings(ctx context.Context, symbols []*treesitter.CodeSymbol) error {
    totalFailed := 0
    totalProcessed := 0
    
    for i := 0; i < len(texts); i += batchSize {
        batch := texts[i:end]
        embeddings, err := idx.embedder.EmbedDocuments(ctx, batch)
        
        if err != nil {
            // Log error but continue with next batch
            slog.Warn("Failed to generate embeddings for batch, skipping",
                "batch_start", i,
                "batch_size", len(batch),
                "error", err)
            totalFailed += len(batch)
            totalProcessed += len(batch)
            continue  // CRITICAL: Process next batch
        }
        
        // Assign embeddings (skip nils)
        for j, embedding := range embeddings {
            if embedding == nil {
                // This embedding failed individually
                slog.Warn("Skipping nil embedding for symbol",
                    "symbol", symbols[symIdx].Name)
                totalFailed++
            } else {
                symbols[symIdx].Embedding = embedding
            }
            totalProcessed++
        }
    }
    
    // Only fail if ALL failed
    if totalFailed == totalProcessed && totalProcessed > 0 {
        return fmt.Errorf("all %d embeddings failed", totalFailed)
    }
    
    return nil  // Partial success is OK
}
```

**Resultado**:
- Símbolos sin embeddings se guardan sin vector
- Búsqueda semántica no los encontrará, pero metadata sí existe
- Mejor que perder todo el archivo

---

### 4. Recuperación de Panics en File Processing

**Ubicación**: `internal/indexer/indexer.go`

```go
func (idx *Indexer) processFiles(...) error {
    for file := range fileChan {
        // Recover from panics to prevent one file from crashing entire process
        func() {
            defer func() {
                if r := recover() {
                    slog.Error("PANIC recovered while processing file",
                        "file", file.RelPath,
                        "panic", r)
                    errChan <- fmt.Errorf("panic processing %s: %v", file.RelPath, r)
                }
            }()
            
            if err := idx.processFileWithParser(...); err != nil {
                slog.Warn("Error processing file, continuing with next",
                    "file", file.RelPath,
                    "error", err)
                errChan <- err
            }
        }()
    }
    
    // ... collect errors ...
    
    // Only fail if ALL files failed
    if failedFiles == totalFiles {
        return fmt.Errorf("all %d files failed", failedFiles)
    }
    
    // Partial success - log but continue
    slog.Info("Indexing completed with partial success",
        "successful_files", totalFiles - failedFiles,
        "failed_files", failedFiles)
    
    return nil  // Success even with some failures
}
```

---

## Logging Estructurado

### Niveles de Severidad

| Nivel | Uso | Ejemplo |
|-------|-----|---------|
| **ERROR** | Fallo individual recuperado | "Failed to embed document, continuing" |
| **WARN** | Resultado parcial o degradado | "Some embeddings failed but continuing" |
| **INFO** | Resumen de operación | "Indexing completed with partial success" |
| **DEBUG** | Detalles de debugging | Stack traces, valores internos |

### Logs Esperados

#### Durante Indexación Normal

```
level=INFO msg="Processing file" file="main.go"
level=INFO msg="Generated embeddings" symbol_count=15
level=INFO msg="File indexed successfully" file="main.go" symbols=15
```

#### Con Fallos Parciales

```
level=WARN msg="Text exceeds limit, truncating" original_length=5000 truncated_to=900
level=WARN msg="Failed to embed document, continuing with next" index=3 error="timeout"
level=WARN msg="Some embeddings failed but continuing" failed_count=2 success_rate="87.5%"
level=INFO msg="File indexed successfully" file="api.go" symbols=13 failed_embeddings=2
```

#### Con Fallo de Archivo Completo

```
level=ERROR msg="Error processing file" file="broken.go" error="parse error"
level=WARN msg="Error processing file, continuing with next" file="broken.go"
level=INFO msg="Indexing completed with partial success" successful_files=284 failed_files=2
```

#### Con Panic Recuperado

```
level=ERROR msg="CRITICAL: Panic recovered in embedSingle" panic="runtime error" text_length=1000
level=WARN msg="Failed to embed document, continuing with next" index=5 error="panic recovered"
level=WARN msg="Some embeddings failed but continuing" failed_count=1 success_rate="95.0%"
```

---

## Estadísticas de Resiliencia

### Métricas Reportadas

Al finalizar indexación:

```
level=INFO msg="Indexing completed with partial success"
    successful_files=286
    failed_files=0
    
level=WARN msg="Some embeddings failed during generation"
    failed_count=12
    total_count=1543
    success_rate="99.2%"
    
level=WARN msg="Some chunk embeddings failed during generation"
    failed_count=3
    total_count=87
    success_rate="96.6%"
```

### Interpretación

| Success Rate | Estado | Acción |
|--------------|--------|--------|
| 100% | ✅ Perfecto | Ninguna |
| 95-99% | ⚠️ Bueno | Revisar logs de fallos |
| 80-94% | ⚠️ Aceptable | Investigar causa |
| 50-79% | ❌ Problemático | Revisar configuración |
| <50% | ❌ Crítico | Revisar modelo/límites |

---

## Validaciones Pre-Embedding

Antes de llamar al modelo (C), se valida:

```go
// Validación exhaustiva antes de llamar a C
if len(text) == 0 {
    return nil, fmt.Errorf("empty text provided")
}

if len(text) > g.maxChars {
    slog.Warn("Text exceeds maxChars, truncating")
    text = text[:g.maxChars]
}

if g.model == nil {
    return nil, fmt.Errorf("model is nil")
}
```

**Previene**:
- Llamadas con texto vacío
- Overflow de tokens
- Null pointer dereference
- Datos inválidos a C

---

## Casos de Uso

### Caso 1: Proyecto con Archivos Problemáticos

**Escenario**: Proyecto con 300 archivos, 5 tienen errores de parsing

**Antes** ❌:
```
Processing file 1/300... OK
Processing file 2/300... OK
Processing file 3/300... ERROR
CRASH - Indexación abortada
```

**Ahora** ✅:
```
Processing file 1/300... OK
Processing file 2/300... OK
Processing file 3/300... ERROR (logged, continuing)
Processing file 4/300... OK
...
Processing file 300/300... OK

Result: 295/300 files indexed successfully (98.3%)
```

---

### Caso 2: Símbolo con Código Muy Largo

**Escenario**: Función con 10,000 líneas que excede límites

**Antes** ❌:
```
Generating embedding... 
GGML_ASSERT failed
SIGABRT
```

**Ahora** ✅:
```
level=WARN msg="Text exceeds limit, truncating" 
    original_length=50000 truncated_to=900
level=INFO msg="Embedding generated successfully (truncated)"
level=INFO msg="Symbol indexed with truncated content"
```

---

### Caso 3: Batch con Varios Fallos

**Escenario**: Batch de 10 símbolos, 2 fallan por timeout

**Antes** ❌:
```
Processing batch...
ERROR: timeout on symbol 3
Batch aborted
```

**Ahora** ✅:
```
Processing symbol 1... OK
Processing symbol 2... OK
Processing symbol 3... ERROR (timeout, continuing)
Processing symbol 4... OK
...
Processing symbol 10... OK

Result: 8/10 symbols embedded successfully (80%)
```

---

## Testing

### Simular Fallos

Para probar la resiliencia:

```go
// En gguf.go, agregar temporalmente:
func (g *GGUFEmbedder) embedSingle(...) {
    // Simular fallo aleatorio
    if rand.Float32() < 0.1 {  // 10% fail rate
        return nil, fmt.Errorf("simulated failure")
    }
    // ... código normal ...
}
```

**Output esperado**:
- 90% de embeddings exitosos
- Logs de fallos individuales
- Indexación completa exitosa
- Estadísticas mostrando 90% success rate

### Simular Panic

```go
// En gguf.go:
func (g *GGUFEmbedder) embedSingle(...) {
    if len(text) > 500 {
        panic("simulated panic for testing")
    }
    // ... código normal ...
}
```

**Output esperado**:
- Panic capturado y loggeado
- Stack trace en logs
- Conversión a error
- Procesamiento continúa

---

## Beneficios

### ✅ Robustez
- Sistema no se detiene por errores individuales
- Recuperación automática de panics
- Datos parciales mejor que nada

### ✅ Visibilidad
- Logs detallados de cada fallo
- Estadísticas de success rate
- Stack traces para debugging

### ✅ Productividad
- No necesita reiniciar indexación completa
- Progreso se preserva
- Solo re-indexar archivos fallidos

### ✅ Debugging
- Logs estructurados facilitan análisis
- Stack traces ayudan a encontrar causa
- Métricas ayudan a identificar problemas sistémicos

---

## Limitaciones

### ⚠️ Embeddings Nulos

Símbolos sin embeddings:
- No aparecerán en búsqueda semántica
- Sí aparecerán en búsqueda por nombre/path
- Se pueden re-indexar posteriormente

### ⚠️ Datos Parciales

Si falla guardar símbolos:
- Archivo se marca como no procesado
- Se debe re-indexar completo
- No hay recuperación parcial en DB

### ⚠️ Panics de C

Algunos panics de C no son recuperables:
- SIGSEGV (segmentation fault)
- Errores de memoria críticos
- En estos casos el sistema aún terminará

---

## Monitoreo Recomendado

### Métricas Clave

1. **Success Rate Global**: >95% es bueno
2. **Archivos Fallidos**: <5% es aceptable
3. **Embeddings Fallidos**: <1% es esperado
4. **Panics Recuperados**: 0 es ideal

### Alertas

Configurar alertas si:
- Success rate <80%
- >10% de archivos fallan
- Se recuperan >5 panics por indexación
- Tiempo de procesamiento >2x normal

---

## Archivos Modificados

```
M  pkg/embedder/gguf.go                       (+78: panic recovery, logging)
M  internal/indexer/indexer_embeddings.go     (+67: partial success handling)
M  internal/indexer/indexer.go                (+45: file-level recovery)
A  docs/fixes/error-recovery-2026-01-12.md    (este archivo)
```

---

## Mejoras Futuras

- [ ] Retry automático con backoff exponencial
- [ ] Guardar lista de símbolos fallidos para re-intentar
- [ ] Métricas exportables a Prometheus/Grafana
- [ ] Dashboard de health del indexer
- [ ] Recuperación de checkpoints (reanudar indexación)

---

**Fecha**: 2026-01-12  
**Autor**: Implementación de error recovery robusto  
**Estado**: ✅ IMPLEMENTADO, COMPILADO Y DOCUMENTADO

**Filosofía**: "The show must go on" - Continuar procesando pase lo que pase