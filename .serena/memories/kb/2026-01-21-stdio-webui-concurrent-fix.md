# Fix: Modo stdio no funcionaba con web-ui activa

**Fecha**: 21 de enero de 2026  
**Tipo**: Bug Fix  
**Módulo**: Transport Layer  
**Relacionado con**: [2026-01-21-dual-transport-http-mcp-support.md](2026-01-21-dual-transport-http-mcp-support.md)

## Problema Identificado

Después de implementar el soporte dual de transportes HTTP JSON API y MCP Streamable HTTP, se descubrió un nuevo problema: cuando **solo** el transporte HTTP JSON API estaba activo (para la web-ui) junto con el modo por defecto stdio, el servidor MCP no iniciaba correctamente el transporte stdio.

### Escenario Problemático

Configuración:
```yaml
http: true
http-addr: "8080"
mcp-http: false  # o no configurado

modules:
  commercial_webui:
    enabled: true
```

**Comportamiento observado:**
- El HTTP JSON API se iniciaba correctamente en el puerto 8080
- La web-ui era accesible en `http://localhost:8080/admin`
- **Pero el transporte stdio MCP nunca se ejecutaba**, imposibilitando el uso del servidor MCP vía stdio

## Análisis de Causa Raíz

En [cmd/remembrances-mcp/main.go](cmd/remembrances-mcp/main.go), la lógica de ejecución de transportes (líneas 493-534) tenía tres ramas:

```go
if hasHTTP && hasMCPHTTP {
    // Caso 1: Ambos transportes HTTP activos
    // ✓ Ejecuta srv.Run() en goroutine (MCP-HTTP)
    // ✓ Ejecuta httpTransport.Start() bloqueante
} else if hasHTTP {
    // Caso 2: Solo HTTP JSON API activo
    // ✗ Solo ejecuta httpTransport.Start()
    // ✗ NUNCA ejecuta srv.Run() → stdio no funciona
} else {
    // Caso 3: Solo MCP (stdio o MCP-HTTP)
    // ✓ Ejecuta srv.Run()
}
```

**El problema estaba en el Caso 2**: cuando solo HTTP estaba activo, se omitía la ejecución de `srv.Run()`, que es el método que inicia el servidor MCP con el transporte configurado (en este caso, stdio).

## Solución Implementada

Se modificó la rama `else if hasHTTP` para ejecutar **ambos** transportes concurrentemente:

```go
} else if hasHTTP {
    // Only HTTP JSON API is enabled, but stdio MCP transport should also run
    slog.Info("Starting HTTP JSON API transport with stdio MCP transport")
    
    // Start MCP server (stdio) in background
    go func() {
        if err := srv.Run(); err != nil {
            slog.Error("MCP stdio server error", "error", err)
        }
    }()
    
    // Run HTTP JSON API (blocking)
    if err := httpTransport.Start(); err != nil && err != http.ErrServerClosed {
        slog.Error("HTTP JSON API transport server error", "error", err)
        os.Exit(1)
    }
}
```

### Cambios Clave

1. **Goroutine para stdio**: Se ejecuta `srv.Run()` en una goroutine para no bloquear el hilo principal
2. **HTTP JSON API bloqueante**: Se mantiene `httpTransport.Start()` en el hilo principal para mantener el proceso vivo
3. **Logging descriptivo**: Se actualizó el mensaje de log para reflejar que ambos transportes están activos

## Matriz de Configuraciones Soportadas

Después del fix, todas las combinaciones funcionan correctamente:

| http | mcp-http | Resultado |
|------|----------|-----------|
| false | false | ✓ Solo stdio (default) |
| false | true | ✓ Solo MCP-HTTP (puerto 3000) |
| true | false | ✓ **HTTP JSON API + stdio** (fix aplicado) |
| true | true | ✓ HTTP JSON API + MCP-HTTP (ambos puertos) |

## Verificación

### Prueba 1: Solo HTTP + stdio
```yaml
http: true
http-addr: "8080"
```

Resultado esperado:
- Puerto 8080 activo para HTTP JSON API y web-ui
- stdio disponible para clientes MCP vía stdin/stdout

### Prueba 2: HTTP + MCP-HTTP + stdio
```yaml
http: true
http-addr: "8080"
mcp-http: true
mcp-http-addr: "3000"
```

Resultado esperado:
- Puerto 8080 para HTTP JSON API
- Puerto 3000 para MCP-HTTP
- stdio **no disponible** (reemplazado por MCP-HTTP)

## Archivos Modificados

- [cmd/remembrances-mcp/main.go](cmd/remembrances-mcp/main.go#L515-L527)

## Lecciones Aprendidas

1. **Transportes independientes**: Cada transporte (stdio, MCP-HTTP, HTTP JSON API) debe poder ejecutarse independientemente
2. **Goroutines para concurrencia**: Cuando múltiples transportes están activos, usar goroutines para los no bloqueantes
3. **Tests de matriz**: Probar todas las combinaciones de configuración, no solo los casos principales
4. **Logging claro**: Los mensajes de log deben indicar exactamente qué transportes están activos

## Impacto

- **Sin breaking changes**: La configuración existente sigue funcionando
- **Mejora de UX**: Los usuarios pueden usar la web-ui y stdio simultáneamente
- **Flexibilidad**: Más opciones de deployment y desarrollo
