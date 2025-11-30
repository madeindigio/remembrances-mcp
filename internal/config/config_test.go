package config

import "testing"

func TestCodeEmbedderGetters(t *testing.T) {
	// Test with no code-specific configuration (should fallback to defaults)
	cfg := &Config{
		GGUFModelPath: "/path/to/default.gguf",
		OllamaModel:   "nomic-embed-text",
		OpenAIModel:   "text-embedding-3-large",
	}

	if got := cfg.GetCodeGGUFModelPath(); got != "/path/to/default.gguf" {
		t.Errorf("GetCodeGGUFModelPath() = %q, want %q", got, "/path/to/default.gguf")
	}
	if got := cfg.GetCodeOllamaModel(); got != "nomic-embed-text" {
		t.Errorf("GetCodeOllamaModel() = %q, want %q", got, "nomic-embed-text")
	}
	if got := cfg.GetCodeOpenAIModel(); got != "text-embedding-3-large" {
		t.Errorf("GetCodeOpenAIModel() = %q, want %q", got, "text-embedding-3-large")
	}
	if cfg.HasCodeSpecificEmbedder() {
		t.Error("HasCodeSpecificEmbedder() = true, want false")
	}
}

func TestCodeEmbedderGettersWithOverrides(t *testing.T) {
	// Test with code-specific configuration overrides
	cfg := &Config{
		GGUFModelPath:     "/path/to/default.gguf",
		OllamaModel:       "nomic-embed-text",
		OpenAIModel:       "text-embedding-3-large",
		CodeGGUFModelPath: "/path/to/coderankembed.gguf",
		CodeOllamaModel:   "jina/jina-embeddings-v2-base-code",
		CodeOpenAIModel:   "text-embedding-3-small",
	}

	if got := cfg.GetCodeGGUFModelPath(); got != "/path/to/coderankembed.gguf" {
		t.Errorf("GetCodeGGUFModelPath() = %q, want %q", got, "/path/to/coderankembed.gguf")
	}
	if got := cfg.GetCodeOllamaModel(); got != "jina/jina-embeddings-v2-base-code" {
		t.Errorf("GetCodeOllamaModel() = %q, want %q", got, "jina/jina-embeddings-v2-base-code")
	}
	if got := cfg.GetCodeOpenAIModel(); got != "text-embedding-3-small" {
		t.Errorf("GetCodeOpenAIModel() = %q, want %q", got, "text-embedding-3-small")
	}
	if !cfg.HasCodeSpecificEmbedder() {
		t.Error("HasCodeSpecificEmbedder() = false, want true")
	}
}

func TestCodeEmbedderGettersPartialOverride(t *testing.T) {
	// Test with only some code-specific configuration
	cfg := &Config{
		GGUFModelPath:   "/path/to/default.gguf",
		OllamaModel:     "nomic-embed-text",
		OpenAIModel:     "text-embedding-3-large",
		CodeOllamaModel: "jina/jina-embeddings-v2-base-code",
	}

	// GGUF should fallback to default
	if got := cfg.GetCodeGGUFModelPath(); got != "/path/to/default.gguf" {
		t.Errorf("GetCodeGGUFModelPath() = %q, want %q", got, "/path/to/default.gguf")
	}
	// Ollama should use override
	if got := cfg.GetCodeOllamaModel(); got != "jina/jina-embeddings-v2-base-code" {
		t.Errorf("GetCodeOllamaModel() = %q, want %q", got, "jina/jina-embeddings-v2-base-code")
	}
	// OpenAI should fallback to default
	if got := cfg.GetCodeOpenAIModel(); got != "text-embedding-3-large" {
		t.Errorf("GetCodeOpenAIModel() = %q, want %q", got, "text-embedding-3-large")
	}
	// Should still be considered as having a code-specific embedder
	if !cfg.HasCodeSpecificEmbedder() {
		t.Error("HasCodeSpecificEmbedder() = false, want true")
	}
}
