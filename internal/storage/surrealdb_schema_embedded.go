package storage

import (
	"context"
	"fmt"
	"log/slog"
)

// applyMigrationEmbedded applies migrations for embedded mode using direct SurrealQL
func (s *SurrealDBStorage) applyMigrationEmbedded(ctx context.Context, version int) error {
	slog.Info("Applying embedded migration", "version", version)

	statements := s.getEmbeddedMigrationStatements(version)
	if statements == nil {
		return fmt.Errorf("unknown migration version: %d", version)
	}

	// Execute each statement
	for _, stmt := range statements {
		_, err := s.query(ctx, stmt, nil)
		if err != nil {
			// Check if it's an "already exists" error and continue
			if s.isAlreadyExistsError(err) {
				slog.Debug("Schema element already exists, continuing", "error", err)
				continue
			}
			return fmt.Errorf("failed to execute migration statement: %w\nStatement: %s", err, stmt)
		}
	}

	slog.Info("Successfully applied embedded migration", "version", version)
	return nil
}

// getEmbeddedMigrationStatements returns the SQL statements for a specific migration version
func (s *SurrealDBStorage) getEmbeddedMigrationStatements(version int) []string {
	switch version {
	case 1:
		return s.getMigrationV1Statements()
	case 2:
		return s.getMigrationV2Statements()
	case 3:
		slog.Debug("Migration V3: Fixing user_stats schema (embedded mode uses direct field updates)")
		return []string{}
	case 4:
		slog.Debug("Migration V4: user_id fields already present in embedded schema")
		return []string{}
	case 5:
		slog.Debug("Migration V5: flexible metadata already present in embedded schema")
		return []string{}
	case 6:
		return s.getMigrationV6Statements()
	case 7:
		return s.getMigrationV7Statements()
	case 8:
		return s.getMigrationV8Statements()
	case 9:
		return s.getMigrationV9Statements()
	case 10:
		return s.getMigrationV10Statements()
	case 11:
		return s.getMigrationV11Statements()
	case 12:
		return s.getMigrationV12Statements()
	default:
		return nil
	}
}

// getMigrationV1Statements returns V1 migration statements (initial schema)
func (s *SurrealDBStorage) getMigrationV1Statements() []string {
	return []string{
		// KV Memories table
		`DEFINE TABLE kv_memories SCHEMAFULL;`,
		`DEFINE FIELD user_id ON kv_memories TYPE string;`,
		`DEFINE FIELD key ON kv_memories TYPE string;`,
		`DEFINE FIELD value ON kv_memories FLEXIBLE TYPE option<string | int | float | bool | object | array>;`,
		`DEFINE FIELD created_at ON kv_memories TYPE datetime DEFAULT time::now();`,
		`DEFINE FIELD updated_at ON kv_memories TYPE datetime DEFAULT time::now();`,
		`DEFINE INDEX idx_kv_user_key ON kv_memories FIELDS user_id, key UNIQUE;`,

		// Vector Memories table
		`DEFINE TABLE vector_memories SCHEMAFULL;`,
		`DEFINE FIELD user_id ON vector_memories TYPE option<string>;`,
		`DEFINE FIELD content ON vector_memories TYPE string;`,
		fmt.Sprintf(`DEFINE FIELD embedding ON vector_memories TYPE array<float, %d>;`, defaultMtreeDim),
		`DEFINE FIELD metadata ON vector_memories FLEXIBLE TYPE object DEFAULT {};`,
		`DEFINE FIELD created_at ON vector_memories TYPE datetime DEFAULT time::now();`,
		`DEFINE FIELD updated_at ON vector_memories TYPE datetime DEFAULT time::now();`,
		fmt.Sprintf(`DEFINE INDEX idx_vector_embedding ON vector_memories FIELDS embedding MTREE DIMENSION %d;`, defaultMtreeDim),

		// Knowledge Base table
		`DEFINE TABLE knowledge_base SCHEMAFULL;`,
		`DEFINE FIELD file_path ON knowledge_base TYPE string;`,
		`DEFINE FIELD content ON knowledge_base TYPE string;`,
		fmt.Sprintf(`DEFINE FIELD embedding ON knowledge_base TYPE array<float, %d>;`, defaultMtreeDim),
		`DEFINE FIELD metadata ON knowledge_base FLEXIBLE TYPE object DEFAULT {};`,
		`DEFINE FIELD created_at ON knowledge_base TYPE datetime DEFAULT time::now();`,
		`DEFINE FIELD updated_at ON knowledge_base TYPE datetime DEFAULT time::now();`,
		`DEFINE INDEX idx_kb_file_path ON knowledge_base FIELDS file_path UNIQUE;`,
		fmt.Sprintf(`DEFINE INDEX idx_kb_embedding ON knowledge_base FIELDS embedding MTREE DIMENSION %d;`, defaultMtreeDim),

		// Entities table
		`DEFINE TABLE entities SCHEMAFULL;`,
		`DEFINE FIELD entity_type ON entities TYPE string;`,
		`DEFINE FIELD name ON entities TYPE string;`,
		`DEFINE FIELD properties ON entities FLEXIBLE TYPE object DEFAULT {};`,
		`DEFINE FIELD created_at ON entities TYPE datetime DEFAULT time::now();`,
		`DEFINE INDEX idx_entity_name ON entities FIELDS name;`,
	}
}

