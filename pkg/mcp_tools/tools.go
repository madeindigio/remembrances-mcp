// Package mcp_tools provides the tool definitions for the MCP server.
package mcp_tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"remembrances-mcp/internal/storage"
	"remembrances-mcp/pkg/embedder"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	mcpserver "github.com/ThinkInAIXYZ/go-mcp/server"
)

// ToolManager manages all MCP tools for the remembrances server
type ToolManager struct {
	storage  storage.StorageWithStats
	embedder embedder.Embedder
}

// NewToolManager creates a new tool manager
func NewToolManager(storage storage.StorageWithStats, embedder embedder.Embedder) *ToolManager {
	return &ToolManager{
		storage:  storage,
		embedder: embedder,
	}
}

// RegisterTools registers all MCP tools with the server
func (tm *ToolManager) RegisterTools(srv *mcpserver.Server) error {
	// Memory operations tools
	srv.RegisterTool(tm.saveFactTool(), tm.saveFactHandler)
	srv.RegisterTool(tm.getFactTool(), tm.getFactHandler)
	srv.RegisterTool(tm.listFactsTool(), tm.listFactsHandler)
	srv.RegisterTool(tm.deleteFactTool(), tm.deleteFactHandler)

	// Vector operations tools
	srv.RegisterTool(tm.addMemoryTool(), tm.addMemoryHandler)
	srv.RegisterTool(tm.searchMemoriesTool(), tm.searchMemoriesHandler)
	srv.RegisterTool(tm.updateMemoryTool(), tm.updateMemoryHandler)
	srv.RegisterTool(tm.deleteMemoryTool(), tm.deleteMemoryHandler)

	// Graph operations tools
	srv.RegisterTool(tm.createEntityTool(), tm.createEntityHandler)
	srv.RegisterTool(tm.createRelationshipTool(), tm.createRelationshipHandler)
	srv.RegisterTool(tm.traverseGraphTool(), tm.traverseGraphHandler)
	srv.RegisterTool(tm.getEntityTool(), tm.getEntityHandler)

	// Knowledge base tools
	srv.RegisterTool(tm.addDocumentTool(), tm.addDocumentHandler)
	srv.RegisterTool(tm.searchDocumentsTool(), tm.searchDocumentsHandler)
	srv.RegisterTool(tm.getDocumentTool(), tm.getDocumentHandler)
	srv.RegisterTool(tm.deleteDocumentTool(), tm.deleteDocumentHandler)

	// Hybrid search and statistics
	srv.RegisterTool(tm.hybridSearchTool(), tm.hybridSearchHandler)
	srv.RegisterTool(tm.getStatsTool(), tm.getStatsHandler)

	slog.Info("Successfully registered all MCP tools")
	return nil
}

