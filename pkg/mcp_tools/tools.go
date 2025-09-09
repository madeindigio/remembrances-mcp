// Package mcp_tools provides the tool definitions for the MCP server.
package mcp_tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/madeindigio/remembrances-mcp/internal/storage"
	"github.com/madeindigio/remembrances-mcp/pkg/embedder"

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
	// Helper to register a tool and return error if creation returned nil
	reg := func(name string, tool *protocol.Tool, handler func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)) error {
		if tool == nil {
			return fmt.Errorf("tool %s creation returned nil", name)
		}
		srv.RegisterTool(tool, handler)
		return nil
	}

	// Delegate to smaller registration groups
	if err := tm.registerRemembranceTools(reg); err != nil {
		return err
	}
	if err := tm.registerVectorTools(reg); err != nil {
		return err
	}
	if err := tm.registerGraphTools(reg); err != nil {
		return err
	}
	if err := tm.registerKBTools(reg); err != nil {
		return err
	}
	if err := tm.registerMiscTools(reg); err != nil {
		return err
	}

	slog.Info("Successfully registered all MCP tools")
	return nil
}

// registration helper groups keep RegisterTools small and readable
func (tm *ToolManager) registerRemembranceTools(reg func(string, *protocol.Tool, func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)) error) error {
	if err := reg("remembrance_save_fact", tm.saveFactTool(), tm.saveFactHandler); err != nil {
		return err
	}
	if err := reg("remembrance_get_fact", tm.getFactTool(), tm.getFactHandler); err != nil {
		return err
	}
	if err := reg("remembrance_list_facts", tm.listFactsTool(), tm.listFactsHandler); err != nil {
		return err
	}
	if err := reg("remembrance_delete_fact", tm.deleteFactTool(), tm.deleteFactHandler); err != nil {
		return err
	}
	return nil
}

func (tm *ToolManager) registerVectorTools(reg func(string, *protocol.Tool, func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)) error) error {
	if err := reg("remembrance_add_vector", tm.addVectorTool(), tm.addVectorHandler); err != nil {
		return err
	}
	if err := reg("remembrance_search_vectors", tm.searchVectorsTool(), tm.searchVectorsHandler); err != nil {
		return err
	}
	if err := reg("remembrance_update_vector", tm.updateVectorTool(), tm.updateVectorHandler); err != nil {
		return err
	}
	if err := reg("remembrance_delete_vector", tm.deleteVectorTool(), tm.deleteVectorHandler); err != nil {
		return err
	}
	return nil
}

func (tm *ToolManager) registerGraphTools(reg func(string, *protocol.Tool, func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)) error) error {
	if err := reg("remembrance_create_entity", tm.createEntityTool(), tm.createEntityHandler); err != nil {
		return err
	}
	if err := reg("remembrance_create_relationship", tm.createRelationshipTool(), tm.createRelationshipHandler); err != nil {
		return err
	}
	if err := reg("remembrance_traverse_graph", tm.traverseGraphTool(), tm.traverseGraphHandler); err != nil {
		return err
	}
	if err := reg("remembrance_get_entity", tm.getEntityTool(), tm.getEntityHandler); err != nil {
		return err
	}
	return nil
}

func (tm *ToolManager) registerKBTools(reg func(string, *protocol.Tool, func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)) error) error {
	if err := reg("kb_add_document", tm.addDocumentTool(), tm.addDocumentHandler); err != nil {
		return err
	}
	if err := reg("kb_search_documents", tm.searchDocumentsTool(), tm.searchDocumentsHandler); err != nil {
		return err
	}
	if err := reg("kb_get_document", tm.getDocumentTool(), tm.getDocumentHandler); err != nil {
		return err
	}
	if err := reg("kb_delete_document", tm.deleteDocumentTool(), tm.deleteDocumentHandler); err != nil {
		return err
	}
	return nil
}

