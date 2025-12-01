// Package storage provides code indexing storage operations for SurrealDB.
// This file contains type definitions for code indexing.
package storage

import (
	"time"

	"github.com/madeindigio/remembrances-mcp/pkg/treesitter"
)

// CodeProject represents a stored code project
type CodeProject struct {
	ID             string                      `json:"id"`
	ProjectID      string                      `json:"project_id"`
	Name           string                      `json:"name"`
	RootPath       string                      `json:"root_path"`
	LanguageStats  map[treesitter.Language]int `json:"language_stats"`
	LastIndexedAt  *time.Time                  `json:"last_indexed_at"`
	IndexingStatus treesitter.IndexingStatus   `json:"indexing_status"`
	WatcherEnabled bool                        `json:"watcher_enabled"`
	CreatedAt      time.Time                   `json:"created_at"`
	UpdatedAt      time.Time                   `json:"updated_at"`
}

// CodeFile represents a stored code file
type CodeFile struct {
	ID           string              `json:"id"`
	ProjectID    string              `json:"project_id"`
	FilePath     string              `json:"file_path"`
	Language     treesitter.Language `json:"language"`
	FileHash     string              `json:"file_hash"`
	SymbolsCount int                 `json:"symbols_count"`
	IndexedAt    time.Time           `json:"indexed_at"`
}

// CodeSymbol represents a stored code symbol
type CodeSymbol struct {
	ID         string                 `json:"id"`
	ProjectID  string                 `json:"project_id"`
	FilePath   string                 `json:"file_path"`
	Language   treesitter.Language    `json:"language"`
	SymbolType treesitter.SymbolType  `json:"symbol_type"`
	Name       string                 `json:"name"`
	NamePath   string                 `json:"name_path"`
	StartLine  int                    `json:"start_line"`
	EndLine    int                    `json:"end_line"`
	StartByte  int                    `json:"start_byte"`
	EndByte    int                    `json:"end_byte"`
	SourceCode *string                `json:"source_code,omitempty"`
	Signature  *string                `json:"signature,omitempty"`
	DocString  *string                `json:"doc_string,omitempty"`
	Embedding  []float32              `json:"embedding,omitempty"`
	ParentID   *string                `json:"parent_id,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// CodeSymbolSearchResult represents a symbol search result with similarity
type CodeSymbolSearchResult struct {
	Symbol     *CodeSymbol `json:"symbol"`
	Similarity float64     `json:"similarity"`
}

// CodeIndexingJob represents a stored indexing job
type CodeIndexingJob struct {
	ID           string                    `json:"id"`
	ProjectID    string                    `json:"project_id"`
	ProjectPath  string                    `json:"project_path"`
	Status       treesitter.IndexingStatus `json:"status"`
	Progress     float64                   `json:"progress"`
	FilesTotal   int                       `json:"files_total"`
	FilesIndexed int                       `json:"files_indexed"`
	StartedAt    time.Time                 `json:"started_at"`
	CompletedAt  *time.Time                `json:"completed_at,omitempty"`
	Error        *string                   `json:"error,omitempty"`
}

// CodeChunk represents a chunk of a large symbol for semantic search
type CodeChunk struct {
	ID          string    `json:"id"`
	SymbolID    string    `json:"symbol_id"`
	ProjectID   string    `json:"project_id"`
	FilePath    string    `json:"file_path"`
	ChunkIndex  int       `json:"chunk_index"`
	ChunkCount  int       `json:"chunk_count"`
	Content     string    `json:"content"`
	StartOffset int       `json:"start_offset"`
	EndOffset   int       `json:"end_offset"`
	Embedding   []float32 `json:"embedding,omitempty"`
	SymbolName  string    `json:"symbol_name"`
	SymbolType  string    `json:"symbol_type"`
	Language    string    `json:"language"`
	CreatedAt   time.Time `json:"created_at"`
}

// CodeChunkSearchResult represents a chunk search result with similarity
type CodeChunkSearchResult struct {
	Chunk      *CodeChunk `json:"chunk"`
	Similarity float64    `json:"similarity"`
}
