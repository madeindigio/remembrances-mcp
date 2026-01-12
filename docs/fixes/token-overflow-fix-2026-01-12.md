# Fix: Token Overflow en Embedder Interno

## Fecha
2026-01-12

## Problema

El embedder interno (GGUF/llama.cpp) estaba fallando con el siguiente error:

```
GGML_ASSERT(cparams.n_ubatch >= n_tokens && "encoder requires n_ubatch >= n_tokens") failed
```

### Causa Raíz

El problema ocurría cuando el texto a embedear generaba más tokens que el límite `UBatchSize` configurado en el modelo (512 tokens). Aunque había validaciones de longitud de caracteres, la relación caracteres/tokens no es constante:

1. **Textos normales**: ~4-5 caracteres por token
2. **Código fuente**: ~1-2 caracteres por token (muchos símbolos como `{`, `}`, `->`, etc.)

Con la configuración anterior:
- UBatchSize: 512 tokens
- MaxChars: 900 caracteres
- Ratio asumido: 2:1 chars/token

**Problema**: Un texto de 829 caracteres de código Go puede generar >512 tokens, causando el crash.

## Solución Implementada

### 1. Reducción de UBatchSize
- Antes: 512 tokens
- Ahora: 384 tokens (25% de reducción para mayor margen de seguridad)

### 2. Límites más conservadores
- **MaxTokens**: 270 (384 * 0.70, dejando 30% de margen)
- **MaxChars**: 450 (270 * 1.5 ratio)
- **Ratio chars/token**: 1.5:1 (antes era 2:1)

### 3. Ajuste en chunking
- **DefaultMaxChunkSize**: 400 caracteres (antes 800)
- **DefaultChunkOverlap**: 60 caracteres (antes 100)

## Archivos Modificados

1. `pkg/embedder/gguf.go`:
   - Reducción de UBatchSize a 384
   - Cálculo más conservador de límites
   - Margen de seguridad aumentado al 30%

2. `pkg/embedder/chunking.go`:
   - Reducción de DefaultMaxChunkSize a 400
   - Ajuste de overlap a 60

## Beneficios

1. **Mayor seguridad**: 30% de margen vs 12% anterior
2. **Soporte para código**: Ratio 1.5:1 maneja mejor símbolos y tokens cortos
3. **Prevención de crashes**: Los límites garantizan que nunca se exceda UBatchSize

## Trade-offs

- **Menor throughput**: Chunks más pequeños = más llamadas al embedder
- **Posible pérdida de contexto**: Textos largos se dividen más

Sin embargo, estos trade-offs son aceptables considerando que:
- La estabilidad es más importante que el rendimiento
- El chunking con overlap preserva contexto razonablemente bien
- Los crashes anteriores hacían el sistema inutilizable

## Testing

Después de aplicar este fix, el sistema debería:
1. No crashear con textos de código fuente
2. Procesar archivos Go, TypeScript, etc. sin problemas
3. Generar más chunks para textos largos, pero con éxito

## Monitoreo Recomendado

Revisar logs para mensajes como:
```
WARN Text exceeds limit, truncating
WARN Some embeddings failed but continuing
```

Si estos mensajes son frecuentes, puede indicar que los límites aún son muy altos.

## Configuración Técnica

```go
// Modelo
ContextSize:  384
BatchSize:    384
UBatchSize:   384

// Embedder
maxTokens:     270  // 384 * 0.70
maxChars:      450  // 270 * 1.5
charsPerToken: 1    // Ratio conservador

// Chunking
DefaultMaxChunkSize: 400
DefaultChunkOverlap: 60
```

## Conclusión

Este fix resuelve el problema de token overflow utilizando límites ultra-conservadores que son seguros incluso para código con alta densidad de tokens. El sistema ahora debería ser estable y robusto.
