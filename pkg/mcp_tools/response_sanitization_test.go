package mcp_tools

import (
	"strings"
	"testing"
	"time"

	"github.com/madeindigio/remembrances-mcp/internal/storage"
)

func TestRemoveInternalRecordIDs(t *testing.T) {
	records := []map[string]interface{}{
		{
			"id":         "code_projects:abc",
			"project_id": "my-project",
		},
	}

	removeInternalRecordIDs(records)

	if _, ok := records[0]["id"]; ok {
		t.Fatalf("expected internal id to be removed")
	}
	if got := records[0]["project_id"]; got != "my-project" {
		t.Fatalf("expected project_id to be preserved, got %v", got)
	}
}

func TestSanitizeDocumentSearchResultsRemovesInternalIDFromJSON(t *testing.T) {
	userID := "user-1"
	results := []storage.DocumentResult{
		{
			Document: &storage.Document{
				ID:        "knowledge_base:abc",
				UserID:    &userID,
				FilePath:  "docs/readme.md",
				Content:   "content",
				Embedding: []float32{1, 2, 3},
				Metadata:  map[string]interface{}{"source": "test"},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Similarity: 0.99,
		},
	}

	sanitizeDocumentSearchResults(results)
	jsonText := MarshalTOON(map[string]interface{}{"results": results})

	if strings.Contains(jsonText, "\"id\":") {
		t.Fatalf("expected internal document id to be omitted from output: %s", jsonText)
	}
}