// getMigrationV2Statements returns V2 migration statements (user_stats table)
func (s *SurrealDBStorage) getMigrationV2Statements() []string {
	return []string{
		`DEFINE TABLE user_stats SCHEMAFULL;`,
		`DEFINE FIELD user_id ON user_stats TYPE string;`,
		`DEFINE FIELD fact_count ON user_stats TYPE int DEFAULT 0;`,
		`DEFINE FIELD vector_count ON user_stats TYPE int DEFAULT 0;`,
		`DEFINE FIELD document_count ON user_stats TYPE int DEFAULT 0;`,
		`DEFINE FIELD entity_count ON user_stats TYPE int DEFAULT 0;`,
		`DEFINE FIELD relationship_count ON user_stats TYPE int DEFAULT 0;`,
		`DEFINE FIELD last_updated ON user_stats TYPE datetime DEFAULT time::now();`,
		`DEFINE INDEX idx_user_stats_user_id ON user_stats FIELDS user_id UNIQUE;`,
	}
}

// getMigrationV6Statements returns V6 migration statements (document chunking)
func (s *SurrealDBStorage) getMigrationV6Statements() []string {
	return []string{
		`DEFINE FIELD chunk_index ON knowledge_base TYPE int DEFAULT 0;`,
		`DEFINE FIELD chunk_count ON knowledge_base TYPE int DEFAULT 0;`,
		`DEFINE FIELD source_file ON knowledge_base TYPE option<string>;`,
	}
}

// getMigrationV7Statements returns V7 migration statements (flexible metadata)
func (s *SurrealDBStorage) getMigrationV7Statements() []string {
	slog.Debug("Migration V7: Fixed metadata/properties fields to be FLEXIBLE")
	return []string{
		// Remove old field definitions
		`REMOVE FIELD metadata ON vector_memories;`,
		`REMOVE FIELD metadata ON knowledge_base;`,
		`REMOVE FIELD properties ON entities;`,
		// Redefine with FLEXIBLE
		`DEFINE FIELD metadata ON vector_memories FLEXIBLE TYPE object DEFAULT {};`,
		`DEFINE FIELD metadata ON knowledge_base FLEXIBLE TYPE object DEFAULT {};`,
		`DEFINE FIELD properties ON entities FLEXIBLE TYPE object DEFAULT {};`,
	}
}

// getMigrationV8Statements returns V8 migration statements (flexible KV value)
func (s *SurrealDBStorage) getMigrationV8Statements() []string {
	slog.Debug("Migration V8: Fixed kv_memories value field to be FLEXIBLE")
	return []string{
		// Remove old field definition
		`REMOVE FIELD value ON kv_memories;`,
		// Redefine with FLEXIBLE
		`DEFINE FIELD value ON kv_memories FLEXIBLE TYPE option<string | int | float | bool | object | array>;`,
	}
}

