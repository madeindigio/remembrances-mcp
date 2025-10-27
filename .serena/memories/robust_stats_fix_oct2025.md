# Robust Stats Fix (October 2025)

## Problem
Stats for all memory types (vector, entity, relationship, document, key-value, total_size_bytes) were incorrect due to wrong table usage and scoping in updateUserStat and GetStats.

## Solution
- updateUserStat now uses the correct table for each stat:
  - vector_count: user-specific, from vector_memories
  - entity_count: global, from entities
  - relationship_count: global, sum of all relationship tables (dynamic)
  - document_count: global, from knowledge_base
  - key_value_count: user-specific, from kv_memories
- total_size_bytes is now calculated:
  - For user: sum content in vector_memories, kv_memories, knowledge_base
  - For global: sum content in entities, all relationship tables, knowledge_base
- Added comments and helpers for maintainability.

## Validation
- Created test data for all memory types.
- Verified stats update and total_size_bytes calculation.
- All fields now robust and correct.

## Next steps
- Maintain this logic for future schema changes.
- See code comments for details.
