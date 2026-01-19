---
title: "Solución de Problemas"
linkTitle: "Troubleshooting"
weight: 40
description: >
  Problemas comunes y soluciones para Remembrances MCP
---

## Problemas de Instalación

### La Compilación Falla con Errores de llama.cpp

**Problema**: El proceso de compilación falla al compilar las dependencias de llama.cpp.

**Soluciones**:

1. **Asegúrate de tener las herramientas de compilación necesarias**:
   ```bash
   # Ubuntu/Debian
   sudo apt-get install build-essential cmake

   # macOS
   xcode-select --install

   # Fedora
   sudo dnf install gcc-c++ cmake
   ```

2. **Verifica la versión de Go** (requiere Go 1.20+):
   ```bash
   go version
   ```

3. **Limpia y recompila**:
   ```bash
   make clean
   make build
   ```

### Falta Soporte de GPU

**Problema**: La aceleración GPU no funciona aunque tengas una GPU compatible.

**Soluciones**:

1. **NVIDIA (CUDA)**:
   ```bash
   # Verificar instalación de CUDA
   nvidia-smi
   nvcc --version
   
   # Asegúrate de que el toolkit CUDA está instalado
   # Ubuntu/Debian
   sudo apt-get install nvidia-cuda-toolkit
   ```

2. **AMD (ROCm)**:
   ```bash
   # Verificar instalación de ROCm
   rocm-smi
   
   # Instalar ROCm si falta
   # Sigue las instrucciones en https://rocm.docs.amd.com/
   ```

3. **Apple Silicon (Metal)**:
   Metal debería funcionar automáticamente en macOS con Apple Silicon. Asegúrate de estar ejecutando la compilación nativa ARM64.

## Problemas en Tiempo de Ejecución

### Errores de Falta de Memoria (OOM)

**Problema**: El servidor se cuelga con errores de memoria al procesar embeddings.

**Soluciones**:

1. **Reduce las capas GPU**:
   ```bash
   # Usa menos capas GPU
   --gguf-gpu-layers 16  # en lugar de 32
   
   # O desactiva GPU completamente
   --gguf-gpu-layers 0
   ```

2. **Usa un modelo más pequeño**:
   - Cambia de `nomic-embed-text-v1.5` a `all-MiniLM-L6-v2`
   - Usa una versión más cuantizada (Q4_K_M en lugar de Q8_0)

3. **Reduce el tamaño de lote** (si aplica):
   ```bash
   --gguf-batch-size 256  # el valor por defecto suele ser mayor
   ```

### El Modelo No Carga

**Problema**: El servidor no arranca con errores de "modelo no encontrado" o similares.

**Soluciones**:

1. **Verifica la ruta del archivo y permisos**:
   ```bash
   ls -lh ./model.gguf
   chmod +r ./model.gguf
   ```

2. **Verifica que el archivo del modelo no esté corrupto**:
   ```bash
   # Comprueba que el tamaño del archivo coincide con lo esperado
   ls -lh ./model.gguf
   
   # Vuelve a descargar si es necesario
   wget https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf
   ```

3. **Usa ruta absoluta**:
   ```bash
   --gguf-model-path /ruta/completa/al/model.gguf
   ```

### Rendimiento Lento

**Problema**: La generación de embeddings o la búsqueda es más lenta de lo esperado.

**Soluciones**:

1. **Activa la aceleración GPU**:
   ```bash
   --gguf-gpu-layers 32
   ```

2. **Aumenta el número de hilos**:
   ```bash
   --gguf-threads 8  # ajusta según el número de núcleos de tu CPU
   ```

3. **Usa un modelo más rápido**:
   - `all-MiniLM-L6-v2` es significativamente más rápido que `nomic-embed-text-v1.5`

4. **Verifica el throttling térmico**:
   ```bash
   # Monitoriza temperaturas de CPU/GPU
   # NVIDIA
   nvidia-smi -l 1
   
   # CPU
   sensors  # Linux
   ```

## Problemas de Base de Datos

### La Conexión a la Base de Datos Falla

**Problema**: No se puede conectar a SurrealDB (embebida o externa).

**Soluciones**:

1. **Para base de datos embebida**:
   ```bash
   # Verifica permisos del archivo
   ls -la ./remembrances.db
   
   # Asegúrate de que el directorio existe y tiene permisos de escritura
   mkdir -p ./data
   chmod 755 ./data
   --db-path ./data/remembrances.db
   ```

