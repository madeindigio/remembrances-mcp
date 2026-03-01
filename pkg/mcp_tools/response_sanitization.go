package mcp_tools

import "github.com/madeindigio/remembrances-mcp/internal/storage"

func removeInternalRecordIDs(records []map[string]interface{}) {
	for _, record := range records {
		delete(record, "id")
	}
}

func sanitizeDocumentSearchResults(results []storage.DocumentResult) {
	for i := range results {
		if results[i].Document != nil {
			results[i].Document.ID = ""
			results[i].Document.Embedding = nil
		}
	}
}
