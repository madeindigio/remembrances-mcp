package storage

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/surrealdb/surrealdb.go"
)

// Default MTREE embedding dimension used in schema. Keep in sync with schema statements.
const defaultMtreeDim = 768

// SchemaElement represents a schema element for migrations
type SchemaElement struct {
	Type      string // "table", "field", "index"
	Statement string // The SurrealQL statement to execute
	OnTable   string // For fields and indexes, the table they belong to
}

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
	targetVersion := 3 // Current target version - updated to fix user_stats field definitions
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
		_, err := surrealdb.Query[[]map[string]interface{}](s.db, `
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
	result, err := surrealdb.Query[[]map[string]interface{}](s.db, `
		SELECT * FROM schema_version ORDER BY version DESC LIMIT 1;
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
	// The CREATE statement returns an array-like result; request the matching type to avoid CBOR unmarshal errors.
	_, err := surrealdb.Query[[]map[string]interface{}](s.db, `
		CREATE schema_version SET version = $version;
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

// applyMigration applies a specific migration version
func (s *SurrealDBStorage) applyMigration(ctx context.Context, version int) error {
	switch version {
	case 1:
		return s.applyMigrationV1(ctx)
	case 2:
		return s.applyMigrationV2(ctx)
	case 3:
		return s.applyMigrationV3(ctx)
	default:
		return fmt.Errorf("unknown migration version: %d", version)
	}
}

// applyMigrationV1 creates the initial schema (version 1)
func (s *SurrealDBStorage) applyMigrationV1(ctx context.Context) error {
	log.Println("Applying migration v1: Creating initial schema")

	elements := []SchemaElement{
		// Tables first
		{Type: "table", Statement: `DEFINE TABLE kv_memories SCHEMAFULL;`},
		{Type: "table", Statement: `DEFINE TABLE vector_memories SCHEMAFULL;`},
		{Type: "table", Statement: `DEFINE TABLE knowledge_base SCHEMAFULL;`},
		{Type: "table", Statement: `DEFINE TABLE entities SCHEMAFULL;`},
		{Type: "table", Statement: `DEFINE TABLE wrote SCHEMALESS;`},
		{Type: "table", Statement: `DEFINE TABLE mentioned_in SCHEMALESS;`},
		{Type: "table", Statement: `DEFINE TABLE related_to SCHEMALESS;`},

		// Fields for kv_memories
		{Type: "field", Statement: `DEFINE FIELD user_id ON kv_memories TYPE string;`, OnTable: "kv_memories"},
		{Type: "field", Statement: `DEFINE FIELD key ON kv_memories TYPE string;`, OnTable: "kv_memories"},
		{Type: "field", Statement: `DEFINE FIELD value ON kv_memories TYPE any;`, OnTable: "kv_memories"},
		{Type: "field", Statement: `DEFINE FIELD created_at ON kv_memories TYPE datetime VALUE time::now();`, OnTable: "kv_memories"},
		// SurrealDB does not support `ON UPDATE` in field definitions in this client/schema version,
		// keep updated_at with a VALUE default and let the application set it on updates.
		{Type: "field", Statement: `DEFINE FIELD updated_at ON kv_memories TYPE datetime VALUE time::now();`, OnTable: "kv_memories"},

		// Fields for vector_memories
		{Type: "field", Statement: `DEFINE FIELD user_id ON vector_memories TYPE string;`, OnTable: "vector_memories"},
		{Type: "field", Statement: `DEFINE FIELD content ON vector_memories TYPE string;`, OnTable: "vector_memories"},
		{Type: "field", Statement: `DEFINE FIELD embedding ON vector_memories TYPE array<float>;`, OnTable: "vector_memories"},
		{Type: "field", Statement: `DEFINE FIELD metadata ON vector_memories TYPE object;`, OnTable: "vector_memories"},
		{Type: "field", Statement: `DEFINE FIELD created_at ON vector_memories TYPE datetime VALUE time::now();`, OnTable: "vector_memories"},
		{Type: "field", Statement: `DEFINE FIELD updated_at ON vector_memories TYPE datetime VALUE time::now();`, OnTable: "vector_memories"},

		// Fields for knowledge_base
		{Type: "field", Statement: `DEFINE FIELD file_path ON knowledge_base TYPE string;`, OnTable: "knowledge_base"},
		{Type: "field", Statement: `DEFINE FIELD content ON knowledge_base TYPE string;`, OnTable: "knowledge_base"},
		{Type: "field", Statement: `DEFINE FIELD embedding ON knowledge_base TYPE array<float>;`, OnTable: "knowledge_base"},
		{Type: "field", Statement: `DEFINE FIELD metadata ON knowledge_base TYPE object;`, OnTable: "knowledge_base"},
		{Type: "field", Statement: `DEFINE FIELD created_at ON knowledge_base TYPE datetime VALUE time::now();`, OnTable: "knowledge_base"},
		{Type: "field", Statement: `DEFINE FIELD updated_at ON knowledge_base TYPE datetime VALUE time::now();`, OnTable: "knowledge_base"},

		// Fields for entities
		{Type: "field", Statement: `DEFINE FIELD name ON entities TYPE string;`, OnTable: "entities"},
		{Type: "field", Statement: `DEFINE FIELD type ON entities TYPE string;`, OnTable: "entities"},
		{Type: "field", Statement: `DEFINE FIELD properties ON entities TYPE object;`, OnTable: "entities"},
		{Type: "field", Statement: `DEFINE FIELD created_at ON entities TYPE datetime VALUE time::now();`, OnTable: "entities"},
		{Type: "field", Statement: `DEFINE FIELD updated_at ON entities TYPE datetime VALUE time::now();`, OnTable: "entities"},

		// Fields for relationship tables
		{Type: "field", Statement: `DEFINE FIELD in ON wrote TYPE record;`, OnTable: "wrote"},
		{Type: "field", Statement: `DEFINE FIELD out ON wrote TYPE record;`, OnTable: "wrote"},
		{Type: "field", Statement: `DEFINE FIELD timestamp ON wrote TYPE datetime VALUE time::now();`, OnTable: "wrote"},
		{Type: "field", Statement: `DEFINE FIELD properties ON wrote TYPE object;`, OnTable: "wrote"},

		{Type: "field", Statement: `DEFINE FIELD in ON mentioned_in TYPE string;`, OnTable: "mentioned_in"},
		{Type: "field", Statement: `DEFINE FIELD out ON mentioned_in TYPE string;`, OnTable: "mentioned_in"},
		{Type: "field", Statement: `DEFINE FIELD timestamp ON mentioned_in TYPE datetime VALUE time::now();`, OnTable: "mentioned_in"},
		{Type: "field", Statement: `DEFINE FIELD properties ON mentioned_in TYPE object;`, OnTable: "mentioned_in"},

		{Type: "field", Statement: `DEFINE FIELD in ON related_to TYPE string;`, OnTable: "related_to"},
		{Type: "field", Statement: `DEFINE FIELD out ON related_to TYPE string;`, OnTable: "related_to"},
		{Type: "field", Statement: `DEFINE FIELD timestamp ON related_to TYPE datetime VALUE time::now();`, OnTable: "related_to"},
		{Type: "field", Statement: `DEFINE FIELD properties ON related_to TYPE object;`, OnTable: "related_to"},

		// Indexes
		{Type: "index", Statement: `DEFINE INDEX idx_kv_user_key ON kv_memories FIELDS user_id, key UNIQUE;`, OnTable: "kv_memories"},
		{Type: "index", Statement: `DEFINE INDEX idx_vector_user ON vector_memories FIELDS user_id;`, OnTable: "vector_memories"},
		{Type: "index", Statement: `DEFINE INDEX idx_embedding ON vector_memories FIELDS embedding MTREE DIMENSION 768 DIST COSINE;`, OnTable: "vector_memories"},
		{Type: "index", Statement: `DEFINE INDEX idx_kb_path ON knowledge_base FIELDS file_path UNIQUE;`, OnTable: "knowledge_base"},
		{Type: "index", Statement: `DEFINE INDEX idx_kb_embedding ON knowledge_base FIELDS embedding MTREE DIMENSION 768 DIST COSINE;`, OnTable: "knowledge_base"},
		{Type: "index", Statement: `DEFINE INDEX idx_entity_name ON entities FIELDS name;`, OnTable: "entities"},
		{Type: "index", Statement: `DEFINE INDEX idx_entity_type ON entities FIELDS type;`, OnTable: "entities"},
	}

	// Apply each schema element with error handling
	for i, element := range elements {
		exists, err := s.checkSchemaElementExists(ctx, element)
		if err != nil {
			log.Printf("Warning: Could not check existence of %s, attempting to create anyway: %v", element.Type, err)
		}

		if !exists {
			log.Printf("Creating %s: %s", element.Type, element.Statement)
			_, err := surrealdb.Query[[]map[string]interface{}](s.db, element.Statement, nil)
			if err != nil {
				// Log warning but don't fail for "already exists" type errors
				if s.isAlreadyExistsError(err) {
					log.Printf("Warning: %s already exists, continuing...", element.Type)
				} else {
					return fmt.Errorf("failed to execute migration statement %d '%s': %w", i+1, element.Statement, err)
				}
			}
		} else {
			log.Printf("Skipping existing %s", element.Type)
		}
	}

	return nil
}

// applyMigrationV2 adds the user_stats table for tracking user-scoped memory statistics
func (s *SurrealDBStorage) applyMigrationV2(ctx context.Context) error {
	log.Println("Applying migration v2: Adding user_stats table")

	elements := []SchemaElement{
		// Add user_stats table
		{Type: "table", Statement: `DEFINE TABLE user_stats SCHEMAFULL;`},

		// Fields for user_stats
		{Type: "field", Statement: `DEFINE FIELD user_id ON user_stats TYPE string;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD key_value_count ON user_stats TYPE int VALUE 0;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD vector_count ON user_stats TYPE int VALUE 0;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD entity_count ON user_stats TYPE int VALUE 0;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD relationship_count ON user_stats TYPE int VALUE 0;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD document_count ON user_stats TYPE int VALUE 0;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD created_at ON user_stats TYPE datetime VALUE time::now();`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD updated_at ON user_stats TYPE datetime VALUE time::now();`, OnTable: "user_stats"},

		// Index for efficient user lookups
		{Type: "index", Statement: `DEFINE INDEX idx_user_stats_user_id ON user_stats FIELDS user_id UNIQUE;`, OnTable: "user_stats"},
	}

	// Apply each schema element with error handling
	for i, element := range elements {
		exists, err := s.checkSchemaElementExists(ctx, element)
		if err != nil {
			log.Printf("Warning: Could not check existence of %s, attempting to create anyway: %v", element.Type, err)
		}

		if !exists {
			log.Printf("Creating %s: %s", element.Type, element.Statement)
			_, err := surrealdb.Query[[]map[string]interface{}](s.db, element.Statement, nil)
			if err != nil {
				// Log warning but don't fail for "already exists" type errors
				if s.isAlreadyExistsError(err) {
					log.Printf("Warning: %s already exists, continuing...", element.Type)
				} else {
					return fmt.Errorf("failed to execute migration statement %d '%s': %w", i+1, element.Statement, err)
				}
			}
		} else {
			log.Printf("Skipping existing %s", element.Type)
		}
	}

	return nil
}

