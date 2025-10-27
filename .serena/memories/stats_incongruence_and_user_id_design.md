# Incongruence in Stats and user_id Design (Oct 2025)

## Problem
- Stats are queried by user_id, but some memory types (entities, relationships, documents) do not store user_id, so their stats are always global and return 0 for per-user queries.
- This is incongruent: querying stats by user_id for these types will often return 0, even if data exists globally.

## Solution Proposal
- Add an optional user_id field to all memory types (vectors, kv, entities, relationships, documents) so that data can be associated with a user if needed.
- If user_id is not set, treat as global; if set, stats can be filtered per user.
- For backward compatibility, user_id should be optional and default to global if missing.

## Knowledge Base (KB) Metadata
- KB documents accept metadata, which may have dynamic fields.
- JSON parsing for metadata must be flexible (map[string]interface{}), not expecting fixed fields.
- Metadata should be stored in SurrealDB as part of the document, associated with the element.

## Next Steps
- Refactor schema and storage logic to support optional user_id in all memory types.
- Update stats logic to aggregate both global and per-user data as needed.
- Ensure KB metadata is parsed and stored as a dynamic object.
