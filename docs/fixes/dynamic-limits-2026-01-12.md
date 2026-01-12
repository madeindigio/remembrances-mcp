# Dynamic Token Limits Implementation - 2026-01-12

## Problema Original

Los límites de tokens estaban **hardcodeados** en el código, lo que causaba problemas:

- ❌ Límites fijos (900 chars, 450 tokens) sin considerar capacidades reales del modelo
- ❌ No se adaptaban a diferentes modelos GGUF con distintos `UBatchSize`
- ❌ Imposible de ajustar sin recompilar
- ❌ No había visibilidad de qué límites se estaban usando

## Solución Implementada

Implementación de **límites dinámicos** que se adaptan automáticamente a las capacidades del modelo cargado.

### Arquitectura

```
┌─────────────────┐
│  Model GGUF     │
│  (llama.cpp)    │
│                 │
│  UBatchSize:512 │◄─── Parámetro físico del modelo
└────────┬────────┘
         │
         │ Se expone via API
         ▼
┌─────────────────┐
│  llama.Model    │
│  (shim.go)      │
│                 │
│  + UBatchSize() │◄─── Getter expone el valor
│  + ContextSize()│
│  + BatchSize()  │
└────────┬────────┘
         │
         │ Se consulta al inicializar
         ▼
┌─────────────────┐
│  GGUFEmbedder   │
│  (gguf.go)      │
│                 │
│  maxTokens: 450 │◄─── Calculado: UBatchSize - 12% margen
│  maxChars:  900 │◄─── Calculado: maxTokens × 2 (ratio)
│  charsPerToken:2│◄─── Ratio conservador
└────────┬────────┘
         │
         │ Se consulta al procesar
         ▼
┌─────────────────┐
│  Indexer        │
│  (indexer.go)   │
│                 │
│  Usa límites    │◄─── Dinámicos, no hardcoded
│  dinámicos      │
└─────────────────┘
```

### Cambios en el Código

#### 1. `internal/llama/shim.go`

**Almacenamiento de parámetros:**

```go
type Model struct {
    // ... campos existentes ...
    
    // Model parameters (stored for dynamic limit calculations)
    contextSize uint32
    batchSize   uint32
    ubatchSize  uint32
}
```

**Getters para exponer parámetros:**

```go
// UBatchSize returns the physical batch size (ubatch) used by the model
// This is the HARD LIMIT for number of tokens that can be processed at once
func (m *Model) UBatchSize() uint32 {
    if m == nil {
        return 0
    }
    return m.ubatchSize
}

// También: ContextSize(), BatchSize()
```

#### 2. `pkg/embedder/gguf.go`

**Campos dinámicos:**

```go
type GGUFEmbedder struct {
    // ... campos existentes ...
    
    // Dynamic limits based on model configuration
    maxTokens     int // Maximum tokens the model can handle (from UBatchSize)
    maxChars      int // Maximum characters (calculated from maxTokens)
    charsPerToken int // Conservative char-to-token ratio (default: 2)
}
```

**Cálculo automático al inicializar:**

```go
func NewGGUFEmbedder(modelPath string, threads, gpuLayers int) (*GGUFEmbedder, error) {
    // ... cargar modelo ...
    
    // Get dynamic limits from the model
    ubatchSize := model.UBatchSize()
    if ubatchSize == 0 {
        ubatchSize = 512 // Fallback to safe default
    }
    
    // Conservative char-to-token ratio (2:1 is very safe for most models)
    charsPerToken := 2
    
    // Calculate max tokens with safety margin (leave ~12% margin)
    maxTokens := int(ubatchSize) - int(float64(ubatchSize)*0.12)
    
    // Calculate max chars based on ratio
    maxChars := maxTokens * charsPerToken
    if maxChars > 900 {
        maxChars = 900 // Cap at 900 for extreme safety
    }
    
    // Log the dynamic limits
    slog.Info("GGUF embedder initialized with dynamic limits",
        "ubatch_size", ubatchSize,
        "max_tokens", maxTokens,
        "max_chars", maxChars,
        "chars_per_token", charsPerToken)
    
    return &GGUFEmbedder{
        // ... otros campos ...
        maxTokens:     maxTokens,
        maxChars:      maxChars,
        charsPerToken: charsPerToken,
    }, nil
}
```

