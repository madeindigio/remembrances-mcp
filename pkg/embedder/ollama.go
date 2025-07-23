package embedder

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/ollama"
)

// OllamaEmbedder implementa la interfaz Embedder utilizando Ollama para generar embeddings.
type OllamaEmbedder struct {
	client    *ollama.LLM
	model     string
	dimension int
}

// NewOllamaEmbedder crea una nueva instancia de OllamaEmbedder.
// url: URL del servidor Ollama (ej: "http://localhost:11434")
// model: nombre del modelo de embedding (ej: "nomic-embed-text", "mxbai-embed-large")
func NewOllamaEmbedder(url, model string) (*OllamaEmbedder, error) {
	if url == "" {
		return nil, fmt.Errorf("ollama URL is required")
	}
	if model == "" {
		return nil, fmt.Errorf("ollama model name is required")
	}

	// Crear cliente Ollama
	client, err := ollama.New(
		ollama.WithServerURL(url),
		ollama.WithModel(model),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama client: %w", err)
	}

	// Determinar dimensión basada en el modelo conocido
	// Estos son valores típicos para modelos comunes
	dimension := getDimensionForModel(model)

	return &OllamaEmbedder{
		client:    client,
		model:     model,
		dimension: dimension,
	}, nil
}

// EmbedDocuments crea embeddings para un lote de textos.
func (o *OllamaEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	// Crear el embedder con el cliente Ollama
	embedder, err := embeddings.NewEmbedder(o.client)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	// Generar embeddings para todos los textos
	embeddings, err := embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("failed to embed documents: %w", err)
	}

	// Convertir de [][]float64 a [][]float32 para consistencia
	result := make([][]float32, len(embeddings))
	for i, emb := range embeddings {
		result[i] = make([]float32, len(emb))
		for j, val := range emb {
			result[i][j] = float32(val)
		}
	}

	return result, nil
}

// EmbedQuery crea un embedding para un único texto (una consulta).
func (o *OllamaEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// Crear el embedder con el cliente Ollama
	embedder, err := embeddings.NewEmbedder(o.client)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	// Generar embedding para la consulta
	embedding, err := embedder.EmbedQuery(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Convertir de []float64 a []float32
	result := make([]float32, len(embedding))
	for i, val := range embedding {
		result[i] = float32(val)
	}

	return result, nil
}

// Dimension devuelve la dimensionalidad de los vectores generados.
func (o *OllamaEmbedder) Dimension() int {
	return o.dimension
}

// getDimensionForModel devuelve la dimensión conocida para modelos específicos.
// Si el modelo no es conocido, devuelve una dimensión por defecto.
func getDimensionForModel(model string) int {
	switch model {
	case "nomic-embed-text":
		return 768
	case "mxbai-embed-large":
		return 1024
	case "all-minilm":
		return 384
	case "sentence-transformers/all-MiniLM-L6-v2":
		return 384
	case "sentence-transformers/all-mpnet-base-v2":
		return 768
	default:
		// Dimensión por defecto para modelos desconocidos
		return 768
	}
}
