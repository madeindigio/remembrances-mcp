package mcp_tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// Vector tool definitions
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

// Vector tool handlers
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
