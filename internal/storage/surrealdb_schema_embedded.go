package storage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/madeindigio/remembrances-mcp/v2/internal/storage/migrations"
)

// applyMigrationEmbedded applies a migration in embedded mode.
//
// It executes exactly the statements the remote path executes. Before RM-7 this
// file carried a parallel, hand-maintained schema that disagreed with the
// migrations package on field names, field presence, index names and embedding
// typing; two ACL schemas that disagree is the exact shape of a leak, so the
// embedded path is now a thin wrapper over the single statement source.
func (s *SurrealDBStorage) applyMigrationEmbedded(ctx context.Context, version int) error {
	slog.Info("Applying embedded migration", "version", version)

	statements := s.getEmbeddedMigrationStatements(version)
	if statements == nil {
		return fmt.Errorf("unknown migration version: %d", version)
	}

	for _, stmt := range statements {
		_, err := s.query(ctx, stmt, nil)
		if err != nil {
			// Every statement carries IF NOT EXISTS or OVERWRITE, so this is
			// belt and braces for engines that still report a redefinition.
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

// getEmbeddedMigrationStatements returns the SurrealQL for a migration version.
// It is a wrapper over the migrations package: one dialect, one statement list,
// exercised by the same tests against embedded 3.x and remote 3.2.
func (s *SurrealDBStorage) getEmbeddedMigrationStatements(version int) []string {
	return migrations.StatementStrings(version, migrations.EmbeddingDimension())
}
