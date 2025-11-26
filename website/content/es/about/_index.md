---
title: "Acerca de Remembrances MCP"
linkTitle: "Acerca de"
---

{{< blocks/cover title="Acerca de Remembrances MCP" image_anchor="center" height="auto" color="primary" >}}

Remembrances MCP es un **servidor Model Context Protocol (MCP)** que proporciona capacidades de memoria a largo plazo para agentes IA. Construido con Go y potenciado por SurrealDB, ofrece una solución flexible que prioriza la privacidad para gestionar la memoria de agentes IA.

{{< /blocks/cover >}}

{{% blocks/lead color="dark" %}}

## ¿Qué Hace Especial a Remembrances MCP?

Los agentes IA tradicionales no tienen estado - olvidan todo entre conversaciones. Remembrances MCP resuelve esto proporcionando **memoria persistente**, **búsqueda semántica** y **mapeo de relaciones** mientras mantiene tus datos privados y seguros.

{{% /blocks/lead %}}

{{% blocks/section color="white" %}}

## Características Principales

{{% blocks/feature icon="fa-lock" title="Embeddings Locales que Priorizan la Privacidad" %}}
Genera embeddings completamente en local usando modelos GGUF. Tus datos nunca salen de tu máquina, garantizando privacidad y seguridad completas.
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-bolt" title="Aceleración GPU" %}}
Aprovecha la aceleración por hardware con soporte para Metal (macOS), CUDA (GPUs NVIDIA) y ROCm (GPUs AMD).
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-database" title="Múltiples Capas de Almacenamiento" %}}
Almacén Clave-Valor para hechos simples, Vector/RAG para búsqueda semántica y Base de Datos de Grafos para mapeo de relaciones.
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-book" title="Gestión de Base de Conocimiento" %}}
Gestiona bases de conocimiento usando simples archivos Markdown, facilitando la organización y mantenimiento del conocimiento de tu IA.
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-plug" title="Integración Flexible" %}}
Soporte para múltiples proveedores de embeddings: Modelos GGUF (local), Ollama (servidor local) y API de OpenAI (basado en la nube).
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-shield-alt" title="Control de Privacidad" %}}
Mantén datos sensibles en local con embeddings GGUF y SurrealDB embebido - sin dependencias en la nube.
{{% /blocks/feature %}}

{{% /blocks/section %}}

{{% blocks/section color="primary" %}}

## ¿Por Qué Elegir Remembrances MCP?

Remembrances MCP potencia tus agentes IA con poderosas capacidades de memoria mientras mantienes el control completo sobre tus datos.

{{% blocks/feature icon="fa-brain" title="Memoria Persistente" %}}
Almacena hechos, conversaciones y conocimiento permanentemente. Tu IA recuerda lo que importa.
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-search" title="Búsqueda Semántica" %}}
Encuentra información relevante usando embeddings vectoriales. Búsqueda inteligente que entiende el contexto.
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-project-diagram" title="Mapeo de Relaciones" %}}
Entiende conexiones entre diferentes piezas de información usando capacidades de base de datos de grafos.
{{% /blocks/feature %}}

{{% /blocks/section %}}

{{% blocks/section color="white" %}}

## Casos de Uso

<div class="row">
<div class="col-md-6">

### 🤖 Asistentes IA Personales
Recuerda preferencias de usuario y conversaciones pasadas para proporcionar una experiencia verdaderamente personalizada.

### 🔬 Asistentes de Investigación
Construye y consulta bases de conocimiento desde documentos, papers y materiales de investigación.

</div>
<div class="col-md-6">

### 💬 Soporte al Cliente
Mantén contexto a través de múltiples interacciones para un mejor servicio al cliente.

### 💻 Herramientas de Desarrollo
Almacena y recupera fragmentos de código, documentación y conocimiento técnico.

</div>
</div>

{{% /blocks/section %}}

{{% blocks/section color="dark" %}}

## Stack Tecnológico

Remembrances MCP está construido con tecnologías modernas y probadas:

- **Lenguaje**: Go 1.20+ para rendimiento y fiabilidad
- **Base de Datos**: SurrealDB (embebida o externa) para almacenamiento de datos flexible
- **Embeddings**: Modelos GGUF vía llama.cpp para embeddings locales que priorizan la privacidad
- **Protocolo**: Model Context Protocol (MCP) para integración perfecta con IA

{{% /blocks/section %}}

{{% blocks/section color="secondary" %}}

## Código Abierto y Comunidad

Remembrances MCP es **código abierto** y está disponible en [GitHub](https://github.com/madeindigio/remembrances-mcp). 

¡Damos la bienvenida a contribuciones de la comunidad! Ya sea que quieras reportar un bug, sugerir una característica o contribuir código, nos encantaría saber de ti.

{{% /blocks/section %}}

{{% blocks/section color="primary" %}}

## Desarrollado por Digio

<div style="text-align: center; padding: 2rem 0;">

Remembrances MCP es desarrollado y mantenido por [**Digio**](https://digio.es), una empresa de desarrollo de software especializada en IA y soluciones innovadoras.

Visítanos en [digio.es](https://digio.es) para conocer más sobre nuestro trabajo.

</div>

{{% /blocks/section %}}
