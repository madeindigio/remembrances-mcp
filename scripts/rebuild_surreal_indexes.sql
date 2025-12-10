-- Rebuild all indexes (drop + recreate) for remembrances-mcp.
-- Adjust namespace/database before running.
USE NS test DB test;

-- v1 indexes
REMOVE INDEX idx_kv_user_key ON kv_memories;
DEFINE INDEX idx_kv_user_key ON kv_memories FIELDS user_id, key UNIQUE;

REMOVE INDEX idx_vector_user ON vector_memories;
DEFINE INDEX idx_vector_user ON vector_memories FIELDS user_id;
REMOVE INDEX idx_embedding ON vector_memories;
DEFINE INDEX idx_embedding ON vector_memories FIELDS embedding MTREE DIMENSION 768 DIST COSINE;

REMOVE INDEX idx_kb_path ON knowledge_base;
DEFINE INDEX idx_kb_path ON knowledge_base FIELDS file_path UNIQUE;
REMOVE INDEX idx_kb_embedding ON knowledge_base;
DEFINE INDEX idx_kb_embedding ON knowledge_base FIELDS embedding MTREE DIMENSION 768 DIST COSINE;

REMOVE INDEX idx_entity_name ON entities;
DEFINE INDEX idx_entity_name ON entities FIELDS name;
REMOVE INDEX idx_entity_type ON entities;
DEFINE INDEX idx_entity_type ON entities FIELDS type;

-- v2 indexes
REMOVE INDEX idx_user_stats_user_id ON user_stats;
DEFINE INDEX idx_user_stats_user_id ON user_stats FIELDS user_id UNIQUE;

-- v9 code indexing indexes
REMOVE INDEX idx_code_project_id ON code_projects;
DEFINE INDEX idx_code_project_id ON code_projects FIELDS project_id UNIQUE;
REMOVE INDEX idx_code_project_name ON code_projects;
DEFINE INDEX idx_code_project_name ON code_projects FIELDS name;
REMOVE INDEX idx_code_project_status ON code_projects;
DEFINE INDEX idx_code_project_status ON code_projects FIELDS indexing_status;

REMOVE INDEX idx_code_file_project_path ON code_files;
DEFINE INDEX idx_code_file_project_path ON code_files FIELDS project_id, file_path UNIQUE;
REMOVE INDEX idx_code_file_project ON code_files;
DEFINE INDEX idx_code_file_project ON code_files FIELDS project_id;
REMOVE INDEX idx_code_file_language ON code_files;
DEFINE INDEX idx_code_file_language ON code_files FIELDS language;
REMOVE INDEX idx_code_file_hash ON code_files;
DEFINE INDEX idx_code_file_hash ON code_files FIELDS file_hash;

REMOVE INDEX idx_code_symbol_project ON code_symbols;
DEFINE INDEX idx_code_symbol_project ON code_symbols FIELDS project_id;
REMOVE INDEX idx_code_symbol_file ON code_symbols;
DEFINE INDEX idx_code_symbol_file ON code_symbols FIELDS project_id, file_path;
REMOVE INDEX idx_code_symbol_name_path ON code_symbols;
DEFINE INDEX idx_code_symbol_name_path ON code_symbols FIELDS project_id, name_path UNIQUE;
REMOVE INDEX idx_code_symbol_type ON code_symbols;
DEFINE INDEX idx_code_symbol_type ON code_symbols FIELDS symbol_type;
REMOVE INDEX idx_code_symbol_language ON code_symbols;
DEFINE INDEX idx_code_symbol_language ON code_symbols FIELDS language;
REMOVE INDEX idx_code_symbol_name ON code_symbols;
DEFINE INDEX idx_code_symbol_name ON code_symbols FIELDS name;
REMOVE INDEX idx_code_symbol_parent ON code_symbols;
DEFINE INDEX idx_code_symbol_parent ON code_symbols FIELDS parent_id;
REMOVE INDEX idx_code_symbol_embedding ON code_symbols;
DEFINE INDEX idx_code_symbol_embedding ON code_symbols FIELDS embedding MTREE DIMENSION 768 DIST COSINE;

REMOVE INDEX idx_code_job_project ON code_indexing_jobs;
DEFINE INDEX idx_code_job_project ON code_indexing_jobs FIELDS project_id;
REMOVE INDEX idx_code_job_status ON code_indexing_jobs;
DEFINE INDEX idx_code_job_status ON code_indexing_jobs FIELDS status;

-- v10 code_chunks indexes
REMOVE INDEX idx_code_chunk_symbol ON code_chunks;
DEFINE INDEX idx_code_chunk_symbol ON code_chunks FIELDS symbol_id;
REMOVE INDEX idx_code_chunk_project ON code_chunks;
DEFINE INDEX idx_code_chunk_project ON code_chunks FIELDS project_id;
REMOVE INDEX idx_code_chunk_file ON code_chunks;
DEFINE INDEX idx_code_chunk_file ON code_chunks FIELDS project_id, file_path;
REMOVE INDEX idx_code_chunk_unique ON code_chunks;
DEFINE INDEX idx_code_chunk_unique ON code_chunks FIELDS symbol_id, chunk_index UNIQUE;
REMOVE INDEX idx_code_chunk_embedding ON code_chunks;
DEFINE INDEX idx_code_chunk_embedding ON code_chunks FIELDS embedding MTREE DIMENSION 768 DIST COSINE;

-- v11 events indexes
REMOVE INDEX idx_events_user ON events;
DEFINE INDEX idx_events_user ON events FIELDS user_id;
REMOVE INDEX idx_events_subject ON events;
DEFINE INDEX idx_events_subject ON events FIELDS subject;
REMOVE INDEX idx_events_created ON events;
DEFINE INDEX idx_events_created ON events FIELDS created_at;
REMOVE INDEX idx_events_user_subject ON events;
DEFINE INDEX idx_events_user_subject ON events FIELDS user_id, subject;
REMOVE INDEX idx_events_embedding ON events;
DEFINE INDEX idx_events_embedding ON events FIELDS embedding MTREE DIMENSION 768 DIST COSINE;
REMOVE INDEX idx_events_content ON events;
DEFINE INDEX idx_events_content ON events FIELDS content SEARCH ANALYZER events_analyzer BM25;
