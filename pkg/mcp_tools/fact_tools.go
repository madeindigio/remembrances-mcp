package mcp_tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// Fact tool definitions
func (tm *ToolManager) saveFactTool() *protocol.Tool {
	tool, err := protocol.NewTool("remembrance_save_fact", `Save a key-value fact for a user.

Explanation: Stores a simple key -> value pair scoped to a user. Values can be strings, numbers, or objects. This is optimized for fast exact-key lookup, not semantic search.

When to call: Use when you need to persist small, structured facts or preferences (e.g. contact info, settings, short user preferences) that will be retrieved by exact key later.

Note: If you are unsure which user_id to use, you may use the current project name as the user_id.

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

Note: If you are unsure which user_id to use, you may use the current project name as the user_id.

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

Note: If you are unsure which user_id to use, you may use the current project name as the user_id.

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

Note: If you are unsure which user_id to use, you may use the current project name as the user_id.

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

// Fact tool handlers
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