// Tool input structs
type SaveFactInput struct {
	UserID string      `json:"user_id"`
	Key    string      `json:"key"`
	Value  interface{} `json:"value"`
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

type AddMemoryInput struct {
	UserID   string                 `json:"user_id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type SearchMemoriesInput struct {
	UserID string `json:"user_id"`
	Query  string `json:"query"`
	Limit  int    `json:"limit,omitempty"`
}

type UpdateMemoryInput struct {
	ID       string                 `json:"id"`
	UserID   string                 `json:"user_id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type DeleteMemoryInput struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
}

type CreateEntityInput struct {
	EntityType string                 `json:"entity_type"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

type CreateRelationshipInput struct {
	FromEntity       string                 `json:"from_entity"`
	ToEntity         string                 `json:"to_entity"`
	RelationshipType string                 `json:"relationship_type"`
	Properties       map[string]interface{} `json:"properties,omitempty"`
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
	FilePath string                 `json:"file_path"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
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

// Tool definitions
func (tm *ToolManager) saveFactTool() *protocol.Tool {
	tool, _ := protocol.NewTool("mem_save_fact", "Save a key-value fact to memory for a specific user", SaveFactInput{})
	return tool
}

func (tm *ToolManager) getFactTool() *protocol.Tool {
	tool, _ := protocol.NewTool("mem_get_fact", "Retrieve a key-value fact from memory for a specific user", GetFactInput{})
	return tool
}

func (tm *ToolManager) listFactsTool() *protocol.Tool {
	tool, _ := protocol.NewTool("mem_list_facts", "List all key-value facts for a specific user", ListFactsInput{})
	return tool
}

func (tm *ToolManager) deleteFactTool() *protocol.Tool {
	tool, _ := protocol.NewTool("mem_delete_fact", "Delete a key-value fact from memory", DeleteFactInput{})
	return tool
}

func (tm *ToolManager) addMemoryTool() *protocol.Tool {
	tool, _ := protocol.NewTool("mem_add_memory", "Add a memory with semantic content that will be automatically embedded for similarity search", AddMemoryInput{})
	return tool
}

func (tm *ToolManager) searchMemoriesTool() *protocol.Tool {
	tool, _ := protocol.NewTool("mem_search_memories", "Search for similar memories using semantic similarity", SearchMemoriesInput{})
	return tool
}

func (tm *ToolManager) updateMemoryTool() *protocol.Tool {
	tool, _ := protocol.NewTool("mem_update_memory", "Update an existing memory with new content and metadata", UpdateMemoryInput{})
	return tool
}

func (tm *ToolManager) deleteMemoryTool() *protocol.Tool {
	tool, _ := protocol.NewTool("mem_delete_memory", "Delete a memory by ID", DeleteMemoryInput{})
	return tool
}

func (tm *ToolManager) createEntityTool() *protocol.Tool {
	tool, _ := protocol.NewTool("mem_create_entity", "Create an entity in the knowledge graph", CreateEntityInput{})
	return tool
}

func (tm *ToolManager) createRelationshipTool() *protocol.Tool {
	tool, _ := protocol.NewTool("mem_create_relationship", "Create a relationship between two entities in the knowledge graph", CreateRelationshipInput{})
	return tool
}

func (tm *ToolManager) traverseGraphTool() *protocol.Tool {
	tool, _ := protocol.NewTool("mem_traverse_graph", "Traverse the knowledge graph starting from an entity", TraverseGraphInput{})
	return tool
}

func (tm *ToolManager) getEntityTool() *protocol.Tool {
	tool, _ := protocol.NewTool("mem_get_entity", "Get details of an entity by ID", GetEntityInput{})
	return tool
}

func (tm *ToolManager) addDocumentTool() *protocol.Tool {
	tool, _ := protocol.NewTool("kb_add_document", "Add a document to the knowledge base with automatic embedding", AddDocumentInput{})
	return tool
}

func (tm *ToolManager) searchDocumentsTool() *protocol.Tool {
	tool, _ := protocol.NewTool("kb_search_documents", "Search for documents in the knowledge base using semantic similarity", SearchDocumentsInput{})
	return tool
}

func (tm *ToolManager) getDocumentTool() *protocol.Tool {
	tool, _ := protocol.NewTool("kb_get_document", "Get a document from the knowledge base by file path", GetDocumentInput{})
	return tool
}

func (tm *ToolManager) deleteDocumentTool() *protocol.Tool {
	tool, _ := protocol.NewTool("kb_delete_document", "Delete a document from the knowledge base", DeleteDocumentInput{})
	return tool
}

func (tm *ToolManager) hybridSearchTool() *protocol.Tool {
	tool, _ := protocol.NewTool("mem_hybrid_search", "Perform a hybrid search across all memory layers (vector, graph, and key-value)", HybridSearchInput{})
	return tool
}

func (tm *ToolManager) getStatsTool() *protocol.Tool {
	tool, _ := protocol.NewTool("mem_get_stats", "Get statistics about stored memories for a user", GetStatsInput{})
	return tool
}

// Tool handlers
func (tm *ToolManager) saveFactHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input SaveFactInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	err := tm.storage.SaveFact(ctx, input.UserID, input.Key, input.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to save fact: %w", err)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Successfully saved fact '%s' for user '%s'", input.Key, input.UserID),
		},
	}, false), nil
}

