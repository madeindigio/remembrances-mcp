---
title: "Events & Logs"
linkTitle: "Events & Logs"
weight: 6
description: >
  Store and search temporal events with semantic search and time-based filters
---

The Events feature provides temporal storage for AI agents to track activities, conversations, logs, and milestones. Unlike regular memories, events are designed for time-ordered data that can be searched both semantically and by time ranges.

## What are Events?

Events are timestamped records that combine:
- **Semantic search**: Find events by meaning using vector embeddings
- **Text search**: BM25 full-text search for keyword matching
- **Time filters**: Query by date ranges or relative time (last 24 hours, last week, etc.)
- **Subject categorization**: Organize events by topic or type

This makes Events perfect for:
- 📝 **Conversation logs**: Track discussion topics and decisions
- 🔍 **Audit trails**: Record actions and changes over time
- 🚀 **Milestones**: Mark important achievements and progress
- ⚠️ **Error tracking**: Log and search through issues and incidents
- 📊 **Activity monitoring**: Track patterns and behaviors

## How to Use Events

### Saving Events

Use `save_event` to store a new event:

```
save_event({
  "user_id": "project-alpha",
  "subject": "conversation:sprint-planning",
  "content": "Discussed new features for Q1: user authentication improvements, dashboard redesign, and API rate limiting. Team agreed to prioritize auth first."
})
```

**Parameters:**
- `user_id` (required): Identifies who or what the event belongs to
- `subject` (required): Category or topic for the event
- `content` (required): The event content (gets embedded for semantic search)
- `metadata` (optional): Additional key-value data

**Subject Patterns:**

We recommend using a prefix pattern for subjects:
- `conversation:topic` – Discussion logs
- `log:category` – General logs
- `audit:action` – Audit trail entries
- `milestone:name` – Achievement markers
- `error:type` – Error/incident logs
- `task:project` – Task tracking

### Searching Events

Use `search_events` to find relevant events:

```
search_events({
  "user_id": "project-alpha",
  "query": "authentication security improvements"
})
```

This performs a **hybrid search** combining:
1. Vector similarity (semantic meaning)
2. BM25 text matching (keyword relevance)

#### Filtering by Subject

```
search_events({
  "user_id": "project-alpha",
  "subject": "conversation:sprint-planning"
})
```

#### Time-Based Filters

**Relative Time:**

```
search_events({
  "user_id": "project-alpha",
  "last_hours": 24
})
```

```
search_events({
  "user_id": "project-alpha",
  "last_days": 7
})
```

```
search_events({
  "user_id": "project-alpha",
  "last_months": 3
})
```

**Date Range:**

```
search_events({
  "user_id": "project-alpha",
  "from_date": "2025-01-01T00:00:00Z",
  "to_date": "2025-01-31T23:59:59Z"
})
```

#### Combining Filters

You can combine subject, query, and time filters:

```
search_events({
  "user_id": "project-alpha",
  "subject": "error:api",
  "query": "timeout connection failed",
  "last_days": 7,
  "limit": 20
})
```

## Use Cases

### Conversation Memory

Track important discussions across multiple sessions:

```
save_event({
  "user_id": "user-123",
  "subject": "conversation:project-review",
  "content": "User expressed concerns about deployment timeline. Agreed to weekly check-ins and MVP scope reduction.",
  "metadata": {"priority": "high", "followup": "true"}
})
```

Later, recall what was discussed:

```
search_events({
  "user_id": "user-123",
  "subject": "conversation:project-review",
  "query": "deployment timeline concerns"
})
```

### Development Logs

Track development activities and decisions:

```
save_event({
  "user_id": "myproject",
  "subject": "log:development",
  "content": "Refactored authentication module to use JWT tokens. Removed session-based auth. Added refresh token rotation."
})
```

### Error Tracking

Log and search through issues:

```
save_event({
  "user_id": "api-service",
  "subject": "error:database",
  "content": "Connection pool exhausted. 50 pending queries. Increased pool size from 10 to 25.",
  "metadata": {"severity": "high", "resolved": "true"}
})
```

Find similar issues:

```
search_events({
  "user_id": "api-service",
  "subject": "error:database",
  "query": "connection pool performance"
})
```

### Milestone Tracking

Mark and find important achievements:

```
save_event({
  "user_id": "product-launch",
  "subject": "milestone:release",
  "content": "Version 2.0 released to production. New features: dark mode, multi-language support, improved search."
})
```

## Best Practices

### Subject Organization

- Use consistent subject patterns across your events
- Keep subjects short but descriptive
- Use prefixes to group related events

### Content Quality

- Write descriptive, searchable content
- Include relevant keywords for better search results
- Add context that helps with semantic matching

### Metadata Usage

- Use metadata for structured data (severity, status, tags)
- Keep metadata simple – complex queries use subject and content
- Useful for filtering or display purposes

### Query Strategies

- Start with broad queries, then narrow down
- Use subject filters when you know the category
- Combine time filters with semantic search for recent relevant events