// applyMigrationV3 fixes the user_stats field definitions by removing VALUE constraints
func (s *SurrealDBStorage) applyMigrationV3(ctx context.Context) error {
	log.Println("Applying migration v3: Fixing user_stats field definitions")

	// Remove the problematic fields that have VALUE constraints
	dropStatements := []string{
		"REMOVE FIELD key_value_count ON user_stats;",
		"REMOVE FIELD vector_count ON user_stats;",
		"REMOVE FIELD entity_count ON user_stats;",
		"REMOVE FIELD relationship_count ON user_stats;",
		"REMOVE FIELD document_count ON user_stats;",
	}

	// Execute drop statements
	for _, stmt := range dropStatements {
		log.Printf("Executing: %s", stmt)
		_, err := surrealdb.Query[[]map[string]interface{}](s.db, stmt, nil)
		if err != nil {
			// Log warning but continue - field might not exist
			log.Printf("Warning: Failed to drop field: %v", err)
		}
	}

	// Define the corrected fields without VALUE constraints
	elements := []SchemaElement{
		{Type: "field", Statement: `DEFINE FIELD key_value_count ON user_stats TYPE int;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD vector_count ON user_stats TYPE int;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD entity_count ON user_stats TYPE int;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD relationship_count ON user_stats TYPE int;`, OnTable: "user_stats"},
		{Type: "field", Statement: `DEFINE FIELD document_count ON user_stats TYPE int;`, OnTable: "user_stats"},
	}

	// Apply the corrected field definitions
	for i, element := range elements {
		log.Printf("Creating corrected field: %s", element.Statement)
		_, err := surrealdb.Query[[]map[string]interface{}](s.db, element.Statement, nil)
		if err != nil {
			// This should succeed since we removed the old definitions
			return fmt.Errorf("failed to execute migration v3 statement %d '%s': %w", i+1, element.Statement, err)
		}
	}

	log.Println("Migration v3 completed successfully")
	return nil
}

