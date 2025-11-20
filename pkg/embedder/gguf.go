package embedder

import (
	"context"
	"fmt"
	"sync"

	llama "github.com/madeindigio/go-llama.cpp"
)

// GGUFEmbedder implementa la interfaz Embedder utilizando modelos GGUF locales
// a través de go-llama.cpp para generar embeddings.
type GGUFEmbedder struct {
	model     *llama.LLama
	modelPath string
	dimension int
	threads   int
	gpuLayers int
	mu        sync.Mutex // Protege el acceso al modelo
}

// GGUFConfig contiene la configuración para el embedder GGUF.
type GGUFConfig struct {
	ModelPath string // Ruta al archivo GGUF del modelo
	Threads   int    // Número de threads a usar (0 = auto-detect)
	GPULayers int    // Número de capas a cargar en GPU (0 = solo CPU)
}

// NewGGUFEmbedder crea una nueva instancia de GGUFEmbedder.
// modelPath: ruta al archivo GGUF del modelo (ej: "/path/to/nomic-embed-text-v1.5.Q4_K_M.gguf")
// threads: número de threads a usar (0 para auto-detect)
// gpuLayers: número de capas a cargar en GPU (0 para solo CPU)
func NewGGUFEmbedder(modelPath string, threads, gpuLayers int) (*GGUFEmbedder, error) {
	if modelPath == "" {
		return nil, fmt.Errorf("model path is required")
	}

	// Configurar opciones del modelo optimizadas para embeddings
	// Context 2048 matches model training, batch sizes allow processing typical documents
	opts := []llama.ModelOption{
		llama.EnableEmbeddings,
		llama.EnableF16Memory,
		llama.SetContext(2048), // Match model's training context
		llama.SetNBatch(2048),  // Allow processing complete documents
		llama.SetNUBatch(2048), // Match batch size for efficiency
	}

	// Añadir capas GPU si se especifican
	if gpuLayers > 0 {
		opts = append(opts, llama.SetGPULayers(gpuLayers))
	}

	// Cargar el modelo
	model, err := llama.New(modelPath, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load GGUF model from %s: %w", modelPath, err)
	}

	// Por defecto, usar el número de CPUs disponibles si threads no se especifica
	if threads <= 0 {
		threads = 8 // Default razonable
	}

	return &GGUFEmbedder{
		model:     model,
		modelPath: modelPath,
		dimension: 0, // Se detectará automáticamente en la primera llamada
		threads:   threads,
		gpuLayers: gpuLayers,
	}, nil
}

// NewGGUFEmbedderFromConfig crea un GGUFEmbedder desde una configuración.
func NewGGUFEmbedderFromConfig(cfg GGUFConfig) (*GGUFEmbedder, error) {
	return NewGGUFEmbedder(cfg.ModelPath, cfg.Threads, cfg.GPULayers)
}

// EmbedDocuments crea embeddings para un lote de textos.
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

	// Procesar cada texto secuencialmente
	// Nota: llama.cpp maneja el paralelismo internamente con threads
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

// EmbedQuery crea un embedding para un único texto (una consulta).
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

// embedSingle genera un embedding para un único texto.
// Usa un mutex para asegurar acceso thread-safe al modelo.
func (g *GGUFEmbedder) embedSingle(ctx context.Context, text string) ([]float32, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Verificar contexto
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Generar embeddings usando llama.cpp
	embeddings, err := g.model.Embeddings(text, llama.SetThreads(g.threads))
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Detectar dimensión en la primera llamada
	if g.dimension == 0 && len(embeddings) > 0 {
		g.dimension = len(embeddings)
	}

	return embeddings, nil
}

// Dimension devuelve la dimensionalidad de los vectores generados.
func (g *GGUFEmbedder) Dimension() int {
	// Si aún no se ha detectado, intentar obtenerla del modelo
	if g.dimension == 0 {
		// Generar un embedding de prueba para detectar la dimensión
		testEmbed, err := g.embedSingle(context.Background(), "test")
		if err == nil && len(testEmbed) > 0 {
			g.dimension = len(testEmbed)
		} else {
			// Si falla, devolver una dimensión por defecto común
			// (768 para nomic-embed-text, 1024 para algunos otros)
			return 768
		}
	}
	return g.dimension
}

// Close libera los recursos del modelo.
func (g *GGUFEmbedder) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.model != nil {
		g.model.Free()
		g.model = nil
	}
	return nil
}

// ModelPath devuelve la ruta al archivo del modelo.
func (g *GGUFEmbedder) ModelPath() string {
	return g.modelPath
}

// Threads devuelve el número de threads configurados.
func (g *GGUFEmbedder) Threads() int {
	return g.threads
}

// GPULayers devuelve el número de capas GPU configuradas.
func (g *GGUFEmbedder) GPULayers() int {
	return g.gpuLayers
}
