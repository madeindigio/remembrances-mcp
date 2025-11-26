---
title: "Remembrances MCP"
linkTitle: "Remembrances MCP"
---

{{< blocks/cover title="Remembrances MCP" image_anchor="center" height="full" color="dark" >}}
<div class="mx-auto">
  <p class="lead mt-5">Long-term memory for AI agents with privacy-first local embeddings</p>
  <div class="mx-auto mt-5">
    <a class="btn btn-lg btn-primary mr-3 mb-4" href="{{< relref "/docs" >}}">
      Get Started <i class="fas fa-arrow-alt-circle-right ml-2"></i>
    </a>
    <a class="btn btn-lg btn-secondary mr-3 mb-4" href="https://github.com/madeindigio/remembrances-mcp">
      View on GitHub <i class="fab fa-github ml-2 "></i>
    </a>
  </div>
</div>
{{< /blocks/cover >}}

{{% blocks/lead color="white" %}}
Remembrances MCP is a **Model Context Protocol (MCP) server** that gives your AI agents persistent, long-term memory. 
It uses **local embeddings** and **SurrealDB** to store and retrieve information securely, privacy-first, and without cloud dependencies.
{{% /blocks/lead %}}

{{< blocks/section color="primary" >}}
<div class="row">

<div class="col-md-4 mb-4">
  <h3 class="text-center mb-3"><i class="fas fa-info-circle fa-2x"></i></h3>
  <h3 class="text-center">About Remembrances</h3>
  <p class="text-center">Learn more about the project, its philosophy, and the team behind it.</p>
  <p class="text-center"><a class="btn btn-light" href="{{< relref "/about" >}}">Read About <i class="fas fa-arrow-right ml-2"></i></a></p>
</div>

<div class="col-md-4 mb-4">
  <h3 class="text-center mb-3"><i class="fas fa-book fa-2x"></i></h3>
  <h3 class="text-center">How it Works</h3>
  <p class="text-center">Dive into the documentation to understand the architecture and integration.</p>
  <p class="text-center"><a class="btn btn-light" href="{{< relref "/docs" >}}">Read Docs <i class="fas fa-arrow-right ml-2"></i></a></p>
</div>

<div class="col-md-4 mb-4">
  <h3 class="text-center mb-3"><i class="fas fa-blog fa-2x"></i></h3>
  <h3 class="text-center">Blog</h3>
  <p class="text-center">Stay updated with the latest news, tutorials, and releases.</p>
  <p class="text-center"><a class="btn btn-light" href="{{< relref "/blog" >}}">Read Blog <i class="fas fa-arrow-right ml-2"></i></a></p>
</div>

</div>
{{< /blocks/section >}}

{{< blocks/section color="dark" >}}
<div class="row">
{{% blocks/feature icon="fa-lock" title="Privacy First" %}}
All embeddings generated locally with GGUF models. No data sent externally.
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-bolt" title="GPU Accelerated" %}}
Support for Metal (macOS), CUDA (NVIDIA), and ROCm (AMD) for blazing-fast performance.
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-database" title="Multiple Storage Layers" %}}
Key-value, vector/RAG, and graph database support with SurrealDB.
{{% /blocks/feature %}}
</div>
{{< /blocks/section >}}

{{< blocks/section color="white" >}}
<div class="col-12">
<h2 class="text-center">Quick Start</h2>
<p class="text-center">Get started with Remembrances MCP in minutes</p>
</div>

<div class="col-12">
<h3>For Linux or MacOSX</h3>
<p>Execute the install script:</p>
<pre><code>curl -fsSL https://raw.githubusercontent.com/madeindigio/remembrances-mcp/main/scripts/install.sh | bash</code></pre>
</div>

<div class="col-12">
<h3>For Windows</h3>
<p>Use the linux version in WSL or with docker:</p>
<pre><code>docker run ....</code></pre>
</div>

{{< /blocks/section >}}
