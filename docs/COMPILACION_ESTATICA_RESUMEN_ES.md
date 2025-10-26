# Resumen: Compilación Estática de kelindar/search

## Resumen Ejecutivo

Después de una investigación exhaustiva del proyecto [kelindar/search](https://github.com/kelindar/search), se ha determinado que **NO es posible compilar la librería nativa estáticamente dentro del binario Go** utilizando la arquitectura actual basada en `purego`.

## ¿Por Qué No Es Posible?

### Arquitectura Actual

```
┌─────────────────────────┐
│   Binario Go            │
│   (remembrances-mcp)    │
│                         │
│   ┌─────────────────┐   │
│   │    purego       │   │
│   │   dlopen()      │───┼───> libllama_go.so
│   └─────────────────┘   │      (filesystem)
│                         │
└─────────────────────────┘
```

1. **purego usa `dlopen()`**: Esta llamada del sistema requiere una **ruta de archivo** en el filesystem
2. **No acepta datos en memoria**: No se pueden cargar librerías desde arrays de bytes embebidos
3. **Carga en tiempo de ejecución**: La librería DEBE existir como archivo físico cuando el programa se ejecuta

### Limitaciones Técnicas

```go
// Código de kelindar/search/loader.go
func load(name string) (uintptr, error) {
    // dlopen() requiere una ruta de archivo válida
    return purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL)
}
```

- `Dlopen()` es una llamada de sistema que opera a nivel de SO
- No hay forma de pasarle datos embebidos
- Siempre busca un archivo `.so`/`.dylib`/`.dll` en el filesystem

## Lo Que SÍ Está Compilado Estáticamente

**Buenas noticias:** La librería `libllama_go.so` YA contiene todo compilado estáticamente dentro de ella:

```
libllama_go.so/dylib/dll
├── llama.cpp (estático) ✓
├── common library (estático) ✓
├── ggml (estático) ✓
└── Solo enlaza con librerías del sistema (libc, libstdc++)
```

### Verificación

```bash
# Linux - verificar dependencias
ldd libllama_go.so
# Salida típica:
#   linux-vdso.so.1
#   libstdc++.so.6 => /lib/x86_64-linux-gnu/libstdc++.so.6
#   libm.so.6 => /lib/x86_64-linux-gnu/libm.so.6
#   libc.so.6 => /lib/x86_64-linux-gnu/libc.so.6

# macOS
otool -L libllama_go.dylib
```

**Resultado:** Solo depende de librerías del sistema, NO de libllama o libcommon externas.

## Proceso de Compilación de kelindar/search

### CMakeLists.txt Estructura

```cmake
# 1. Dependencias se compilan como librerías ESTÁTICAS
set(BUILD_SHARED_LIBS OFF CACHE BOOL "Build static libraries" FORCE)
add_subdirectory(llama.cpp)        # → libllama.a
add_subdirectory(llama.cpp/common) # → libcommon.a

# 2. Librería final es COMPARTIDA pero enlaza estáticamente
add_library(llama_go SHARED ${SOURCES})
target_link_libraries(llama_go PRIVATE
    llama    # Enlace estático
    common   # Enlace estático
)
```

### Resultado

- **Entrada**: Múltiples archivos C++ y librerías estáticas
- **Salida**: UN SOLO archivo `.so`/`.dylib`/`.dll`
- **Contenido**: Todo el código de llama.cpp empaquetado dentro

## Estrategias de Distribución Disponibles

### ✅ Opción 1: Distribución Separada (RECOMENDADA)

**Distribuir dos archivos:**
- `remembrances-mcp` (binario Go)
- `libllama_go.so` (librería nativa)

**Instalación:**
```bash
# Usando el script de instalación
sudo ./scripts/install-with-library.sh

# O manualmente
sudo cp remembrances-mcp /usr/local/bin/
sudo cp libllama_go.so /usr/local/lib/
sudo ldconfig  # Linux
```

**Ventajas:**
- ✅ Separación limpia de componentes
- ✅ Fácil de actualizar
- ✅ Siguiendo mejores prácticas
- ✅ La librería puede ser compartida por múltiples programas

**Desventajas:**
- ❌ Dos archivos que gestionar
- ❌ Requiere instalación a nivel de sistema o configuración de PATH

### ✅ Opción 2: Extracción en Tiempo de Ejecución

**Embeber la librería en el binario Go y extraerla al arrancar:**

```go
package main

import (
    _ "embed"
    "os"
    "path/filepath"
)

//go:embed libllama_go.so
var nativeLib []byte

func init() {
    // Extraer a directorio temporal
    tmpDir := os.TempDir()
    libPath := filepath.Join(tmpDir, "libllama_go.so")
    
    // Escribir archivo
    os.WriteFile(libPath, nativeLib, 0755)
    
    // Configurar PATH para que purego lo encuentre
    os.Setenv("LD_LIBRARY_PATH", tmpDir+":"+os.Getenv("LD_LIBRARY_PATH"))
}
```

**Ventajas:**
- ✅ Un solo binario para distribuir
- ✅ Extracción automática
- ✅ Funciona con purego

**Desventajas:**
- ❌ Escribe en filesystem (directorio temporal)
- ❌ Binario más grande (~50-100 MB adicionales)
- ❌ Posibles problemas de permisos
- ❌ Consideraciones de seguridad (archivos temporales)
- ❌ Limpieza del archivo temporal

### ✅ Opción 3: Script Lanzador

**Crear un script wrapper:**

```bash
#!/bin/bash
# remembrances-mcp-launcher

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export LD_LIBRARY_PATH="${SCRIPT_DIR}:${LD_LIBRARY_PATH}"
exec "${SCRIPT_DIR}/remembrances-mcp" "$@"
```

**Ventajas:**
- ✅ Simple y transparente
- ✅ No necesita instalación del sistema
- ✅ Fácil de debuggear

**Desventajas:**
- ❌ Requiere shell script
- ❌ Tres archivos (script, binario, librería)

### ✅ Opción 4: Docker/Contenedores

**Dockerfile:**
```dockerfile
FROM ubuntu:22.04
COPY libllama_go.so /usr/local/lib/
COPY remembrances-mcp /usr/local/bin/
RUN ldconfig
ENTRYPOINT ["/usr/local/bin/remembrances-mcp"]
```

**Ventajas:**
- ✅ Distribución consistente
- ✅ Todas las dependencias incluidas
- ✅ Multiplataforma

**Desventajas:**
- ❌ Requiere Docker
- ❌ Mayor tamaño de distribución
- ❌ Overhead de contenedor

### ✅ Opción 5: Instaladores por Plataforma

**Crear paquetes nativos:**
- Linux: `.deb`, `.rpm`, AppImage
- macOS: `.pkg`, `.app` bundle
- Windows: `.msi`, instalador `.exe`

**Ventajas:**
- ✅ Experiencia nativa por plataforma
- ✅ Gestión automática de dependencias
- ✅ Integración con gestor de paquetes
- ✅ Profesional

**Desventajas:**
- ❌ Proceso de build más complejo
- ❌ Mantenimiento específico por plataforma

## Scripts Disponibles

### 1. Descargar Librería Nativa

```bash
# Descargar automáticamente la versión correcta para tu plataforma
./scripts/download-native-library.sh

# Descargar a directorio específico
./scripts/download-native-library.sh --dir /tmp

# Descargar versión específica
./scripts/download-native-library.sh --version v0.4.0
```

### 2. Instalar con Librería

```bash
# Instalación completa (binario + librería)
sudo ./scripts/install-with-library.sh

# Instalar en directorio personalizado
sudo ./scripts/install-with-library.sh --prefix /opt/remembrances-mcp
```

## Alternativa: Volver a CGO

**Si la compilación estática es absolutamente necesaria:**

```go
// Requeriría:
// #cgo LDFLAGS: -L. -lllama_go -lstdc++ -static
// #include "llama-go.h"
import "C"

func loadModel(path string) {
    C.load_model(C.CString(path))
}
```

**Intercambios:**
- ✅ Verdadero binario estático
- ✅ Sin dependencias externas
- ❌ Requiere compilador C++
- ❌ Cross-compilación compleja
- ❌ Builds más lentos
- ❌ Pierde los beneficios de purego

**Conclusión:** NO RECOMENDADO - contradice el propósito del proyecto

## Recomendaciones

### Para Desarrollo y Pruebas
**Usar Opción 3 (Script Lanzador):**
```bash
# Estructura del proyecto
proyecto/
├── remembrances-mcp          # Binario Go
├── libllama_go.so            # Librería nativa
└── run.sh                    # Script lanzador
```

### Para Producción
**Usar Opción 5 (Instaladores):**
- Crear paquetes `.deb`/`.rpm` para Linux
- Crear `.pkg` para macOS
- Crear `.msi` para Windows
- Automatizar con GitHub Actions

### Para Distribución Rápida
**Usar Opción 1 (Distribución Separada):**
- Documentar claramente el requisito de la librería
- Proporcionar scripts de instalación
- Incluir instrucciones en README

## Verificación de Instalación

### Linux
```bash
# Verificar que la librería está en el path
ldconfig -p | grep llama_go

# Verificar dependencias del binario
ldd remembrances-mcp

# Probar ejecución
remembrances-mcp --version
```

### macOS
```bash
# Verificar ubicación de la librería
ls -l /usr/local/lib/libllama_go.dylib

# Verificar dependencias
otool -L remembrances-mcp

# Probar ejecución
remembrances-mcp --version
```

### Windows
```bash
# Verificar que la DLL está en PATH o mismo directorio
where llama_go.dll

# Probar ejecución
remembrances-mcp.exe --version
```

## Conclusión

1. **Compilación estática en Go NO es posible** con la arquitectura actual de kelindar/search
2. **La librería `.so` YA está compilada estáticamente** (contiene todas sus dependencias)
3. **Múltiples estrategias de distribución viables** están disponibles
4. **La distribución separada es simple y efectiva** para la mayoría de casos
5. **El enfoque actual es estándar** en la industria para proyectos Go con componentes nativos

## Referencias

- [Análisis Completo (Inglés)](./KELINDAR_SEARCH_STATIC_LINKING.md)
- [Repositorio kelindar/search](https://github.com/kelindar/search)
- [Documentación purego](https://github.com/ebitengine/purego)
- [Script de Instalación](../scripts/install-with-library.sh)
- [Script de Descarga](../scripts/download-native-library.sh)

## Soporte

Si encuentras problemas:
1. Verifica que la librería está instalada: `ldconfig -p | grep llama_go`
2. Verifica permisos: `ls -l /usr/local/lib/libllama_go.so`
3. Revisa variables de entorno: `echo $LD_LIBRARY_PATH`
4. Consulta los logs de errores del programa

---

**Última actualización:** Octubre 2024
**Estado:** Proyecto funcional con kelindar/search