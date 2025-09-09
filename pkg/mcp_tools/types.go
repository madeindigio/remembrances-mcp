package mcp_tools

// Tool input structs
type SaveFactInput struct {
	UserID string `json:"user_id"`
	Key    string `json:"key"`
	Value  string `json:"value"`
}

type GetFactInput struct {
	UserID string `json:"user_id"`
	Key    string `json:"key"`
}

type ListFactsInput struct {
	UserID string `json:"user_id"`
}

type DeleteFactInput struct {
	UserID string `json:"user_id"`
	Key    string `json:"key"`
}

type AddVectorInput struct {
	UserID   string            `json:"user_id"`
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type SearchVectorsInput struct {
	UserID string `json:"user_id"`
	Query  string `json:"query"`
	Limit  int    `json:"limit,omitempty"`
}

type UpdateVectorInput struct {
	ID       string            `json:"id"`
	UserID   string            `json:"user_id"`
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type DeleteVectorInput struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
}

type CreateEntityInput struct {
	EntityType string            `json:"entity_type"`
	Name       string            `json:"name"`
	Properties map[string]string `json:"properties,omitempty"`
}

type CreateRelationshipInput struct {
	FromEntity       string            `json:"from_entity"`
	ToEntity         string            `json:"to_entity"`
	RelationshipType string            `json:"relationship_type"`
	Properties       map[string]string `json:"properties,omitempty"`
}

type TraverseGraphInput struct {
	StartEntity      string `json:"start_entity"`
	RelationshipType string `json:"relationship_type,omitempty"`
	Depth            int    `json:"depth,omitempty"`
}

type GetEntityInput struct {
	EntityID string `json:"entity_id"`
}

type AddDocumentInput struct {
	FilePath string            `json:"file_path"`
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type SearchDocumentsInput struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

type GetDocumentInput struct {
	FilePath string `json:"file_path"`
}

type DeleteDocumentInput struct {
	FilePath string `json:"file_path"`
}

type HybridSearchInput struct {
	UserID   string   `json:"user_id"`
	Query    string   `json:"query"`
	Entities []string `json:"entities,omitempty"`
	Limit    int      `json:"limit,omitempty"`
}

type GetStatsInput struct {
	UserID string `json:"user_id"`
}

const (
	errParseArgs         = "failed to parse arguments: %w"
	errGenEmbedding      = "failed to generate embedding: %w"
	errGenQueryEmbedding = "failed to generate query embedding: %w"
)

// Helper function to convert map[string]string to map[string]interface{}
func stringMapToInterfaceMap(m map[string]string) map[string]interface{} {
	if m == nil {
		return nil
	}
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
