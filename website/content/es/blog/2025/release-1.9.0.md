---
title: "Versión 1.9.0: Indexación de Código y Ahorro Inteligente de Tokens"
linkTitle: Versión 1.9.0
date: 2025-11-29
author: Remembrances MCP Team
description: >
  Remembrances MCP 1.9.0 introduce una potente indexación de código con tree-sitter y un sistema de ayuda inteligente que reduce el consumo de tokens en un 85%.
tags: [release, features, code-indexing]
---

Estamos emocionados de anunciar **Remembrances MCP 1.9.0**, una versión repleta de funcionalidades que trae dos capacidades principales: un potente **Sistema de Indexación de Código** para búsqueda semántica de código y un nuevo **sistema de ayuda how_to_use** que reduce drásticamente el consumo de tokens.

## 🔍 Sistema de Indexación de Código

La función estrella de esta versión es el **Sistema de Indexación de Código** – una solución completa para que los agentes de IA comprendan, busquen y naveguen por bases de código usando búsqueda semántica.

### ¿Qué Puedes Hacer?

**Buscar Código por Significado:**
Pregunta por "autenticación de usuario y validación de contraseña" y encuentra funciones de login relevantes, validadores de contraseñas y módulos de seguridad – incluso si no contienen esas palabras exactas.

**Navegar Grandes Bases de Código:**
Obtén vistas generales instantáneas de estructuras de archivos, encuentra todas las implementaciones de una interfaz, rastrea referencias a una función y comprende jerarquías de llamadas.

**Manipular Código Inteligentemente:**
Recupera implementaciones de símbolos, reemplaza cuerpos de funciones e inserta nuevo código en ubicaciones específicas con total conocimiento del contexto.

### Más de 14 Lenguajes Soportados

Hemos integrado **tree-sitter** para un análisis AST preciso en una amplia gama de lenguajes:

- **Go, Rust, C/C++** – Programación de sistemas
- **TypeScript, JavaScript** – Desarrollo web
- **Python, Ruby, PHP** – Lenguajes de scripting
- **Java, C#, Kotlin, Scala** – Lenguajes empresariales
- **Swift** – Desarrollo móvil
- ¡Y más!

### Cómo Funciona

1. **Indexa tu proyecto:**
   ```
   code_index_project({
     "project_path": "/ruta/al/proyecto",
     "project_name": "Mi App"
   })
   ```

2. **Busca semánticamente:**
   ```
   code_semantic_search({
     "project_id": "mi-app",
     "query": "pooling de conexiones a base de datos"
   })
   ```

3. **Encuentra y navega símbolos:**
   ```
   code_find_symbol({
     "project_id": "mi-app",
     "name_path_pattern": "DatabasePool/getConnection"
   })
   ```

El indexador extrae todos los símbolos significativos – clases, funciones, métodos, interfaces – y crea embeddings vectoriales para búsqueda por similitud semántica. Los cambios se rastrean y re-indexan automáticamente.

## 💡 Sistema de Ayuda Inteligente (how_to_use)

Con más de 37 herramientas disponibles, cargar la documentación completa al inicio de cada conversación consumía ~15,000+ tokens antes de que comenzara cualquier trabajo real. Eso es costoso e ineficiente.

### La Solución: Documentación Bajo Demanda

La nueva herramienta `how_to_use` proporciona documentación exactamente cuando la necesitas:

| Antes | Después | Ahorro |
|-------|---------|--------|
| ~15,000 tokens por adelantado | ~2,500 tokens | **~85% de reducción** |

### Cómo Funciona

Cada herramienta ahora tiene una descripción mínima de 1-2 líneas. Cuando tu agente de IA necesita más información:

```
how_to_use("code_semantic_search")
```

Esto carga solo la documentación para esa herramienta específica – descripciones completas de parámetros, ejemplos y herramientas relacionadas.

También puedes obtener vistas generales por categoría:
```
how_to_use("code")      # Todas las herramientas de indexación de código
how_to_use("memory")    # Todas las herramientas de memoria
how_to_use("kb")        # Todas las herramientas de base de conocimiento
```

### Por Qué Esto Importa

- **Menores costes:** Menos tokens por conversación significa facturas de API más bajas
- **Respuestas más rápidas:** Menos contexto para procesar significa respuestas iniciales más rápidas
- **Mejor enfoque:** Los agentes de IA ven la documentación relevante cuando la necesitan

## Empezando

### Actualizar

Descarga la última versión desde [GitHub Releases](https://github.com/madeindigio/remembrances-mcp/releases/tag/v1.9.0) y reemplaza tu binario existente.

### Prueba la Indexación de Código

1. Inicia Remembrances MCP
2. Pide a tu IA que indexe un proyecto:
   ```
   "Indexa mi proyecto en /ruta/al/proyecto"
   ```
3. Busca en tu código:
   ```
   "Encuentra código relacionado con autenticación de usuario"
   ```

### Explora el Sistema de Ayuda

Pide a tu IA que ejecute:
```
how_to_use()
```

Para ver una vista general de todas las capacidades disponibles.

## Próximos Pasos

Continuamos mejorando Remembrances MCP con:
- Más soporte de lenguajes para indexación de código
- Funciones avanzadas de análisis de código
- Optimizaciones de rendimiento para grandes bases de código

¡Gracias a todos los que proporcionaron feedback y solicitudes de funcionalidades. Vuestras aportaciones dan forma al futuro de Remembrances MCP!

---

*¿Encontraste un problema? ¿Tienes una solicitud de funcionalidad? ¡Abre un issue en [GitHub](https://github.com/madeindigio/remembrances-mcp/issues)!*
