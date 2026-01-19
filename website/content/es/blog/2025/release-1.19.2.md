---
title: "Lanzamiento 1.19.2"
date: 2026-01-19
categories: [release]
tags: [release, update, code-indexing, languages]
---

¡La versión **1.19.2** de Remembrances-MCP ya está disponible! Este lanzamiento amplía el soporte de lenguajes para indexación de código y mejora la estabilidad.

## Novedades

### Soporte Ampliado de Lenguajes para Indexación de Código

Remembrances-MCP ahora comprende e indexa código en aún más lenguajes y frameworks:

- **Svelte:** Soporte completo para componentes Svelte, permitiendo mejor comprensión de código en proyectos basados en Svelte
- **MDX:** Indexa y busca archivos MDX, perfecto para flujos de trabajo de documentación como código
- **Markdown:** Soporte mejorado para archivos Markdown estándar
- **Vue:** Soporte completo de componentes Vue.js para mejor navegación de proyectos
- **Lua:** Soporte para scripts Lua, ideal para sistemas embebidos y desarrollo de juegos

Estas adiciones hacen de Remembrances-MCP una herramienta más versátil para proyectos multi-lenguaje y stacks de desarrollo web moderno.

### Cómo Usar la Indexación de Código con Nuevos Lenguajes

Simplemente indexa tu proyecto con la herramienta `code_project_index`, y Remembrances-MCP automáticamente:

1. Detectará archivos en los lenguajes recientemente soportados
2. Analizará y extraerá símbolos, funciones y estructura
3. Creará embeddings buscables para búsqueda semántica de código
4. Habilitará autocompletado inteligente y comprensión de código

Casos de uso de ejemplo:

- **Sitios de documentación:** Indexa tu documentación basada en MDX para búsqueda inteligente
- **Proyectos Vue/Svelte:** Navega grandes bibliotecas de componentes con facilidad
- **Scripts Lua:** Busca y comprende configuraciones Lua embebidas
- **Bases de código multi-lenguaje:** Trabaja sin problemas entre JavaScript, TypeScript, Vue, Svelte y Markdown en el mismo proyecto

### Estabilidad Mejorada

- **Mejor manejo de excepciones:** Manejo de errores mejorado al indexar código previene fallos con archivos mal formados
- **Manejo de archivos ignorados:** Mejor gestión de archivos ignorados durante el proceso de indexación

## ¿Por Qué Actualizar?

- **Trabaja con frameworks modernos:** Soporte completo para Svelte y Vue lo hace esencial para desarrolladores web modernos
- **Mejores flujos de documentación:** El soporte MDX habilita potentes capacidades de documentación como código
- **Indexación más robusta:** Manejo mejorado de errores asegura indexación confiable incluso con bases de código imperfectas

Descarga la nueva versión aquí:

[Descargar Remembrances-MCP v1.19.2](https://github.com/madeindigio/remembrances-mcp/releases/tag/v1.19.2)

¡Actualiza ahora e indexa todo tu stack tecnológico!
