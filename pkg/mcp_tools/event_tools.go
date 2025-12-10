package mcp_tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/madeindigio/remembrances-mcp/internal/storage"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// Event tool definitions

func (tm *ToolManager) saveEventTool() *protocol.Tool {
	tool, err := protocol.NewTool("save_event", `Store a temporal event with semantic search. Use how_to_use("save_event") for details.`, SaveEventInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "save_event", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) searchEventsTool() *protocol.Tool {
	tool, err := protocol.NewTool("search_events", `Search events with hybrid text+vector search and time filters. Use how_to_use("search_events") for details.`, SearchEventsInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "search_events", "err", err)
		return nil
	}
	return tool
}

// Event tool handlers

func (tm *ToolManager) saveEventHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input SaveEventInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf(errParseArgs, err)
	}

	// Generate embedding for content
	embedding, err := tm.embedder.EmbedDocuments(ctx, []string{input.Content})
	if err != nil {
		return nil, fmt.Errorf(errGenEmbedding, err)
	}
	if len(embedding) == 0 {
		return nil, fmt.Errorf("failed to generate embedding: empty result")
	}

	// Save event
	eventID, createdAt, err := tm.storage.SaveEvent(ctx, input.UserID, input.Subject, input.Content, embedding[0], input.Metadata.AsMap())
	if err != nil {
		return nil, fmt.Errorf("failed to save event: %w", err)
	}

	result := map[string]interface{}{
		"id":         eventID,
		"user_id":    input.UserID,
		"subject":    input.Subject,
		"created_at": createdAt.Format(time.RFC3339),
		"status":     "saved",
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{Type: "text", Text: MarshalTOON(result)},
	}, false), nil
}

func (tm *ToolManager) searchEventsHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input SearchEventsInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf(errParseArgs, err)
	}

	// Build search params
	params := storage.EventSearchParams{
		UserID:  input.UserID,
		Subject: input.Subject,
		Query:   input.Query,
		Limit:   input.Limit,
	}

	// Parse date filters - normalize to UTC and truncate to seconds
	if input.FromDate != "" {
		t, err := time.Parse(time.RFC3339, input.FromDate)
		if err != nil {
			slog.Warn("invalid from_date format, ignoring", "from_date", input.FromDate, "err", err)
		} else {
			t = t.UTC().Truncate(time.Second)
			params.FromDate = &t
		}
	}
	if input.ToDate != "" {
		t, err := time.Parse(time.RFC3339, input.ToDate)
		if err != nil {
			slog.Warn("invalid to_date format, ignoring", "to_date", input.ToDate, "err", err)
		} else {
			t = t.UTC().Truncate(time.Second)
			params.ToDate = &t
		}
	}

	// Relative time filters
	if input.LastHours > 0 {
		params.LastHours = &input.LastHours
	}
	if input.LastDays > 0 {
		params.LastDays = &input.LastDays
	}
	if input.LastMonths > 0 {
		params.LastMonths = &input.LastMonths
	}

	// Generate query embedding if query is provided
	if input.Query != "" {
		embedding, err := tm.embedder.EmbedQuery(ctx, input.Query)
		if err != nil {
			slog.Warn("failed to generate query embedding, falling back to text search", "err", err)
		} else {
			params.Embedding = embedding
		}
	}

	// Search events
	results, err := tm.storage.SearchEvents(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to search events: %w", err)
	}

	// Format results
	output := make([]map[string]interface{}, len(results))
	for i, r := range results {
		output[i] = map[string]interface{}{
			"id":         r.Event.ID,
			"user_id":    r.Event.UserID,
			"subject":    r.Event.Subject,
			"content":    r.Event.Content,
			"metadata":   r.Event.Metadata,
			"created_at": r.Event.CreatedAt.Format(time.RFC3339),
			"relevance":  r.Relevance,
		}
	}

	response := map[string]interface{}{
		"count":  len(results),
		"events": output,
	}

	if len(results) == 0 {
		suggestions := tm.FindUserAlternatives(ctx, "events", input.UserID)
		yamlText := CreateEmptyResultTOON(
			fmt.Sprintf("No events found for user '%s' with current filters", input.UserID),
			suggestions,
		)
		return protocol.NewCallToolResult([]protocol.Content{
			&protocol.TextContent{Type: "text", Text: yamlText},
		}, false), nil
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{Type: "text", Text: MarshalTOON(response)},
	}, false), nil
}
