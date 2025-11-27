---
title: "Documentación"
linkTitle: "Documentación"
weight: 20
---

¡Bienvenido a la documentación de Remembrances!

Esta documentación te ayudará a comenzar con Remembrances, configurarlo según tus necesidades e integrarlo con tus agentes IA.

## Qué es Remembrances? y Por Qué Usarlo?

Remembrances es un servidor Model Context Protocol (MCP) que proporciona memoria a largo plazo a tus agentes IA. Utiliza embeddings para búsqueda semántica y SurrealDB para almacenar y recuperar información de manera segura y privada, sin depender de servicios en la nube.

## Qué son los embeddings?

Los modelos de inteligencia artificial consiguen entender cualquier idioma con el que han sido entrenados, pero no entienden el significado de las palabras como lo hacemos los humanos. Para que un modelo de IA pueda entender el significado de un texto, este debe ser convertido a una representación matemática llamada "embedding". 
Un embedding es un vector de números que captura el significado semántico del texto. Los embeddings permiten a los modelos de IA comparar y relacionar diferentes textos basándose en su significado, facilitando tareas como la búsqueda semántica y la recuperación de información relevante. Así una misma palabra en diferentes idiomas y todas las palabras que son sinónimo de ésta tendrán embeddings similares, y por tanto podremos buscar mejor la información, no por coincidencia literal de palabras sino por significado de lo que queremos expresar.

Toda la información que Remembrances almacena y recupera para tus agentes IA es convertida a embeddings para que éstos puedan entenderla y utilizarla eficazmente.

## Y qué es SurrealDB, y qué tiene de especial?

En la búsqueda de una base de datos que cumpliera con los requisitos de Remembrances (almacenamiento de vectores/embeddings, base de datos de grafos, clave-valor, rendimiento, facilidad de uso, licencia abierta, etc) encontramos SurrealDB, una base de datos multi-modelo que cumple con todos estos requisitos y más. [SurrealDB](https://surrealdb.com/) nos permite almacenar y recuperar información de manera eficiente y flexible, adaptándose a las necesidades de Remembrances y sus usuarios.

Existen otras soluciones que combinan almacenamiento de vectores y bases de datos tradicionales, pero SurrealDB destaca por su rendimiento, flexibilidad y facilidad de uso, lo que la convierte en una opción ideal para Remembrances. Además, SurrealDB puede ser embebida directamente en la aplicación, eliminando la necesidad de configurar y mantener un servidor de base de datos separado, lo que simplifica enormemente la instalación y el uso de Remembrances. 

Pero si quieres que todo un equipo de trabajo comparta la misma base de datos de conocimiento de Remembrances, puedes configurar SurrealDB para que funcione conectado con un servidor SurrealDB que puede estar en tu red local o en la nube, y así todos los usuarios compartirán la misma base de datos de conocimiento, de forma segura y privada. Porque SurrealDB soporta múltiples capas de seguridad y autenticación, puedes estar seguro de que tus datos estarán protegidos.

## Pero, y cómo funciona?

Fácil, Remembrances expone una API compatible con el protocolo MCP (Model Context Protocol), lo que permite a tus agentes IA interactuar con él para almacenar y recuperar información de manera eficiente. Cuando un agente IA necesita recordar algo, envía una solicitud a Remembrances, que convierte la información en embeddings y la almacena en SurrealDB. Cuando el agente necesita recuperar información, Remembrances utiliza búsqueda semántica para encontrar los datos más relevantes y los devuelve al agente.

Almacenamos la información en diferentes capas de almacenamiento según su naturaleza y uso previsto:

- Capa Clave-Valor: Para datos simples y de acceso rápido.
- Capa Vector/RAG: Para búsqueda semántica utilizando embeddings.
- Capa de Grafos: Para relaciones complejas entre datos, imagina una red de conocimientos donde cada nodo es un concepto y las aristas representan las relaciones entre ellos. A Lucía le gusta leer libros de ciencia ficción, y su autor favorito es Isaac Asimov. En la capa de grafos, tendríamos nodos para "Lucía", "libros de ciencia ficción" e "Isaac Asimov", con aristas que conectan a Lucía con sus gustos y autores favoritos. Esto permite a Remembrances entender y navegar por las relaciones entre diferentes piezas de información, facilitando respuestas más contextuales y relevantes para los agentes IA.
- Capa de Archivos o base de datos de conocimiento: Permite leer ficheros en markdown y almacenarlos en una carpeta (útil para trabajar con proyectos de código) pero al mismo tiempo permite búsqueda semántica sobre el contenido de dichos ficheros.
- Capa de archivos de conocimiento avanzado (próximamente): Permite leer ficheros en formatos más complejos como PDF, DOCX, etc y almacenarlos en una carpeta para búsqueda semántica.
- Capa de archivos multimedia (próximamente): Para almacenar y buscar en imágenes, audio y vídeo utilizando embeddings multimodales.
- Capa de Series Temporales (próximamente): Para datos que cambian con el tiempo, como eventos o registros históricos.
