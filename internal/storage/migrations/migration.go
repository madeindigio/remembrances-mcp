package migrations

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/surrealdb/surrealdb.go"
)

// Migration represents a database migration
type Migration interface {
	Apply(ctx context.Context, db *surrealdb.DB) error
	Version() int
	Description() string
}

// SchemaElement represents a schema element for migrations
type SchemaElement struct {
	Type      string // "table", "field", "index"
	Statement string // The SurrealQL statement to execute
	OnTable   string // For fields and indexes, the table they belong to
}

// MigrationBase provides common functionality for all migrations
type MigrationBase struct {
	db *surrealdb.DB
}

// NewMigrationBase creates a new migration base with the given database connection
func NewMigrationBase(db *surrealdb.DB) *MigrationBase {
	return &MigrationBase{db: db}
}

// checkSchemaElementExists checks if a schema element (table, field, index) already exists
func (m *MigrationBase) checkSchemaElementExists(ctx context.Context, element SchemaElement) (bool, error) {
	switch element.Type {
	case "table":
		return m.CheckTableExists(ctx, m.extractTableName(element.Statement))
	case "field":
		tableName := element.OnTable
		fieldName := m.extractFieldName(element.Statement)
		return m.checkFieldExists(ctx, tableName, fieldName)
	case "index":
		tableName := element.OnTable
		indexName := m.extractIndexName(element.Statement)
		return m.checkIndexExists(ctx, tableName, indexName)
	default:
		return false, fmt.Errorf("unknown schema element type: %s", element.Type)
	}
}

// CheckTableExists checks if a table exists
func (m *MigrationBase) CheckTableExists(ctx context.Context, tableName string) (bool, error) {
	// The INFO FOR DB; command returns a QueryResult array. Use array-based unmarshalling.
	result, err := surrealdb.Query[[]map[string]interface{}](ctx, m.db, `INFO FOR DB;`, nil)
	if err != nil {
		// If unmarshalling fails, it might be because the DB is empty or returning a different format
		// In this case, assume the table doesn't exist and let the caller create it
		slog.Debug("Failed to query DB info (likely empty database)", "error", err)
		return false, nil
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
func (m *MigrationBase) checkFieldExists(ctx context.Context, tableName, fieldName string) (bool, error) {
	query := fmt.Sprintf("INFO FOR TABLE %s;", tableName)
	result, err := surrealdb.Query[[]map[string]interface{}](ctx, m.db, query, nil)
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
func (m *MigrationBase) checkIndexExists(ctx context.Context, tableName, indexName string) (bool, error) {
	query := fmt.Sprintf("INFO FOR TABLE %s;", tableName)
	result, err := surrealdb.Query[[]map[string]interface{}](ctx, m.db, query, nil)
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

// IsAlreadyExistsError checks if an error is due to an element already existing
func (m *MigrationBase) IsAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "already exists") ||
		strings.Contains(errStr, "already defined") ||
		strings.Contains(errStr, "duplicate")
}

// extractTableName extracts table name from DEFINE TABLE statement
func (m *MigrationBase) extractTableName(statement string) string {
	// Example: "DEFINE TABLE kv_memories SCHEMAFULL;"
	parts := strings.Fields(statement)
	if len(parts) >= 3 && parts[0] == "DEFINE" && parts[1] == "TABLE" {
		return parts[2]
	}
	return ""
}

// extractFieldName extracts field name from DEFINE FIELD statement
func (m *MigrationBase) extractFieldName(statement string) string {
	// Example: "DEFINE FIELD user_id ON kv_memories TYPE string;"
	parts := strings.Fields(statement)
	if len(parts) >= 3 && parts[0] == "DEFINE" && parts[1] == "FIELD" {
		return parts[2]
	}
	return ""
}

// extractIndexName extracts index name from DEFINE INDEX statement
func (m *MigrationBase) extractIndexName(statement string) string {
	// Example: "DEFINE INDEX idx_kv_user_key ON kv_memories FIELDS user_id, key UNIQUE;"
	parts := strings.Fields(statement)
	if len(parts) >= 3 && parts[0] == "DEFINE" && parts[1] == "INDEX" {
		return parts[2]
	}
	return ""
}

// ApplyElements applies a list of schema elements with error handling
func (m *MigrationBase) ApplyElements(ctx context.Context, elements []SchemaElement) error {
	// Apply each schema element with error handling
	for i, element := range elements {
		exists, err := m.checkSchemaElementExists(ctx, element)
		if err != nil {
			slog.Warn("Could not check existence of element, attempting to create anyway", "type", element.Type, "error", err)
		}

		if !exists {
			slog.Debug("Creating schema element", "type", element.Type, "statement", element.Statement)
			_, err := surrealdb.Query[[]map[string]interface{}](ctx, m.db, element.Statement, nil)
			if err != nil {
				// Log warning but don't fail for "already exists" type errors
				if m.IsAlreadyExistsError(err) {
					slog.Debug("Schema element already exists, continuing", "type", element.Type)
				} else {
					return fmt.Errorf("failed to execute migration statement %d '%s': %w", i+1, element.Statement, err)
				}
			}
		} else {
			slog.Debug("Skipping existing schema element", "type", element.Type)
		}
	}

	return nil
}
