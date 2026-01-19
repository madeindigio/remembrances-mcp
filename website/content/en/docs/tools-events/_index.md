---
title: "Events Tools"
linkTitle: "Events Tools"
weight: 13
description: >
  Temporal event storage and hybrid search
---

The events system provides temporal storage for logs, conversations, auditing, and historical data with hybrid search (text + semantic).

## Available Tools

### save_event
Store an event with automatic timestamp and semantic categorization by subject.

### search_events
Query events using hybrid search (BM25 + vector similarity) with temporal filtering.

## Key Features

- **Automatic timestamps**: Each event is timestamped at creation
- **Categorization by subject**: Organize events with descriptive subject patterns
- **Hybrid search**: Combines text search (BM25) with semantic search (vectors)
- **Temporal filtering**: Query events by absolute dates or relative ranges
- **Flexible metadata**: Add additional context via JSON metadata

## Recommended Subject Patterns

Use descriptive patterns to categorize your events:

| Pattern | Use | Example |
|---------|-----|---------|
| `conversation:session_id` | Conversation logs | `conversation:chat_001` |
| `log:category` | Application logs | `log:build`, `log:deploy` |
| `audit:action` | Audit events | `audit:file_modified` |
| `milestone:name` | Project milestones | `milestone:v1.0_released` |
| `error:type` | Error events | `error:runtime`, `error:validation` |

## Recommended Prompts

### For Saving Events

**Conversation logs:**
```
Save this conversation with subject "conversation:session_123"
```

```
Record this user message in the current conversation
```

**Application logs:**
```
Save this build log with subject "log:build" and metadata status: "success"
```

```
Record production deployment event with current timestamp
```

**Auditing:**
```
Save an audit event: user modified config.yaml file
```

```
Record admin user's system access with subject "audit:login"
```

**Errors:**
```
Save this validation error with subject "error:validation"
```

```
Record database connection error with full stack trace
```

### For Searching Events

**By semantic content:**
```
Search for events related to authentication errors
```

```
Find conversations about production deployment
```

**By subject:**
```
Show me all events of type "log:build"
```

```
Retrieve conversation with session_id "chat_001"
```

**By time range:**
```
Search for errors from the last 24 hours
```

```
Show me all build events from the last 7 days
```

```
Find audit events from the last month
```

**By absolute dates:**
```
Search for events between 2025-01-01 and 2025-01-31
```

```
Show me deploy logs since January 15th
```

**Combined search:**
```
Search for "error:runtime" events from the last 48 hours mentioning "database"
```

```
Find conversations about microservices from the last week
```

## Temporal Queries

### Absolute Dates (RFC3339 format)

```json
{
  "from_date": "2025-01-01T00:00:00Z",
  "to_date": "2025-12-31T23:59:59Z"
}
```

### Relative Ranges (mutually exclusive)

Choose **one** of these parameters:

- `last_hours: 24` - Last 24 hours
- `last_days: 7` - Last 7 days
- `last_months: 3` - Last 3 months

## Hybrid Search

When you provide a query, the system performs hybrid search:

1. **BM25 text matching (50% weight)** - Finds events containing search terms
2. **Vector similarity (50% weight)** - Finds semantically related events

This ensures finding both exact matches and related content.

## Common Use Cases

### 1. Conversation History

**Save:**
```
Save each message with subject="conversation:session_123"
```

**Retrieve:**
```
Search for all events with subject "conversation:session_123"
```

### 2. Build Logs

**Save:**
```
Record build event with subject="log:build" and metadata {status: "success", duration: 45}
```

**Query:**
```
Search for failed builds from the last week
Show me all build events for project X
```

### 3. Security Auditing

**Save:**
```
Record user access with subject="audit:user_login" and metadata {user: "admin", ip: "192.168.1.10"}
```

**Query:**
```
Search for all admin user accesses in January
Find audit events related to file modifications
```

### 4. Error Tracking

**Save:**
```
Save error with subject="error:database" and full stack trace
```

**Query:**
```
Search for database errors from the last 48 hours
Find all "runtime" type errors from the last month
```

### 5. Project Milestones

**Save:**
```
Record milestone with subject="milestone:v2.0_released" and metadata {version: "2.0.0", features: [...]}
```

**Query:**
```
Show me all project milestones
Search for releases from the last 6 months
```

## Best Practices

### Subject Organization

- Use consistent conventions (namespace:identifier)
- Group related events with same prefix
- Include unique identifiers when relevant
- Keep subjects descriptive but concise

### Useful Metadata

```json
{
  "user_id": "admin",
  "action": "file_edit",
  "file_path": "/config/app.yaml",
  "status": "success",
  "duration_ms": 150
}
```

- Include relevant context for future filtering
- Use consistent data types
- Add information that facilitates debugging
- Don't duplicate information from subject or content

### Efficient Searches

**Specific:**
```
Search for exact events using subject and time range
```

**Exploratory:**
```
Use semantic search to find related events
```

**Combined:**
```
Combine subject + query + time range for maximum precision
```

## Workflow Integration

### CI/CD Pipeline

```
# During build
Save build start event
Save test results with metrics
Save successful deployment event

# For analysis
Search for failed builds from last week
Find production deployments from last month
```

### Application Debugging

```
# During execution
Save errors with full stack trace
Record important warnings

# For debugging
Search for errors related to module X
Find warnings before the error occurred
```

### Usage Analysis

```
# Activity tracking
Save user action events
Record performance metrics

# For analysis
Search for usage patterns in last month
Find actions by specific user
```

## See More

For detailed documentation of each tool:

```
how_to_use("events")
how_to_use("save_event")
how_to_use("search_events")
```

Also check the [Events](/en/docs/events/) documentation for more technical details.
