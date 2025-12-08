# Plan de Desarrollo: TOON Format + Levenshtein Suggestions

**Branch**: `feature/llama-libs-inclded-inbinary`  
**Fecha**: 8 de diciembre de 2025  
**Estado**: En planificación

---

## 📋 Resumen Ejecutivo

Este plan implementa dos nuevas características principales para el proyecto remembrances-mcp:

1. **Formato TOON**: Reemplazar las respuestas YAML en todas las tools MCP por el formato TOON, un formato más condensado y eficiente. Usa la librería `github.com/toon-format/toon-go`.

2. **Sugerencias Levenshtein ("Quiso decir...")**: Cuando no se encuentran resultados en operaciones GET de facts, entities o knowledge base, usar el algoritmo de distancia Levenshtein para sugerir opciones similares, compatible con el sistema actual de alternativas por counts.

---

## 🎯 Objetivos

- [ ] Todas las respuestas MCP en formato TOON
- [ ] API REST mantiene formato JSON (sin cambios)
- [ ] Sistema de sugerencias "did_you_mean" basado en Levenshtein
- [ ] Compatibilidad con sistema actual de alternativas (counts de otros IDs)
- [ ] Tests completos y documentación actualizada

---

## 📦 Dependencias Externas

| Librería | Propósito | URL |
|----------|-----------|-----|
| `toon-go` | Serialización formato TOON | https://github.com/toon-format/toon-go |
| `levenshtein` | Distancia de edición | https://github.com/agnivade/levenshtein |

---

## 🔄 Fases del Plan

Cada fase está documentada en detalle como un fact en remembrances con `user_id='plan'`.

### Fase 1: Integración del formato TOON
**Fact**: `get_fact(user_id='plan', key='phase-1-toon-format-integration')`

- Añadir dependencia toon-go
- Crear utilidades de serialización TOON
- Refactorizar todos los handlers MCP
- Mantener API REST sin cambios

### Fase 2: Integración de librería Levenshtein
**Fact**: `get_fact(user_id='plan', key='phase-2-levenshtein-library')`

- Seleccionar e integrar librería
- Crear módulo de similitud de strings
- Tests unitarios

### Fase 3: Extender sistema de alternativas
**Fact**: `get_fact(user_id='plan', key='phase-3-extend-alternatives-system')`

- Refactorizar alternatives.go
- Crear estructura `AlternativeSuggestions`
- Integrar con respuestas vacías

### Fase 4: Implementar sugerencias en Facts
**Fact**: `get_fact(user_id='plan', key='phase-4-facts-suggestions')`

- Modificar getFactHandler
- Modificar listFactsHandler
- Modificar deleteFactHandler

### Fase 5: Implementar sugerencias en Entities (Graph)
**Fact**: `get_fact(user_id='plan', key='phase-5-entity-suggestions')`

- Modificar getEntityHandler
- Modificar traverseGraphHandler
- Modificar createRelationshipHandler

### Fase 6: Implementar sugerencias en Knowledge Base
**Fact**: `get_fact(user_id='plan', key='phase-6-kb-suggestions')`

- Modificar getDocumentHandler
- Modificar deleteDocumentHandler
- Modificar searchDocumentsHandler

### Fase 7: Implementar sugerencias en Code Tools
**Fact**: `get_fact(user_id='plan', key='phase-7-code-tools-suggestions')`

- Modificar code_indexing_tools.go
- Modificar code_search_tools_handlers.go
- Modificar code_manipulation_tools.go

### Fase 8: Implementar sugerencias en Vectors y Events
**Fact**: `get_fact(user_id='plan', key='phase-8-vectors-events-suggestions')`

- Modificar vector_tools.go
- Modificar event_tools.go

### Fase 9: Testing de integración y documentación
**Fact**: `get_fact(user_id='plan', key='phase-9-integration-testing')`

- Tests de integración completos
- Tests de rendimiento
- Actualizar documentación
- Cleanup

---

## 📊 Archivos Principales Afectados

```
go.mod                                    # Nuevas dependencias
pkg/mcp_tools/
├── toon_utils.go                         # NUEVO - Utilidades TOON
├── string_similarity.go                  # NUEVO - Levenshtein
├── alternatives.go                       # Extender
├── fact_tools.go                         # Modificar
├── graph_tools.go                        # Modificar
├── kb_tools.go                           # Modificar
├── event_tools.go                        # Modificar
├── vector_tools.go                       # Modificar
├── code_indexing_tools.go                # Modificar
├── code_search_tools_handlers.go         # Modificar
├── code_manipulation_tools.go            # Modificar
└── yaml_utils.go                         # Deprecar/Eliminar

tests/
├── test_toon_responses.py                # NUEVO
└── test_suggestions.py                   # NUEVO
```

---

## 📝 Ejemplo de Respuesta Esperada

### Antes (YAML actual)
```yaml
message: No fact found for key 'preferencias' and user 'usr1'
alternatives:
  - user1 (15)
  - user2 (8)
```

### Después (TOON con sugerencias)
```toon
message: No fact found for key 'preferencias' and user 'usr1'
did_you_mean:
  - key: preferences (distance: 2)
  - key: preferencia (distance: 1)
available_user_ids:
  - user1 (15 facts)
  - user2 (8 facts)
```

---

## ⚠️ Notas Importantes

1. **API REST sin cambios**: El transporte HTTP debe seguir devolviendo JSON
2. **Compatibilidad**: El sistema actual de alternatives (counts) se mantiene
3. **Performance**: Limitar candidatos para Levenshtein (máx 100-500)
4. **Umbral distancia**: Sugerir solo si distancia ≤ 3-5 caracteres

---

## 🔗 Comandos Útiles

```bash
# Ver todas las fases del plan
list_facts(user_id='plan')

# Ver una fase específica
get_fact(user_id='plan', key='phase-1-toon-format-integration')

# Ver resumen actualizado
last_to_remember(user_id='plan')
```

---

## 📅 Progreso

| Fase | Estado | Fecha Inicio | Fecha Fin |
|------|--------|--------------|-----------|
| 1 | ⏳ Pendiente | - | - |
| 2 | ⏳ Pendiente | - | - |
| 3 | ⏳ Pendiente | - | - |
| 4 | ⏳ Pendiente | - | - |
| 5 | ⏳ Pendiente | - | - |
| 6 | ⏳ Pendiente | - | - |
| 7 | ⏳ Pendiente | - | - |
| 8 | ⏳ Pendiente | - | - |
| 9 | ⏳ Pendiente | - | - |