// getMigrationV9Statements returns V9 migration statements (code indexing schema)
func (s *SurrealDBStorage) getMigrationV9Statements() []string {
	slog.Debug("Migration V9: Creating code indexing schema")
	return []string{
		// code_projects table
		`DEFINE TABLE code_projects SCHEMAFULL;`,
		`DEFINE FIELD project_id ON code_projects TYPE string;`,
		`DEFINE FIELD name ON code_projects TYPE string;`,
		`DEFINE FIELD root_path ON code_projects TYPE string;`,
		`DEFINE FIELD language_stats ON code_projects FLEXIBLE TYPE option<object>;`,
		`DEFINE FIELD last_indexed_at ON code_projects TYPE option<datetime>;`,
		`DEFINE FIELD indexing_status ON code_projects TYPE string DEFAULT 'pending';`,
		`DEFINE FIELD created_at ON code_projects TYPE datetime DEFAULT time::now();`,
		`DEFINE FIELD updated_at ON code_projects TYPE datetime DEFAULT time::now();`,
		`DEFINE INDEX idx_code_project_id ON code_projects FIELDS project_id UNIQUE;`,
		`DEFINE INDEX idx_code_project_name ON code_projects FIELDS name;`,
		`DEFINE INDEX idx_code_project_status ON code_projects FIELDS indexing_status;`,

		// code_files table
		`DEFINE TABLE code_files SCHEMAFULL;`,
		`DEFINE FIELD project_id ON code_files TYPE string;`,
		`DEFINE FIELD file_path ON code_files TYPE string;`,
		`DEFINE FIELD language ON code_files TYPE string;`,
		`DEFINE FIELD file_hash ON code_files TYPE string;`,
		`DEFINE FIELD symbols_count ON code_files TYPE int DEFAULT 0;`,
		`DEFINE FIELD indexed_at ON code_files TYPE datetime DEFAULT time::now();`,
		`DEFINE INDEX idx_code_file_project_path ON code_files FIELDS project_id, file_path UNIQUE;`,
		`DEFINE INDEX idx_code_file_project ON code_files FIELDS project_id;`,
		`DEFINE INDEX idx_code_file_language ON code_files FIELDS language;`,
		`DEFINE INDEX idx_code_file_hash ON code_files FIELDS file_hash;`,

		// code_symbols table
		`DEFINE TABLE code_symbols SCHEMAFULL;`,
		`DEFINE FIELD project_id ON code_symbols TYPE string;`,
		`DEFINE FIELD file_path ON code_symbols TYPE string;`,
		`DEFINE FIELD language ON code_symbols TYPE string;`,
		`DEFINE FIELD symbol_type ON code_symbols TYPE string;`,
		`DEFINE FIELD name ON code_symbols TYPE string;`,
		`DEFINE FIELD name_path ON code_symbols TYPE string;`,
		`DEFINE FIELD start_line ON code_symbols TYPE int;`,
		`DEFINE FIELD end_line ON code_symbols TYPE int;`,
		`DEFINE FIELD start_byte ON code_symbols TYPE int;`,
		`DEFINE FIELD end_byte ON code_symbols TYPE int;`,
		`DEFINE FIELD source_code ON code_symbols TYPE option<string>;`,
		`DEFINE FIELD signature ON code_symbols TYPE option<string>;`,
		`DEFINE FIELD doc_string ON code_symbols TYPE option<string>;`,
		fmt.Sprintf(`DEFINE FIELD embedding ON code_symbols TYPE option<array<float, %d>>;`, defaultMtreeDim),
		`DEFINE FIELD parent_id ON code_symbols TYPE option<string>;`,
		`DEFINE FIELD metadata ON code_symbols FLEXIBLE TYPE option<object>;`,
		`DEFINE FIELD created_at ON code_symbols TYPE datetime DEFAULT time::now();`,
		`DEFINE FIELD updated_at ON code_symbols TYPE datetime DEFAULT time::now();`,
		`DEFINE INDEX idx_code_symbol_project ON code_symbols FIELDS project_id;`,
		`DEFINE INDEX idx_code_symbol_file ON code_symbols FIELDS project_id, file_path;`,
		`DEFINE INDEX idx_code_symbol_name_path ON code_symbols FIELDS project_id, name_path UNIQUE;`,
		`DEFINE INDEX idx_code_symbol_type ON code_symbols FIELDS symbol_type;`,
		`DEFINE INDEX idx_code_symbol_language ON code_symbols FIELDS language;`,
		`DEFINE INDEX idx_code_symbol_name ON code_symbols FIELDS name;`,
		`DEFINE INDEX idx_code_symbol_parent ON code_symbols FIELDS parent_id;`,
		fmt.Sprintf(`DEFINE INDEX idx_code_symbol_embedding ON code_symbols FIELDS embedding MTREE DIMENSION %d;`, defaultMtreeDim),

		// code_indexing_jobs table
		`DEFINE TABLE code_indexing_jobs SCHEMAFULL;`,
		`DEFINE FIELD project_id ON code_indexing_jobs TYPE string;`,
		`DEFINE FIELD project_path ON code_indexing_jobs TYPE string;`,
		`DEFINE FIELD status ON code_indexing_jobs TYPE string DEFAULT 'pending';`,
		`DEFINE FIELD progress ON code_indexing_jobs TYPE float DEFAULT 0.0;`,
		`DEFINE FIELD files_total ON code_indexing_jobs TYPE int DEFAULT 0;`,
		`DEFINE FIELD files_indexed ON code_indexing_jobs TYPE int DEFAULT 0;`,
		`DEFINE FIELD started_at ON code_indexing_jobs TYPE datetime DEFAULT time::now();`,
		`DEFINE FIELD completed_at ON code_indexing_jobs TYPE option<datetime>;`,
		`DEFINE FIELD error ON code_indexing_jobs TYPE option<string>;`,
		`DEFINE INDEX idx_code_job_project ON code_indexing_jobs FIELDS project_id;`,
		`DEFINE INDEX idx_code_job_status ON code_indexing_jobs FIELDS status;`,
	}
}

