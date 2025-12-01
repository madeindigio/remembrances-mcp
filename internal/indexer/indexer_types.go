// Package indexer provides types for the indexing service.
package indexer

import (
	"time"

	"github.com/madeindigio/remembrances-mcp/pkg/treesitter"
)

// IndexerConfig holds configuration for the indexer
type IndexerConfig struct {
	// Number of concurrent file processors
	Concurrency int

	// Batch size for embedding generation
	EmbeddingBatchSize int

	// Whether to store source code in symbols
	StoreSourceCode bool

	// Maximum source code length to store
	MaxSourceCodeLength int

	// Scanner configuration
	Scanner *FileScanner
}

// DefaultIndexerConfig returns sensible defaults
func DefaultIndexerConfig() IndexerConfig {
	return IndexerConfig{
		Concurrency:         4,
		EmbeddingBatchSize:  10,
		StoreSourceCode:     true,
		MaxSourceCodeLength: 10000,
		Scanner:             NewFileScanner(),
	}
}

// IndexingProgress tracks the progress of an indexing operation
type IndexingProgress struct {
	ProjectID    string
	Status       treesitter.IndexingStatus
	FilesTotal   int
	FilesIndexed int
	SymbolsFound int
	CurrentFile  string
	StartedAt    time.Time
	UpdatedAt    time.Time
	Error        *string
}
