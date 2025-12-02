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
	tool, err := protocol.NewTool("add_vector", `Add a semantic remembrance (text -> embedding). Use how_to_use("add_vector") for details.`, AddVectorInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "add_vector", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) searchVectorsTool() *protocol.Tool {
	tool, err := protocol.NewTool("search_vectors", `Search remembrances by semantic similarity. Use how_to_use("search_vectors") for details.`, SearchVectorsInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "search_vectors", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) updateVectorTool() *protocol.Tool {
	tool, err := protocol.NewTool("update_vector", `Update an existing semantic remembrance. Use how_to_use("update_vector") for details.`, UpdateVectorInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "update_vector", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) deleteVectorTool() *protocol.Tool {
	tool, err := protocol.NewTool("delete_vector", `Delete a semantic remembrance by ID. Use how_to_use("delete_vector") for details.`, DeleteVectorInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "delete_vector", "err", err)
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

	err = tm.storage.IndexVector(ctx, input.UserID, input.Content, embedding, input.Metadata.AsMap())
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

	err = tm.storage.UpdateVector(ctx, input.ID, input.UserID, input.Content, embedding, input.Metadata.AsMap())
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