// checkSchemaElementExists checks if a schema element (table, field, index) already exists
func (s *SurrealDBStorage) checkSchemaElementExists(ctx context.Context, element SchemaElement) (bool, error) {
	switch element.Type {
	case "table":
		return s.checkTableExists(ctx, s.extractTableName(element.Statement))
	case "field":
		tableName := element.OnTable
		fieldName := s.extractFieldName(element.Statement)
		return s.checkFieldExists(ctx, tableName, fieldName)
	case "index":
		tableName := element.OnTable
		indexName := s.extractIndexName(element.Statement)
		return s.checkIndexExists(ctx, tableName, indexName)
	default:
		return false, fmt.Errorf("unknown schema element type: %s", element.Type)
	}
}

// checkTableExists checks if a table exists
func (s *SurrealDBStorage) checkTableExists(ctx context.Context, tableName string) (bool, error) {
	// The INFO FOR DB; command returns a QueryResult array. Use array-based unmarshalling.
	result, err := surrealdb.Query[[]map[string]interface{}](s.db, `INFO FOR DB;`, nil)
	if err != nil {
		return false, err
	}

	if result == nil || len(*result) == 0 {
		return false, nil
	}

	// Get the QueryResult wrapper and its typed Result map
	qr := (*result)[0]
	if qr.Status != "OK" {
		return false, nil
	}

	resMap := qr.Result
	if len(resMap) == 0 {
		return false, nil
	}

	// Extract the result data from the first element
	resultData := resMap[0]

	// Expect a "tables" key in the returned map
	tablesRaw, ok := resultData["tables"]
	if !ok || tablesRaw == nil {
		return false, nil
	}

	tables, ok := tablesRaw.(map[string]interface{})
	if !ok {
		return false, nil
	}

	_, exists := tables[tableName]
	return exists, nil
}

