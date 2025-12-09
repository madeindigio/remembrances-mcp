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
	tool, err := protocol.NewTool("save_fact", `Store a key-value fact for a user. Use how_to_use("save_fact") for details.`, SaveFactInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "save_fact", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) getFactTool() *protocol.Tool {
	tool, err := protocol.NewTool("get_fact", `Retrieve a fact by exact key. Use how_to_use("get_fact") for details.`, GetFactInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "get_fact", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) listFactsTool() *protocol.Tool {
	tool, err := protocol.NewTool("list_facts", `List all facts for a user. Use how_to_use("list_facts") for details.`, ListFactsInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "list_facts", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) deleteFactTool() *protocol.Tool {
	tool, err := protocol.NewTool("delete_fact", `Delete a fact by key. Use how_to_use("delete_fact") for details.`, DeleteFactInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "delete_fact", "err", err)
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
		suggestions := tm.FindKeyAlternatives(ctx, input.UserID, "kv_memories", input.Key)
		payload := CreateEmptyResultTOON(
			fmt.Sprintf("No fact found for key '%s' and user '%s'", input.Key, input.UserID),
			suggestions,
		)
		return protocol.NewCallToolResult([]protocol.Content{
			&protocol.TextContent{Type: "text", Text: payload},
		}, false), nil
	}

	response := map[string]interface{}{
		"user_id": input.UserID,
		"key":     input.Key,
		"value":   value,
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{Type: "text", Text: MarshalTOON(response)},
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

	if len(facts) == 0 {
		suggestions := tm.FindUserAlternatives(ctx, "kv_memories", input.UserID)
		payload := CreateEmptyResultTOON(
			fmt.Sprintf("No facts found for user '%s'", input.UserID),
			suggestions,
		)
		return protocol.NewCallToolResult([]protocol.Content{
			&protocol.TextContent{Type: "text", Text: payload},
		}, false), nil
	}

	response := map[string]interface{}{
		"user_id": input.UserID,
		"count":   len(facts),
		"facts":   facts,
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{Type: "text", Text: MarshalTOON(response)},
	}, false), nil
}

func (tm *ToolManager) deleteFactHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input DeleteFactInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf(errParseArgs, err)
	}

	// Check existence to provide suggestions when missing
	current, err := tm.storage.GetFact(ctx, input.UserID, input.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to check fact before delete: %w", err)
	}

	if current == nil {
		suggestions := tm.FindKeyAlternatives(ctx, input.UserID, "kv_memories", input.Key)
		payload := CreateEmptyResultTOON(
			fmt.Sprintf("No fact found for key '%s' and user '%s'", input.Key, input.UserID),
			suggestions,
		)
		return protocol.NewCallToolResult([]protocol.Content{
			&protocol.TextContent{Type: "text", Text: payload},
		}, false), nil
	}

	err = tm.storage.DeleteFact(ctx, input.UserID, input.Key)
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