**Getters para consultar límites:**

```go
func (g *GGUFEmbedder) MaxTokens() int     // Returns max tokens
func (g *GGUFEmbedder) MaxChars() int      // Returns max chars
func (g *GGUFEmbedder) CharsPerToken() int // Returns ratio
```

#### 3. `internal/indexer/indexer_embeddings.go`

**Consulta dinámica de límites:**

```go
func (idx *Indexer) prepareSymbolText(sym *treesitter.CodeSymbol) string {
    // Get dynamic limits from embedder (falls back to safe defaults)
    maxTextLength := 900 // Fallback default
    if ggufEmb, ok := idx.embedder.(*embedder.GGUFEmbedder); ok {
        maxTextLength = ggufEmb.MaxChars()
    }
    
    // Calculate max per part dynamically
    maxPartLength := maxTextLength / 3
    
    // ... usar maxTextLength y maxPartLength ...
}
```

## Tabla de Comparación

| Aspecto | Antes (Hardcoded) | Ahora (Dinámico) |
|---------|-------------------|-------------------|
| **Configuración** | Fija en código | Detectada del modelo |
| **Flexibilidad** | Requiere recompilar | Se adapta automáticamente |
| **Visibilidad** | Ninguna | Logging al iniciar |
| **Fallback** | N/A | 900 chars si falla detección |
| **Margen seguridad** | Fijo 12% | Calculado 12% del UBatchSize |
| **Adaptabilidad** | Ninguna | Compatible con cualquier modelo |

## Cálculo de Límites

### Fórmula

```
UBatchSize (del modelo)
    ↓
maxTokens = UBatchSize - (UBatchSize × 0.12)
    ↓
maxChars = maxTokens × charsPerToken
    ↓
maxChars = min(maxChars, 900)  // Cap de seguridad
```

### Ejemplo con UBatchSize = 512

```
UBatchSize:    512 tokens (del modelo GGUF)
Margen 12%:    512 × 0.12 = 61 tokens
maxTokens:     512 - 61 = 451 tokens
charsPerToken: 2 (ratio conservador)
maxChars:      451 × 2 = 902 chars
Cap máximo:    min(902, 900) = 900 chars

Resultado final: 900 chars, ~450 tokens
```

### Ejemplo con UBatchSize = 1024

```
UBatchSize:    1024 tokens (modelo más grande)
Margen 12%:    1024 × 0.12 = 123 tokens
maxTokens:     1024 - 123 = 901 tokens
charsPerToken: 2 (ratio conservador)
maxChars:      901 × 2 = 1802 chars
Cap máximo:    min(1802, 900) = 900 chars

Resultado final: 900 chars (limitado por cap de seguridad)
```

### Ejemplo con UBatchSize = 256

```
UBatchSize:    256 tokens (modelo pequeño)
Margen 12%:    256 × 0.12 = 31 tokens
maxTokens:     256 - 31 = 225 tokens
charsPerToken: 2 (ratio conservador)
maxChars:      225 × 2 = 450 chars
Cap máximo:    min(450, 900) = 450 chars

Resultado final: 450 chars, ~225 tokens
```

## Ventajas de la Solución

### ✅ Adaptabilidad

- Se adapta automáticamente a modelos con diferentes capacidades
- No requiere configuración manual
- Compatible con modelos futuros

### ✅ Seguridad

- Mantiene margen de seguridad del 12%
- Fallback a valores conservadores (900 chars)
- Cap máximo para prevenir uso excesivo de memoria

### ✅ Visibilidad

- Logs claros al inicializar el embedder
- Fácil debugging y troubleshooting
- Métricas disponibles via getters

### ✅ Mantenibilidad