func (tm *ToolManager) getFactHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input GetFactInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	value, err := tm.storage.GetFact(ctx, input.UserID, input.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to get fact: %w", err)
	}

	if value == nil {
		return protocol.NewCallToolResult([]protocol.Content{
			&protocol.TextContent{
				Type: "text",
				Text: fmt.Sprintf("No fact found for key '%s' and user '%s'", input.Key, input.UserID),
			},
		}, false), nil
	}

	valueBytes, _ := json.Marshal(value)

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Fact '%s' for user '%s': %s", input.Key, input.UserID, string(valueBytes)),
		},
	}, false), nil
}

func (tm *ToolManager) listFactsHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input ListFactsInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	facts, err := tm.storage.ListFacts(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to list facts: %w", err)
	}

	factsBytes, _ := json.MarshalIndent(facts, "", "  ")

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Facts for user '%s':\n%s", input.UserID, string(factsBytes)),
		},
	}, false), nil
}

func (tm *ToolManager) deleteFactHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input DeleteFactInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	err := tm.storage.DeleteFact(ctx, input.UserID, input.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to delete fact: %w", err)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Successfully deleted fact '%s' for user '%s'", input.Key, input.UserID),
		},
	}, false), nil
}

func (tm *ToolManager) addMemoryHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input AddMemoryInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Generate embedding for the content
	embedding, err := tm.embedder.EmbedQuery(ctx, input.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	memoryID, err := tm.storage.IndexVector(ctx, input.UserID, input.Content, embedding, input.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to add memory: %w", err)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Successfully added memory with ID '%s' for user '%s'", memoryID, input.UserID),
		},
	}, false), nil
}

func (tm *ToolManager) searchMemoriesHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input SearchMemoriesInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if input.Limit == 0 {
		input.Limit = 10
	}

	// Generate embedding for the query
	queryEmbedding, err := tm.embedder.EmbedQuery(ctx, input.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	results, err := tm.storage.SearchSimilar(ctx, input.UserID, queryEmbedding, input.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search memories: %w", err)
	}

	resultsBytes, _ := json.MarshalIndent(results, "", "  ")

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Found %d similar memories for query '%s':\n%s", len(results), input.Query, string(resultsBytes)),
		},
	}, false), nil
}

func (tm *ToolManager) updateMemoryHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input UpdateMemoryInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Generate new embedding for the updated content
	embedding, err := tm.embedder.EmbedQuery(ctx, input.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	err = tm.storage.UpdateVector(ctx, input.ID, input.UserID, input.Content, embedding, input.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to update memory: %w", err)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Successfully updated memory '%s' for user '%s'", input.ID, input.UserID),
		},
	}, false), nil
}

func (tm *ToolManager) deleteMemoryHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input DeleteMemoryInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	err := tm.storage.DeleteVector(ctx, input.ID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete memory: %w", err)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Successfully deleted memory '%s' for user '%s'", input.ID, input.UserID),
		},
	}, false), nil
}

func (tm *ToolManager) createEntityHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CreateEntityInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	entityID, err := tm.storage.CreateEntity(ctx, input.EntityType, input.Name, input.Properties)
	if err != nil {
		return nil, fmt.Errorf("failed to create entity: %w", err)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Successfully created entity '%s' of type '%s' with ID '%s'", input.Name, input.EntityType, entityID),
		},
	}, false), nil
}

func (tm *ToolManager) createRelationshipHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CreateRelationshipInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	err := tm.storage.CreateRelationship(ctx, input.FromEntity, input.ToEntity, input.RelationshipType, input.Properties)
	if err != nil {
		return nil, fmt.Errorf("failed to create relationship: %w", err)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Successfully created '%s' relationship from '%s' to '%s'", input.RelationshipType, input.FromEntity, input.ToEntity),
		},
	}, false), nil
}

