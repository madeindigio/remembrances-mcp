package embedder

import (
	"context"
	"fmt"
)

// DEPRECATED: This file is deprecated and will be removed in a future version.
// Use search.go (kelindar/search) instead, which supports BERT models in GGUF format.
//
// Migration Guide:
// - Old: NewLlamaEmbedder(path, dim, threads, gpu, ctx)
// - New: NewSearchEmbedderWithDimension(path, dim, gpu)
//
// Reason for deprecation: go-llama.cpp does not support BERT-based embedding models
// (like nomic-embed-text), only LLaMA/Mistral architectures which are not suitable
// for embeddings. kelindar/search provides proper BERT support without cgo.

// LlamaEmbedder is a deprecated embedder that redirects to SearchEmbedder.
// DEPRECATED: Use SearchEmbedder instead.
type LlamaEmbedder struct {
	embedder *SearchEmbedder
}

// NewLlamaEmbedder creates a new LlamaEmbedder instance that internally uses SearchEmbedder.
// DEPRECATED: Use NewSearchEmbedder or NewSearchEmbedderWithDimension instead.
// This function is maintained for backward compatibility but will be removed.
//
// modelPath: path to .gguf model file (must be BERT-based model)
// dimension: embedding dimension (typically 768 for BERT models)
// threads: IGNORED - kelindar/search handles threading automatically
// gpuLayers: number of GPU layers (0 for CPU only)
// context: IGNORED - not needed for embedding models
func NewLlamaEmbedder(modelPath string, dimension int, threads int, gpuLayers int, context int) (*LlamaEmbedder, error) {
	// Redirect to SearchEmbedder with appropriate parameters
	searchEmbedder, err := NewSearchEmbedderWithDimension(modelPath, dimension, gpuLayers)
	if err != nil {
		return nil, fmt.Errorf("llama embedder (redirected to search): %w", err)
	}

	return &LlamaEmbedder{
		embedder: searchEmbedder,
	}, nil
}

// EmbedDocuments creates embeddings for a batch of texts.
func (l *LlamaEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if l.embedder == nil {
		return nil, fmt.Errorf("embedder not initialized")
	}
	return l.embedder.EmbedDocuments(ctx, texts)
}

// EmbedQuery creates an embedding for a single text (a query).
func (l *LlamaEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if l.embedder == nil {
		return nil, fmt.Errorf("embedder not initialized")
	}
	return l.embedder.EmbedQuery(ctx, text)
}

// Dimension returns the dimensionality of the generated vectors.
func (l *LlamaEmbedder) Dimension() int {
	if l.embedder == nil {
		return 0
	}
	return l.embedder.Dimension()
}

// Close releases the resources of the underlying embedder.
func (l *LlamaEmbedder) Close() error {
	if l.embedder == nil {
		return nil
	}
	return l.embedder.Close()
}

// GetModelPath returns the path of the loaded model.
func (l *LlamaEmbedder) GetModelPath() string {
	if l.embedder == nil {
		return ""
	}
	return l.embedder.GetModelPath()
}

// GetThreads returns 0 (not used in kelindar/search).
// DEPRECATED: This method exists for compatibility only.
func (l *LlamaEmbedder) GetThreads() int {
	return 0 // Not applicable
}

// GetGPULayers returns the number of configured GPU layers.
func (l *LlamaEmbedder) GetGPULayers() int {
	if l.embedder == nil {
		return 0
	}
	return l.embedder.GetGPULayers()
}

// GetContext returns 0 (not used in kelindar/search).
// DEPRECATED: This method exists for compatibility only.
func (l *LlamaEmbedder) GetContext() int {
	return 0 // Not applicable
}

// IsInitialized returns true if the embedder is initialized.
func (l *LlamaEmbedder) IsInitialized() bool {
	if l.embedder == nil {
		return false
	}
	return l.embedder.IsInitialized()
}
