package mcp_tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"gopkg.in/yaml.v3"
)

// Remember tool definitions
func (tm *ToolManager) toRememberTool() *protocol.Tool {
	tool, err := protocol.NewTool("to_remember", `Store important information for future sessions. Use how_to_use("to_remember") for details.`, ToRememberInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "to_remember", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) lastToRememberTool() *protocol.Tool {
	tool, err := protocol.NewTool("last_to_remember", `Retrieve stored context and recent work for a user. Use how_to_use("last_to_remember") for details.`, LastToRememberInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "last_to_remember", "err", err)
		return nil
	}
	return tool
}

// Remember tool handlers
func (tm *ToolManager) toRememberHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input ToRememberInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf(errParseArgs, err)
	}

	// Store the content as a special fact
	err := tm.storage.SaveFact(ctx, input.UserID, "__to_remember__", input.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to save to_remember fact: %w", err)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Successfully stored important information for user '%s'. This will be retrievable via 'last_to_remember'.", input.UserID),
		},
	}, false), nil
}

func (tm *ToolManager) lastToRememberHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input LastToRememberInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf(errParseArgs, err)
	}

	// Collect all the information
	result := make(map[string]interface{})

	// 1. Get the "to_remember" fact
	toRemember, err := tm.storage.GetFact(ctx, input.UserID, "__to_remember__")
	if err == nil && toRemember != nil {
		result["to_remember"] = toRemember
	}

	// 2. Get all facts
	facts, err := tm.storage.ListFacts(ctx, input.UserID)
	if err == nil && len(facts) > 0 {
		// Filter out the __to_remember__ key since we already included it
		filteredFacts := make(map[string]interface{})
		for k, v := range facts {
			if k != "__to_remember__" {
				filteredFacts[k] = v
			}
		}
		if len(filteredFacts) > 0 {
			result["facts"] = filteredFacts
		}
	}

	// 3. Get recent vectors (limit to 10 most recent by using a broad search)
	// Note: We don't have a direct "list all vectors" method, so we use stats to check if any exist
	stats, err := tm.storage.GetStats(ctx, input.UserID)
	if err == nil && stats.VectorCount > 0 {
		result["vectors_count"] = stats.VectorCount
		result["vectors_note"] = "Use search_vectors to query specific semantic memories"
	}

	// 4. Get graph entities - query the database directly for recent entities
	graphInfo, err := tm.getRecentGraphEntities(ctx, input.UserID)
	if err == nil && len(graphInfo) > 0 {
		result["graph_entities"] = graphInfo
	}

	// 5. Get the 5 most recent documents from knowledge base
	recentDocs, err := tm.getRecentDocuments(ctx, 5)
	if err == nil && len(recentDocs) > 0 {
		result["recent_documents"] = recentDocs
	}

	// Add metadata
	result["retrieved_at"] = time.Now().Format(time.RFC3339)
	result["user_id"] = input.UserID
	result["note"] = "This information may be of interest to remember what you have been working on most recently or what is important to remember"

	// Convert to YAML
	yamlBytes, err := yaml.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to YAML: %w", err)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: string(yamlBytes),
		},
	}, false), nil
}

// Helper function to get recent graph entities
func (tm *ToolManager) getRecentGraphEntities(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	// Query recent entities from the graph
	query := `SELECT id, type, name, properties, created_at, updated_at
	          FROM entities
	          WHERE user_id = $user_id
	          ORDER BY updated_at DESC
	          LIMIT 10`

	params := map[string]interface{}{
		"user_id": userID,
	}

	// We need to access the underlying SurrealDB storage to execute custom queries
	// This is a simplified approach - in production you might want to add a method to the Storage interface
	result, err := tm.executeCustomQuery(ctx, query, params)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Helper function to get recent documents
func (tm *ToolManager) getRecentDocuments(ctx context.Context, limit int) ([]string, error) {
	// Query recent documents from the knowledge base
	query := fmt.Sprintf(`SELECT file_path, created_at, updated_at
	                      FROM knowledge_base
	                      ORDER BY updated_at DESC
	                      LIMIT %d`, limit)

	result, err := tm.executeCustomQuery(ctx, query, nil)
	if err != nil {
		return nil, err
	}

	// Extract just the file paths
	filePaths := make([]string, 0, len(result))
	for _, doc := range result {
		if filePath, ok := doc["file_path"].(string); ok {
			filePaths = append(filePaths, filePath)
		}
	}

	return filePaths, nil
}

// Helper function to execute custom queries
// This is a workaround since we don't have direct access to query methods
// In a production environment, you'd want to add these methods to the Storage interface
func (tm *ToolManager) executeCustomQuery(ctx context.Context, query string, params map[string]interface{}) ([]map[string]interface{}, error) {
	// This is a placeholder - we'll need to check if the storage implementation
	// exposes a query method or if we need to add one to the interface

	// For now, return an empty result to avoid compilation errors
	// The actual implementation would depend on accessing the underlying storage

	// Try to do a type assertion to see if we can access the query method
	type querier interface {
		Query(ctx context.Context, query string, params map[string]interface{}) ([]map[string]interface{}, error)
	}

	if q, ok := tm.storage.(querier); ok {
		return q.Query(ctx, query, params)
	}

	// Return empty result if we can't query
	return []map[string]interface{}{}, nil
}

// Additional helper to format entity info for YAML output
func formatEntityInfo(entities []map[string]interface{}) []map[string]interface{} {
	formatted := make([]map[string]interface{}, 0, len(entities))
	for _, entity := range entities {
		info := make(map[string]interface{})
		if id, ok := entity["id"]; ok {
			info["id"] = id
		}
		if entityType, ok := entity["type"]; ok {
			info["type"] = entityType
		}
		if name, ok := entity["name"]; ok {
			info["name"] = name
		}
		// Only include non-empty properties
		if props, ok := entity["properties"].(map[string]interface{}); ok && len(props) > 0 {
			info["properties"] = props
		}
		formatted = append(formatted, info)
	}
	return formatted
}
