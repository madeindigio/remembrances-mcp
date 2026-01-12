# Tree-sitter Lua Warning Fix

## Issue
Al compilar remembrances-mcp, se mostraba el siguiente warning del parser de Lua:

```
parser.c:254:17: warning: null character(s) preserved in literal
  254 |   [anon_sym_] = " ",
      |                 ^
```

## Root Cause
El warning era causado por un carácter nulo en un literal de cadena en el archivo `parser.c` generado por tree-sitter para el lenguaje Lua. Este archivo es parte del paquete `github.com/madeindigio/go-tree-sitter/lua`.

## Solution
El maintainer del fork (Jose F. Rives Lirola / digiogithub) solucionó el problema en el commit:

- **Commit**: `85069ecc07acf200906cf61866b656ac54492ff6`
- **Fecha**: January 12, 2026 at 14:49:30 UTC+01:00
- **Mensaje**: "fix: Remove warning lua grammar"

## Changes Applied

### Updated go.mod
```go
github.com/madeindigio/go-tree-sitter v0.0.0-20260112134930-85069ecc07ac
```

Cambió desde: `v0.0.0-20260112132453-61cbcfc43e18`

### Build Result
✅ **Build completado sin warnings**
✅ **Binario funcional**

## Verification Steps
```bash
cd /www/MCP/remembrances-mcp
go mod tidy
go build -mod=mod -o /tmp/remembrances-mcp ./cmd/remembrances-mcp
# Output: Sin warnings ni errores
```

## Notes
- No fue necesario agregar flags de compilación adicionales
- El fix fue aplicado directamente en el upstream del fork
- El archivo temporal `pkg/treesitter/cgo_flags.go` que se había creado fue eliminado ya que no es necesario
