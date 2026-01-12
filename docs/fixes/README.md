# Fixes Documentation

Este directorio contiene documentación sobre fixes críticos aplicados al proyecto Remembrances MCP.

## Índice de Fixes

### 2026-01-12: Token Overflow en Indexación

**Archivo:** [token-overflow-fix-2026-01-12.md](./token-overflow-fix-2026-01-12.md)

**Problema:** Crash durante indexación con error `GGML_ASSERT(cparams.n_ubatch >= n_tokens)` 

**Solución:** 
- Incrementar límites de contexto de 2048 a 8192 tokens
- Ajustar límites de caracteres con ratio conservador 3:1
- Agregar protecciones en múltiples capas

**Archivos modificados:**
- `pkg/embedder/gguf.go`
- `internal/indexer/indexer_embeddings.go`

**Estado:** ✅ Aplicado y compilado exitosamente

---

## Aplicar un Fix

Si encuentras un error relacionado con uno de los fixes documentados:

1. **Verifica que el fix esté aplicado:**
   ```bash
   git log --oneline --grep="token overflow"
   ```

2. **Si no está aplicado, revisa los archivos modificados:**
   ```bash
   git diff HEAD -- pkg/embedder/gguf.go internal/indexer/indexer_embeddings.go
   ```

3. **Recompila el proyecto:**
   ```bash
   cd remembrances-mcp
   go build -o remembrances-mcp ./cmd/remembrances-mcp
   ```

4. **Ejecuta tests si están disponibles:**
   ```bash
   go test ./pkg/embedder/... -v
   ```

## Reportar un Nuevo Problema

Si encuentras un nuevo bug crítico que requiere documentación:

1. Crea un archivo en este directorio: `nombre-descriptivo-YYYY-MM-DD.md`
2. Usa el template del fix de token overflow como referencia
3. Incluye:
   - Descripción del problema
   - Análisis de la causa raíz
   - Solución implementada
   - Impacto y consideraciones
   - Instrucciones de testing
   - Referencias

## Convenciones

- **Nombres de archivo:** `descripcion-del-fix-YYYY-MM-DD.md`
- **Formato:** Markdown con secciones claras
- **Código:** Incluir snippets de antes/después
- **Testing:** Siempre incluir cómo verificar el fix

## Contacto

Para preguntas sobre fixes específicos, revisar los commits relacionados en el repositorio.