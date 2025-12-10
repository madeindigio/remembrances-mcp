# SurrealDB Index Rebuild Guide

This note explains how to rebuild all SurrealDB indexes for Remembrances-MCP using the bundled script `scripts/rebuild_surreal_indexes.sql`.

## What the script does
- Drops and recreates every index defined in the project migrations: MTREE (vectors), SEARCH (full-text), and regular indexes across tables such as `vector_memories`, `knowledge_base`, `code_*` tables, and `events`.
- Starts with `USE NS test DB test;` — adjust `NS/DB` if your deployment uses different names.

## Why you’d run it
- To clear `Index is corrupted: TreeStore::load` errors caused by corrupted MTREE/SEARCH indexes.
- To ensure index definitions match the current SurrealDB version (e.g., after upgrading to 2.3.x+ or 2.2.x+).
- To normalize indexes if the same data directory was opened by different SurrealDB builds (embedded vs external) and formats drifted.

## How to run manually (one-shot)
Replace connection/auth/NS/DB as needed:

```
surreal sql \
  --conn ws://localhost:8000 \
  --user root \
  --pass root \
  --ns test \
  --db test \
  --file scripts/rebuild_surreal_indexes.sql
```

## How to run automatically at startup
- **Wrapper approach**: start SurrealDB, wait for readiness, then execute the same `surreal sql ... --file scripts/rebuild_surreal_indexes.sql` before starting Remembrances.
- **Systemd/compose hook**: add a preStart/postStart or `depends_on` step that runs the SQL file once the DB is reachable.

## File location
- Script path: `scripts/rebuild_surreal_indexes.sql`
- Remember to adjust `USE NS ... DB ...;` in the script for your environment.

## Minimal subset (if you only want MTREE/SEARCH)
Keep these statements from the script:
- `idx_embedding` on `vector_memories`
- `idx_kb_embedding` on `knowledge_base`
- `idx_code_chunk_embedding` on `code_chunks`
- `idx_code_symbol_embedding` on `code_symbols`
- `idx_events_embedding` and `idx_events_content` on `events`