2. **Para SurrealDB externa**:
   ```bash
   # Verifica que SurrealDB está ejecutándose
   curl http://localhost:8000/health
   
   # Comprueba los parámetros de conexión
   --surrealdb-url ws://localhost:8000
   --surrealdb-user root
   --surrealdb-pass root
   ```

### Corrupción de Base de Datos

**Problema**: Errores de base de datos o datos inconsistentes después de un crash.

**Soluciones**:

1. **Respalda y recrea**:
   ```bash
   # Respalda datos existentes
   cp ./remembrances.db ./remembrances.db.backup
   
   # Elimina base de datos corrupta
   rm ./remembrances.db
   
   # Reinicia - creará una base de datos nueva
   ./remembrances-mcp --gguf-model-path ./model.gguf
   ```

2. **Ejecuta con logging de debug** para identificar problemas:
   ```bash
   --log-level debug
   ```

## Problemas de Conexión MCP

### Claude Desktop No Conecta

**Problema**: Claude Desktop no reconoce o no conecta con Remembrances MCP.

**Soluciones**:

1. **Verifica la ubicación del archivo de configuración**:
   - macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
   - Linux: `~/.config/claude/claude_desktop_config.json`

2. **Verifica la sintaxis JSON**:
   ```bash
   # Valida JSON
   cat ~/.config/claude/claude_desktop_config.json | python -m json.tool
   ```

3. **Usa rutas absolutas** en la configuración:
   ```json
   {
     "mcpServers": {
       "remembrances": {
         "command": "/usr/local/bin/remembrances-mcp",
         "args": [
           "--gguf-model-path",
           "/home/usuario/models/nomic-embed-text-v1.5.Q4_K_M.gguf"
         ]
       }
     }
   }
   ```

4. **Reinicia Claude Desktop** después de cambios en la configuración.

### Problemas de MCP Streamable HTTP / API HTTP

**Problema**: No se puede conectar vía MCP Streamable HTTP (tools MCP) o la API JSON HTTP.

**Soluciones**:

1. **Verifica si el puerto está en uso**:
   ```bash
   # Verificar disponibilidad del puerto
   lsof -i :3000  # MCP Streamable HTTP por defecto
   lsof -i :8080  # HTTP por defecto
   ```

2. **Usa un puerto diferente**:
   ```bash
   --mcp-http --mcp-http-addr ":3001"
   --http --http-addr ":8081"
   ```

3. **Verifica la configuración del firewall**:
   ```bash
   # Permitir puerto (Linux con ufw)
   sudo ufw allow 8080/tcp
   ```

## Problemas de Embeddings

### Resultados de Búsqueda Inconsistentes

**Problema**: Los resultados de búsqueda varían o no coinciden con el contenido esperado.

**Soluciones**:

1. **Asegura un modelo de embeddings consistente** - no mezcles embeddings de diferentes modelos

2. **Verifica que las dimensiones de embeddings coincidan**:
   - `nomic-embed-text-v1.5`: 768 dimensiones
   - `all-MiniLM-L6-v2`: 384 dimensiones

3. **Re-indexa después de cambiar de modelo**:
   ```bash
   # Puede que necesites re-generar embeddings de todo el contenido si cambias de modelo
   ```

### Los Embeddings No Se Generan

**Problema**: El contenido se almacena pero los embeddings están vacíos o faltan.

**Soluciones**:

1. **Verifica la configuración del embedder**:
   ```bash
   # Verifica que el modelo está especificado
   --gguf-model-path ./model.gguf
   # O
   --ollama-model nomic-embed-text
   # O
   --openai-key sk-xxx
   ```

2. **Activa logging de debug**:
   ```bash
   --log-level debug
   ```

## Obtener Ayuda

Si sigues experimentando problemas:

1. **Revisa los logs** con modo debug:
   ```bash
   --log-level debug
   ```

2. **Busca issues existentes** en [GitHub Issues](https://github.com/madeindigio/remembrances-mcp/issues)

3. **Abre un nuevo issue** con:
   - Sistema operativo y versión
   - Versión de Go (`go version`)
   - Tipo de GPU (si aplica)
   - Mensaje de error completo
   - Pasos para reproducir

## Ver También

- [Primeros Pasos](../getting-started/) - Guía de instalación
- [Configuración](../configuration/) - Opciones de configuración
- [Modelos GGUF](../gguf-models/) - Selección y optimización de modelos