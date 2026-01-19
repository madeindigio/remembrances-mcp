---
title: "Herramientas de Memoria"
linkTitle: "Memory Tools"
weight: 11
description: >
  Sistema de memoria de 3 capas: facts, vectors y graph
---

El sistema de memoria de Remembrances proporciona tres capas complementarias de almacenamiento, cada una optimizada para diferentes tipos de datos y patrones de acceso.

## Las Tres Capas de Memoria

### 1. Capa Clave-Valor (Facts)
Almacenamiento simple de pares clave-valor para recuperación exacta.

**Herramientas:**
- `save_fact`: Guardar un fact clave-valor
- `get_fact`: Recuperar un fact por clave exacta
- `list_facts`: Listar todos los facts de un usuario
- `delete_fact`: Eliminar un fact específico

**Cuándo usar:** Configuraciones, preferencias, datos estructurados con claves conocidas.

### 2. Capa Vectorial/RAG (Vectors)
Almacena contenido con embeddings para búsqueda por similitud semántica.

**Herramientas:**
- `add_vector`: Añadir contenido con embedding automático
- `search_vectors`: Buscar por similitud semántica
- `update_vector`: Actualizar contenido y regenerar embedding
- `delete_vector`: Eliminar una entrada vectorial

**Cuándo usar:** Notas, ideas, contenido que buscarás por significado.

### 3. Capa de Grafos (Graph)
Crea entidades y relaciones para datos estructurados con conexiones.

**Herramientas:**
- `create_entity`: Crear una entidad tipada (persona, proyecto, concepto)
- `create_relationship`: Vincular dos entidades
- `traverse_graph`: Explorar conexiones entre entidades
- `get_entity`: Obtener detalles de una entidad por ID

**Cuándo usar:** Entidades con relaciones (personas, proyectos, conceptos).

## Herramientas Adicionales

### Búsqueda Híbrida
- `hybrid_search`: Busca simultáneamente en las tres capas de memoria
- `get_stats`: Obtiene estadísticas de uso de memoria

### Contexto de Sesión
- `to_remember`: Almacena contexto importante para sesiones futuras
- `last_to_remember`: Recupera contexto almacenado y actividad reciente

## Prompts Recomendados

### Capa Facts (Clave-Valor)

```
Guarda la preferencia del usuario para el tema oscuro
```

```
Almacena la configuración de la API key para el servicio de email
```

```
Recupera todas las configuraciones del proyecto actual
```

```
¿Cuál es el valor guardado para la clave "database_connection"?
```

### Capa Vectors (Semántica)

```
Guarda esta nota sobre la reunión de planificación del sprint
```

```
Busca información relacionada con optimización de rendimiento de base de datos
```

```
Encuentra todas las notas sobre arquitectura de microservicios
```

```
Actualiza la nota anterior sobre el módulo de autenticación con esta nueva información
```

### Capa Graph (Relaciones)

```
Crea una entidad de tipo "persona" para Ana García, desarrolladora senior
```

```
Establece una relación "trabaja_en" entre Ana García y el proyecto Ecommerce
```

```
Muéstrame todas las personas que trabajan en el proyecto Ecommerce
```

```
Encuentra todas las conexiones del proyecto API Gateway (hasta 2 niveles)
```

### Búsqueda Híbrida

```
Busca todo lo relacionado con "autenticación OAuth" en todas las capas de memoria
```

```
Encuentra información sobre el desarrollador Juan y sus proyectos
```

## Mejores Prácticas

### Selección de Capa

**Usa Facts cuando:**
- Necesitas recuperación exacta por clave
- Los datos son simples pares clave-valor
- Conoces la clave exacta que necesitas buscar
- Ejemplos: configuraciones, flags, contadores

**Usa Vectors cuando:**
- Buscarás por significado, no por clave exacta
- El contenido es texto libre (notas, descripciones)
- Quieres encontrar contenido similar
- Ejemplos: notas de reunión, aprendizajes, ideas

**Usa Graph cuando:**
- Los datos tienen relaciones significativas
- Necesitas navegar conexiones
- La estructura es importante
- Ejemplos: personas y proyectos, conceptos relacionados, dependencias

### User ID

**Nota importante:** Si no estás seguro qué `user_id` usar, utiliza el nombre del proyecto actual como user_id. Esto permite organizar la memoria por proyecto o contexto.

### Organización de Datos

- **Nomenclatura consistente**: Usa convenciones claras para claves y tipos de entidad
- **Metadatos útiles**: Añade metadatos que faciliten filtrado y organización
- **Limpieza regular**: Elimina datos obsoletos para mantener la memoria eficiente

## Casos de Uso Comunes

### Gestión de Configuración
```
Guarda la URL del servidor de staging como "staging_server_url"
Guarda el timeout de la API como "api_timeout" con valor 30
```

### Memoria de Conversación
```
Guarda esta decisión importante sobre la arquitectura del sistema
Busca lo que discutimos sobre el sistema de caché
```

### Mapeo de Conocimiento
```
Crea una entidad "proyecto" llamada "Sistema de Facturación"
Crea una entidad "persona" para el líder técnico María Rodríguez
Relaciona a María Rodríguez con Sistema de Facturación como "líder_técnico"
Muéstrame todos los proyectos liderados por María
```

### Seguimiento de Aprendizajes
```
Guarda este aprendizaje sobre patrones de diseño en Go
Busca lo que aprendí sobre concurrencia
```

## Búsqueda Híbrida Avanzada

La búsqueda híbrida combina las tres capas:

```
hybrid_search({
  "query": "microservicios",
  "user_id": "proyecto-ecommerce",
  "limit": 10
})
```

Esto buscará:
- **Facts** con claves que contengan "microservicios"
- **Vectors** semánticamente similares a "microservicios"
- **Entidades** del graph relacionadas con "microservicios"

## Ver Más

Para documentación detallada de cada herramienta, usa el sistema de ayuda:

```
how_to_use("memory")
how_to_use("save_fact")
how_to_use("add_vector")
how_to_use("create_entity")
how_to_use("hybrid_search")
```
