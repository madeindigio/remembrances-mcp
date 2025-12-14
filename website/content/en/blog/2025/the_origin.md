---
title: "The Origin of the Project"
date: 2025-12-16
categories: [history]
tags: [history, origin]
---

This project was born in mid-2025 after working extensively with AI assistants and having to provide context over and over again. The idea was to create a tool that would make information management and retrieval efficient and effective. There were different solutions at that time, but most were based on traditional text-only approaches:

* Solutions based on markdown files or similar.
* Solutions based on relational databases like SQLite.

However, after working with RAG (Retrieval-Augmented Generation) in some projects for semantic information search, I believed something better could be done. Around that time, the *paper and open-source project* **[Mem0](https://github.com/mem0ai/mem0)** appeared, proposing an MCP server approach that stored information in structured knowledge graphs and other elements.

The first version emerged from combining these ideas with semantic search, which in Remembrances we call "hybrid search," allowing ingestion of markdown or plain text documents, but also structured information.

I was aware of different multimodal databases that were emerging, and some older ones, that allowed storing vector information (for RAG), graphs, key-value, etc. The idea was to use a system that integrated all of this and could scale in the future, since Mem0 and other alternatives propose using different servers for each data type, which complicates installation and maintenance.

This is where **[SurrealDB](https://surrealdb.com/)** came in, a multimodal database that allowed storing all types of data on a single server, with a simple and powerful API. After some testing, I decided to use SurrealDB as the foundation for the project.

The project grew and evolved. At that time, we also discovered and started working with another very useful tool for AI code assistant development, **[Serena MCP](https://github.com/oraios/serena)**. Serena allows analyzing project source code and extracting structured information using LSP (Language Server Protocol). This fit perfectly with Remembrances' idea of managing structured and semantic information. But it had performance and scalability limitations, since each project and programming language required the corresponding LSP to be installed, and it did not index information with RAG, which greatly limited search capabilities.

So I decided to integrate Serena's ideas into Remembrances, creating a system that could analyze source code, extract structured information, and store it in SurrealDB, all with semantic search capabilities using RAG.

The latest versions of Remembrances can index projects with more than 7k files in just a few minutes, with advanced search capabilities and a scalable, flexible architecture. The process does not block work and runs in the background. Serena needs much more time, uses Python, and a file index system that is fully loaded into memory to work, which greatly limits its scalability. With Remembrances, you can index more than one project in its database and search all of them simultaneously, for example, because they are dependencies, libraries, or simply projects from which you want to extract ideas, patterns, or solutions.
Remembrances indexes using AST (abstract syntax tree), which is included in the binary itself, supporting multiple languages, and does not require installing additional LSPs, greatly simplifying installation and maintenance. It can read different types of code files in different languages in the same project at the same time and extract relevant information from each.

I added this functionality because we saw that with Serena, the indexing time for large projects was very high, and the user experience was not good. We also faced the challenge of applying it to large projects with over 15 years of development, thousands of files, and multiple languages, where relevant information was scattered and not always well documented.

The result is spectacular. Remembrances helps code assistants locate answers in seconds—answers to questions asked in very simple, natural language, without needing much context, since the system already has all the information indexed and structured and can retrieve it efficiently to provide an answer as if it were an expert in the project. The result in our daily work is impressive; it allows us to "see" again what our memory had forgotten and consider parts of the code we had not taken into account.

Remembrances is a constantly evolving project, and we continue working to improve its capabilities and features. The idea is to keep integrating new technologies and approaches to make Remembrances an even more powerful and useful tool for developers and software teams.
