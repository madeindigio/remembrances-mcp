# Soporte Dual de Transportes HTTP JSON API y MCP Streamable HTTP

**Fecha**: 21 de enero de 2026  
**Tipo**: Feature Implementation  
**Módulo**: Transport Layer

## Problema Identificado

El módulo comercial `webui` se inicializaba correctamente en el puerto configurado en `http-addr`, pero la configuración de `mcp-http` y `mcp-http-addr` era ignorada. Esto se debía a que la lógica de selección de transporte en `main.go` usaba un bloque `if-else if-else`, permitiendo solo un transporte a la vez:

- Si `http: true` → Solo HTTP JSON API
- Si `mcp-http: true` → Solo MCP Streamable HTTP  
- Ninguno → stdio (default)

## Solución Implementada

Se modificó la lógica de inicialización de transportes en [cmd/remembrances-mcp/main.go](cmd/remembrances-mcp/main.go) para permitir la **ejecución simultánea** de ambos transportes:

### Cambios Principales

1. **Separación de la lógica de transportes** (líneas 159-219):
   - Se eliminó el `if-else if` que hacía mutuamente excluyentes HTTP y MCP-HTTP
   - Se configura MCP Streamable HTTP de forma independiente cuando `cfg.MCPStreamableHTTP` es true
   - El HTTP JSON API se configura después, independientemente del MCP transport

2. **Nueva lógica de ejecución** (líneas 493-527):
   ```go
   hasHTTP := cfg.HTTP && httpTransport != nil
   hasMCPHTTP := mcpHTTPTransport != nil
   
   if hasHTTP && hasMCPHTTP {
       // Ambos activos: MCP-HTTP en goroutine, HTTP JSON API bloqueante
       go func() {
           if err := srv.Run(); err != nil {
               slog.Error("MCP Streamable HTTP server error", "error", err)
           }
       }()
       if err := httpTransport.Start(); err != nil && err != http.ErrServerClosed {
           slog.Error("HTTP JSON API transport server error", "error", err)
           os.Exit(1)
       }
   } else if hasHTTP {
       // Solo HTTP JSON API
   } else {
       // Solo MCP (stdio o HTTP)
   }
   ```

3. **Actualización de configuración**: Se habilitó `mcp-http: true` en el archivo de configuración.

## Resultado

Ahora el servidor puede ejecutar simultáneamente:

- **Puerto 8080**: HTTP JSON API + Web UI comercial en `/admin`
- **Puerto 3000**: MCP Streamable HTTP en `/mcp`

### Verificación

```bash
$ ss -tlnp | grep -E ':(3000|8080)'
LISTEN 0 4096 *:3000 *:*    users:(("remembrances-mc",pid=104406,fd=2818))
LISTEN 0 4096 *:8080 *:*    users:(("remembrances-mc",pid=104406,fd=2817))
```

### Configuración de Ejemplo

```yaml
# HTTP JSON API (incluye módulos web como commercial_webui)
http: true
http-addr: ":8080"

# MCP Streamable HTTP (protocolo MCP estándar)
mcp-http: true
mcp-http-addr: "3000"
mcp-http-endpoint: "/mcp"

modules:
  commercial_webui:
    enabled: true
    config:
      port: "8080"  # Nota: este valor no se usa, usa http-addr
```

## Beneficios

1. **Compatibilidad completa**: Soporta clientes MCP estándar y aplicaciones web personalizadas
2. **Flexibilidad**: Cada transporte puede habilitarse/deshabilitarse independientemente
3. **Sin breaking changes**: La configuración existente sigue funcionando
4. **Mejor logging**: Mensajes claros indicando qué transportes están activos

## Archivos Modificados

- [cmd/remembrances-mcp/main.go](cmd/remembrances-mcp/main.go#L159-L527)
- [.ai/remembrances.config.yaml](.ai/remembrances.config.yaml#L14-L20)

## Notas Técnicas

- El transporte MCP-HTTP se ejecuta en una goroutine cuando ambos están activos
- El HTTP JSON API es bloqueante (main thread) para mantener el proceso vivo
- El shutdown es coordinado para ambos transportes
- No hay conflicto de puertos ya que cada transporte usa su propio puerto configurado
