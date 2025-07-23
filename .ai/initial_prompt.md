Develop a project in Golang whose main objective is to serve as an MCP server. For this, I want to use the mcp-go library: https://mcp-go.dev/getting-started/

It should implement a series of MCP tools or functions similar to the logic and functionality of the Mem0 project at the memory level but without a management environment; everything will operate at the tools function level: https://github.com/mem0ai/mem0

The Mem0 project, whose memory functionality I want to replicate, has several memory layers with different types of layered storage, including key-value (KV), RAG or vector database, and graph-based database. Conduct a study of its functionality to clone its behavior in the Golang project. You can use https://gitingest.com/ to read the project's documentation.

Additionally, I want to introduce the ability to generate and modify `.md` files that serve as a knowledge base to remember important aspects of a project. This feature will be activated when the MCP server is called at startup with the `--knowledge-base` parameter and the path to the root folder where it will create, modify, and query, via reading and RAG, everything important to consider for the project being worked on. The rest of the layers will function as defined in Mem0.

For generating embeddings for RAG, use langchaingo with support for Ollama locally and OpenAI-compatible APIs.

The binary can also be called with the `--sse` parameter to activate SSE support instead of MCP's stdio, and a `--rest-api-serve` parameter that will enable a series of endpoints to call the functionality externally, allowing any external application to use an HTTP API instead of the MCP protocol.

Another important detail is that for storing key-value databases, vector embeddings for RAG, and graph databases, you must use the SurrealDB SDK for Golang, prioritizing the use of an embedded database system but allowing configuration of an external SurrealDB database via environment variables. This is the SDK documentation: https://surrealdb.com/docs/sdk/golang and https://pkg.go.dev/github.com/surrealdb/surrealdb.go. With the `--dbpath` parameter, the root folder for generating the database structure will be specified.

Additionally, generate documentation for the entire project code and a `README.md`.
