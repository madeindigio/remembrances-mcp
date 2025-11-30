package mcp_tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// Graph tool definitions
func (tm *ToolManager) createEntityTool() *protocol.Tool {
	tool, err := protocol.NewTool("remembrance_create_entity", `Create an entity in the knowledge graph. Use how_to_use("remembrance_create_entity") for details.`, CreateEntityInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "remembrance_create_entity", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) createRelationshipTool() *protocol.Tool {
	tool, err := protocol.NewTool("remembrance_create_relationship", `Create a relationship between two entities. Use how_to_use("remembrance_create_relationship") for details.`, CreateRelationshipInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "remembrance_create_relationship", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) traverseGraphTool() *protocol.Tool {
	tool, err := protocol.NewTool("remembrance_traverse_graph", `Traverse the knowledge graph from a start entity. Use how_to_use("remembrance_traverse_graph") for details.`, TraverseGraphInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "remembrance_traverse_graph", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) getEntityTool() *protocol.Tool {
	tool, err := protocol.NewTool("remembrance_get_entity", `Get a graph entity by ID. Use how_to_use("remembrance_get_entity") for details.`, GetEntityInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "remembrance_get_entity", "err", err)
		return nil
	}
	return tool
}

// Graph tool handlers
func (tm *ToolManager) createEntityHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input CreateEntityInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf(errParseArgs, err)
	}

	err := tm.storage.CreateEntity(ctx, input.EntityType, input.Name, input.Properties.AsMap())
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

	err := tm.storage.CreateRelationship(ctx, input.FromEntity, input.ToEntity, input.RelationshipType, input.Properties.AsMap())
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
