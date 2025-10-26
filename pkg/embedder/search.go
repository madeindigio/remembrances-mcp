package embedder

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/kelindar/search"
)

// SearchEmbedder implementa la interfaz Embedder utilizando kelindar/search para generar embeddings.
// Esta implementación soporta modelos BERT en formato GGUF sin necesidad de cgo.
type SearchEmbedder struct {
	vectorizer  *search.Vectorizer
	modelPath   string
	dimension   int
	gpuLayers   int
	mu          sync.RWMutex
	initialized bool
}

// NewSearchEmbedder crea una nueva instancia de SearchEmbedder.
// modelPath: ruta al archivo del modelo .gguf (debe ser un modelo BERT compatible)
// gpuLayers: número de capas para GPU (por defecto: 0 para CPU)
//
// Nota: kelindar/search detecta automáticamente la dimensión del modelo,
// pero necesitamos especificarla manualmente para SurrealDB (debe ser 768).
func NewSearchEmbedder(modelPath string, gpuLayers int) (*SearchEmbedder, error) {
	if modelPath == "" {
		return nil, fmt.Errorf("model path is required")
	}

	if gpuLayers < 0 {
		gpuLayers = 0
	}

	// Crear el vectorizer con kelindar/search
	// El segundo parámetro es el número de capas GPU (0 para CPU)
	vectorizer, err := search.NewVectorizer(modelPath, gpuLayers)
	if err != nil {
		return nil, fmt.Errorf("failed to load model with kelindar/search: %w", err)
	}

	// kelindar/search usa modelos BERT que típicamente generan embeddings de 768 dimensiones
	// Para MiniLM-L6-v2 y modelos similares, la dimensión es 384
	// Para BERT-base y modelos más grandes, la dimensión es 768
	// Necesitamos detectar esto o permitir que el usuario lo especifique
	dimension := 768 // Valor por defecto para la mayoría de modelos BERT

	return &SearchEmbedder{
		vectorizer:  vectorizer,
		modelPath:   modelPath,
		dimension:   dimension,
		gpuLayers:   gpuLayers,
		initialized: true,
	}, nil
}

// NewSearchEmbedderWithDimension crea una nueva instancia de SearchEmbedder con dimensión especificada.
// Útil cuando se conoce la dimensión exacta del modelo (384 para MiniLM, 768 para BERT-base, etc.)
func NewSearchEmbedderWithDimension(modelPath string, dimension int, gpuLayers int) (*SearchEmbedder, error) {
	if modelPath == "" {
		return nil, fmt.Errorf("model path is required")
	}

	if dimension <= 0 {
		return nil, fmt.Errorf("dimension must be positive")
	}

	if gpuLayers < 0 {
		gpuLayers = 0
	}

	vectorizer, err := search.NewVectorizer(modelPath, gpuLayers)
	if err != nil {
		return nil, fmt.Errorf("failed to load model with kelindar/search: %w", err)
	}

	return &SearchEmbedder{
		vectorizer:  vectorizer,
		modelPath:   modelPath,
		dimension:   dimension,
		gpuLayers:   gpuLayers,
		initialized: true,
	}, nil
}

// EmbedDocuments crea embeddings para un lote de textos.
func (s *SearchEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
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
			// kelindar/search usa EmbedText para generar embeddings
			embedding, err := s.vectorizer.EmbedText(text)
			if err != nil {
				return nil, fmt.Errorf("failed to embed document %d: %w", i, err)
			}

			// Verificar y ajustar dimensión si es necesario
			if len(embedding) != s.dimension {
				// Si el embedding es más largo, truncar
				if len(embedding) > s.dimension {
					embedding = embedding[:s.dimension]
				} else {
					// Si es más corto, hacer padding con ceros
					padded := make([]float32, s.dimension)
					copy(padded, embedding)
					embedding = padded
				}
			}

			embeddings[i] = embedding
		}
	}

	return embeddings, nil
}

// EmbedQuery crea un embedding para un único texto (una consulta).
func (s *SearchEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
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
		embedding, err := s.vectorizer.EmbedText(text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed query: %w", err)
		}

		// Verificar y ajustar dimensión si es necesario
		if len(embedding) != s.dimension {
			if len(embedding) > s.dimension {
				embedding = embedding[:s.dimension]
			} else {
				padded := make([]float32, s.dimension)
				copy(padded, embedding)
				embedding = padded
			}
		}

		return embedding, nil
	}
}

// Dimension devuelve la dimensionalidad de los vectores generados.
func (s *SearchEmbedder) Dimension() int {
	return s.dimension
}

// Close libera los recursos del vectorizer.
func (s *SearchEmbedder) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized && s.vectorizer != nil {
		s.vectorizer.Close()
		s.initialized = false
	}

	return nil
}

// GetModelPath devuelve la ruta del modelo cargado.
func (s *SearchEmbedder) GetModelPath() string {
	return s.modelPath
}

// GetGPULayers devuelve el número de capas GPU configuradas.
func (s *SearchEmbedder) GetGPULayers() int {
	return s.gpuLayers
}

// IsInitialized devuelve true si el embedder está inicializado.
func (s *SearchEmbedder) IsInitialized() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.initialized
}

// SetDimension permite cambiar la dimensión esperada después de la inicialización.
// Útil si se descubre que la dimensión real del modelo es diferente.
func (s *SearchEmbedder) SetDimension(dimension int) error {
	if dimension <= 0 {
		return fmt.Errorf("dimension must be positive")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.dimension = dimension
	return nil
}

// DetectDimension intenta detectar la dimensión real del modelo generando
// un embedding de prueba. Esto es útil para modelos donde no conocemos
// la dimensión de antemano.
func (s *SearchEmbedder) DetectDimension(ctx context.Context) (int, error) {
	if !s.initialized {
		return 0, fmt.Errorf("embedder not initialized")
	}

	// Generar un embedding de prueba con texto simple
	testText := "test"
	embedding, err := s.vectorizer.EmbedText(testText)
	if err != nil {
		return 0, fmt.Errorf("failed to detect dimension: %w", err)
	}

	detectedDim := len(embedding)

	// Actualizar la dimensión del embedder
	s.mu.Lock()
	s.dimension = detectedDim
	s.mu.Unlock()

	return detectedDim, nil
}

// GetInfo devuelve información sobre el embedder actual.
func (s *SearchEmbedder) GetInfo() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"type":        "kelindar/search",
		"model_path":  s.modelPath,
		"dimension":   s.dimension,
		"gpu_layers":  s.gpuLayers,
		"initialized": s.initialized,
		"backend":     "kelindar/search (llama.cpp via purego)",
		"cgo":         false,
		"num_cpu":     runtime.NumCPU(),
	}
}