// getMigrationV10Statements returns V10 migration statements (code chunks table)
func (s *SurrealDBStorage) getMigrationV10Statements() []string {
	slog.Debug("Migration V10: Creating code_chunks table")
	return []string{
		// code_chunks table
		`DEFINE TABLE code_chunks SCHEMAFULL;`,
		`DEFINE FIELD symbol_id ON code_chunks TYPE string;`,
		`DEFINE FIELD project_id ON code_chunks TYPE string;`,
		`DEFINE FIELD file_path ON code_chunks TYPE string;`,
		`DEFINE FIELD chunk_index ON code_chunks TYPE int;`,
		`DEFINE FIELD chunk_count ON code_chunks TYPE int;`,
		`DEFINE FIELD content ON code_chunks TYPE string;`,
		`DEFINE FIELD start_offset ON code_chunks TYPE int;`,
		`DEFINE FIELD end_offset ON code_chunks TYPE int;`,
		fmt.Sprintf(`DEFINE FIELD embedding ON code_chunks TYPE option<array<float, %d>>;`, defaultMtreeDim),
		`DEFINE FIELD symbol_name ON code_chunks TYPE string;`,
		`DEFINE FIELD symbol_type ON code_chunks TYPE string;`,
		`DEFINE FIELD language ON code_chunks TYPE string;`,
		`DEFINE FIELD created_at ON code_chunks TYPE datetime DEFAULT time::now();`,
		`DEFINE INDEX idx_code_chunk_symbol ON code_chunks FIELDS symbol_id;`,
		`DEFINE INDEX idx_code_chunk_project ON code_chunks FIELDS project_id;`,
		`DEFINE INDEX idx_code_chunk_file ON code_chunks FIELDS project_id, file_path;`,
		`DEFINE INDEX idx_code_chunk_unique ON code_chunks FIELDS symbol_id, chunk_index UNIQUE;`,
		fmt.Sprintf(`DEFINE INDEX idx_code_chunk_embedding ON code_chunks FIELDS embedding MTREE DIMENSION %d;`, defaultMtreeDim),
	}
}

// getMigrationV11Statements returns V11 migration statements (events table)
func (s *SurrealDBStorage) getMigrationV11Statements() []string {
	slog.Debug("Migration V11: Creating events table")
	return []string{
		// events table
		`DEFINE TABLE events SCHEMAFULL;`,
		`DEFINE FIELD user_id ON events TYPE string;`,
		`DEFINE FIELD subject ON events TYPE string;`,
		`DEFINE FIELD content ON events TYPE string;`,
		fmt.Sprintf(`DEFINE FIELD embedding ON events TYPE array<float, %d>;`, defaultMtreeDim),
		`DEFINE FIELD metadata ON events FLEXIBLE TYPE option<object>;`,
		`DEFINE FIELD created_at ON events TYPE datetime DEFAULT time::now();`,
		// Indexes
		`DEFINE INDEX idx_events_user ON events FIELDS user_id;`,
		`DEFINE INDEX idx_events_subject ON events FIELDS subject;`,
		`DEFINE INDEX idx_events_created ON events FIELDS created_at;`,
		`DEFINE INDEX idx_events_user_subject ON events FIELDS user_id, subject;`,
		fmt.Sprintf(`DEFINE INDEX idx_events_embedding ON events FIELDS embedding MTREE DIMENSION %d DIST COSINE;`, defaultMtreeDim),
		// Full-text search with BM25
		`DEFINE ANALYZER events_analyzer TOKENIZERS blank, class FILTERS lowercase, snowball(english);`,
		`DEFINE INDEX idx_events_content ON events FIELDS content SEARCH ANALYZER events_analyzer BM25;`,
	}
}

// getMigrationV12Statements returns V12 migration statements (watcher_enabled field)
func (s *SurrealDBStorage) getMigrationV12Statements() []string {
	slog.Debug("Migration V12: Adding watcher_enabled field to code_projects")
	return []string{
		`DEFINE FIELD watcher_enabled ON code_projects TYPE bool DEFAULT false;`,
	}
}
