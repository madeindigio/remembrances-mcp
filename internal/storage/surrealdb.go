package storage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	embeddedlibs "github.com/madeindigio/remembrances-mcp/internal/embedded"
	embedded "github.com/madeindigio/surrealdb-embedded-golang"
	"github.com/surrealdb/surrealdb.go"
)

// SurrealDBStorage implements the Storage interface using SurrealDB
type SurrealDBStorage struct {
	db          *surrealdb.DB
	embeddedDB  *embedded.DB
	config      *ConnectionConfig
	useEmbedded bool

	embeddedLoader *embeddedlibs.Loader
	embeddedLibs   *embeddedlibs.ExtractResult
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
		URL:             os.Getenv("SURREALDB_URL"),
		Username:        os.Getenv("SURREALDB_USER"),
		Password:        os.Getenv("SURREALDB_PASS"),
		DBPath:          dbPath,
		Namespace:       namespace,
		Database:        database,
		UseEmbeddedLibs: true,
		Timeout:         30 * time.Second,
	}

	return NewSurrealDBStorage(config)
}

// Connect establishes connection to SurrealDB (embedded or remote)
func (s *SurrealDBStorage) Connect(ctx context.Context) error {
	var err error

	if s.config.UseEmbeddedLibs && s.embeddedLoader == nil {
		libs, loader, loadErr := embeddedlibs.ExtractAndLoad(ctx, s.config.EmbeddedLibsDir)
		if loadErr != nil {
			if errors.Is(loadErr, embeddedlibs.ErrPlatformUnsupported) {
				slog.Warn("Embedded libraries not available for this platform; falling back to system lookup", "platform", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
			} else {
				return fmt.Errorf("prepare embedded libraries: %w", loadErr)
			}
		} else {
			s.embeddedLibs = libs
			s.embeddedLoader = loader
			slog.Info("Embedded libraries loaded", "platform", libs.Platform, "dir", libs.Directory)
		}
	}

	// Priority: if DBPath is set, use embedded; otherwise use remote URL
	if s.config.DBPath != "" && s.config.URL == "" {
		// Use embedded SurrealDB with configurable backend (memory, rocksdb, surrealkv)
		slog.Info("Connecting to embedded SurrealDB", "url", s.config.DBPath)
		s.embeddedDB, err = embedded.NewFromURL(s.config.DBPath)
		if err != nil {
			return fmt.Errorf("failed to connect to embedded SurrealDB: %w", err)
		}

		if err = s.embeddedDB.Use(s.config.Namespace, s.config.Database); err != nil {
			return fmt.Errorf("failed to use namespace/database: %w", err)
		}

		s.useEmbedded = true
		slog.Info("Successfully connected to embedded SurrealDB")
	} else if s.config.URL != "" {
		// Use remote SurrealDB
		slog.Info("Connecting to remote SurrealDB", "url", s.config.URL)
		s.db, err = surrealdb.New(s.config.URL)
		if err != nil {
			return fmt.Errorf("failed to connect to remote SurrealDB: %w", err)
		}

		if s.config.Username != "" && s.config.Password != "" {
			_, err = s.db.SignIn(map[string]interface{}{
				"user": s.config.Username,
				"pass": s.config.Password,
			})
			if err != nil {
				return fmt.Errorf("failed to authenticate with SurrealDB: %w", err)
			}
		}

		if err = s.db.Use(s.config.Namespace, s.config.Database); err != nil {
			return fmt.Errorf("failed to use namespace/database: %w", err)
		}

		s.useEmbedded = false
		slog.Info("Successfully connected to remote SurrealDB")
	} else {
		return fmt.Errorf("either DBPath or URL must be configured")
	}

	return nil
}

// Close closes the database connection
func (s *SurrealDBStorage) Close() error {
	var errs []error

	if s.useEmbedded {
		if s.embeddedDB != nil {
			if err := s.embeddedDB.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	} else {
		if s.db != nil {
			if err := s.db.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if s.embeddedLoader != nil {
		if err := s.embeddedLoader.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// Ping checks if the database connection is alive
func (s *SurrealDBStorage) Ping(ctx context.Context) error {
	if s.useEmbedded {
		if s.embeddedDB == nil {
			return fmt.Errorf("database connection not established")
		}
		// Execute a simple query to check connection
		_, err := s.embeddedDB.Query("SELECT 1", nil)
		return err
	} else {
		if s.db == nil {
			return fmt.Errorf("database connection not established")
		}
		_, err := surrealdb.Query[[]map[string]interface{}](s.db, "SELECT 1", nil)
		return err
	}
}
