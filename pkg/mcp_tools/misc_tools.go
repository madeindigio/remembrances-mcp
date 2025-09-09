package mcp_tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// Miscellaneous tool definitions
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

// Miscellaneous tool handlers
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
