---
title: "Documentation"
linkTitle: "Documentation"
weight: 20
---

Welcome to the Remembrances documentation!

This documentation will help you get started with Remembrances, configure it according to your needs, and integrate it with your AI agents.

## What is Remembrances? and Why Use It?

Remembrances is a Model Context Protocol (MCP) server that provides long-term memory to your AI agents. It uses embeddings for semantic search and SurrealDB to store and retrieve information securely and privately, without depending on cloud services.

## What are embeddings?

AI models can understand any language they have been trained on, but they don't understand the meaning of words the way humans do. For an AI model to understand the meaning of a text, it must be converted to a mathematical representation called an "embedding".
An embedding is a vector of numbers that captures the semantic meaning of the text. Embeddings allow AI models to compare and relate different texts based on their meaning, facilitating tasks such as semantic search and retrieval of relevant information. Thus, the same word in different languages and all words that are synonyms of it will have similar embeddings, and therefore we can better search for information, not by literal word coincidence but by the meaning of what we want to express.

All the information that Remembrances stores and retrieves for your AI agents is converted to embeddings so that they can understand and use it effectively.

## And what is SurrealDB, and what makes it special?

In the search for a database that met Remembrances' requirements (vector/embedding storage, graph database, key-value, performance, ease of use, open license, etc.), we found SurrealDB, a multi-model database that meets all these requirements and more. [SurrealDB](https://surrealdb.com/) allows us to store and retrieve information efficiently and flexibly, adapting to Remembrances' needs and its users.

There are other solutions that combine vector storage and traditional databases, but SurrealDB stands out for its performance, flexibility, and ease of use, making it an ideal choice for Remembrances. Additionally, SurrealDB can be embedded directly into the application, eliminating the need to configure and maintain a separate database server, which greatly simplifies the installation and use of Remembrances.

But if you want an entire work team to share the same Remembrances knowledge database, you can configure SurrealDB to work connected to a SurrealDB server that can be on your local network or in the cloud, so all users will share the same knowledge database, securely and privately. Because SurrealDB supports multiple layers of security and authentication, you can be sure your data will be protected.

## But, how does it work?

Easy, Remembrances exposes an API compatible with the MCP (Model Context Protocol), which allows your AI agents to interact with it to store and retrieve information efficiently. When an AI agent needs to remember something, it sends a request to Remembrances, which converts the information into embeddings and stores it in SurrealDB. When the agent needs to retrieve information, Remembrances uses semantic search to find the most relevant data and returns it to the agent.

We store information in different storage layers according to its nature and intended use:

- Key-Value Layer: For simple data and fast access.
- Vector/RAG Layer: For semantic search using embeddings.
- Graph Layer: For complex relationships between data, imagine a knowledge network where each node is a concept and the edges represent the relationships between them. Lucía likes to read science fiction books, and her favorite author is Isaac Asimov. In the graph layer, we would have nodes for "Lucía", "science fiction books", and "Isaac Asimov", with edges connecting Lucía to her tastes and favorite authors. This allows Remembrances to understand and navigate the relationships between different pieces of information, facilitating more contextual and relevant responses for AI agents.
- File Layer or Knowledge Database: Allows reading markdown files and storing them in a folder (useful for working with code projects) but at the same time allows semantic search on the content of said files.
- Advanced Knowledge Files Layer (coming soon): Allows reading files in more complex formats such as PDF, DOCX, etc., and storing them in a folder for semantic search.
- Multimedia Files Layer (coming soon): For storing and searching in images, audio, and video using multimodal embeddings.
- Time Series Layer (coming soon): For data that changes over time, such as events or historical records.