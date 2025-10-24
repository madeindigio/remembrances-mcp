package embedder

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	llama "github.com/go-skynet/go-llama.cpp"
)

// LlamaEmbedder implementa la interfaz Embedder utilizando llama.cpp para generar embeddings.
type LlamaEmbedder struct {
	model       *llama.LLama
	modelPath   string
	dimension   int
	threads     int
	gpuLayers   int
	context     int
	mu          sync.RWMutex
	initialized bool
}

// NewLlamaEmbedder crea una nueva instancia de LlamaEmbedder.
// modelPath: ruta al archivo del modelo .gguf
// dimension: dimensión de los embeddings (generalmente 768 para modelos de embeddings)
// threads: número de hilos a usar (por defecto: número de CPUs)
// gpuLayers: número de capas para GPU (por defecto: 0)
// context: tamaño del contexto (por defecto: 512)
func NewLlamaEmbedder(modelPath string, dimension int, threads int, gpuLayers int, context int) (*LlamaEmbedder, error) {
	if modelPath == "" {
		return nil, fmt.Errorf("model path is required")
	}
	if dimension <= 0 {
		return nil, fmt.Errorf("dimension must be positive")
	}
	if threads <= 0 {
		threads = runtime.NumCPU()
	}
	if context <= 0 {
		context = 512
	}

	// Crear el modelo llama.cpp con embeddings habilitados
	model, err := llama.New(
		modelPath,
		llama.EnableEmbeddings,
		llama.SetContext(context),
		llama.SetGPULayers(gpuLayers),
		llama.EnableF16Memory,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load llama.cpp model: %w", err)
	}

	return &LlamaEmbedder{
		model:       model,
		modelPath:   modelPath,
		dimension:   dimension,
		threads:     threads,
		gpuLayers:   gpuLayers,
		context:     context,
		initialized: true,
	}, nil
}

// EmbedDocuments crea embeddings para un lote de textos.
func (l *LlamaEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if !l.initialized {
		return nil, fmt.Errorf("embedder not initialized")
	}
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	// Generar embeddings para cada texto
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			embedding, err := l.model.Embeddings(text, llama.SetThreads(l.threads))
			if err != nil {
				return nil, fmt.Errorf("failed to embed document %d: %w", i, err)
			}

			// Asegurar que el embedding tenga la dimensión correcta
			if len(embedding) != l.dimension {
				// Truncar o padding según sea necesario
				if len(embedding) > l.dimension {
					embedding = embedding[:l.dimension]
				} else {
					// Padding con ceros si es más corto
					padded := make([]float32, l.dimension)
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
func (l *LlamaEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if !l.initialized {
		return nil, fmt.Errorf("embedder not initialized")
	}
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		embedding, err := l.model.Embeddings(text, llama.SetThreads(l.threads))
		if err != nil {
			return nil, fmt.Errorf("failed to embed query: %w", err)
		}

		// Asegurar que el embedding tenga la dimensión correcta
		if len(embedding) != l.dimension {
			if len(embedding) > l.dimension {
				embedding = embedding[:l.dimension]
			} else {
				// Padding con ceros si es más corto
				padded := make([]float32, l.dimension)
				copy(padded, embedding)
				embedding = padded
			}
		}

		return embedding, nil
	}
}

// Dimension devuelve la dimensionalidad de los vectores generados.
func (l *LlamaEmbedder) Dimension() int {
	return l.dimension
}

// Close libera los recursos del modelo llama.cpp.
func (l *LlamaEmbedder) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.initialized && l.model != nil {
		l.model.Free()
		l.initialized = false
	}

	return nil
}

// GetModelPath devuelve la ruta del modelo cargado.
func (l *LlamaEmbedder) GetModelPath() string {
	return l.modelPath
}

// GetThreads devuelve el número de hilos configurados.
func (l *LlamaEmbedder) GetThreads() int {
	return l.threads
}

// GetGPULayers devuelve el número de capas GPU configuradas.
func (l *LlamaEmbedder) GetGPULayers() int {
	return l.gpuLayers
}

// GetContext devuelve el tamaño del contexto configurado.
func (l *LlamaEmbedder) GetContext() int {
	return l.context
}

// IsInitialized devuelve true si el embedder está inicializado.
func (l *LlamaEmbedder) IsInitialized() bool {
	return l.initialized
}
