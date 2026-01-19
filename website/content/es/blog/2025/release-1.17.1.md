---
title: "Lanzamiento 1.17.1"
date: 2025-12-20
categories: [release]
tags: [release, update, changelog, http, streaming]
---

¡La versión **1.17.1** de Remembrances-MCP ya está disponible! Esta actualización introduce una nueva funcionalidad importante para comunicación HTTP y varias mejoras en el proceso de instalación.

## Novedades

### Soporte para Modo HTTP con Streaming

La característica estrella de este lanzamiento es el **modo HTTP con streaming**, que permite comunicación en tiempo real mediante el protocolo HTTP. Esto habilita:

- **Respuestas en tiempo real:** Obtén resultados progresivos mientras se generan, sin esperar respuestas completas
- **Mejor integración:** Integración más sencilla con aplicaciones web y clientes basados en HTTP
- **Experiencia de usuario mejorada:** Visualiza resultados mientras se transmiten, perfecto para operaciones de larga duración

Para usar el modo HTTP con streaming, simplemente configura el puerto apropiado en tu configuración. El sistema manejará automáticamente el streaming sin el problema del carácter ':' que existía en implementaciones tempranas.

### Scripts de Instalación Mejorados

Hemos mejorado significativamente la experiencia de instalación en todas las plataformas:

- **Detección de CPU AVX-512:** El instalador ahora detecta automáticamente si tu CPU soporta instrucciones AVX-512 y descarga la versión optimizada para máximo rendimiento
- **Mejor soporte para macOS:** Corregidos problemas de compilación con la versión embebida en sistemas macOS
- **Mejor detección CUDA:** Detección mejorada de bibliotecas CUDA para sistemas Linux
- **Detección de binarios más confiable:** Corregidos errores cuando el instalador intenta detectar binarios existentes

### Correcciones de Errores

- Resueltos problemas de compatibilidad con clientes MCP cuando llama.cpp genera información de depuración
- Solucionados problemas de ubicación de bibliotecas compartidas en versiones embebidas y no embebidas
- Mejoradas opciones de compilación embebida para macOS

## ¿Por Qué Actualizar?

- **Mantente actualizado con streaming HTTP:** Aprovecha protocolos de comunicación en tiempo real modernos
- **Instalación más fácil:** Deja que el instalador detecte y configure automáticamente la mejor versión para tu hardware
- **Mayor estabilidad:** Múltiples correcciones de errores aseguran un funcionamiento más fluido en todas las plataformas

Descarga la nueva versión aquí:

[Descargar Remembrances-MCP v1.17.1](https://github.com/madeindigio/remembrances-mcp/releases/tag/v1.17.1)

¡Actualiza ahora y experimenta streaming en tiempo real!
