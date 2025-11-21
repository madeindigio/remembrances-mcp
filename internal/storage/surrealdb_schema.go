package storage

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/madeindigio/remembrances-mcp/internal/storage/migrations"
)

// Default MTREE embedding dimension used in schema. Keep in sync with schema statements.
const defaultMtreeDim = 768

// InitializeSchema creates all required tables and indexes
func (s *SurrealDBStorage) InitializeSchema(ctx context.Context) error {
	log.Println("Initializing SurrealDB schema...")

	// First, ensure schema_version table exists to track migrations
	err := s.ensureSchemaVersionTable(ctx)
	if err != nil {
		return fmt.Errorf("failed to create schema version table: %w", err)
	}

	// Get current schema version
	currentVersion, err := s.getCurrentSchemaVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current schema version: %w", err)
	}

	// Run migrations if needed
	targetVersion := 7 // v7: fix metadata to allow flexible fields
	if currentVersion < targetVersion {
		log.Printf("Running schema migrations from version %d to %d", currentVersion, targetVersion)
		err = s.runMigrations(ctx, currentVersion, targetVersion)
		if err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}
	} else {
		log.Printf("Schema is up to date (version %d)", currentVersion)
	}

	log.Println("Schema initialization completed")
	return nil
}

// ensureSchemaVersionTable creates the schema_version table if it doesn't exist
func (s *SurrealDBStorage) ensureSchemaVersionTable(ctx context.Context) error {
	// First check if the table exists
	exists, err := s.checkTableExists(ctx, "schema_version")
	if err != nil {
		// If we can't check, try to create anyway
		log.Printf("Warning: Could not check if schema_version table exists: %v", err)
	}

	if !exists {
		// Create the schema version table
		_, err := s.query(ctx, `
			DEFINE TABLE schema_version SCHEMAFULL;
			DEFINE FIELD version ON schema_version TYPE int;
			DEFINE FIELD applied_at ON schema_version TYPE datetime VALUE time::now();
		`, nil)

		if err != nil {
			// Check if it's an "already exists" error
			if s.isAlreadyExistsError(err) {
				log.Println("Schema version table already exists, continuing...")
				return nil
			}
			return fmt.Errorf("failed to create schema_version table: %w", err)
		}
		log.Println("Created schema_version table")
	} else {
		log.Println("Schema version table already exists")
	}

	return nil
}

// getCurrentSchemaVersion returns the current schema version, 0 if no version is set
func (s *SurrealDBStorage) getCurrentSchemaVersion(ctx context.Context) (int, error) {
	// Query the single version record with fixed ID
	result, err := s.query(ctx, `
		SELECT * FROM schema_version:current;
	`, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to query schema version: %w", err)
	}

	if result == nil || len(*result) == 0 {
		return 0, nil // No version set, start from 0
	}

	queryResult := (*result)[0]
	if queryResult.Status != "OK" || queryResult.Result == nil || len(queryResult.Result) == 0 {
		return 0, nil // No version set, start from 0
	}

	raw := queryResult.Result[0]["version"]
	switch v := raw.(type) {
	case float64:
		return int(v), nil
	case float32:
		return int(v), nil
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case uint64:
		return int(v), nil
	case string:
		// Try to parse numeric string
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed, nil
		}
		return 0, fmt.Errorf("invalid version format in schema_version table: non-numeric string")
	default:
		return 0, fmt.Errorf("invalid version format in schema_version table")
	}
}

// setSchemaVersion updates the schema version
func (s *SurrealDBStorage) setSchemaVersion(ctx context.Context, version int) error {
	// Use UPSERT with a fixed ID to ensure only one version record exists
	_, err := s.query(ctx, `
		UPSERT schema_version:current SET version = $version;
	`, map[string]interface{}{
		"version": version,
	})
	return err
}

// runMigrations executes migrations from currentVersion to targetVersion
func (s *SurrealDBStorage) runMigrations(ctx context.Context, currentVersion, targetVersion int) error {
	for version := currentVersion; version < targetVersion; version++ {
		nextVersion := version + 1
		log.Printf("Applying migration to version %d", nextVersion)

		err := s.applyMigration(ctx, nextVersion)
		if err != nil {
			return fmt.Errorf("failed to apply migration to version %d: %w", nextVersion, err)
		}

		err = s.setSchemaVersion(ctx, nextVersion)
		if err != nil {
			return fmt.Errorf("failed to update schema version to %d: %w", nextVersion, err)
		}
	}
	return nil
}

// applyMigration applies a specific migration version using the new migration structure
func (s *SurrealDBStorage) applyMigration(ctx context.Context, version int) error {
	// For embedded mode, we need to use a different approach
	// Migrations need to be executed through our query helper
	if s.useEmbedded {
		return s.applyMigrationEmbedded(ctx, version)
	}

	// Remote mode: use traditional migrations with s.db
	var migration migrations.Migration

	switch version {
	case 1:
		migration = migrations.NewMigrationV1(s.db)
	case 2:
		migration = migrations.NewMigrationV2(s.db)
	case 3:
		migration = migrations.NewMigrationV3(s.db)
	case 4:
		migration = migrations.NewMigrationV4(s.db)
	case 5:
		migration = migrations.NewMigrationV5(s.db)
	case 6:
		migration = migrations.NewV6DocumentChunks(s.db)
	case 7:
		migration = migrations.NewV7FlexibleMetadataFix(s.db)
	default:
		return fmt.Errorf("unknown migration version: %d", version)
	}

	return migration.Apply(ctx, s.db)
}

