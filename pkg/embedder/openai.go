package embedder

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
)

// OpenAIEmbedder implementa la interfaz Embedder utilizando OpenAI o APIs compatibles.
type OpenAIEmbedder struct {
	client    *openai.LLM
	model     string
	dimension int
}

// NewOpenAIEmbedder crea una nueva instancia de OpenAIEmbedder.
// apiKey: clave de API de OpenAI o servicio compatible
// baseURL: URL base para APIs compatibles con OpenAI (opcional, usa OpenAI por defecto)
// model: nombre del modelo de embedding (ej: "text-embedding-3-large", "text-embedding-ada-002")
func NewOpenAIEmbedder(apiKey, baseURL, model string) (*OpenAIEmbedder, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	if model == "" {
		return nil, fmt.Errorf("model name is required")
	}

	// Configurar opciones del cliente
	opts := []openai.Option{
		openai.WithToken(apiKey),
		openai.WithModel(model),
	}

	// Si se proporciona una URL base personalizada, configurarla
	if baseURL != "" {
		opts = append(opts, openai.WithBaseURL(baseURL))
	}

	// Crear cliente OpenAI
	client, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	// Determinar dimensión basada en el modelo
	dimension := getDimensionForOpenAIModel(model)

	return &OpenAIEmbedder{
		client:    client,
		model:     model,
		dimension: dimension,
	}, nil
}

// EmbedDocuments crea embeddings para un lote de textos.
func (o *OpenAIEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	// Crear el embedder con el cliente OpenAI
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
func (o *OpenAIEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// Crear el embedder con el cliente OpenAI
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
func (o *OpenAIEmbedder) Dimension() int {
	return o.dimension
}

// getDimensionForOpenAIModel devuelve la dimensión conocida para modelos de OpenAI.
func getDimensionForOpenAIModel(model string) int {
	switch model {
	case "text-embedding-3-large":
		return 3072
	case "text-embedding-3-small":
		return 1536
	case "text-embedding-ada-002":
		return 1536
	case "text-davinci-002":
		return 12288
	case "text-curie-001":
		return 4096
	default:
		// Dimensión por defecto para modelos desconocidos
		// Usar el tamaño más común de OpenAI
		return 1536
	}
}
