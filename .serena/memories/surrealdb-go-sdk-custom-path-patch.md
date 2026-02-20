# Parche custom del SDK SurrealDB Go: soporte para URL con path prefix

## Versiones afectadas

- **SDK**: `github.com/surrealdb/surrealdb.go` **v1.0.0**
- **SurrealDB Server**: 2.3.10
- **Go**: 1.23.4

## Problema

El SDK oficial de SurrealDB para Go (v1.0.0) **descarta el path de la URL** al conectar vía WebSocket o HTTP. Esto impide conectar a instancias de SurrealDB que estén detrás de un reverse proxy con un path prefix (por ejemplo `wss://host/surreal/`).

### Causa raíz en el SDK

En `pkg/connection/config.go`, la función `NewConfig` construye `BaseURL` usando solo scheme + host:

```go
BaseURL: fmt.Sprintf("%s://%s", u.Scheme, u.Host)
```

Luego en `pkg/connection/gorillaws/connection.go`, el método `Connect` siempre añade `/rpc` al BaseURL:

```go
conn, res, err := DefaultDialer.DialContext(ctx, fmt.Sprintf("%s/rpc", c.BaseURL), nil)
```

Por tanto, una URL como `wss://gpu01.digio.es/surreal/` se convierte en `wss://gpu01.digio.es/rpc` en vez de `wss://gpu01.digio.es/surreal/rpc`.

### Comportamiento observado

- **Surrealist** (cliente GUI) funciona porque permite especificar la URL completa incluyendo el path.
- **El SDK Go** falla la conexión porque intenta conectar a `/rpc` directamente en el host, sin el path prefix.

## Solución implementada

Se creó la función `ConnectRemoteSurrealDB` en `internal/storage/surrealdb.go` que reemplaza a `surrealdb.FromEndpointURLString`. Esta función:

1. Parsea la URL con `url.ParseRequestURI`
2. Crea la configuración con `connection.NewConfig(u)` (del SDK)
3. **Restaura el path** en `conf.BaseURL` si la URL tiene un path prefix
4. Crea la conexión gorillaws o http según el scheme
5. Conecta mediante `surrealdb.FromConnection(ctx, con)`

### Código de la función

```go
func ConnectRemoteSurrealDB(ctx context.Context, connectionURL string) (*surrealdb.DB, error) {
    u, err := url.ParseRequestURI(connectionURL)
    if err != nil {
        return nil, fmt.Errorf("failed to parse URL: %w", err)
    }

    conf := connection.NewConfig(u)

    // Preserve path prefix for reverse proxy setups.
    if u.Path != "" && u.Path != "/" && u.Path != "/rpc" {
        path := strings.TrimSuffix(u.Path, "/")
        path = strings.TrimSuffix(path, "/rpc")
        if path != "" {
            conf.BaseURL = fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, path)
        }
    }

    if confErr := conf.Validate(); confErr != nil {
        return nil, fmt.Errorf("invalid connection config: %w", confErr)
    }

    var con connection.Connection
    switch u.Scheme {
    case "http", "https":
        con = sdkhttp.New(conf)
    case "ws", "wss":
        con = gorillaws.New(conf)
    default:
        return nil, fmt.Errorf("unsupported URL scheme: %s", u.Scheme)
    }

    return surrealdb.FromConnection(ctx, con)
}
```

### Imports necesarios

```go
import (
    "net/url"
    "strings"
    "github.com/surrealdb/surrealdb.go"
    "github.com/surrealdb/surrealdb.go/pkg/connection"
    "github.com/surrealdb/surrealdb.go/pkg/connection/gorillaws"
    sdkhttp "github.com/surrealdb/surrealdb.go/pkg/connection/http"
)
```

## Archivos modificados

- **`internal/storage/surrealdb.go`**: Se añadió la función `ConnectRemoteSurrealDB` y se reemplazó la llamada a `surrealdb.FromEndpointURLString()` en el método `Connect`.
- **`modules/commercial/db-sync-server/connection.go`**: Se actualizó para usar `storage.ConnectRemoteSurrealDB()` en lugar de `surrealdb.FromEndpointURLString()`.

## Tabla de comportamiento

| URL en config                      | SDK original (BaseURL)     | Con el parche (BaseURL)           | Dial final (WebSocket)             |
|------------------------------------|----------------------------|-----------------------------------|------------------------------------|
| `wss://gpu01.digio.es/surreal/`    | `wss://gpu01.digio.es`     | `wss://gpu01.digio.es/surreal`    | `wss://gpu01.digio.es/surreal/rpc` |
| `ws://localhost:8000`              | `ws://localhost:8000`      | `ws://localhost:8000`             | `ws://localhost:8000/rpc`          |
| `wss://host:8000/rpc`             | `wss://host:8000`          | `wss://host:8000`                 | `wss://host:8000/rpc`             |
| `https://host/db/`                | `https://host`             | `https://host/db`                 | N/A (HTTP, usa BaseURL directo)    |

## Notas

- Las URLs **sin path custom** siguen funcionando exactamente igual que antes.
- Si el path termina en `/rpc`, se trata como una URL directa al endpoint RPC y no se duplica.
- El parche no modifica el SDK en sí (no usa `replace` en go.mod), sino que evita usar `FromEndpointURLString` y construye la conexión manualmente usando las APIs públicas del SDK.
- Si se actualiza el SDK a una versión futura que corrija este comportamiento, la función `ConnectRemoteSurrealDB` seguirá siendo compatible ya que usa las interfaces públicas `connection.Connection` y `surrealdb.FromConnection`.
