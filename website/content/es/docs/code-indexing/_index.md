---
title: "Indexación de Código"
linkTitle: "Indexación de Código"
weight: 5
description: >
  Indexa tu base de código para búsqueda semántica y navegación inteligente
---

La función de Indexación de Código permite a los agentes de IA comprender, buscar y navegar por tu código mediante búsqueda semántica. En lugar de una simple coincidencia de texto, tu asistente de IA puede encontrar código por significado – busca "autenticación de usuario" y encuentra las funciones de login, gestores de sesión y módulos de seguridad relevantes.

## ¿Qué es la Indexación de Código?

La Indexación de Código analiza tu código fuente usando [tree-sitter](https://tree-sitter.github.io/tree-sitter/) para un análisis preciso en múltiples lenguajes de programación. Extrae símbolos significativos (clases, funciones, métodos, interfaces) y crea embeddings vectoriales para búsqueda semántica.

Esto significa que puedes:
- **Buscar código por significado**: Encuentra "manejo de errores para operaciones de base de datos" y obtén los bloques try-catch relevantes, manejadores de errores y código de logging
- **Navegar bases de código grandes**: Encuentra rápidamente implementaciones, referencias y código relacionado
- **Obtener contexto inteligente**: Ayuda a los agentes de IA a entender la estructura y patrones de tu código

## Lenguajes Soportados

Remembrances soporta **más de 14 lenguajes de programación** sin configuración adicional:

| Lenguaje | Extensiones | Símbolos Extraídos |
|----------|------------|-------------------|
| Go | `.go` | funciones, métodos, structs, interfaces, constantes |
| TypeScript | `.ts`, `.tsx` | clases, funciones, métodos, interfaces, tipos |
| JavaScript | `.js`, `.jsx`, `.mjs` | clases, funciones, métodos, exports |
| Python | `.py` | clases, funciones, métodos, decoradores |
| Java | `.java` | clases, métodos, interfaces, enums |
| C# | `.cs` | clases, métodos, interfaces, structs |
| Rust | `.rs` | funciones, structs, traits, impls, enums |
| C/C++ | `.c`, `.h`, `.cpp`, `.hpp` | funciones, structs, clases |
| Ruby | `.rb` | clases, módulos, métodos |
| PHP | `.php` | clases, funciones, métodos, interfaces |
| Swift | `.swift` | clases, structs, protocolos, funciones |
| Kotlin | `.kt` | clases, funciones, interfaces |
| Scala | `.scala` | clases, objetos, traits, funciones |
| Bash | `.sh`, `.bash` | funciones |

## Cómo Usar la Indexación de Código

### 1. Indexar un Proyecto

Usa la herramienta `code_index_project` para comenzar a indexar tu código:

```
code_index_project({
  "project_path": "/ruta/a/tu/proyecto",
  "project_name": "Mi Proyecto",
  "languages": ["go", "typescript", "python"]
})
```

La indexación se ejecuta en segundo plano. Para proyectos grandes, puedes verificar el progreso con `code_index_status`.

### 2. Buscar Código

**Búsqueda Semántica** – Encuentra código describiendo lo que buscas:

```
code_semantic_search({
  "project_id": "mi-proyecto",
  "query": "autenticación de usuario y gestión de sesiones",
  "limit": 10
})
```

Esto devuelve fragmentos de código relevantes ordenados por similitud semántica, incluso si no contienen las palabras exactas de tu búsqueda.

**Buscar Símbolos** – Busca por nombre de función, clase o método:

```
code_find_symbol({
  "project_id": "mi-proyecto",
  "name_path_pattern": "UserService/authenticate",
  "include_body": true
})
```

### 3. Navegar la Estructura del Código

**Vista General de Archivo** – Ve todos los símbolos en un archivo:

```
code_get_file_symbols({
  "project_id": "mi-proyecto",
  "relative_path": "src/services/auth.go"
})
```

**Buscar Referencias** – Encuentra dónde se usa un símbolo:

```
code_find_references({
  "project_id": "mi-proyecto",
  "symbol_name": "validateToken"
})
```

**Obtener Jerarquía de Llamadas** – Ve qué llama a una función y qué funciones llama:

```
code_get_call_hierarchy({
  "project_id": "mi-proyecto",
  "symbol_name": "processPayment"
})
```

### 4. Gestionar Proyectos

**Listar Proyectos** – Ver todos los proyectos indexados:

```
code_list_projects()
```

**Estadísticas del Proyecto** – Ver detalles de indexación:

```
code_get_project_stats({
  "project_id": "mi-proyecto"
})
```

**Re-indexar un Archivo** – Actualizar después de cambios:

```
code_reindex_file({
  "project_id": "mi-proyecto",
  "relative_path": "src/services/auth.go"
})
```

## Manipulación de Código

Más allá de la búsqueda y navegación, Remembrances proporciona herramientas para modificar código:

**Obtener Cuerpo del Símbolo** – Recuperar la implementación completa:

```
code_get_symbol_body({
  "project_id": "mi-proyecto",
  "symbol_name": "validateToken"
})
```

**Reemplazar Cuerpo del Símbolo** – Actualizar una implementación:

```
code_replace_symbol_body({
  "project_id": "mi-proyecto",
  "symbol_name": "validateToken",
  "new_body": "func validateToken(token string) bool {\n  // Nueva implementación\n}"
})
```

**Insertar Símbolo** – Añadir nuevo código en una ubicación específica:

```
code_insert_symbol({
  "project_id": "mi-proyecto",
  "file_path": "src/services/auth.go",
  "position": "after:validateToken",
  "code": "func refreshToken(token string) (string, error) {\n  // Implementación\n}"
})
```

## Mejores Prácticas

### Organización de Proyectos

- Indexa bases de código relacionadas como proyectos separados
- Usa nombres de proyecto descriptivos para fácil identificación
- Incluye solo los lenguajes con los que trabajas activamente

### Búsquedas Eficientes

- Usa búsqueda semántica para consultas conceptuales ("manejo de errores", "validación de datos")
- Usa búsqueda de símbolos para funciones o clases específicas que conoces por nombre
- Combina búsquedas para reducir resultados

### Mantener el Índice Actualizado

- El indexador detecta cambios en archivos y re-indexa automáticamente cuando la vigilancia está habilitada
- Para actualizaciones manuales, usa `code_reindex_file` después de cambios significativos
- Elimina y re-indexa un proyecto si has hecho cambios estructurales importantes

## Modelos de Embedding Especializados para Código

Para resultados óptimos en búsqueda de código, puedes configurar un modelo de embedding dedicado especializado en código. Consulta la página de [Configuración](/es/docs/configuration/) para detalles sobre cómo configurar modelos de embedding específicos para código como CodeRankEmbed o Jina Code Embeddings.
