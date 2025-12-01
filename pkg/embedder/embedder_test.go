package embedder

import (
	"testing"
)

// MockConfig implements MainConfig interface for testing
type MockConfig struct {
	ollamaURL     string
	ollamaModel   string
	openaiKey     string
	openaiURL     string
	openaiModel   string
	ggufModelPath string
	ggufThreads   int
	ggufGPULayers int
}

func (m *MockConfig) GetOllamaURL() string     { return m.ollamaURL }
func (m *MockConfig) GetOllamaModel() string   { return m.ollamaModel }
func (m *MockConfig) GetOpenAIKey() string     { return m.openaiKey }
func (m *MockConfig) GetOpenAIURL() string     { return m.openaiURL }
func (m *MockConfig) GetOpenAIModel() string   { return m.openaiModel }
func (m *MockConfig) GetGGUFModelPath() string { return m.ggufModelPath }
func (m *MockConfig) GetGGUFThreads() int      { return m.ggufThreads }
func (m *MockConfig) GetGGUFGPULayers() int    { return m.ggufGPULayers }

func TestNewEmbedderFromConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		expectType  string
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
		},
		{
			name: "ollama config",
			config: &Config{
				OllamaURL:   "http://localhost:11434",
				OllamaModel: "nomic-embed-text",
			},
			expectError: false,
			expectType:  "ollama",
		},
		{
			name: "openai config",
			config: &Config{
				OpenAIKey:   "test-api-key",
				OpenAIModel: "text-embedding-3-large",
			},
			expectError: false,
			expectType:  "openai",
		},
		{
			name: "ollama priority over openai",
			config: &Config{
				OllamaURL:   "http://localhost:11434",
				OllamaModel: "nomic-embed-text",
				OpenAIKey:   "test-api-key",
				OpenAIModel: "text-embedding-3-large",
			},
			expectError: false,
			expectType:  "ollama",
		},
		{
			name: "ollama URL without model",
			config: &Config{
				OllamaURL: "http://localhost:11434",
			},
			expectError: true,
		},
		{
			name:        "no configuration",
			config:      &Config{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			embedder, err := NewEmbedderFromConfig(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if embedder == nil {
				t.Errorf("expected embedder but got nil")
				return
			}

			// Verify embedder type
			switch tt.expectType {
			case "ollama":
				if _, ok := embedder.(*OllamaEmbedder); !ok {
					t.Errorf("expected OllamaEmbedder, got %T", embedder)
				}
			case "openai":
				if _, ok := embedder.(*OpenAIEmbedder); !ok {
					t.Errorf("expected OpenAIEmbedder, got %T", embedder)
				}
			}

			// Test dimension method
			if dim := embedder.Dimension(); dim <= 0 {
				t.Errorf("expected positive dimension, got %d", dim)
			}
		})
	}
}

func TestNewEmbedderFromMainConfig(t *testing.T) {
	tests := []struct {
		name        string
		mainConfig  MainConfig
		expectError bool
		expectType  string
	}{
		{
			name:        "nil main config",
			mainConfig:  nil,
			expectError: true,
		},
		{
			name: "ollama main config",
			mainConfig: &MockConfig{
				ollamaURL:   "http://localhost:11434",
				ollamaModel: "nomic-embed-text",
			},
			expectError: false,
			expectType:  "ollama",
		},
		{
			name: "openai main config",
			mainConfig: &MockConfig{
				openaiKey:   "test-api-key",
				openaiModel: "text-embedding-3-large",
			},
			expectError: false,
			expectType:  "openai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			embedder, err := NewEmbedderFromMainConfig(tt.mainConfig)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if embedder == nil {
				t.Errorf("expected embedder but got nil")
				return
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
		},
		{
			name: "valid ollama config",
			config: &Config{
				OllamaURL:   "http://localhost:11434",
				OllamaModel: "nomic-embed-text",
			},
			expectError: false,
		},
		{
			name: "valid openai config",
			config: &Config{
				OpenAIKey:   "test-api-key",
				OpenAIModel: "text-embedding-3-large",
			},
			expectError: false,
		},
		{
			name: "invalid ollama URL",
			config: &Config{
				OllamaURL:   "invalid-url",
				OllamaModel: "nomic-embed-text",
			},
			expectError: true,
		},
		{
			name: "ollama URL without model",
			config: &Config{
				OllamaURL: "http://localhost:11434",
			},
			expectError: true,
		},
		{
			name: "empty openai key",
			config: &Config{
				OpenAIKey: "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGetEmbedderType(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected string
	}{
		{
			name:     "nil config",
			config:   nil,
			expected: "none",
		},
		{
			name: "ollama config",
			config: &Config{
				OllamaURL: "http://localhost:11434",
			},
			expected: "ollama",
		},
		{
			name: "openai config",
			config: &Config{
				OpenAIKey: "test-api-key",
			},
			expected: "openai",
		},
		{
			name: "ollama priority",
			config: &Config{
				OllamaURL: "http://localhost:11434",
				OpenAIKey: "test-api-key",
			},
			expected: "ollama",
		},
		{
			name:     "empty config",
			config:   &Config{},
			expected: "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetEmbedderType(tt.config)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestEmbedderDimensions(t *testing.T) {
	tests := []struct {
		model    string
		expected int
	}{
		{"nomic-embed-text", 768},
		{"mxbai-embed-large", 1024},
		{"all-minilm", 384},
		{"unknown-model", 768}, // default
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			dim := getDimensionForModel(tt.model)
			if dim != tt.expected {
				t.Errorf("expected dimension %d for model %s, got %d", tt.expected, tt.model, dim)
			}
		})
	}
}

func TestOpenAIEmbedderDimensions(t *testing.T) {
	tests := []struct {
		model    string
		expected int
	}{
		{"text-embedding-3-large", 3072},
		{"text-embedding-3-small", 1536},
		{"text-embedding-ada-002", 1536},
		{"unknown-model", 1536}, // default
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			dim := getDimensionForOpenAIModel(tt.model)
			if dim != tt.expected {
				t.Errorf("expected dimension %d for model %s, got %d", tt.expected, tt.model, dim)
			}
		})
	}
}

// TestEmbedderInterface ensures our implementations satisfy the interface
func TestEmbedderInterface(t *testing.T) {
	var _ Embedder = (*OllamaEmbedder)(nil)
	var _ Embedder = (*OpenAIEmbedder)(nil)
}

// Benchmark test to check performance if needed
func BenchmarkNewEmbedderFromConfig(b *testing.B) {
	cfg := &Config{
		OllamaURL:   "http://localhost:11434",
		OllamaModel: "nomic-embed-text",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewEmbedderFromConfig(cfg)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}
