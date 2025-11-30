// Package mcp_tools provides help/documentation tools.
package mcp_tools

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

//go:embed docs/*.txt docs/tools/*.txt
var docsFS embed.FS

// HowToUseInput represents input for how_to_use tool
type HowToUseInput struct {
	Topic string `json:"topic,omitempty" description:"Optional topic: 'memory', 'kb', 'code', or a specific tool name. If omitted, returns overview."`
}

// howToUseTool creates the how_to_use tool definition
func (tm *ToolManager) howToUseTool() *protocol.Tool {
	tool, err := protocol.NewTool("how_to_use", `Get help on remembrances-mcp tools. Call with no args for overview, or specify a topic.`, HowToUseInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "how_to_use", "err", err)
		return nil
	}
	return tool
}

// howToUseHandler handles the how_to_use tool calls
func (tm *ToolManager) howToUseHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input HowToUseInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf(errParseArgs, err)
	}

	topic := strings.TrimSpace(strings.ToLower(input.Topic))

	var content string
	var err error

	switch topic {
	case "", "overview":
		content, err = readDocFile("docs/overview.txt")
	case "memory", "remembrance":
		content, err = readDocFile("docs/memory_group.txt")
	case "kb", "knowledge", "knowledge_base", "documents":
		content, err = readDocFile("docs/kb_group.txt")
	case "code", "indexing", "code_indexing":
		content, err = readDocFile("docs/code_group.txt")
	default:
		// Try to find a specific tool documentation
		content, err = readDocFile(fmt.Sprintf("docs/tools/%s.txt", topic))
		if err != nil {
			// Tool not found, provide helpful error
			content = fmt.Sprintf(`Unknown topic: "%s"

Available topics:
  - "memory" or "remembrance": Memory tools (facts, vectors, graph)
  - "kb" or "knowledge_base": Knowledge base tools
  - "code" or "indexing": Code indexing and search tools
  - Or specify a tool name like "remembrance_save_fact", "kb_add_document", etc.

Call how_to_use() with no arguments for a complete overview.`, topic)
			err = nil
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to read documentation: %w", err)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: content,
		},
	}, false), nil
}

// readDocFile reads a file from the embedded docs filesystem
func readDocFile(path string) (string, error) {
	data, err := docsFS.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
