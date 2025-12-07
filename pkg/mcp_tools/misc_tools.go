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
	tool, err := protocol.NewTool("hybrid_search", `Search across vectors, graph, and facts. Use how_to_use("hybrid_search") for details.`, HybridSearchInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "hybrid_search", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) getStatsTool() *protocol.Tool {
	tool, err := protocol.NewTool("get_stats", `Get memory statistics for a user. Use how_to_use("get_stats") for details.`, GetStatsInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "get_stats", "err", err)
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

	if results.TotalResults == 0 {
		alt := tm.userAlternatives(ctx, "vector_memories")
		yamlText := CreateEmptyResultYAML(fmt.Sprintf("Hybrid search for '%s' returned no results for user '%s'", input.Query, input.UserID), alt)
		return protocol.NewCallToolResult([]protocol.Content{
			&protocol.TextContent{Type: "text", Text: yamlText},
		}, false), nil
	}

	response := map[string]interface{}{
		"user_id":        input.UserID,
		"query":          input.Query,
		"entities":       input.Entities,
		"limit":          input.Limit,
		"total_results":  results.TotalResults,
		"query_time":     results.QueryTime.String(),
		"vector_results": results.VectorResults,
		"graph_results":  results.GraphResults,
		"facts":          results.Facts,
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{Type: "text", Text: MarshalYAML(response)},
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

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{Type: "text", Text: MarshalYAML(stats)},
	}, false), nil
}
