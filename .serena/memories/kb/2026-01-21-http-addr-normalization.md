# Normalización del Formato de Dirección HTTP

**Fecha**: 21 de enero de 2026  
**Tipo**: Code Improvement  
**Módulo**: Configuration / HTTP Transport

## Problema

La configuración `http-addr` requería especificar los dos puntos ":" antes del número de puerto (ej: `:8080`), lo cual no es estándar y difiere del formato usado en `mcp-http-addr` que acepta solo el número de puerto.

## Solución Implementada

Se aplicó la función `normalizeBindAddr` (que ya existía para MCP-HTTP) también al HTTPAddr, permitiendo especificar solo el número de puerto.

### Cambios en el Código

**Archivo**: [cmd/remembrances-mcp/main.go](cmd/remembrances-mcp/main.go#L410-L417)

```go
// Antes
if addr == "" {
    addr = ":8080"
}

// Después  
addr = normalizeBindAddr(addr, "8080")
```

La función `normalizeBindAddr` automáticamente:
- Añade ":" si solo se proporciona un número de puerto
- Mantiene el formato completo si se proporciona "host:port"
- Usa el valor por defecto si está vacío

### Formatos Aceptados

Ahora `http-addr` acepta cualquiera de estos formatos:

- `"8080"` → normalizado a `:8080`
- `":8080"` → mantiene `:8080`
- `"localhost:8080"` → mantiene `localhost:8080`
- `"0.0.0.0:8080"` → mantiene `0.0.0.0:8080`
- `""` (vacío) → usa default `:8080`

## Archivos Actualizados

1. **[cmd/remembrances-mcp/main.go](cmd/remembrances-mcp/main.go#L415)**: Aplicar normalización
2. **[config.sample.yaml](config.sample.yaml#L28)**: Actualizado ejemplo a `"8080"` con comentario explicativo
3. **[.ai/remembrances.config.yaml](.ai/remembrances.config.yaml#L27)**: Actualizado a `"8080"`
4. **[README.md](README.md)**: Actualizados todos los ejemplos de uso

## Ejemplos de Configuración

### YAML
```yaml
# Formato simplificado (recomendado)
http: true
http-addr: "8080"

# También funciona el formato anterior
http-addr: ":8080"

# O especificar host
http-addr: "localhost:8080"
```

### Línea de Comandos
```bash
# Nuevo formato estándar
./remembrances-mcp --http --http-addr="8080"

# Formato anterior (aún compatible)
./remembrances-mcp --http --http-addr=":8080"
```

## Beneficios

1. **Consistencia**: Mismo formato que `mcp-http-addr`
2. **Simplicidad**: No requiere recordar añadir ":"
3. **Compatibilidad**: Acepta ambos formatos (con y sin ":")
4. **Estándar**: Formato más común en otras aplicaciones
5. **Sin breaking changes**: El formato antiguo sigue funcionando

## Verificación

```bash
$ grep "address=" /tmp/remembrances-test.log
time=... level=INFO msg="MCP Streamable HTTP transport enabled" address=:3000 endpoint=/mcp
time=... level=INFO msg="Starting HTTP transport server" address=:8080

$ ss -tlnp | grep -E ':(3000|8080)'
LISTEN 0 4096 *:3000 *:*
LISTEN 0 4096 *:8080 *:*
```

Ambos puertos se normalizan correctamente a formato `:puerto` internamente.
