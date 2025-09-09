package mcp_tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// Knowledge Base tool definitions
func (tm *ToolManager) addDocumentTool() *protocol.Tool {
	tool, err := protocol.NewTool("kb_add_document", `Add a document to the knowledge base with automatic embedding.

Explanation: Embeds the document content and stores it together with file path and metadata for semantic document search.

When to call: Use when onboarding reference documents, manuals, or files you want to query semantically.

Example arguments/values:
	file_path: "/kb/guide.pdf"
	content: "Full text of the document..."
	metadata: { source: "import" }
`, AddDocumentInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "kb_add_document", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) searchDocumentsTool() *protocol.Tool {
	tool, err := protocol.NewTool("kb_search_documents", `Search knowledge-base documents by semantic similarity.

Explanation: Embeds the query and returns matching documents ranked by semantic relevance.

When to call: Use to find relevant reference documents or passages given a question or topic.

Example arguments/values:
	query: "how to configure authentication"
	limit: 5
`, SearchDocumentsInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "kb_search_documents", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) getDocumentTool() *protocol.Tool {
	tool, err := protocol.NewTool("kb_get_document", `Retrieve a stored document by file path.

Explanation: Returns the document metadata and content (embedding omitted in responses).

When to call: Use when you know the exact document path and need its contents or metadata.

Example arguments/values:
	file_path: "/kb/guide.pdf"
`, GetDocumentInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "kb_get_document", "err", err)
		return nil
	}
	return tool
}

func (tm *ToolManager) deleteDocumentTool() *protocol.Tool {
	tool, err := protocol.NewTool("kb_delete_document", `Delete a document from the knowledge base by file path.

Explanation: Removes the stored document and its embedding.

When to call: Use to remove outdated or sensitive documents.

Example arguments/values:
	file_path: "/kb/guide.pdf"
`, DeleteDocumentInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "kb_delete_document", "err", err)
		return nil
	}
	return tool
}

// Knowledge Base tool handlers
func (tm *ToolManager) addDocumentHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input AddDocumentInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf(errParseArgs, err)
	}

	// Generate embedding for the document content
	embedding, err := tm.embedder.EmbedQuery(ctx, input.Content)
	if err != nil {
		return nil, fmt.Errorf(errGenEmbedding, err)
	}

	err = tm.storage.SaveDocument(ctx, input.FilePath, input.Content, embedding, stringMapToInterfaceMap(input.Metadata))
	if err != nil {
		return nil, fmt.Errorf("failed to add document: %w", err)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Successfully added document '%s' to knowledge base", input.FilePath),
		},
	}, false), nil
}

func (tm *ToolManager) searchDocumentsHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input SearchDocumentsInput
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

	results, err := tm.storage.SearchDocuments(ctx, queryEmbedding, input.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}

	resultsBytes, _ := json.MarshalIndent(results, "", "  ")

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Found %d documents for query '%s':\n%s", len(results), input.Query, string(resultsBytes)),
		},
	}, false), nil
}

func (tm *ToolManager) getDocumentHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input GetDocumentInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf(errParseArgs, err)
	}

	document, err := tm.storage.GetDocument(ctx, input.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	if document == nil {
		return protocol.NewCallToolResult([]protocol.Content{
			&protocol.TextContent{
				Type: "text",
				Text: fmt.Sprintf("No document found at path '%s'", input.FilePath),
			},
		}, false), nil
	}

	// Don't include embedding in response (too large)
	doc := *document
	doc.Embedding = nil
	docBytes, _ := json.MarshalIndent(doc, "", "  ")

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Document '%s':\n%s", input.FilePath, string(docBytes)),
		},
	}, false), nil
}

func (tm *ToolManager) deleteDocumentHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input DeleteDocumentInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf(errParseArgs, err)
	}

	err := tm.storage.DeleteDocument(ctx, input.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to delete document: %w", err)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Successfully deleted document '%s'", input.FilePath),
		},
	}, false), nil
}
