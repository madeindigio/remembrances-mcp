---
title: "El origen del projecto"
date: 2025-12-16
categories: [historia]
tags: [historia, origen]
---

Este proyecto nació a mediados de 2025 después de trabajar bastante con asistentes de AI y tener que dar contexto una y otra vez. La idea era crear una herramienta que facilitara la gestión y recuperación de información de manera eficiente y efectiva. Existían diferentes soluciones en aquel momento, pero la mayoría basadas en enfoques tradicionales en solo texto:

* Soluciones basadas en ficheros markdown o similares.
* Soluciones basadas en bases de datos relacionales SQLite

Sin embargo, tras haber trabajado con RAG en algunos proyectos para búsquedas semánticas de la información, creía que se podía hacer algo mejor. También apareció en ese tiempo *el paper y proyecto de software libre* **[Mem0](https://github.com/mem0ai/mem0)** que proponía un enfoque de servidor MCP que almacenase la información de forma estructurada en grafos de conocimiento y otros elementos.

La primera versión surge de convinar esas ideas con la búsqueda semántica, que en remembrances llamamos "búsqueda híbrida", pudiendo ingestar documentos markdown o de texto plano, pero también información estructurada.

Conocía diferentes bases de datos multimodales que estaban apareciendo y algunas anteriores, que permitían almacenar información vectorial (para RAG), grafos, clave-valor, etc... La idea es poder usar un sistema que integrase todo eso y permitiese escalar en el futuro, ya que Mem0 y otras alternativas proponen utilizar diferentes servidores para cada tipo de dato, lo que complica la instalación y mantenimiento.

Es aquí donde apareció **[SurrealDB](https://surrealdb.com/)**, una base de datos multimodal que permitía almacenar todo tipo de datos en un solo servidor, con una API sencilla y potente. Tras algunas pruebas, decidí usar SurrealDB como base para el proyecto.

El proyecto fue creciendo y evolucionando. En ese momento empezamos a descubrir y trabajar con otra herramienta muy útil para desarrollo con asistentes de código para AI, **[Serena MCP](https://github.com/oraios/serena)**. Serena permite analizar el código fuente de proyectos y extraer información estructurada usando LSP (Language Server Protocol). Esto encajaba perfectamente con la idea de Remembrances de gestionar información estructurada y semántica. Pero tenía limitaciones de rendimiento y escalabilidad, ya que cada proyecto y lenguaje de programación, requería instalado el LSP correspondiente, además no indexaba con RAG la información, lo que limitaba mucho las capacidades de búsqueda.

Así que decidí integrar las ideas de Serena en Remembrances, creando un sistema que pudiera analizar código fuente, extraer información estructurada y almacenarla en SurrealDB, todo ello con capacidades de búsqueda semántica usando RAG.

Las últimas versiones de Remembrances, consiguen indexar proyectos de más de 7k ficheros en pocos minutos, con capacidades de búsqueda avanzadas y una arquitectura escalable y flexible, el proceso no bloquea el trabajo, y se mantiene en segundo plano. Serena necesita mucho más tiempo, además de utilizar Python y un sistema de fichero índice que se carga completamente en memoria para trabajar, lo que limita mucho su escalabilidad. 
Con Remembrances, se pueden indexar más de un proyecto en su base de datos, y buscar en todos ellos de forma simultánea, por ejemplo porque sean dependencias, librerías o simpmlemente proyectos de los que quieras extraer ideas, patrones o soluciones.
Remembrances indexa utilizando AST (abstract syntax tree) esto está incluido en el propio binario soportando multiples lenguajes, y no requiere instalar LSPs adicionales, lo que simplifica mucho la instalación y mantenimiento, es capaz de leer al mismo tiempo diferentes tipos de ficheros de código en diferentes lenguajes en un mismo proyecto y extraer la información relevante de cada uno.

Añadí esta funcionalidad porque vimos que con Serena, el tiempo de indexación de grandes proyectos era muy alto, y la experiencia de usuario no era buena. Además teníamos el reto de poder aplicarlo en grandes proyectos de más de 15 años de desarrollo con miles de ficheros y múltiples lenguajes, donde la información relevante estaba dispersa y no siempre bien documentada.

El resultado es expectacular, Remembrances permite ayudar a localizar la respuesta a los asistentes de código en segundos, respuestas a preguntas formuladas de forma muy simple y en lenguaje natural, sin necesidad de dar mucho contexto, ya que el sistema ya tiene toda la información indexada y estructurada y puede recuperarla de forma eficiente para dar una respuesta como si fuera un experto en el proyecto, el resultado en nuestro trabajo diario es impresionante, nos permite "ver" de nuevo aquello que nuestra memoria había olvidado, y considerar partes de código que no habíamos tenido en cuenta.

Remembrances es un proyecto en constante evolución, y seguimos trabajando para mejorar sus capacidades y funcionalidades. La idea es seguir integrando nuevas tecnologías y enfoques para hacer de Remembrances una herramienta aún más poderosa y útil para desarrolladores y equipos de software.

