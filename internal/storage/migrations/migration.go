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

// ApplyElements executes a list of schema elements in order.
//
// SurrealDB 3.x makes the old INFO FOR DB existence probe redundant: every
// statement this package emits carries IF NOT EXISTS or OVERWRITE, so applying
// it twice is a no-op rather than an error. Dropping the probe also removes the
// class of bug where a mis-classified element (the v11 analyzer registered as an
// index) made the probe look for something that could never be found.
func (m *MigrationBase) ApplyElements(ctx context.Context, elements []SchemaElement) error {
	for i, element := range elements {
		slog.Debug("Applying schema element", "type", element.Type, "statement", element.Statement)
		_, err := surrealdb.Query[[]map[string]interface{}](ctx, m.db, element.Statement, nil)
		if err != nil {
			if m.IsAlreadyExistsError(err) {
				slog.Debug("Schema element already exists, continuing", "type", element.Type)
				continue
			}
			return fmt.Errorf("failed to execute migration statement %d '%s': %w", i+1, element.Statement, err)
		}
	}
	return nil
}
