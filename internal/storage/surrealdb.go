package storage

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/surrealdb/surrealdb.go"
)

// SurrealDBStorage implements the Storage interface using SurrealDB
type SurrealDBStorage struct {
	db     *surrealdb.DB
	config *ConnectionConfig
}

// SchemaElement represents a schema element for migrations
type SchemaElement struct {
	Type      string // "table", "field", "index"
	Statement string // The SurrealQL statement to execute
	OnTable   string // For fields and indexes, the table they belong to
}

// NewSurrealDBStorage creates a new SurrealDB storage instance
func NewSurrealDBStorage(config *ConnectionConfig) *SurrealDBStorage {
	if config.Namespace == "" {
		config.Namespace = "test"
	}
	if config.Database == "" {
		config.Database = "test"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &SurrealDBStorage{
		config: config,
	}
}

// NewSurrealDBStorageFromEnv creates a SurrealDB storage instance from environment variables
func NewSurrealDBStorageFromEnv(dbPath string) *SurrealDBStorage {
	namespace := os.Getenv("SURREALDB_NAMESPACE")
	if namespace == "" {
		namespace = "test"
	}

	database := os.Getenv("SURREALDB_DATABASE")
	if database == "" {
		database = "test"
	}

	config := &ConnectionConfig{
		URL:       os.Getenv("SURREALDB_URL"),
		Username:  os.Getenv("SURREALDB_USER"),
		Password:  os.Getenv("SURREALDB_PASS"),
		DBPath:    dbPath,
		Namespace: namespace,
		Database:  database,
		Timeout:   30 * time.Second,
	}

	return NewSurrealDBStorage(config)
}

// Connect establishes connection to SurrealDB (embedded or remote)
func (s *SurrealDBStorage) Connect(ctx context.Context) error {
	var err error

	// Determine connection type: remote vs embedded
	if s.config.URL != "" {
		// Connect to remote SurrealDB instance
		log.Printf("Connecting to remote SurrealDB at %s", s.config.URL)
		s.db, err = surrealdb.New(s.config.URL)
		if err != nil {
			return fmt.Errorf("failed to connect to remote SurrealDB: %w", err)
		}

		// Authenticate if credentials provided
		if s.config.Username != "" && s.config.Password != "" {
			_, err = s.db.SignIn(map[string]interface{}{
				"user": s.config.Username,
				"pass": s.config.Password,
			})
			if err != nil {
				return fmt.Errorf("failed to authenticate with SurrealDB: %w", err)
			}
		}
	} else {
		// Use embedded SurrealDB
		dbURL := fmt.Sprintf("rocksdb://%s", s.config.DBPath)
		log.Printf("Connecting to embedded SurrealDB at %s", dbURL)
		s.db, err = surrealdb.New(dbURL)
		if err != nil {
			return fmt.Errorf("failed to connect to embedded SurrealDB: %w", err)
		}
	}

	// Use the configured namespace and database
	err = s.db.Use(s.config.Namespace, s.config.Database)
	if err != nil {
		return fmt.Errorf("failed to use namespace/database: %w", err)
	}

	return nil
}

// Close closes the database connection
func (s *SurrealDBStorage) Close() error {
	if s.db != nil {
		s.db.Close()
	}
	return nil
}

// Ping checks if the database connection is alive
func (s *SurrealDBStorage) Ping(ctx context.Context) error {
	if s.db == nil {
		return fmt.Errorf("database connection not established")
	}

	// Simple query to test connection
	_, err := surrealdb.Query[map[string]interface{}](s.db, "SELECT 1", nil)
	return err
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
	targetVersion := 1 // Current target version
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
		_, err := surrealdb.Query[map[string]interface{}](s.db, `
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

		{Type: "field", Statement: `DEFINE FIELD in ON mentioned_in TYPE record;`, OnTable: "mentioned_in"},
		{Type: "field", Statement: `DEFINE FIELD out ON mentioned_in TYPE record;`, OnTable: "mentioned_in"},
		{Type: "field", Statement: `DEFINE FIELD timestamp ON mentioned_in TYPE datetime VALUE time::now();`, OnTable: "mentioned_in"},
		{Type: "field", Statement: `DEFINE FIELD properties ON mentioned_in TYPE object;`, OnTable: "mentioned_in"},

		{Type: "field", Statement: `DEFINE FIELD in ON related_to TYPE record;`, OnTable: "related_to"},
		{Type: "field", Statement: `DEFINE FIELD out ON related_to TYPE record;`, OnTable: "related_to"},
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
			_, err := surrealdb.Query[map[string]interface{}](s.db, element.Statement, nil)
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
	// The INFO FOR DB; command returns a single result as a map. Use map-based unmarshalling.
	result, err := surrealdb.Query[map[string]interface{}](s.db, `INFO FOR DB;`, nil)
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
	// Expect a "tables" key in the returned map
	tablesRaw, ok := resMap["tables"]
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
	result, err := surrealdb.Query[map[string]interface{}](s.db, query, nil)
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
	fieldsRaw, ok := resMap["fields"]
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
	result, err := surrealdb.Query[map[string]interface{}](s.db, query, nil)
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
	indexesRaw, ok := resMap["indexes"]
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

// SaveFact saves a key-value fact for a user
func (s *SurrealDBStorage) SaveFact(ctx context.Context, userID, key string, value interface{}) error {
	data := map[string]interface{}{
		"user_id": userID,
		"key":     key,
		"value":   value,
	}

	recordID := fmt.Sprintf("kv_memories:['%s:%s']", userID, key)
	_, err := surrealdb.Create[map[string]interface{}](s.db, recordID, data)
	if err != nil {
		return fmt.Errorf("failed to save fact: %w", err)
	}

	return nil
}

// GetFact retrieves a key-value fact for a user
func (s *SurrealDBStorage) GetFact(ctx context.Context, userID, key string) (interface{}, error) {
	recordID := fmt.Sprintf("kv_memories:['%s:%s']", userID, key)

	result, err := surrealdb.Select[map[string]interface{}](s.db, recordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get fact: %w", err)
	}

	// Handle empty result
	if result == nil {
		return nil, nil
	}

	// Extract value from result
	return (*result)["value"], nil
}

// UpdateFact updates an existing key-value fact
func (s *SurrealDBStorage) UpdateFact(ctx context.Context, userID, key string, value interface{}) error {
	recordID := fmt.Sprintf("kv_memories:['%s:%s']", userID, key)

	data := map[string]interface{}{
		"value": value,
	}

	_, err := surrealdb.Update[map[string]interface{}](s.db, recordID, data)
	if err != nil {
		return fmt.Errorf("failed to update fact: %w", err)
	}

	return nil
}

// DeleteFact deletes a key-value fact
func (s *SurrealDBStorage) DeleteFact(ctx context.Context, userID, key string) error {
	recordID := fmt.Sprintf("kv_memories:['%s:%s']", userID, key)

	_, err := surrealdb.Delete[map[string]interface{}](s.db, recordID)
	if err != nil {
		return fmt.Errorf("failed to delete fact: %w", err)
	}

	return nil
}

// ListFacts retrieves all facts for a user
func (s *SurrealDBStorage) ListFacts(ctx context.Context, userID string) (map[string]interface{}, error) {
	query := "SELECT key, value FROM kv_memories WHERE user_id = $user_id"
	params := map[string]interface{}{
		"user_id": userID,
	}

	result, err := surrealdb.Query[[]map[string]interface{}](s.db, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list facts: %w", err)
	}

	facts := make(map[string]interface{})

	if result != nil && len(*result) > 0 {
		queryResult := (*result)[0]
		if queryResult.Status == "OK" && queryResult.Result != nil {
			resultSlice := queryResult.Result
			for _, item := range resultSlice {
				if key, hasKey := item["key"].(string); hasKey {
					if value, hasValue := item["value"]; hasValue {
						facts[key] = value
					}
				}
			}
		}
	}

	return facts, nil
}

// IndexVector stores a vector embedding with content and metadata
func (s *SurrealDBStorage) IndexVector(ctx context.Context, userID, content string, embedding []float32, metadata map[string]interface{}) (string, error) {
	data := map[string]interface{}{
		"user_id":   userID,
		"content":   content,
		"embedding": embedding,
		"metadata":  metadata,
	}

	result, err := surrealdb.Create[map[string]interface{}](s.db, "vector_memories", data)
	if err != nil {
		return "", fmt.Errorf("failed to index vector: %w", err)
	}

	// Extract ID from result
	if result != nil {
		if id, ok := (*result)["id"].(string); ok {
			return id, nil
		}
	}

	return "", fmt.Errorf("failed to extract ID from result")
}

// SearchSimilar performs vector similarity search
func (s *SurrealDBStorage) SearchSimilar(ctx context.Context, userID string, queryEmbedding []float32, limit int) ([]VectorResult, error) {
	query := `
		SELECT id, content, vector::similarity::cosine(embedding, $query_embedding) AS similarity, metadata, created_at, updated_at
		FROM vector_memories 
		WHERE user_id = $user_id AND embedding <|$limit|> $query_embedding
		ORDER BY similarity DESC
	`

	params := map[string]interface{}{
		"user_id":         userID,
		"query_embedding": queryEmbedding,
		"limit":           limit,
	}

	result, err := surrealdb.Query[[]map[string]interface{}](s.db, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar vectors: %w", err)
	}

	return s.parseVectorResults(result)
}

// UpdateVector updates an existing vector memory
func (s *SurrealDBStorage) UpdateVector(ctx context.Context, id, userID, content string, embedding []float32, metadata map[string]interface{}) error {
	data := map[string]interface{}{
		"content":   content,
		"embedding": embedding,
		"metadata":  metadata,
	}

	_, err := surrealdb.Update[map[string]interface{}](s.db, id, data)
	if err != nil {
		return fmt.Errorf("failed to update vector: %w", err)
	}

	return nil
}

// DeleteVector deletes a vector memory
func (s *SurrealDBStorage) DeleteVector(ctx context.Context, id, userID string) error {
	_, err := surrealdb.Delete[map[string]interface{}](s.db, id)
	if err != nil {
		return fmt.Errorf("failed to delete vector: %w", err)
	}

	return nil
}

// CreateEntity creates a new entity in the graph
func (s *SurrealDBStorage) CreateEntity(ctx context.Context, entityType, name string, properties map[string]interface{}) (string, error) {
	data := map[string]interface{}{
		"type":       entityType,
		"name":       name,
		"properties": properties,
	}

	result, err := surrealdb.Create[map[string]interface{}](s.db, "entities", data)
	if err != nil {
		return "", fmt.Errorf("failed to create entity: %w", err)
	}

	// Extract ID from result
	if result != nil {
		if id, ok := (*result)["id"].(string); ok {
			return id, nil
		}
	}

	return "", fmt.Errorf("failed to extract ID from result")
}

// CreateRelationship creates a relationship between two entities
func (s *SurrealDBStorage) CreateRelationship(ctx context.Context, fromEntity, toEntity, relationshipType string, properties map[string]interface{}) error {
	query := fmt.Sprintf("RELATE $from->%s->$to CONTENT $content", relationshipType)

	content := properties
	if content == nil {
		content = make(map[string]interface{})
	}
	content["timestamp"] = time.Now()

	params := map[string]interface{}{
		"from":    fromEntity,
		"to":      toEntity,
		"content": content,
	}

	_, err := surrealdb.Query[map[string]interface{}](s.db, query, params)
	if err != nil {
		return fmt.Errorf("failed to create relationship: %w", err)
	}

	return nil
}

// TraverseGraph traverses the graph starting from an entity
func (s *SurrealDBStorage) TraverseGraph(ctx context.Context, startEntity, relationshipType string, depth int) ([]GraphResult, error) {
	var query string

	if relationshipType == "" {
		// Traverse all relationship types
		query = fmt.Sprintf("SELECT ->* FROM %s", startEntity)
	} else {
		// Traverse specific relationship type
		query = fmt.Sprintf("SELECT ->%s->entities FROM %s", relationshipType, startEntity)
	}

	params := map[string]interface{}{
		"start_entity": startEntity,
		"depth":        depth,
	}

	result, err := surrealdb.Query[[]map[string]interface{}](s.db, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to traverse graph: %w", err)
	}

	return s.parseGraphResults(result)
}

// GetEntity retrieves an entity by ID
func (s *SurrealDBStorage) GetEntity(ctx context.Context, entityID string) (*Entity, error) {
	result, err := surrealdb.Select[map[string]interface{}](s.db, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	if result == nil {
		return nil, nil
	}

	resultMap := *result
	entity := &Entity{
		ID:         getString(resultMap, "id"),
		Type:       getString(resultMap, "type"),
		Name:       getString(resultMap, "name"),
		Properties: getMap(resultMap, "properties"),
		CreatedAt:  getTime(resultMap, "created_at"),
		UpdatedAt:  getTime(resultMap, "updated_at"),
	}
	return entity, nil
}

// DeleteEntity deletes an entity and its relationships
func (s *SurrealDBStorage) DeleteEntity(ctx context.Context, entityID string) error {
	_, err := surrealdb.Delete[map[string]interface{}](s.db, entityID)
	if err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	return nil
}

// SaveDocument saves a knowledge base document
func (s *SurrealDBStorage) SaveDocument(ctx context.Context, filePath, content string, embedding []float32, metadata map[string]interface{}) error {
	data := map[string]interface{}{
		"file_path": filePath,
		"content":   content,
		"embedding": embedding,
		"metadata":  metadata,
	}

	recordID := fmt.Sprintf("knowledge_base:['%s']", strings.ReplaceAll(filePath, "/", "_"))
	_, err := surrealdb.Create[map[string]interface{}](s.db, recordID, data)
	if err != nil {
		return fmt.Errorf("failed to save document: %w", err)
	}

	return nil
}

// SearchDocuments performs similarity search on knowledge base documents
func (s *SurrealDBStorage) SearchDocuments(ctx context.Context, queryEmbedding []float32, limit int) ([]DocumentResult, error) {
	query := `
		SELECT id, file_path, content, embedding, metadata, created_at, updated_at,
		       vector::similarity::cosine(embedding, $query_embedding) AS similarity
		FROM knowledge_base 
		WHERE embedding <|$limit|> $query_embedding
		ORDER BY similarity DESC
	`

	params := map[string]interface{}{
		"query_embedding": queryEmbedding,
		"limit":           limit,
	}

	result, err := surrealdb.Query[[]map[string]interface{}](s.db, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}

	return s.parseDocumentResults(result)
}

// DeleteDocument deletes a knowledge base document
func (s *SurrealDBStorage) DeleteDocument(ctx context.Context, filePath string) error {
	recordID := fmt.Sprintf("knowledge_base:['%s']", strings.ReplaceAll(filePath, "/", "_"))

	_, err := surrealdb.Delete[map[string]interface{}](s.db, recordID)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

// GetDocument retrieves a knowledge base document by file path
func (s *SurrealDBStorage) GetDocument(ctx context.Context, filePath string) (*Document, error) {
	recordID := fmt.Sprintf("knowledge_base:['%s']", strings.ReplaceAll(filePath, "/", "_"))

	result, err := surrealdb.Select[map[string]interface{}](s.db, recordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	if result == nil {
		return nil, nil
	}

	resultMap := *result
	// Convert embedding from interface{} to []float32
	var embedding []float32
	if embeddingSlice, ok := resultMap["embedding"].([]interface{}); ok {
		embedding = make([]float32, len(embeddingSlice))
		for i, v := range embeddingSlice {
			if f, ok := v.(float64); ok {
				embedding[i] = float32(f)
			} else if f, ok := v.(float32); ok {
				embedding[i] = f
			}
		}
	}

	document := &Document{
		ID:        getString(resultMap, "id"),
		FilePath:  getString(resultMap, "file_path"),
		Content:   getString(resultMap, "content"),
		Embedding: embedding,
		Metadata:  getMap(resultMap, "metadata"),
		CreatedAt: getTime(resultMap, "created_at"),
		UpdatedAt: getTime(resultMap, "updated_at"),
	}
	return document, nil
}

// HybridSearch performs a combined search across vector, graph, and key-value stores
func (s *SurrealDBStorage) HybridSearch(ctx context.Context, userID string, queryEmbedding []float32, entities []string, limit int) (*HybridSearchResult, error) {
	start := time.Now()

	// Perform vector search
	vectorResults, err := s.SearchSimilar(ctx, userID, queryEmbedding, limit)
	if err != nil {
		return nil, fmt.Errorf("failed vector search: %w", err)
	}

	// Perform graph traversal for each entity
	var graphResults []GraphResult
	for _, entity := range entities {
		results, err := s.TraverseGraph(ctx, entity, "", 2) // depth 2
		if err != nil {
			log.Printf("Warning: failed to traverse graph for entity %s: %v", entity, err)
			continue
		}
		graphResults = append(graphResults, results...)
	}

	// Get relevant facts
	facts, err := s.ListFacts(ctx, userID)
	if err != nil {
		log.Printf("Warning: failed to get facts: %v", err)
		facts = make(map[string]interface{})
	}

	result := &HybridSearchResult{
		VectorResults: vectorResults,
		GraphResults:  graphResults,
		Facts:         facts,
		TotalResults:  len(vectorResults) + len(graphResults) + len(facts),
		QueryTime:     time.Since(start),
	}

	return result, nil
}

// GetStats returns statistics about stored memories
func (s *SurrealDBStorage) GetStats(ctx context.Context, userID string) (*MemoryStats, error) {
	stats := &MemoryStats{}

	// Count key-value memories
	kvQuery := "SELECT count() FROM kv_memories WHERE user_id = $user_id GROUP ALL"
	kvResult, err := surrealdb.Query[[]map[string]interface{}](s.db, kvQuery, map[string]interface{}{"user_id": userID})
	if err == nil {
		if count := s.extractCount(kvResult); count >= 0 {
			stats.KeyValueCount = count
		}
	}

	// Count vector memories
	vectorQuery := "SELECT count() FROM vector_memories WHERE user_id = $user_id GROUP ALL"
	vectorResult, err := surrealdb.Query[[]map[string]interface{}](s.db, vectorQuery, map[string]interface{}{"user_id": userID})
	if err == nil {
		if count := s.extractCount(vectorResult); count >= 0 {
			stats.VectorCount = count
		}
	}

	// Count entities
	entityQuery := "SELECT count() FROM entities GROUP ALL"
	entityResult, err := surrealdb.Query[[]map[string]interface{}](s.db, entityQuery, nil)
	if err == nil {
		if count := s.extractCount(entityResult); count >= 0 {
			stats.EntityCount = count
		}
	}

	// Count relationships
	relQuery := "SELECT count() FROM wrote GROUP ALL"
	relResult, err := surrealdb.Query[[]map[string]interface{}](s.db, relQuery, nil)
	if err == nil {
		if count := s.extractCount(relResult); count >= 0 {
			stats.RelationshipCount = count
		}
	}

	// Count knowledge base documents
	kbQuery := "SELECT count() FROM knowledge_base GROUP ALL"
	kbResult, err := surrealdb.Query[[]map[string]interface{}](s.db, kbQuery, nil)
	if err == nil {
		if count := s.extractCount(kbResult); count >= 0 {
			stats.DocumentCount = count
		}
	}

	return stats, nil
}

// Helper function to extract count from query result
func (s *SurrealDBStorage) extractCount(result *[]surrealdb.QueryResult[[]map[string]interface{}]) int {
	if result != nil && len(*result) > 0 {
		queryResult := (*result)[0]
		if queryResult.Status == "OK" && queryResult.Result != nil {
			resultSlice := queryResult.Result
			if len(resultSlice) > 0 {
				if count, ok := resultSlice[0]["count"].(float64); ok {
					return int(count)
				}
			}
		}
	}
	return -1
}

// parseVectorResults converts raw query results to VectorResult structs
func (s *SurrealDBStorage) parseVectorResults(result *[]surrealdb.QueryResult[[]map[string]interface{}]) ([]VectorResult, error) {
	var results []VectorResult

	if result != nil && len(*result) > 0 {
		queryResult := (*result)[0]
		if queryResult.Status == "OK" && queryResult.Result != nil {
			resultSlice := queryResult.Result
			for _, itemMap := range resultSlice {
				vectorResult := VectorResult{
					ID:         getString(itemMap, "id"),
					Content:    getString(itemMap, "content"),
					Similarity: getFloat64(itemMap, "similarity"),
					Metadata:   getMap(itemMap, "metadata"),
					CreatedAt:  getTime(itemMap, "created_at"),
					UpdatedAt:  getTime(itemMap, "updated_at"),
				}
				results = append(results, vectorResult)
			}
		}
	}

	return results, nil
}

// parseGraphResults converts raw query results to GraphResult structs
func (s *SurrealDBStorage) parseGraphResults(result *[]surrealdb.QueryResult[[]map[string]interface{}]) ([]GraphResult, error) {
	var results []GraphResult

	if result != nil && len(*result) > 0 {
		queryResult := (*result)[0]
		if queryResult.Status == "OK" && queryResult.Result != nil {
			resultSlice := queryResult.Result
			for _, itemMap := range resultSlice {
				// Parse entity information
				entity := &Entity{
					ID:         getString(itemMap, "id"),
					Type:       getString(itemMap, "type"),
					Name:       getString(itemMap, "name"),
					Properties: getMap(itemMap, "properties"),
					CreatedAt:  getTime(itemMap, "created_at"),
					UpdatedAt:  getTime(itemMap, "updated_at"),
				}

				graphResult := GraphResult{
					Entity: entity,
					Path:   []string{entity.ID}, // Simplified path for now
					Depth:  1,                   // Simplified depth for now
				}
				results = append(results, graphResult)
			}
		}
	}

	return results, nil
}

// parseDocumentResults converts raw query results to DocumentResult structs
func (s *SurrealDBStorage) parseDocumentResults(result *[]surrealdb.QueryResult[[]map[string]interface{}]) ([]DocumentResult, error) {
	var results []DocumentResult

	if result != nil && len(*result) > 0 {
		queryResult := (*result)[0]
		if queryResult.Status == "OK" && queryResult.Result != nil {
			resultSlice := queryResult.Result
			for _, itemMap := range resultSlice {
				// Convert embedding from interface{} to []float32
				var embedding []float32
				if embeddingSlice, ok := itemMap["embedding"].([]interface{}); ok {
					embedding = make([]float32, len(embeddingSlice))
					for i, v := range embeddingSlice {
						if f, ok := v.(float64); ok {
							embedding[i] = float32(f)
						} else if f, ok := v.(float32); ok {
							embedding[i] = f
						}
					}
				}

				document := &Document{
					ID:        getString(itemMap, "id"),
					FilePath:  getString(itemMap, "file_path"),
					Content:   getString(itemMap, "content"),
					Embedding: embedding,
					Metadata:  getMap(itemMap, "metadata"),
					CreatedAt: getTime(itemMap, "created_at"),
					UpdatedAt: getTime(itemMap, "updated_at"),
				}

				similarity := getFloat64(itemMap, "similarity")

				documentResult := DocumentResult{
					Document:   document,
					Similarity: similarity,
					Score:      similarity, // Use similarity as score for now
				}
				results = append(results, documentResult)
			}
		}
	}

	return results, nil
}

// Helper functions for type conversion
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getFloat64(m map[string]interface{}, key string) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	if val, ok := m[key].(float32); ok {
		return float64(val)
	}
	return 0
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if val, ok := m[key].(map[string]interface{}); ok {
		return val
	}
	return make(map[string]interface{})
}

func getTime(m map[string]interface{}, key string) time.Time {
	if val, ok := m[key].(string); ok {
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			return t
		}
	}
	return time.Time{}
}