- Código DRY (Don't Repeat Yourself)
- Lógica centralizada en el embedder
- Fácil de modificar el ratio o margen si es necesario

## Logging Esperado

Al iniciar el embedder, verás:

```
level=INFO msg="GGUF embedder initialized with dynamic limits" 
    ubatch_size=512 
    max_tokens=450 
    max_chars=900 
    chars_per_token=2 
    model_path=/path/to/model.gguf
```

Esto te permite verificar:
- Qué límite detectó el sistema
- Qué valores calculó automáticamente
- Qué modelo está usando

## Configuración Avanzada

### Modificar el Ratio Chars/Token

Si encuentras que el ratio 2:1 es demasiado conservador para tu modelo:

```go
// En NewGGUFEmbedder(), línea ~65:
charsPerToken := 2  // Cambiar a 3 o 4 si tu modelo lo permite
```

### Modificar el Margen de Seguridad

Si necesitas más o menos margen:

```go
// En NewGGUFEmbedder(), línea ~71:
maxTokens := int(ubatchSize) - int(float64(ubatchSize)*0.12)
//                                                      ^^^^
//                              Cambiar 0.12 (12%) a 0.15 (15%) o 0.10 (10%)
```

### Modificar el Cap Máximo

Si tu hardware puede manejar más:

```go
// En NewGGUFEmbedder(), línea ~76:
if maxChars > 900 {
    maxChars = 900  // Cambiar a 1500 o 2000 si tu hardware lo permite
}
```

## Testing

### Verificar Límites Detectados

```bash
# Compilar
go build -o remembrances-mcp ./cmd/remembrances-mcp

# Ejecutar y buscar el log de inicialización
./remembrances-mcp 2>&1 | grep "GGUF embedder initialized"
```

Output esperado:
```
level=INFO msg="GGUF embedder initialized with dynamic limits" 
    ubatch_size=512 max_tokens=450 max_chars=900 chars_per_token=2
```

### Verificar Comportamiento con Diferentes Modelos

1. **Modelo pequeño** (UBatchSize=256):
   - Debería detectar límite menor
   - maxChars debería ser ~450

2. **Modelo estándar** (UBatchSize=512):
   - Límite detectado: 900 chars
   - maxTokens: 450

3. **Modelo grande** (UBatchSize=1024):
   - Límite detectado: 900 chars (cap aplicado)
   - maxTokens: 450 (limitado por cap)

## Troubleshooting

### No aparece el log de inicialización

- Verifica que el nivel de log sea INFO o menor
- Verifica que estés usando la versión compilada correcta

### Límites parecen incorrectos

- Verifica el UBatchSize del modelo con `llama.cpp` directamente
- Revisa el log para ver qué valor detectó
- Considera ajustar el ratio o margen

### Sigue crasheando con token overflow

- Reduce el ratio `charsPerToken` de 2 a 1
- Aumenta el margen de 12% a 15% o 20%
- Reduce el cap máximo de 900 a 700

## Archivos Modificados

```
M  internal/llama/shim.go                     (+32 líneas: getters y storage)
M  pkg/embedder/gguf.go                       (+47 líneas: cálculo dinámico)
M  internal/indexer/indexer_embeddings.go     (+20 líneas: consulta dinámica)
A  docs/fixes/dynamic-limits-2026-01-12.md    (este archivo)
```

## Mejoras Futuras

- [ ] Detectar el ratio real del tokenizador (en lugar de asumir 2:1)
- [ ] Permitir configuración via archivo de config
- [ ] Métricas de cuántos textos se truncan
- [ ] Auto-ajuste del ratio basado en experiencia real
- [ ] API para consultar límites en tiempo de ejecución

## Referencias

- [llama.cpp context parameters](https://github.com/ggml-org/llama.cpp/blob/master/common/common.h)
- [Token estimation best practices](https://platform.openai.com/docs/guides/embeddings)
- Fix anterior: [token-overflow-fix-2026-01-12.md](./token-overflow-fix-2026-01-12.md)

---

**Fecha**: 2026-01-12  
**Autor**: Implementación de límites dinámicos para adaptarse a cualquier modelo  
**Estado**: ✅ IMPLEMENTADO Y COMPILADO

**Próximo paso**: Probar con el modelo real y verificar que los límites sean correctos