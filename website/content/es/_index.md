---
title: "Remembrances"
linkTitle: "Remembrances"
---

{{< blocks/cover title="Remembrances" image_anchor="center" height="full" color="dark" >}}
<div class="mx-auto">
  <p class="lead mt-5">Memoria a largo plazo para agentes IA con embeddings locales que priorizan la privacidad y opcionalmente base de datos compartida self-hosted</p>
  <div class="mx-auto mt-5">
    <a class="btn btn-lg btn-primary mr-3 mb-4" href="{{< relref "/docs" >}}">
      Comenzar <i class="fas fa-arrow-alt-circle-right ml-2"></i>
    </a>&nbsp;
    <a class="btn btn-lg btn-secondary mr-3 mb-4" href="https://github.com/madeindigio/remembrances-mcp">
      Ver en GitHub <i class="fab fa-github ml-2 "></i>
    </a>
  </div>
</div>
{{< /blocks/cover >}}

{{% blocks/lead color="white" %}}
Remembrances MCP es un **servidor Model Context Protocol (MCP)** que dota a tus agentes IA de memoria persistente a largo plazo. 
Utiliza **embeddings locales** y **SurrealDB** para almacenar y recuperar información de forma segura, privada y sin dependencias en la nube.
{{% /blocks/lead %}}

{{< blocks/section color="primary" >}}
<div class="row">

<div class="col-md-4 mb-4">
  <h3 class="text-center mb-3"><i class="fas fa-info-circle fa-2x"></i></h3>
  <h3 class="text-center">Acerca de Remembrances</h3>
  <p class="text-center">Aprende más sobre el proyecto, su filosofía y el equipo detrás de él.</p>
  <p class="text-center"><a class="btn btn-light" href="{{< relref "/about" >}}">Leer Acerca de <i class="fas fa-arrow-right ml-2"></i></a></p>
</div>

<div class="col-md-4 mb-4">
  <h3 class="text-center mb-3"><i class="fas fa-book fa-2x"></i></h3>
  <h3 class="text-center">Cómo Funciona</h3>
  <p class="text-center">Sumérgete en la documentación para entender la arquitectura e integración.</p>
  <p class="text-center"><a class="btn btn-light" href="{{< relref "/docs" >}}">Leer Docs <i class="fas fa-arrow-right ml-2"></i></a></p>
</div>

<div class="col-md-4 mb-4">
  <h3 class="text-center mb-3"><i class="fas fa-blog fa-2x"></i></h3>
  <h3 class="text-center">Blog</h3>
  <p class="text-center">Mantente actualizado con las últimas noticias, tutoriales y lanzamientos.</p>
  <p class="text-center"><a class="btn btn-light" href="{{< relref "/blog" >}}">Leer Blog <i class="fas fa-arrow-right ml-2"></i></a></p>
</div>

</div>
{{< /blocks/section >}}

{{< blocks/section color="dark" >}}
<div class="row">
{{% blocks/feature icon="fa-lock" title="Privacidad Primero" %}}
Todos los embeddings se generan localmente con modelos GGUF. Sin envío de datos externos.
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-bolt" title="Aceleración GPU" %}}
Soporte para Metal (macOS), CUDA (NVIDIA) y ROCm (AMD) para un rendimiento ultrarrápido.
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-database" title="Múltiples Capas de Almacenamiento" %}}
Soporte para clave-valor, vector/RAG y base de datos de grafos con SurrealDB.
{{% /blocks/feature %}}
</div>
{{< /blocks/section >}}

{{< blocks/section color="white" >}}
<div class="col-12">
<h2 class="text-center">Inicio Rápido</h2>
<p class="text-center">Comienza con Remembrances MCP en minutos</p>
</div>

<div class="col-12">
    <h3>Para Linux o MacOSX</h3>
    <p>Ejecuta el script de instalación:</p>
<pre><code>

    curl -fsSL https://raw.githubusercontent.com/madeindigio/remembrances-mcp/main/scripts/install.sh | bash

</code></pre>
</div>

<div class="col-12">
    <h3>Para Windows</h3>
    <p>Usa la versión de Linux en WSL o con Docker:</p>
<pre><code>

    docker run -it --rm \
    -v /path/to/data:/data \
    -v /path/to/kb:/knowledge-base \
    ghcr.io/madeindigio/remembrances-mcp:latest
    
</code></pre>
</div>

{{< /blocks/section >}}
