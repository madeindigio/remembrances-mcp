---
title: "Herramientas de Base de Conocimiento"
linkTitle: "KB Tools"
weight: 10
description: >
  Almacenamiento y búsqueda semántica de documentos
---

El sistema de Base de Conocimiento (KB) permite almacenar documentos con búsqueda semántica automática, ideal para mantener documentación, notas y cualquier contenido que necesites recuperar por significado.

## Herramientas Disponibles

### kb_add_document
Añade un documento con generación automática de embeddings para búsqueda semántica.

### kb_search_documents
Busca documentos por similitud semántica a una consulta en lenguaje natural.

### kb_get_document
Recupera un documento específico por su ruta de archivo.

### kb_delete_document
Elimina un documento de la base de conocimiento.

## Flujo de Trabajo Típico

1. **Añadir documentos**: Usa `kb_add_document` con contenido y ruta de archivo
2. **Buscar**: Utiliza `kb_search_documents` con consultas en lenguaje natural
3. **Recuperar**: Emplea `kb_get_document` para obtener el contenido completo cuando lo necesites
4. **Mantener**: Usa `kb_delete_document` para eliminar documentación obsoleta

## Características

- **Generación automática de embeddings**: El sistema convierte automáticamente tus documentos en vectores para búsqueda semántica
- **Ruta de archivo como identificador**: Cada documento se identifica por su ruta (ej: `guides/authentication.md`)
- **Metadatos opcionales**: Añade metadatos para organización y filtrado
- **Sincronización con archivos Markdown**: Si configuras una ruta de base de conocimiento, los documentos se sincronizan con archivos `.md`
- **Fragmentación automática**: Los documentos grandes se dividen automáticamente en chunks manejables

## Prompts Recomendados

### Para Añadir Documentos

```
Añade este documento a la base de conocimiento en la ruta "arquitectura/microservicios.md":
[contenido del documento]
```

```
Guarda la siguiente guía de estilo en "team/style-guide.md" con metadatos 
category: "guidelines", team: "frontend"
```

### Para Buscar Documentos

```
Busca documentación sobre cómo implementar autenticación con JWT
```

```
Encuentra información relacionada con despliegue en producción y configuración de CI/CD
```

```
¿Qué documentos tenemos sobre buenas prácticas de testing?
```

### Para Gestionar Documentos

```
Muéstrame el contenido completo del documento en "api/endpoints.md"
```

```
Elimina el documento obsoleto en "legacy/old-api.md"
```

## Mejores Prácticas

### Organización de Rutas

- Usa rutas descriptivas y jerárquicas: `proyectos/nombre/subseccion.md`
- Agrupa documentos relacionados en carpetas lógicas
- Mantén nombres consistentes y fáciles de recordar

### Contenido de Documentos

- Escribe títulos claros y descriptivos
- Incluye contexto relevante en el documento
- Usa markdown para formato consistente
- Añade metadatos relevantes para facilitar filtrado

### Búsqueda Efectiva

- Formula consultas en lenguaje natural describiendo lo que buscas
- Los resultados incluyen scores de relevancia para ordenación
- La búsqueda es semántica: encuentra documentos por significado, no solo por palabras exactas
- Combina múltiples búsquedas para refinar resultados

## Casos de Uso Comunes

### Documentación de Proyecto

Mantén toda la documentación técnica del proyecto accesible:

```
Añade la documentación de arquitectura
Busca información sobre el módulo de pagos
Actualiza la guía de despliegue
```

### Base de Conocimiento de Equipo

Centraliza conocimiento compartido del equipo:

```
Guarda las decisiones de diseño importantes
Busca documentos sobre convenciones de código
Recupera la guía de onboarding
```

### Notas y Referencias

Almacena notas de investigación y referencias:

```
Añade estos apuntes sobre optimización de rendimiento
Busca información que guardé sobre GraphQL
```

## Integración con Filesystem

Si configuras `--knowledge-base` o `GOMEM_KNOWLEDGE_BASE` con una ruta de directorio:

- Los documentos se guardan como archivos `.md` en el filesystem
- Puedes editar archivos directamente y se sincronizarán
- Ideal para integración con Git y control de versiones
- Los documentos permanecen accesibles incluso fuera de Remembrances

## Ver Más

Para documentación detallada de cada herramienta, usa el sistema de ayuda integrado:

```
how_to_use("kb_add_document")
how_to_use("kb_search_documents")
how_to_use("kb_get_document")
how_to_use("kb_delete_document")
```
