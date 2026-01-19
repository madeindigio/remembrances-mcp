---
title: "Herramientas de Indexación de Código"
linkTitle: "Code Tools"
weight: 12
description: >
  Indexación y búsqueda semántica de código fuente
---

Las herramientas de código permiten indexar proyectos completos para búsqueda semántica, navegación inteligente y manipulación de código mediante análisis AST con Tree-sitter.

## Categorías de Herramientas

### Herramientas de Indexación

Gestiona la indexación de proyectos de código:

- `code_index_project`: Iniciar indexación de un proyecto (ejecución asíncrona)
- `code_index_status`: Verificar progreso del trabajo de indexación
- `code_list_projects`: Listar todos los proyectos indexados
- `code_delete_project`: Eliminar un proyecto y sus datos
- `code_reindex_file`: Actualizar el índice de un archivo específico
- `code_get_project_stats`: Obtener estadísticas del proyecto
- `code_get_file_symbols`: Listar símbolos en un archivo específico

### Herramientas de Búsqueda

Encuentra código usando diferentes métodos:

- `code_get_symbols_overview`: Obtener estructura de archivo de alto nivel (¡úsala primero!)
- `code_find_symbol`: Buscar por nombre o patrón de ruta
- `code_search_symbols_semantic`: Búsqueda en lenguaje natural
- `code_search_pattern`: Búsqueda por patrón de texto/regex
- `code_find_references`: Encontrar usos de un símbolo
- `code_hybrid_search`: Búsqueda combinada semántica + patrón

### Herramientas de Manipulación

Modifica código fuente:

- `code_replace_symbol`: Reemplazar el cuerpo completo de un símbolo
- `code_insert_after_symbol`: Añadir código después de un símbolo
- `code_insert_before_symbol`: Añadir código antes de un símbolo
- `code_delete_symbol`: Eliminar un símbolo del archivo

## Prompts Recomendados

### Para Indexar Proyectos

```
Indexa el proyecto en /ruta/al/proyecto con nombre "mi-proyecto"
```

```
Indexa solo archivos Go y TypeScript del proyecto en /src/backend
```

```
¿Cuál es el estado de indexación del proyecto "api-service"?
```

```
Muéstrame las estadísticas del proyecto "frontend-app"
```

### Para Buscar Código (Búsqueda Semántica)

```
Encuentra código que maneje autenticación de usuarios y gestión de sesiones
```

```
Busca funciones relacionadas con validación de datos de formularios
```

```
¿Dónde está el código que procesa pagos con tarjeta de crédito?
```

```
Encuentra implementaciones de manejo de errores para operaciones de base de datos
```

### Para Buscar Símbolos Específicos

```
Encuentra la definición de la clase UserService
```

```
Busca el método authenticate en el módulo de auth
```

```
Muéstrame todos los símbolos en el archivo src/services/payment.go
```

```
Encuentra dónde se usa la función validateToken
```

### Para Búsqueda Híbrida

```
Busca "authentication" usando búsqueda híbrida (semántica + patrón de texto)
```

```
Encuentra código relacionado con cache que contenga la palabra "redis"
```

### Para Manipular Código

```
Reemplaza el método login con esta nueva implementación: [código]
```

```
Inserta esta nueva función después del método validateUser
```

```
Elimina el método obsoleto oldAuthentication
```

## Flujo de Trabajo Típico

### 1. Indexar Proyecto

```
Indexa el proyecto backend en /home/user/projects/api
```

Espera a que la indexación termine:

```
Verifica el estado de indexación del proyecto "api"
```

### 2. Explorar Estructura

Primero obtén una visión general del archivo:

```
Muéstrame la estructura del archivo src/auth/handler.go
```

Luego busca código específico:

```
Encuentra funciones relacionadas con JWT en el proyecto api
```

### 3. Buscar y Navegar

Búsqueda semántica por concepto:

```
Busca código que implemente validación de email
```

O búsqueda por nombre específico:

```
Encuentra la función processPayment
```

Encuentra usos:

```
¿Dónde se llama a la función sendEmail?
```

### 4. Modificar y Actualizar

Realiza cambios:

```
Reemplaza el método validateInput con: [nuevo código]
```

Actualiza el índice:

```
Re-indexa el archivo src/validation/input.go
```

## Lenguajes Soportados

El sistema soporta más de 14 lenguajes sin configuración adicional:

| Lenguaje | Extensiones | Símbolos Extraídos |
|----------|------------|-------------------|
| Go | `.go` | funciones, métodos, structs, interfaces |
| TypeScript | `.ts`, `.tsx` | clases, funciones, métodos, interfaces |
| JavaScript | `.js`, `.jsx`, `.mjs` | clases, funciones, métodos |
| Python | `.py` | clases, funciones, métodos |
| Java | `.java` | clases, métodos, interfaces |
| C# | `.cs` | clases, métodos, interfaces |
| Rust | `.rs` | funciones, structs, traits |
| C/C++ | `.c`, `.h`, `.cpp`, `.hpp` | funciones, structs, clases |
| Ruby | `.rb` | clases, módulos, métodos |
| PHP | `.php` | clases, funciones, métodos |
| Swift | `.swift` | clases, structs, protocolos |
| Kotlin | `.kt` | clases, funciones, interfaces |

## Mejores Prácticas

### Organización de Proyectos

- Indexa bases de código relacionadas como proyectos separados
- Usa nombres de proyecto descriptivos para fácil identificación
- Incluye solo los lenguajes con los que trabajas activamente

### Búsquedas Eficientes

**Para búsqueda conceptual:**
```
Busca funciones que manejen caché de datos
Encuentra código relacionado con validación de formularios
```

**Para símbolos específicos:**
```
Encuentra la clase UserRepository
Busca el método findById
```

**Para exploración de código:**
```
Muéstrame la estructura de auth/service.go
Lista todos los símbolos en payment/processor.ts
```

### Mantener el Índice Actualizado

- El indexador detecta cambios automáticamente si está habilitada la vigilancia
- Para actualizaciones manuales, usa `code_reindex_file` después de cambios significativos
- Re-indexa el proyecto completo si has hecho cambios estructurales importantes

## Modelos de Embedding Especializados

Para resultados óptimos en búsqueda de código, configura un modelo especializado:

**Modelos recomendados:**
- GGUF: `coderankembed.Q4_K_M.gguf` - CodeRankEmbed optimizado para código
- Ollama: `jina/jina-embeddings-v2-base-code` - Jina Code Embeddings
- OpenAI: `text-embedding-3-large` - Funciona bien también para código

**Configuración:**

```bash
# Modelo general + modelo específico para código
export GOMEM_GGUF_MODEL_PATH="/path/to/nomic-embed-text-v1.5.Q4_K_M.gguf"
export GOMEM_CODE_GGUF_MODEL_PATH="/path/to/coderankembed.Q4_K_M.gguf"
```

Consulta la sección de [Configuración](/es/docs/configuration/) para más detalles.

## Ver Más

Para documentación detallada de cada herramienta:

```
how_to_use("code")
how_to_use("code_index_project")
how_to_use("code_search_symbols_semantic")
how_to_use("code_find_symbol")
how_to_use("code_hybrid_search")
```

También consulta:
- [Indexación de Código](/es/docs/code-indexing/) - Guía completa del usuario
