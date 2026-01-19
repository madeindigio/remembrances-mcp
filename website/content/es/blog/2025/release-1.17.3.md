---
title: "Lanzamiento 1.17.3"
date: 2025-12-20
categories: [release]
tags: [release, update, bugfix, stability]
---

¡La versión **1.17.3** de Remembrances-MCP ya está disponible! Este es un lanzamiento enfocado en estabilidad que aborda problemas importantes de compatibilidad y registro de eventos.

## Correcciones

### Gestión de Sesiones Mejorada en Modo HTTP con Streaming

Hemos mejorado el modo HTTP con streaming con mejor manejo de sesiones:

- **Mejor registro:** Los problemas de falta de sesión ahora se registran correctamente en modo HTTP con streaming, facilitando el diagnóstico de problemas de conexión
- **Depuración mejorada:** Mayor visibilidad del ciclo de vida de las sesiones ayuda a resolver problemas de integración

### Compatibilidad Mejorada con Clientes MCP

Corregido un problema crítico que afectaba a algunos clientes MCP:

- **Mejor manejo de esquemas:** Las herramientas sin propiedades ahora generan correctamente esquemas cuando el tipo de herramienta es "object"
- **Soporte más amplio de clientes:** Esta corrección asegura compatibilidad con una gama más amplia de implementaciones de clientes MCP
- **Menos errores de integración:** Elimina errores que ocurrían previamente en ciertas configuraciones de clientes

## Por Qué Esta Actualización Es Importante

Aunque este lanzamiento no introduce nuevas características, mejora significativamente la confiabilidad y compatibilidad de la funcionalidad existente:

- **Streaming listo para producción:** El modo HTTP con streaming ahora es más robusto y está listo para uso en producción
- **Mejor diagnóstico de problemas:** El registro mejorado facilita identificar y resolver problemas
- **Mayor compatibilidad:** Funciona con más implementaciones de clientes MCP desde el inicio

## ¿Quién Debería Actualizar?

Esta actualización es especialmente importante si:

- Usas el modo HTTP con streaming en tus despliegues
- Experimentas problemas de compatibilidad con ciertos clientes MCP
- Necesitas mejor registro de eventos para entornos de producción

Descarga la nueva versión aquí:

[Descargar Remembrances-MCP v1.17.3](https://github.com/madeindigio/remembrances-mcp/releases/tag/v1.17.3)

¡Actualiza ahora para una experiencia más estable y compatible!
