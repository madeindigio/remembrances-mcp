package embedder

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
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

	// Dynamic limits based on model configuration
	maxTokens     int // Maximum tokens the model can handle (from UBatchSize)
	maxChars      int // Maximum characters (calculated from maxTokens)
	charsPerToken int // Conservative char-to-token ratio (default: 2)

	mu sync.Mutex // Protects access to the model/context
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
		Threads:      threads,
		ThreadsBatch: threads,
		GPULayers:    gpuLayers,
		ContextSize:  384, // Reduced from 512 for safety
		BatchSize:    384, // Reduced from 512 for safety
		UBatchSize:   384, // CRITICAL: Must be >= n_tokens in any single call
		Pooling:      llama.PoolingMean,
		Attention:    llama.AttentionNonCausal,
		Normalize:    2,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load GGUF model from %s: %w", modelPath, err)
	}

	// Get dynamic limits from the model
	ubatchSize := model.UBatchSize()
	if ubatchSize == 0 {
		ubatchSize = 384 // Fallback to safe default (reduced from 512)
	}

	// ULTRA conservative char-to-token ratio for code and special characters
	// Many programming tokens can be 1:1 or worse (e.g., "}", "{", "->")
	// Use 1.5:1 ratio to be extra safe
	charsPerToken := 1

	// Calculate max tokens with LARGE safety margin (leave ~30% margin)
	// This accounts for tokenizer overhead and special tokens
	maxTokens := int(ubatchSize) - int(float64(ubatchSize)*0.30)
	if maxTokens <= 0 {
		maxTokens = 270 // Fallback (384 * 0.70)
	}

	// Calculate max chars based on conservative ratio
	// For 384 ubatch: 270 tokens * 1.5 chars/token = 405 chars max
	maxChars := int(float64(maxTokens) * 1.5)
	if maxChars > 450 {
		maxChars = 450 // Hard cap for extreme safety
	}

	embedder := &GGUFEmbedder{
		model:         model,
		modelPath:     modelPath,
		dimension:     0, // Detected on first call
		threads:       threads,
		gpuLayers:     gpuLayers,
		maxTokens:     maxTokens,
		maxChars:      maxChars,
		charsPerToken: charsPerToken,
	}

	// Log the dynamic limits for debugging
	slog.Info("GGUF embedder initialized with dynamic limits",
		"ubatch_size", ubatchSize,
		"max_tokens", maxTokens,
		"max_chars", maxChars,
		"chars_per_token", charsPerToken,
		"model_path", modelPath)

	return embedder, nil
}

// NewGGUFEmbedderFromConfig creates a GGUF embedder from a config struct.
func NewGGUFEmbedderFromConfig(cfg GGUFConfig) (*GGUFEmbedder, error) {
	return NewGGUFEmbedder(cfg.ModelPath, cfg.Threads, cfg.GPULayers)
}

// EmbedDocuments generates embeddings for a batch of texts.
// This function continues processing even if individual embeddings fail.
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
	var lastError error
	failedCount := 0

	// Process sequentially; llama.cpp handles parallelism internally.
	for i, text := range texts {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if text == "" {
			slog.Warn("Empty text at index, skipping", "index", i)
			failedCount++
			continue
		}

		// CRITICAL: Limit text length to prevent token overflow
		// Use dynamic limit based on model's actual UBatchSize
		if len(text) > g.maxChars {
			slog.Warn("Text exceeds limit, truncating",
				"index", i,
				"original_length", len(text),
				"truncated_to", g.maxChars)
			text = text[:g.maxChars]
		}

		embedding, err := g.embedSingle(ctx, text)
		if err != nil {
			// Log the error but CONTINUE processing other texts
			slog.Error("Failed to embed document, continuing with next",
				"index", i,
				"error", err,
				"text_length", len(text))
			lastError = err
			failedCount++
			// Set nil embedding for failed item
			result[i] = nil
			continue
		}

		result[i] = embedding
	}

	// If ALL embeddings failed, return error
	if failedCount == len(texts) {
		return nil, fmt.Errorf("all %d embeddings failed, last error: %w", failedCount, lastError)
	}

	// If SOME embeddings failed, log warning but return partial results
	if failedCount > 0 {
		slog.Warn("Some embeddings failed but continuing",
			"failed_count", failedCount,
			"total_count", len(texts),
			"success_rate", fmt.Sprintf("%.1f%%", float64(len(texts)-failedCount)/float64(len(texts))*100))
	}

	return result, nil
}

// EmbedQuery generates an embedding for a single text query.
// This function includes error recovery and detailed logging.
func (g *GGUFEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	originalLength := len(text)

	// CRITICAL: Limit text length to prevent token overflow
	// Use dynamic limit based on model's actual UBatchSize
	if len(text) > g.maxChars {
		slog.Warn("Query text exceeds limit, truncating",
			"original_length", originalLength,
			"truncated_to", g.maxChars)
		text = text[:g.maxChars]
	}

	embedding, err := g.embedSingle(ctx, text)
	if err != nil {
		slog.Error("Failed to embed query",
			"error", err,
			"text_length", len(text),
			"original_length", originalLength)
		return nil, err
	}

	return embedding, nil
}

// embedSingle generates an embedding for a single text.
// A mutex is used to ensure the underlying llama context is accessed safely.
// This function includes panic recovery to prevent crashes from propagating.
func (g *GGUFEmbedder) embedSingle(ctx context.Context, text string) (embeddings []float32, err error) {
	// Recover from panics to prevent program termination
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			slog.Error("CRITICAL: Panic recovered in embedSingle",
				"panic", r,
				"text_length", len(text),
				"stack", string(stack))
			err = fmt.Errorf("panic recovered during embedding: %v", r)
			embeddings = nil
		}
	}()

	g.mu.Lock()
	defer g.mu.Unlock()

	// Verificar contexto
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Validación exhaustiva antes de llamar a C
	if len(text) == 0 {
		return nil, fmt.Errorf("empty text provided")
	}

	if len(text) > g.maxChars {
		slog.Warn("Text exceeds maxChars limit, this should not happen",
			"text_length", len(text),
			"max_chars", g.maxChars)
		text = text[:g.maxChars]
	}

	// Validación del modelo
	if g.model == nil {
		return nil, fmt.Errorf("model is nil")
	}

	// Llamada al modelo con manejo de error
	embeddings, err = g.model.Embed(ctx, text, g.threads)
	if err != nil {
		slog.Error("Failed to generate embeddings",
			"error", err,
			"text_length", len(text),
			"max_chars", g.maxChars,
			"max_tokens", g.maxTokens)
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

// MaxTokens returns the maximum number of tokens this embedder can handle
func (g *GGUFEmbedder) MaxTokens() int {
	return g.maxTokens
}

// MaxChars returns the maximum number of characters this embedder can handle
func (g *GGUFEmbedder) MaxChars() int {
	return g.maxChars
}

// CharsPerToken returns the conservative char-to-token ratio used
func (g *GGUFEmbedder) CharsPerToken() int {
	return g.charsPerToken
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