func (tm *ToolManager) registerMiscTools(reg func(string, *protocol.Tool, func(context.Context, *protocol.CallToolRequest) (*protocol.CallToolResult, error)) error) error {
	if err := reg("remembrance_hybrid_search", tm.hybridSearchTool(), tm.hybridSearchHandler); err != nil {
		return err
	}
	if err := reg("remembrance_get_stats", tm.getStatsTool(), tm.getStatsHandler); err != nil {
		return err
	}
	return nil
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

// Tool definitions
func (tm *ToolManager) saveFactTool() *protocol.Tool {
	tool, err := protocol.NewTool("remembrance_save_fact", `Save a key-value fact for a user.

Explanation: Stores a simple key -> value pair scoped to a user. Values can be strings, numbers, or objects. This is optimized for fast exact-key lookup, not semantic search.

When to call: Use when you need to persist small, structured facts or preferences (e.g. contact info, settings, short user preferences) that will be retrieved by exact key later.

Example arguments/values:
	user_id: "user123"
	key: "favorite_color"
	value: "blue"
`, SaveFactInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "remembrance_save_fact", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) getFactTool() *protocol.Tool {
	tool, err := protocol.NewTool("remembrance_get_fact", `Retrieve a key-value fact for a user by key.

Explanation: Returns the stored value for the given user/key. If not found, returns nil.

When to call: Use when you know the exact key you stored and need the precise value back (no semantic matching).

Example arguments/values:
	user_id: "user123"
	key: "favorite_color"
`, GetFactInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "remembrance_get_fact", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) listFactsTool() *protocol.Tool {
	tool, err := protocol.NewTool("remembrance_list_facts", `List all key-value facts for a user.

Explanation: Returns all facts previously saved for the specified user as a map of keys to values.

When to call: Use when you need an overview of stored preferences or when initializing a user session.

Example arguments/values:
	user_id: "user123"
`, ListFactsInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "remembrance_list_facts", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) deleteFactTool() *protocol.Tool {
	tool, err := protocol.NewTool("remembrance_delete_fact", `Delete a user-scoped key-value fact.

Explanation: Permanently removes the specified key for the user.

When to call: Use to correct mistakes or to forget outdated personal data (e.g., user requested deletion).

Example arguments/values:
	user_id: "user123"
	key: "favorite_color"
`, DeleteFactInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "remembrance_delete_fact", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) addVectorTool() *protocol.Tool {
	tool, err := protocol.NewTool("remembrance_add_vector", `Add a semantic remembrance (text -> embedding).

Explanation: Converts the provided text into an embedding and stores it with optional metadata for later semantic retrieval.

When to call: Use for storing notes, messages, or any content you may later find by conceptual similarity (e.g., meeting notes, ideas, long-form content).

Example arguments/values:
	user_id: "user123"
	content: "Met Alice about project X; action: follow up on budget."
	metadata: { source: "meeting" }
`, AddVectorInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "remembrance_add_vector", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) searchVectorsTool() *protocol.Tool {
	tool, err := protocol.NewTool("remembrance_search_vectors", `Search remembrances by semantic similarity.

Explanation: Embeds the query and returns the closest stored vectors for the user.

When to call: Use when you want results related by meaning (e.g., find notes about "budget" even if the note doesn't contain the word). Set "limit" to control result count.

Example arguments/values:
	user_id: "user123"
	query: "follow up on project budget"
	limit: 5
`, SearchVectorsInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "remembrance_search_vectors", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) updateVectorTool() *protocol.Tool {
	tool, err := protocol.NewTool("remembrance_update_vector", `Update an existing semantic remembrance.

Explanation: Recomputes embedding for the new content and updates metadata. Requires the vector's ID and the owning user.

When to call: Use when correcting or improving previously stored content.

Example arguments/values:
	id: "vec_abc123"
	user_id: "user123"
	content: "Updated meeting notes..."
	metadata: { edited_by: "user123" }
`, UpdateVectorInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "remembrance_update_vector", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) deleteVectorTool() *protocol.Tool {
	tool, err := protocol.NewTool("remembrance_delete_vector", `Delete a semantic remembrance by ID.

Explanation: Removes the vector record and its embedding. Requires the vector ID and user for authorization/scoping.

When to call: Use to remove obsolete or sensitive semantic items.

Example arguments/values:
	id: "vec_abc123"
	user_id: "user123"
`, DeleteVectorInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "remembrance_delete_vector", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) createEntityTool() *protocol.Tool {
	tool, err := protocol.NewTool("remembrance_create_entity", `Create an entity in the knowledge graph.

Explanation: Adds a typed entity (person, place, concept) with properties to the graph store and returns its ID.

When to call: Use when capturing structured objects you want to link (e.g., contacts, organizations, projects).

Example arguments/values:
	entity_type: "person"
	name: "Alice"
	properties: { email: "alice@example.com" }
`, CreateEntityInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "remembrance_create_entity", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) createRelationshipTool() *protocol.Tool {
	tool, err := protocol.NewTool("remembrance_create_relationship", `Create a relationship between two graph entities.

Explanation: Links two existing entity IDs with a typed relationship and optional properties.

When to call: Use to model connections (e.g., person->works_at->organization, person->knows->person).

Example arguments/values:
	from_entity: "entity_1"
	to_entity: "entity_2"
	relationship_type: "works_at"
	properties: { since: "2023-01-01" }
`, CreateRelationshipInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "remembrance_create_relationship", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) traverseGraphTool() *protocol.Tool {
	tool, err := protocol.NewTool("remembrance_traverse_graph", `Traverse the knowledge graph from a start entity.

Explanation: Performs breadth-limited traversal following relationships and returns connected entities/edges.

When to call: Use when you want to discover related entities (e.g., find colleagues of a person or projects linked to an org). "depth" controls traversal breadth.

Example arguments/values:
	start_entity: "entity_1"
	relationship_type: "works_at"
	depth: 2
`, TraverseGraphInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "remembrance_traverse_graph", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) getEntityTool() *protocol.Tool {
	tool, err := protocol.NewTool("remembrance_get_entity", `Get a graph entity by ID.

Explanation: Returns the stored entity record including properties and metadata.

When to call: Use when you need the full data for a specific entity (e.g., when rendering a contact card).

Example arguments/values:
	entity_id: "entity_1"
`, GetEntityInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "remembrance_get_entity", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) addDocumentTool() *protocol.Tool {
	tool, err := protocol.NewTool("kb_add_document", `Add a document to the knowledge base with automatic embedding.

Explanation: Embeds the document content and stores it together with file path and metadata for semantic document search.

When to call: Use when onboarding reference documents, manuals, or files you want to query semantically.

Example arguments/values:
	file_path: "/kb/guide.pdf"
	content: "Full text of the document..."
	metadata: { source: "import" }
`, AddDocumentInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "kb_add_document", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) searchDocumentsTool() *protocol.Tool {
	tool, err := protocol.NewTool("kb_search_documents", `Search knowledge-base documents by semantic similarity.

Explanation: Embeds the query and returns matching documents ranked by semantic relevance.

When to call: Use to find relevant reference documents or passages given a question or topic.

Example arguments/values:
	query: "how to configure authentication"
	limit: 5
`, SearchDocumentsInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "kb_search_documents", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) getDocumentTool() *protocol.Tool {
	tool, err := protocol.NewTool("kb_get_document", `Retrieve a stored document by file path.

Explanation: Returns the document metadata and content (embedding omitted in responses).

When to call: Use when you know the exact document path and need its contents or metadata.

Example arguments/values:
	file_path: "/kb/guide.pdf"
`, GetDocumentInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "kb_get_document", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) deleteDocumentTool() *protocol.Tool {
	tool, err := protocol.NewTool("kb_delete_document", `Delete a document from the knowledge base by file path.

Explanation: Removes the stored document and its embedding.

When to call: Use to remove outdated or sensitive documents.

Example arguments/values:
	file_path: "/kb/guide.pdf"
`, DeleteDocumentInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "kb_delete_document", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) hybridSearchTool() *protocol.Tool {
	tool, err := protocol.NewTool("remembrance_hybrid_search", `Perform a hybrid search across vectors, graph, and key-value facts.

Explanation: Combines semantic vector search, graph traversal (filtered by entities), and exact-fact lookup to produce a consolidated result set and timing stats.

When to call: Use when you need the broadest coverage for a query that may be answered by facts, documents, graph links, or semantic memories.

Example arguments/values:
	user_id: "user123"
	query: "Who worked on project X and what notes exist?"
	entities: [ "person", "project" ]
	limit: 10
`, HybridSearchInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "remembrance_hybrid_search", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) getStatsTool() *protocol.Tool {
	tool, err := protocol.NewTool("remembrance_get_stats", `Get memory statistics for a user.

Explanation: Returns counts for facts, vectors, documents, entities, relationships and other usage metrics.

When to call: Use for monitoring, quota checks, or to provide an overview dashboard for a user.

Example arguments/values:
	user_id: "user123"
`, GetStatsInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "remembrance_get_stats", "err", err)
		return nil
	}
	return tool
}

// Tool handlers
func (tm *ToolManager) saveFactHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input SaveFactInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf(errParseArgs, err)
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
		return nil, fmt.Errorf(errParseArgs, err)
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
		return nil, fmt.Errorf(errParseArgs, err)
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
		return nil, fmt.Errorf(errParseArgs, err)
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

func (tm *ToolManager) addVectorHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input AddVectorInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf(errParseArgs, err)
	}

	// Generate embedding for the content
	embedding, err := tm.embedder.EmbedQuery(ctx, input.Content)
	if err != nil {
		return nil, fmt.Errorf(errGenEmbedding, err)
	}

	err = tm.storage.IndexVector(ctx, input.UserID, input.Content, embedding, stringMapToInterfaceMap(input.Metadata))
	if err != nil {
		return nil, fmt.Errorf("failed to add remembrance: %w", err)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Successfully added remembrance for user '%s'", input.UserID),
		},
	}, false), nil
}

func (tm *ToolManager) searchVectorsHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input SearchVectorsInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf(errParseArgs, err)
	}

	if input.Limit == 0 {
		input.Limit = 10
	}

	// Generate embedding for the query
	queryEmbedding, err := tm.embedder.EmbedQuery(ctx, input.Query)
	if err != nil {
		return nil, fmt.Errorf(errGenQueryEmbedding, err)
	}

	results, err := tm.storage.SearchSimilar(ctx, input.UserID, queryEmbedding, input.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search remembrances: %w", err)
	}

	resultsBytes, _ := json.MarshalIndent(results, "", "  ")

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Found %d similar remembrances for query '%s':\n%s", len(results), input.Query, string(resultsBytes)),
		},
	}, false), nil
}

func (tm *ToolManager) updateVectorHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input UpdateVectorInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf(errParseArgs, err)
	}

	// Generate new embedding for the updated content
	embedding, err := tm.embedder.EmbedQuery(ctx, input.Content)
	if err != nil {
		return nil, fmt.Errorf(errGenEmbedding, err)
	}

	err = tm.storage.UpdateVector(ctx, input.ID, input.UserID, input.Content, embedding, stringMapToInterfaceMap(input.Metadata))
	if err != nil {
		return nil, fmt.Errorf("failed to update remembrance: %w", err)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Successfully updated remembrance '%s' for user '%s'", input.ID, input.UserID),
		},
	}, false), nil
}

func (tm *ToolManager) deleteVectorHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input DeleteVectorInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf(errParseArgs, err)
	}

	err := tm.storage.DeleteVector(ctx, input.ID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete remembrance: %w", err)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Successfully deleted remembrance '%s' for user '%s'", input.ID, input.UserID),
		},
	}, false), nil
}

