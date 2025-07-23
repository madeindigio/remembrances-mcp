// Package embedder provides functionalities for creating vector embeddings of text.
package embedder

import (
	"context"
)

// Embedder define la interfaz para cualquier servicio que pueda crear
// embeddings vectoriales a partir de texto.
type Embedder interface {
	// EmbedDocuments crea embeddings para un lote de textos.
	// Devuelve un slice de vectores de embeddings, uno por cada texto de entrada.
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)

	// EmbedQuery crea un embedding para un único texto (una consulta).
	// Es optimizado para consultas de búsqueda y puede diferir ligeramente
	// del comportamiento de EmbedDocuments en algunos modelos.
	EmbedQuery(ctx context.Context, text string) ([]float32, error)

	// Dimension devuelve la dimensionalidad de los vectores generados.
	// Es crucial para configurar dinámicamente los índices vectoriales.
	Dimension() int
}
