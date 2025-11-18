package mcp_tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"gopkg.in/yaml.v3"
)

// Knowledge Base tool definitions
func (tm *ToolManager) addDocumentTool() *protocol.Tool {
	tool, err := protocol.NewTool("kb_add_document", `Add a document to the knowledge base with automatic embedding.

Explanation: Embeds the document content and stores it together with file path and metadata for semantic document search.

When to call: Use when onboarding reference documents, manuals, or files you want to query semantically.

Example arguments/values:
	file_path: "guide.md"
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
	file_path: "guide.md"
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
	file_path: "guide.md"
`, DeleteDocumentInput{})
	if err != nil {
		slog.Error("failed to create tool", "name", "kb_delete_document", "err", err)
		return nil
	}
	return tool
}

// Knowledge Base tool handlers
// saveMarkdownFile saves a document as a markdown file in the knowledge base directory
func (tm *ToolManager) saveMarkdownFile(filePath, content string) error {
	if tm.knowledgeBasePath == "" {
		// Knowledge base path not configured, skip filesystem storage
		return nil
	}

	// Ensure the file has .md extension
	if !strings.HasSuffix(filePath, ".md") {
		filePath = filePath + ".md"
	}

	// Create the full path
	fullPath := filepath.Join(tm.knowledgeBasePath, filePath)

	// Create directories if they don't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write the content to file
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	return nil
}

// removeMarkdownFile removes a markdown file from the knowledge base directory
func (tm *ToolManager) removeMarkdownFile(filePath string) error {
	if tm.knowledgeBasePath == "" {
		// Knowledge base path not configured, skip filesystem operation
		return nil
	}

	// Ensure the file has .md extension
	if !strings.HasSuffix(filePath, ".md") {
		filePath = filePath + ".md"
	}

	// Create the full path
	fullPath := filepath.Join(tm.knowledgeBasePath, filePath)

	// Log the attempted removal
	slog.Info("Attempting to remove markdown file", "file_path", filePath, "full_path", fullPath)

	// Remove the file if it exists
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove file %s: %w", fullPath, err)
	}

	slog.Info("Successfully removed markdown file", "file_path", filePath, "full_path", fullPath)
	return nil
}

// readMarkdownFile reads a markdown file from the knowledge base directory
func (tm *ToolManager) readMarkdownFile(filePath string) (string, error) {
	if tm.knowledgeBasePath == "" {
		// Knowledge base path not configured, return empty content
		return "", nil
	}

	// Ensure the file has .md extension
	if !strings.HasSuffix(filePath, ".md") {
		filePath = filePath + ".md"
	}

	// Create the full path
	fullPath := filepath.Join(tm.knowledgeBasePath, filePath)

	// Read the file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // File doesn't exist, return empty content
		}
		return "", fmt.Errorf("failed to read file %s: %w", fullPath, err)
	}

	return string(content), nil
}

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

	// Save to database
	err = tm.storage.SaveDocument(ctx, input.FilePath, input.Content, embedding, input.Metadata.AsMap())
	if err != nil {
		return nil, fmt.Errorf("failed to add document to database: %w", err)
	}

	// Save to filesystem as markdown file (if knowledge base path is configured)
	if err := tm.saveMarkdownFile(input.FilePath, input.Content); err != nil {
		slog.Warn("failed to save document to filesystem", "file_path", input.FilePath, "error", err)
		// Don't fail the operation if filesystem save fails, but log it
	}

	var message string
	if tm.knowledgeBasePath != "" {
		message = fmt.Sprintf("Successfully added document '%s' to knowledge base (database and filesystem)", input.FilePath)
	} else {
		message = fmt.Sprintf("Successfully added document '%s' to knowledge base (database only)", input.FilePath)
	}

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: message,
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

	// Omit embeddings from the response
	for _, result := range results {
		if result.Document != nil {
			result.Document.Embedding = nil
		}
	}

	resultsBytes, _ := yaml.Marshal(results)

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

	// First try to get from database
	document, err := tm.storage.GetDocument(ctx, input.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	if document != nil {
		// Found in database, return it
		// Don't include embedding in response (too large)
		doc := *document
		doc.Embedding = nil
		docBytes, _ := json.MarshalIndent(doc, "", "  ")

		return protocol.NewCallToolResult([]protocol.Content{
			&protocol.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Document '%s' (from database):\n%s", input.FilePath, string(docBytes)),
			},
		}, false), nil
	}

	// Not found in database, try to read from filesystem
	if tm.knowledgeBasePath != "" {
		content, err := tm.readMarkdownFile(input.FilePath)
		if err != nil {
			slog.Warn("failed to read document from filesystem", "file_path", input.FilePath, "error", err)
		} else if content != "" {
			// Found in filesystem, return it as a simple content response
			return protocol.NewCallToolResult([]protocol.Content{
				&protocol.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Document '%s' (from filesystem):\n%s", input.FilePath, content),
				},
			}, false), nil
		}
	}

	// Not found in either location
	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: fmt.Sprintf("No document found at path '%s' in database or filesystem", input.FilePath),
		},
	}, false), nil
}

func (tm *ToolManager) deleteDocumentHandler(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	var input DeleteDocumentInput
	if err := json.Unmarshal(request.RawArguments, &input); err != nil {
		return nil, fmt.Errorf(errParseArgs, err)
	}

	slog.Info("Processing delete document request", "file_path", input.FilePath)

	// Delete from database
	err := tm.storage.DeleteDocument(ctx, input.FilePath)
	if err != nil {
		slog.Error("Failed to delete document from database", "file_path", input.FilePath, "error", err)
		return nil, fmt.Errorf("failed to delete document from database: %w", err)
	}

	slog.Info("Successfully deleted document from database", "file_path", input.FilePath)

	// Remove from filesystem (if knowledge base path is configured)
	if err := tm.removeMarkdownFile(input.FilePath); err != nil {
		slog.Warn("failed to remove document from filesystem", "file_path", input.FilePath, "error", err)
		// Don't fail the operation if filesystem removal fails, but log it
	}

	var message string
	if tm.knowledgeBasePath != "" {
		message = fmt.Sprintf("Successfully deleted document '%s' from knowledge base (database and filesystem)", input.FilePath)
	} else {
		message = fmt.Sprintf("Successfully deleted document '%s' from knowledge base (database only)", input.FilePath)
	}

	slog.Info("Completed delete document request", "file_path", input.FilePath, "message", message)

	return protocol.NewCallToolResult([]protocol.Content{
		&protocol.TextContent{
			Type: "text",
			Text: message,
		},
	}, false), nil
}