// applyMigrationEmbedded applies migrations for embedded mode using direct SurrealQL
func (s *SurrealDBStorage) applyMigrationEmbedded(ctx context.Context, version int) error {
	log.Printf("Applying embedded migration version %d", version)

	var statements []string

	switch version {
	case 1:
		// V1: Initial schema with all tables
		statements = []string{
			// KV Memories table
			`DEFINE TABLE kv_memories SCHEMAFULL;`,
			`DEFINE FIELD user_id ON kv_memories TYPE string;`,
			`DEFINE FIELD key ON kv_memories TYPE string;`,
			`DEFINE FIELD value ON kv_memories TYPE option<string | int | float | bool | object | array>;`,
			`DEFINE FIELD created_at ON kv_memories TYPE datetime DEFAULT time::now();`,
			`DEFINE FIELD updated_at ON kv_memories TYPE datetime DEFAULT time::now();`,
			`DEFINE INDEX idx_kv_user_key ON kv_memories FIELDS user_id, key UNIQUE;`,

			// Vector Memories table
			fmt.Sprintf(`DEFINE TABLE vector_memories SCHEMAFULL;`),
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
	case 2:
		// V2: Add user_stats table
		statements = []string{
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
	case 3:
		// V3: Fix user_stats field types
		log.Println("Migration V3: Fixing user_stats schema (embedded mode uses direct field updates)")
	case 4:
		// V4: Add user_id everywhere (already in V1 for embedded)
		log.Println("Migration V4: user_id fields already present in embedded schema")
	case 5:
		// V5: Flexible metadata/properties (already in V1 for embedded)
		log.Println("Migration V5: flexible metadata already present in embedded schema")
	case 6:
		// V6: Document chunking support
		statements = []string{
			`DEFINE FIELD chunk_index ON knowledge_base TYPE int DEFAULT 0;`,
			`DEFINE FIELD chunk_count ON knowledge_base TYPE int DEFAULT 0;`,
			`DEFINE FIELD source_file ON knowledge_base TYPE option<string>;`,
		}
	case 7:
		// V7: Fix metadata/properties fields to be FLEXIBLE
		// This allows dynamic nested fields in metadata objects
		statements = []string{
			// Remove old field definitions
			`REMOVE FIELD metadata ON vector_memories;`,
			`REMOVE FIELD metadata ON knowledge_base;`,
			`REMOVE FIELD properties ON entities;`,
			// Redefine with FLEXIBLE
			`DEFINE FIELD metadata ON vector_memories FLEXIBLE TYPE object DEFAULT {};`,
			`DEFINE FIELD metadata ON knowledge_base FLEXIBLE TYPE object DEFAULT {};`,
			`DEFINE FIELD properties ON entities FLEXIBLE TYPE object DEFAULT {};`,
		}
		log.Println("Migration V7: Fixed metadata/properties fields to be FLEXIBLE (allows dynamic nested fields)")
	default:
		return fmt.Errorf("unknown migration version: %d", version)
	}

	// Execute each statement
	for _, stmt := range statements {
		_, err := s.query(ctx, stmt, nil)
		if err != nil {
			// Check if it's an "already exists" error and continue
			if s.isAlreadyExistsError(err) {
				log.Printf("Schema element already exists, continuing: %v", err)
				continue
			}
			return fmt.Errorf("failed to execute migration statement: %w\nStatement: %s", err, stmt)
		}
	}

	log.Printf("Successfully applied embedded migration version %d", version)
	return nil
}

// checkTableExists checks if a table exists
func (s *SurrealDBStorage) checkTableExists(ctx context.Context, tableName string) (bool, error) {
	if s.useEmbedded {
		// For embedded mode, try a simple query on the table
		result, err := s.query(ctx, fmt.Sprintf("SELECT * FROM %s LIMIT 1", tableName), nil)
		if err != nil {
			// If error contains "table" and "not found" or similar, table doesn't exist
			errStr := err.Error()
			if strings.Contains(errStr, "does not exist") || strings.Contains(errStr, "not found") {
				return false, nil
			}
			// Other error
			return false, err
		}
		// If query succeeded, table exists
		return result != nil, nil
	}

	// Remote mode: use migration base
	migrationBase := migrations.NewMigrationBase(s.db)
	return migrationBase.CheckTableExists(ctx, tableName)
}

// isAlreadyExistsError checks if an error is due to an element already existing
func (s *SurrealDBStorage) isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// Check for common "already exists" error messages
	return strings.Contains(errStr, "already exists") ||
		strings.Contains(errStr, "already defined") ||
		strings.Contains(errStr, "duplicate")
}
