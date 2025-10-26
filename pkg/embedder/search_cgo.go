//go:build static
// +build static

package embedder

// #cgo CFLAGS: -I${SRCDIR}
// #cgo LDFLAGS: -L${SRCDIR}/../../dist/lib -lllama_go -lstdc++ -lm
// #include <stdlib.h>
// #include "search_static.h"
import "C"
import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

// SearchCGOEmbedder implements the Embedder interface using CGO for static linking.
// This implementation is only compiled when the 'static' build tag is used.
// It provides the same functionality as SearchEmbedder but allows for static compilation.
type SearchCGOEmbedder struct {
	model       C.model_ptr
	context     C.context_ptr
	modelPath   string
	dimension   int
	gpuLayers   int
	mu          sync.RWMutex
	initialized bool
}

// NewSearchEmbedder creates a new SearchCGOEmbedder instance.
// This is the static build version that uses CGO instead of purego.
//
// Parameters:
//   - modelPath: Path to the GGUF BERT model file
//   - gpuLayers: Number of layers to offload to GPU (0 for CPU only)
//
// Note: This function is only available when building with -tags static
func NewSearchEmbedder(modelPath string, gpuLayers int) (*SearchCGOEmbedder, error) {
	if modelPath == "" {
		return nil, fmt.Errorf("model path is required")
	}

	if gpuLayers < 0 {
		gpuLayers = 0
	}

	// Initialize library (log level: 2 = WARN)
	C.init_library(2)

	// Load model
	cPath := C.CString(modelPath)
	defer C.free(unsafe.Pointer(cPath))

	model := C.load_model(cPath, C.uint(gpuLayers))
	if model == nil {
		return nil, fmt.Errorf("failed to load model: %s", modelPath)
	}

	// Get embedding dimension
	dimension := int(C.get_embedding_size(model))
	if dimension <= 0 {
		C.free_model(model)
		return nil, fmt.Errorf("invalid embedding dimension: %d", dimension)
	}

	// Create context for embeddings
	ctx := C.create_context(model, 0, 1) // 0 = default size, 1 = embeddings mode
	if ctx == nil {
		C.free_model(model)
		return nil, fmt.Errorf("failed to create context")
	}

	embedder := &SearchCGOEmbedder{
		model:       model,
		context:     ctx,
		modelPath:   modelPath,
		dimension:   dimension,
		gpuLayers:   gpuLayers,
		initialized: true,
	}

	// Set finalizer to ensure cleanup
	runtime.SetFinalizer(embedder, func(e *SearchCGOEmbedder) {
		e.Close()
	})

	return embedder, nil
}

// NewSearchEmbedderWithDimension creates a new SearchCGOEmbedder with a specified dimension.
// The dimension is detected automatically but can be overridden if needed.
func NewSearchEmbedderWithDimension(modelPath string, dimension int, gpuLayers int) (*SearchCGOEmbedder, error) {
	embedder, err := NewSearchEmbedder(modelPath, gpuLayers)
	if err != nil {
		return nil, err
	}

	// Override dimension if specified
	if dimension > 0 && dimension != embedder.dimension {
		embedder.mu.Lock()
		embedder.dimension = dimension
		embedder.mu.Unlock()
	}

	return embedder, nil
}

// EmbedDocuments creates embeddings for a batch of texts.
func (s *SearchCGOEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if !s.initialized {
		return nil, fmt.Errorf("embedder not initialized")
	}
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			emb, err := s.embedTextInternal(text)
			if err != nil {
				return nil, fmt.Errorf("failed to embed document %d: %w", i, err)
			}
			embeddings[i] = emb
		}
	}

	return embeddings, nil
}

// EmbedQuery creates an embedding for a single text (query).
func (s *SearchCGOEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if !s.initialized {
		return nil, fmt.Errorf("embedder not initialized")
	}
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return s.embedTextInternal(text)
	}
}

// embedTextInternal performs the actual embedding using CGO calls.
// Must be called with lock held.
func (s *SearchCGOEmbedder) embedTextInternal(text string) ([]float32, error) {
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))

	embeddings := make([]float32, s.dimension)
	var tokenCount C.uint

	result := C.embed_text(
		s.context,
		cText,
		(*C.float)(unsafe.Pointer(&embeddings[0])),
		&tokenCount,
	)

	if result != 0 {
		return nil, fmt.Errorf("embedding failed with code: %d", result)
	}

	// Verify dimension matches
	if len(embeddings) != s.dimension {
		// Adjust if needed
		if len(embeddings) > s.dimension {
			embeddings = embeddings[:s.dimension]
		} else {
			padded := make([]float32, s.dimension)
			copy(padded, embeddings)
			embeddings = padded
		}
	}

	return embeddings, nil
}

// Dimension returns the dimensionality of the vectors generated.
func (s *SearchCGOEmbedder) Dimension() int {
	return s.dimension
}

// Close frees the resources associated with the embedder.
func (s *SearchCGOEmbedder) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		if s.context != nil {
			C.free_context(s.context)
			s.context = nil
		}
		if s.model != nil {
			C.free_model(s.model)
			s.model = nil
		}
		s.initialized = false
	}

	return nil
}

// GetModelPath returns the path of the loaded model.
func (s *SearchCGOEmbedder) GetModelPath() string {
	return s.modelPath
}

// GetGPULayers returns the number of GPU layers configured.
func (s *SearchCGOEmbedder) GetGPULayers() int {
	return s.gpuLayers
}

// IsInitialized returns true if the embedder is initialized.
func (s *SearchCGOEmbedder) IsInitialized() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.initialized
}

// SetDimension allows changing the expected dimension after initialization.
func (s *SearchCGOEmbedder) SetDimension(dimension int) error {
	if dimension <= 0 {
		return fmt.Errorf("dimension must be positive")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.dimension = dimension
	return nil
}

// DetectDimension attempts to detect the actual dimension by generating
// a test embedding. This is already done during initialization.
func (s *SearchCGOEmbedder) DetectDimension(ctx context.Context) (int, error) {
	if !s.initialized {
		return 0, fmt.Errorf("embedder not initialized")
	}

	// The dimension is already detected during initialization
	return s.dimension, nil
}

// GetInfo returns information about the embedder.
func (s *SearchCGOEmbedder) GetInfo() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"type":        "search-cgo-static",
		"model_path":  s.modelPath,
		"dimension":   s.dimension,
		"gpu_layers":  s.gpuLayers,
		"initialized": s.initialized,
		"backend":     "kelindar/search compatible (CGO static)",
		"cgo":         true,
		"static":      true,
		"num_cpu":     runtime.NumCPU(),
	}
}