func (tm *ToolManager) traverseGraphHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input TraverseGraphInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if input.Depth == 0 {
		input.Depth = 2
	}

	results, err := tm.storage.TraverseGraph(ctx, input.StartEntity, input.RelationshipType, input.Depth)
	if err != nil {
		return nil, fmt.Errorf("failed to traverse graph: %w", err)
	}

	resultsBytes, _ := json.MarshalIndent(results, "", "  ")

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Graph traversal from '%s' found %d results:\n%s", input.StartEntity, len(results), string(resultsBytes)),
		},
	}, false), nil
}

func (tm *ToolManager) getEntityHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input GetEntityInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	entity, err := tm.storage.GetEntity(ctx, input.EntityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	if entity == nil {
		return protocol.NewCallToolResult([]protocol.Content{
			&protocol.TextContent{
				Type: "text",
				Text: fmt.Sprintf("No entity found with ID '%s'", input.EntityID),
			},
		}, false), nil
	}

	entityBytes, _ := json.MarshalIndent(entity, "", "  ")

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Entity '%s':\n%s", input.EntityID, string(entityBytes)),
		},
	}, false), nil
}

func (tm *ToolManager) addDocumentHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input AddDocumentInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Generate embedding for the document content
	embedding, err := tm.embedder.EmbedQuery(ctx, input.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	err = tm.storage.SaveDocument(ctx, input.FilePath, input.Content, embedding, input.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to add document: %w", err)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Successfully added document '%s' to knowledge base", input.FilePath),
		},
	}, false), nil
}

func (tm *ToolManager) searchDocumentsHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input SearchDocumentsInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if input.Limit == 0 {
		input.Limit = 10
	}

	// Generate embedding for the query
	queryEmbedding, err := tm.embedder.EmbedQuery(ctx, input.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	results, err := tm.storage.SearchDocuments(ctx, queryEmbedding, input.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}

	resultsBytes, _ := json.MarshalIndent(results, "", "  ")

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Found %d documents for query '%s':\n%s", len(results), input.Query, string(resultsBytes)),
		},
	}, false), nil
}

func (tm *ToolManager) getDocumentHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input GetDocumentInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	document, err := tm.storage.GetDocument(ctx, input.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	if document == nil {
		return protocol.NewCallToolResult([]protocol.Content{
			&protocol.TextContent{
				Type: "text",
				Text: fmt.Sprintf("No document found at path '%s'", input.FilePath),
			},
		}, false), nil
	}

	// Don't include embedding in response (too large)
	doc := *document
	doc.Embedding = nil
	docBytes, _ := json.MarshalIndent(doc, "", "  ")

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Document '%s':\n%s", input.FilePath, string(docBytes)),
		},
	}, false), nil
}

func (tm *ToolManager) deleteDocumentHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input DeleteDocumentInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	err := tm.storage.DeleteDocument(ctx, input.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to delete document: %w", err)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Successfully deleted document '%s'", input.FilePath),
		},
	}, false), nil
}

func (tm *ToolManager) hybridSearchHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input HybridSearchInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if input.Limit == 0 {
		input.Limit = 10
	}

	// Generate embedding for the query
	queryEmbedding, err := tm.embedder.EmbedQuery(ctx, input.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	results, err := tm.storage.HybridSearch(ctx, input.UserID, queryEmbedding, input.Entities, input.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to perform hybrid search: %w", err)
	}

	resultsBytes, _ := json.MarshalIndent(results, "", "  ")

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Hybrid search for '%s' returned %d total results in %v:\n%s", input.Query, results.TotalResults, results.QueryTime, string(resultsBytes)),
		},
	}, false), nil
}

func (tm *ToolManager) getStatsHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input GetStatsInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	stats, err := tm.storage.GetStats(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	statsBytes, _ := json.MarshalIndent(stats, "", "  ")

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Memory statistics for user '%s':\n%s", input.UserID, string(statsBytes)),
		},
	}, false), nil
}
