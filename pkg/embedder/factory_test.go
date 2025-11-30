package embedder

import "testing"

// mockMainConfig implements MainConfig for testing
type mockMainConfig struct {
	ggufPath   string
	ggufThread int
	ggufGPU    int
	ollamaURL  string
	ollamaM    string
	openaiK    string
	openaiU    string
	openaiM    string
}

func (m *mockMainConfig) GetGGUFModelPath() string { return m.ggufPath }
func (m *mockMainConfig) GetGGUFThreads() int      { return m.ggufThread }
func (m *mockMainConfig) GetGGUFGPULayers() int    { return m.ggufGPU }
func (m *mockMainConfig) GetOllamaURL() string     { return m.ollamaURL }
func (m *mockMainConfig) GetOllamaModel() string   { return m.ollamaM }
func (m *mockMainConfig) GetOpenAIKey() string     { return m.openaiK }
func (m *mockMainConfig) GetOpenAIURL() string     { return m.openaiU }
func (m *mockMainConfig) GetOpenAIModel() string   { return m.openaiM }

// mockCodeMainConfig extends mockMainConfig to implement CodeMainConfig
type mockCodeMainConfig struct {
	mockMainConfig
	codeGGUF   string
	codeOllama string
	codeOpenai string
}

func (m *mockCodeMainConfig) GetCodeGGUFModelPath() string { return m.codeGGUF }
func (m *mockCodeMainConfig) GetCodeOllamaModel() string   { return m.codeOllama }
func (m *mockCodeMainConfig) GetCodeOpenAIModel() string   { return m.codeOpenai }
func (m *mockCodeMainConfig) HasCodeSpecificEmbedder() bool {
	return m.codeGGUF != "" || m.codeOllama != "" || m.codeOpenai != ""
}

func TestNewCodeEmbedderFromMainConfig_NoCodeSpecific(t *testing.T) {
	// When no code-specific config is set, should return nil (use default)
	cfg := &mockCodeMainConfig{
		mockMainConfig: mockMainConfig{
			ggufPath:  "/path/to/default.gguf",
			ollamaURL: "http://localhost:11434",
			ollamaM:   "nomic-embed-text",
		},
		codeGGUF:   "",
		codeOllama: "",
		codeOpenai: "",
	}

	embedder, err := NewCodeEmbedderFromMainConfig(cfg)
	if err != nil {
		t.Fatalf("NewCodeEmbedderFromMainConfig() error = %v, want nil", err)
	}
	if embedder != nil {
		t.Error("NewCodeEmbedderFromMainConfig() returned non-nil embedder, want nil when no code-specific config")
	}
}

func TestNewCodeEmbedderFromMainConfig_NilConfig(t *testing.T) {
	// Should return error for nil config
	_, err := NewCodeEmbedderFromMainConfig(nil)
	if err == nil {
		t.Error("NewCodeEmbedderFromMainConfig(nil) error = nil, want error")
	}
}

func TestGetEmbedderType_Extended(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want string
	}{
		{
			name: "nil config",
			cfg:  nil,
			want: "none",
		},
		{
			name: "gguf priority",
			cfg: &Config{
				GGUFModelPath: "/path/to/model.gguf",
				OllamaURL:     "http://localhost:11434",
				OpenAIKey:     "sk-xxx",
			},
			want: "gguf",
		},
		{
			name: "ollama fallback",
			cfg: &Config{
				OllamaURL: "http://localhost:11434",
				OpenAIKey: "sk-xxx",
			},
			want: "ollama",
		},
		{
			name: "openai fallback",
			cfg: &Config{
				OpenAIKey: "sk-xxx",
			},
			want: "openai",
		},
		{
			name: "no config",
			cfg:  &Config{},
			want: "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetEmbedderType(tt.cfg); got != tt.want {
				t.Errorf("GetEmbedderType() = %v, want %v", got, tt.want)
			}
		})
	}
}