// checkFieldExists checks if a field exists on a table
func (s *SurrealDBStorage) checkFieldExists(ctx context.Context, tableName, fieldName string) (bool, error) {
	query := fmt.Sprintf("INFO FOR TABLE %s;", tableName)
	result, err := surrealdb.Query[[]map[string]interface{}](s.db, query, nil)
	if err != nil {
		return false, err
	}

	if result == nil || len(*result) == 0 {
		return false, nil
	}

	qr := (*result)[0]
	if qr.Status != "OK" {
		return false, nil
	}

	resMap := qr.Result
	if len(resMap) == 0 {
		return false, nil
	}

	// Extract the result data from the first element
	resultData := resMap[0]

	fieldsRaw, ok := resultData["fields"]
	if !ok || fieldsRaw == nil {
		return false, nil
	}

	fields, ok := fieldsRaw.(map[string]interface{})
	if !ok {
		return false, nil
	}

	_, exists := fields[fieldName]
	return exists, nil
}

// checkIndexExists checks if an index exists on a table
func (s *SurrealDBStorage) checkIndexExists(ctx context.Context, tableName, indexName string) (bool, error) {
	query := fmt.Sprintf("INFO FOR TABLE %s;", tableName)
	result, err := surrealdb.Query[[]map[string]interface{}](s.db, query, nil)
	if err != nil {
		return false, err
	}

	if result == nil || len(*result) == 0 {
		return false, nil
	}

	qr := (*result)[0]
	if qr.Status != "OK" {
		return false, nil
	}

	resMap := qr.Result
	if len(resMap) == 0 {
		return false, nil
	}

	// Extract the result data from the first element
	resultData := resMap[0]

	indexesRaw, ok := resultData["indexes"]
	if !ok || indexesRaw == nil {
		return false, nil
	}

	indexes, ok := indexesRaw.(map[string]interface{})
	if !ok {
		return false, nil
	}

	_, exists := indexes[indexName]
	return exists, nil
}

// isAlreadyExistsError checks if an error is due to an element already existing
func (s *SurrealDBStorage) isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "already exists") ||
		strings.Contains(errStr, "already defined") ||
		strings.Contains(errStr, "duplicate")
}

// extractTableName extracts table name from DEFINE TABLE statement
func (s *SurrealDBStorage) extractTableName(statement string) string {
	// Example: "DEFINE TABLE kv_memories SCHEMAFULL;"
	parts := strings.Fields(statement)
	if len(parts) >= 3 && parts[0] == "DEFINE" && parts[1] == "TABLE" {
		return parts[2]
	}
	return ""
}

// extractFieldName extracts field name from DEFINE FIELD statement
func (s *SurrealDBStorage) extractFieldName(statement string) string {
	// Example: "DEFINE FIELD user_id ON kv_memories TYPE string;"
	parts := strings.Fields(statement)
	if len(parts) >= 3 && parts[0] == "DEFINE" && parts[1] == "FIELD" {
		return parts[2]
	}
	return ""
}

// extractIndexName extracts index name from DEFINE INDEX statement
func (s *SurrealDBStorage) extractIndexName(statement string) string {
	// Example: "DEFINE INDEX idx_kv_user_key ON kv_memories FIELDS user_id, key UNIQUE;"
	parts := strings.Fields(statement)
	if len(parts) >= 3 && parts[0] == "DEFINE" && parts[1] == "INDEX" {
		return parts[2]
	}
	return ""
}
