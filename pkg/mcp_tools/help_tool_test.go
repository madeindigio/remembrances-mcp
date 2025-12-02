package mcp_tools

import (
	"strings"
	"testing"
)

// TestReadDocFileOverview tests reading the overview documentation
func TestReadDocFileOverview(t *testing.T) {
	content, err := readDocFile("docs/overview.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check overview content markers
	if !strings.Contains(content, "REMEMBRANCES-MCP TOOL OVERVIEW") {
		t.Error("overview should contain main title")
	}
	if !strings.Contains(content, "MEMORY") {
		t.Error("overview should mention MEMORY category")
	}
	if !strings.Contains(content, "KNOWLEDGE BASE") {
		t.Error("overview should mention KNOWLEDGE BASE category")
	}
	if !strings.Contains(content, "CODE") {
		t.Error("overview should mention CODE category")
	}
}

// TestReadDocFileMemoryGroup tests reading memory group documentation
func TestReadDocFileMemoryGroup(t *testing.T) {
	content, err := readDocFile("docs/memory_group.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(content, "MEMORY TOOLS GROUP") {
		t.Error("memory group should contain group title")
	}
	if !strings.Contains(content, "save_fact") {
		t.Error("memory group should mention fact tools")
	}
}

// TestReadDocFileKBGroup tests reading kb group documentation
func TestReadDocFileKBGroup(t *testing.T) {
	content, err := readDocFile("docs/kb_group.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(content, "KNOWLEDGE BASE TOOLS GROUP") {
		t.Error("kb group should contain group title")
	}
}

// TestReadDocFileCodeGroup tests reading code group documentation
func TestReadDocFileCodeGroup(t *testing.T) {
	content, err := readDocFile("docs/code_group.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(content, "CODE TOOLS GROUP") {
		t.Error("code group should contain group title")
	}
}

// TestReadDocFileSpecificTools tests reading individual tool documentation
func TestReadDocFileSpecificTools(t *testing.T) {
	testCases := []struct {
		filename    string
		shouldMatch string
	}{
		{"docs/tools/save_fact.txt", "PURPOSE"},
		{"docs/tools/kb_add_document.txt", "PURPOSE"},
		{"docs/tools/index_repository.txt", "PURPOSE"},
		{"docs/tools/hybrid_search.txt", "PURPOSE"},
		{"docs/tools/get_fact.txt", "PURPOSE"},
		{"docs/tools/search_vectors.txt", "PURPOSE"},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			content, err := readDocFile(tc.filename)
			if err != nil {
				t.Fatalf("failed to read %s: %v", tc.filename, err)
			}

			if !strings.Contains(content, tc.shouldMatch) {
				t.Errorf("file %s should contain %s", tc.filename, tc.shouldMatch)
			}
		})
	}
}

// TestReadDocFileNotFound tests error handling for missing files
func TestReadDocFileNotFound(t *testing.T) {
	_, err := readDocFile("docs/tools/nonexistent_tool.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

// TestDocFilesEmbedded tests that all expected doc files are embedded
func TestDocFilesEmbedded(t *testing.T) {
	expectedFiles := []string{
		"docs/overview.txt",
		"docs/memory_group.txt",
		"docs/kb_group.txt",
		"docs/code_group.txt",
		"docs/tools/save_fact.txt",
		"docs/tools/get_fact.txt",
		"docs/tools/list_facts.txt",
		"docs/tools/delete_fact.txt",
		"docs/tools/add_vector.txt",
		"docs/tools/search_vectors.txt",
		"docs/tools/update_vector.txt",
		"docs/tools/delete_vector.txt",
		"docs/tools/create_entity.txt",
		"docs/tools/get_entity.txt",
		"docs/tools/create_relationship.txt",
		"docs/tools/traverse_graph.txt",
		"docs/tools/kb_add_document.txt",
		"docs/tools/kb_get_document.txt",
		"docs/tools/kb_search_documents.txt",
		"docs/tools/kb_delete_document.txt",
		"docs/tools/to_remember.txt",
		"docs/tools/last_to_remember.txt",
		"docs/tools/get_stats.txt",
		"docs/tools/hybrid_search.txt",
		"docs/tools/index_repository.txt",
		"docs/tools/index_directory.txt",
		"docs/tools/list_indexed_files.txt",
		"docs/tools/get_indexing_status.txt",
		"docs/tools/reindex_file.txt",
		"docs/tools/clear_index.txt",
		"docs/tools/get_index_stats.txt",
		"docs/tools/search_code.txt",
		"docs/tools/search_symbols.txt",
		"docs/tools/get_file_summary.txt",
		"docs/tools/get_symbol_context.txt",
		"docs/tools/find_references.txt",
		"docs/tools/find_implementations.txt",
		"docs/tools/extract_code_block.txt",
		"docs/tools/get_function_signature.txt",
		"docs/tools/list_imports.txt",
		"docs/tools/find_similar_code.txt",
	}

	for _, file := range expectedFiles {
		t.Run(file, func(t *testing.T) {
			content, err := docsFS.ReadFile(file)
			if err != nil {
				t.Fatalf("failed to read embedded file %s: %v", file, err)
			}
			if len(content) == 0 {
				t.Errorf("embedded file %s is empty", file)
			}
		})
	}
}

// TestHowToUseInputStruct tests the input struct definition
func TestHowToUseInputStruct(t *testing.T) {
	// Test that the struct can be created
	input := HowToUseInput{Topic: "memory"}
	if input.Topic != "memory" {
		t.Error("input struct should hold topic value")
	}

	// Test empty topic
	emptyInput := HowToUseInput{}
	if emptyInput.Topic != "" {
		t.Error("empty input should have empty topic")
	}
}
