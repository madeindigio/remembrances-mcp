---
title: "Memory Tools"
linkTitle: "Memory Tools"
weight: 11
description: >
  3-layer memory system: facts, vectors, and graph
---

Remembrances' memory system provides three complementary storage layers, each optimized for different data types and access patterns.

## The Three Memory Layers

### 1. Key-Value Layer (Facts)
Simple key-value pair storage for exact retrieval.

**Tools:**
- `save_fact`: Store a key-value fact
- `get_fact`: Retrieve a fact by exact key
- `list_facts`: List all facts for a user
- `delete_fact`: Delete a specific fact

**When to use:** Settings, preferences, structured data with known keys.

### 2. Vector/RAG Layer (Vectors)
Stores content with embeddings for semantic similarity search.

**Tools:**
- `add_vector`: Add content with automatic embedding
- `search_vectors`: Search by semantic similarity
- `update_vector`: Update content and regenerate embedding
- `delete_vector`: Remove a vector entry

**When to use:** Notes, ideas, content you'll search by meaning.

### 3. Graph Layer
Creates entities and relationships for structured data with connections.

**Tools:**
- `create_entity`: Create a typed entity (person, project, concept)
- `create_relationship`: Link two entities
- `traverse_graph`: Explore entity connections
- `get_entity`: Get entity details by ID

**When to use:** Entities with relationships (people, projects, concepts).

## Additional Tools

### Hybrid Search
- `hybrid_search`: Search across all three memory layers simultaneously
- `get_stats`: Get memory usage statistics

### Session Context
- `to_remember`: Store important context for future sessions
- `last_to_remember`: Retrieve stored context and recent activity

## Recommended Prompts

### Facts Layer (Key-Value)

```
Save the user preference for dark theme
```

```
Store the API key configuration for the email service
```

```
Retrieve all configurations for the current project
```

```
What's the saved value for the key "database_connection"?
```

### Vectors Layer (Semantic)

```
Save this note about the sprint planning meeting
```

```
Search for information related to database performance optimization
```

```
Find all notes about microservices architecture
```

```
Update the previous note about the authentication module with this new information
```

### Graph Layer (Relationships)

```
Create a "person" entity for Ana García, senior developer
```

```
Establish a "works_on" relationship between Ana García and the Ecommerce project
```

```
Show me all people working on the Ecommerce project
```

```
Find all connections of the API Gateway project (up to 2 levels)
```

### Hybrid Search

```
Search for everything related to "OAuth authentication" across all memory layers
```

```
Find information about developer Juan and his projects
```

## Best Practices

### Layer Selection

**Use Facts when:**
- You need exact retrieval by key
- Data is simple key-value pairs
- You know the exact key to search for
- Examples: configurations, flags, counters

**Use Vectors when:**
- You'll search by meaning, not exact key
- Content is free text (notes, descriptions)
- You want to find similar content
- Examples: meeting notes, learnings, ideas

**Use Graph when:**
- Data has meaningful relationships
- You need to navigate connections
- Structure is important
- Examples: people and projects, related concepts, dependencies

### User ID

**Important note:** If you're unsure which `user_id` to use, use the current project name as the user_id. This allows organizing memory by project or context.

### Data Organization

- **Consistent naming**: Use clear conventions for keys and entity types
- **Useful metadata**: Add metadata that facilitates filtering and organization
- **Regular cleanup**: Delete obsolete data to keep memory efficient

## Common Use Cases

### Configuration Management
```
Save the staging server URL as "staging_server_url"
Save the API timeout as "api_timeout" with value 30
```

### Conversation Memory
```
Save this important decision about the system architecture
Search for what we discussed about the caching system
```

### Knowledge Mapping
```
Create a "project" entity called "Billing System"
Create a "person" entity for tech lead María Rodríguez
Relate María Rodríguez to Billing System as "tech_lead"
Show me all projects led by María
```

### Learning Tracking
```
Save this learning about design patterns in Go
Search for what I learned about concurrency
```

## Advanced Hybrid Search

Hybrid search combines all three layers:

```
hybrid_search({
  "query": "microservices",
  "user_id": "ecommerce-project",
  "limit": 10
})
```

This will search:
- **Facts** with keys containing "microservices"
- **Vectors** semantically similar to "microservices"
- **Graph entities** related to "microservices"

## See More

For detailed documentation of each tool, use the help system:

```
how_to_use("memory")
how_to_use("save_fact")
how_to_use("add_vector")
how_to_use("create_entity")
how_to_use("hybrid_search")
```