func (tm *ToolManager) createEntityHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CreateEntityInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf(errParseArgs, err)
	}

	err := tm.storage.CreateEntity(ctx, input.EntityType, input.Name, stringMapToInterfaceMap(input.Properties))
	if err != nil {
		return nil, fmt.Errorf("failed to create entity: %w", err)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Successfully created entity '%s' of type '%s'", input.Name, input.EntityType),
		},
	}, false), nil
}

func (tm *ToolManager) createRelationshipHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CreateRelationshipInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf(errParseArgs, err)
	}

	err := tm.storage.CreateRelationship(ctx, input.FromEntity, input.ToEntity, input.RelationshipType, stringMapToInterfaceMap(input.Properties))
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
		return nil, fmt.Errorf(errParseArgs, err)
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
		return nil, fmt.Errorf(errParseArgs, err)
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
		return nil, fmt.Errorf(errParseArgs, err)
	}

	// Generate embedding for the document content
	embedding, err := tm.embedder.EmbedQuery(ctx, input.Content)
	if err != nil {
		return nil, fmt.Errorf(errGenEmbedding, err)
	}

	err = tm.storage.SaveDocument(ctx, input.FilePath, input.Content, embedding, stringMapToInterfaceMap(input.Metadata))
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
		return nil, fmt.Errorf(errParseArgs, err)
	}

	if input.Limit == 0 {
		input.Limit = 10
	}

	// Generate embedding for the query
	queryEmbedding, err := tm.embedder.EmbedQuery(ctx, input.Query)
	if err != nil {
		return nil, fmt.Errorf(errGenQueryEmbedding, err)
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
		return nil, fmt.Errorf(errParseArgs, err)
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
		return nil, fmt.Errorf(errParseArgs, err)
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
		return nil, fmt.Errorf(errParseArgs, err)
	}

	if input.Limit == 0 {
		input.Limit = 10
	}

	// Generate embedding for the query
	queryEmbedding, err := tm.embedder.EmbedQuery(ctx, input.Query)
	if err != nil {
		return nil, fmt.Errorf(errGenQueryEmbedding, err)
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
		return nil, fmt.Errorf(errParseArgs, err)
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
