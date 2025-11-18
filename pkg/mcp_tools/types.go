package mcp_tools

import (
	"bytes"
	"encoding/json"
)

// FlexibleObject captures dynamic metadata/properties with JSON marshalling helpers.
type FlexibleObject map[string]interface{}

// UnmarshalJSON decodes arbitrary JSON objects while preserving number precision.
func (f *FlexibleObject) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*f = nil
		return nil
	}

	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()

	raw := make(map[string]interface{})
	if err := decoder.Decode(&raw); err != nil {
		return err
	}

	*f = FlexibleObject(raw)
	return nil
}

// MarshalJSON re-encodes the dynamic object.
func (f FlexibleObject) MarshalJSON() ([]byte, error) {
	if f == nil {
		return []byte("null"), nil
	}
	return json.Marshal(map[string]interface{}(f))
}

// AsMap returns the underlying map for storage operations.
func (f FlexibleObject) AsMap() map[string]interface{} {
	if f == nil {
		return nil
	}
	return map[string]interface{}(f)
}

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
	UserID   string         `json:"user_id"`
	Content  string         `json:"content"`
	Metadata FlexibleObject `json:"metadata,omitempty"`
}

type SearchVectorsInput struct {
	UserID string `json:"user_id"`
	Query  string `json:"query"`
	Limit  int    `json:"limit,omitempty"`
}

type UpdateVectorInput struct {
	ID       string         `json:"id"`
	UserID   string         `json:"user_id"`
	Content  string         `json:"content"`
	Metadata FlexibleObject `json:"metadata,omitempty"`
}

type DeleteVectorInput struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
}

type CreateEntityInput struct {
	EntityType string         `json:"entity_type"`
	Name       string         `json:"name"`
	Properties FlexibleObject `json:"properties,omitempty"`
}

type CreateRelationshipInput struct {
	FromEntity       string         `json:"from_entity"`
	ToEntity         string         `json:"to_entity"`
	RelationshipType string         `json:"relationship_type"`
	Properties       FlexibleObject `json:"properties,omitempty"`
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
	FilePath string         `json:"file_path"`
	Content  string         `json:"content"`
	Metadata FlexibleObject `json:"metadata,omitempty"`
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
