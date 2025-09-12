package storage

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/madeindigio/remembrances-mcp/internal/storage/migrations"
	"github.com/surrealdb/surrealdb.go"
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

// applyMigration applies a specific migration version using the new migration structure
func (s *SurrealDBStorage) applyMigration(ctx context.Context, version int) error {
	var migration migrations.Migration

	switch version {
	case 1:
		migration = migrations.NewMigrationV1(s.db)
	case 2:
		migration = migrations.NewMigrationV2(s.db)
	case 3:
		migration = migrations.NewMigrationV3(s.db)
	default:
		return fmt.Errorf("unknown migration version: %d", version)
	}

	return migration.Apply(ctx, s.db)
}

// checkTableExists checks if a table exists
func (s *SurrealDBStorage) checkTableExists(ctx context.Context, tableName string) (bool, error) {
	migrationBase := migrations.NewMigrationBase(s.db)
	return migrationBase.CheckTableExists(ctx, tableName)
}

// isAlreadyExistsError checks if an error is due to an element already existing
func (s *SurrealDBStorage) isAlreadyExistsError(err error) bool {
	migrationBase := migrations.NewMigrationBase(s.db)
	return migrationBase.IsAlreadyExistsError(err)
}
