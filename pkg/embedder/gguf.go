package embedder

import (
	"context"
	"fmt"
	"sync"

	"github.com/madeindigio/remembrances-mcp/internal/llama"
)

// GGUFEmbedder implements Embedder using local GGUF models via llama.cpp.
type GGUFEmbedder struct {
	model     *llama.Model
	modelPath string
	dimension int
	threads   int
	gpuLayers int
	mu        sync.Mutex // Protects access to the model/context
}

// GGUFConfig holds configuration for the GGUF embedder.
type GGUFConfig struct {
	ModelPath string // Path to the GGUF model file
	Threads   int    // Threads to use (<=0 uses a reasonable default)
	GPULayers int    // GPU layers to offload (0 = CPU only)
}

// NewGGUFEmbedder creates a new GGUFEmbedder.
func NewGGUFEmbedder(modelPath string, threads, gpuLayers int) (*GGUFEmbedder, error) {
	if modelPath == "" {
		return nil, fmt.Errorf("model path is required")
	}

	if threads <= 0 {
		threads = 8
	}

	model, err := llama.LoadModel(context.Background(), modelPath, llama.Options{
		Threads:     threads,
		ThreadsBatch: threads,
		GPULayers:   gpuLayers,
		ContextSize: 2048,
		BatchSize:   2048,
		UBatchSize:  2048,
		Pooling:     llama.PoolingMean,
		Attention:   llama.AttentionNonCausal,
		Normalize:   2,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load GGUF model from %s: %w", modelPath, err)
	}

	return &GGUFEmbedder{
		model:     model,
		modelPath: modelPath,
		dimension: 0, // Detected on first call
		threads:   threads,
		gpuLayers: gpuLayers,
	}, nil
}

// NewGGUFEmbedderFromConfig creates a GGUF embedder from a config struct.
func NewGGUFEmbedderFromConfig(cfg GGUFConfig) (*GGUFEmbedder, error) {
	return NewGGUFEmbedder(cfg.ModelPath, cfg.Threads, cfg.GPULayers)
}

// EmbedDocuments generates embeddings for a batch of texts.
func (g *GGUFEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	// Limit batch size to prevent memory exhaustion
	const maxBatchSize = 10
	if len(texts) > maxBatchSize {
		return nil, fmt.Errorf("batch size %d exceeds maximum allowed %d", len(texts), maxBatchSize)
	}

	result := make([][]float32, len(texts))

	// Process sequentially; llama.cpp handles parallelism internally.
	for i, text := range texts {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if text == "" {
			return nil, fmt.Errorf("text at index %d is empty", i)
		}

		// Limit text length to prevent memory issues
		const maxTextLength = 8000 // ~2000 tokens with typical 4:1 char/token ratio
		if len(text) > maxTextLength {
			text = text[:maxTextLength]
		}

		embedding, err := g.embedSingle(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed document at index %d: %w", i, err)
		}

		result[i] = embedding
	}

	return result, nil
}

// EmbedQuery generates an embedding for a single text query.
func (g *GGUFEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// Limit text length to prevent memory issues
	const maxTextLength = 8000 // ~2000 tokens with typical 4:1 char/token ratio
	if len(text) > maxTextLength {
		text = text[:maxTextLength]
	}

	return g.embedSingle(ctx, text)
}

// embedSingle generates an embedding for a single text.
// A mutex is used to ensure the underlying llama context is accessed safely.
func (g *GGUFEmbedder) embedSingle(ctx context.Context, text string) ([]float32, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Verificar contexto
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	embeddings, err := g.model.Embed(ctx, text, g.threads)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Detectar dimensión en la primera llamada
	if g.dimension == 0 && len(embeddings) > 0 {
		g.dimension = len(embeddings)
	}

	return embeddings, nil
}

// Dimension returns the embedding vector dimension.
func (g *GGUFEmbedder) Dimension() int {
	// If not detected yet, try to infer it.
	if g.dimension == 0 {
		// Generate a test embedding to detect the dimension.
		testEmbed, err := g.embedSingle(context.Background(), "test")
		if err == nil && len(testEmbed) > 0 {
			g.dimension = len(testEmbed)
		} else {
			// Common fallback for many embedding models.
			return 768
		}
	}
	return g.dimension
}

// Close releases model resources.
func (g *GGUFEmbedder) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.model != nil {
		_ = g.model.Close()
		g.model = nil
	}
	return nil
}

// ModelPath returns the model file path.
func (g *GGUFEmbedder) ModelPath() string {
	return g.modelPath
}

// Threads returns the configured number of threads.
func (g *GGUFEmbedder) Threads() int {
	return g.threads
}

// GPULayers returns the configured number of GPU layers.
func (g *GGUFEmbedder) GPULayers() int {
	return g.gpuLayers
}
