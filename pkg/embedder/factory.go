package embedder

import (
	"fmt"
	"os"
	"strings"
)

// Config representa la configuración necesaria para crear un embedder.
type Config struct {
	// Ollama configuration
	OllamaURL   string
	OllamaModel string

	// OpenAI configuration
	OpenAIKey     string
	OpenAIBaseURL string
	OpenAIModel   string

	// GGUF configuration
	GGUFModelPath string
	GGUFThreads   int
	GGUFGPULayers int
}

// NewEmbedderFromConfig crea una instancia de Embedder basada en la configuración disponible.
// Prioridad: GGUF (local) > Ollama > OpenAI
// Retorna error si no se encuentra ninguna configuración válida.
func NewEmbedderFromConfig(cfg *Config) (Embedder, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	// Prioridad 1: GGUF (modelo local - más eficiente)
	if cfg.GGUFModelPath != "" {
		return NewGGUFEmbedder(cfg.GGUFModelPath, cfg.GGUFThreads, cfg.GGUFGPULayers)
	}

	// Prioridad 2: Ollama (si URL está disponible)
	if cfg.OllamaURL != "" {
		if cfg.OllamaModel == "" {
			return nil, fmt.Errorf("ollama URL provided but model is missing")
		}
		return NewOllamaEmbedder(cfg.OllamaURL, cfg.OllamaModel)
	}

	// Prioridad 3: OpenAI (si API key está disponible)
	if cfg.OpenAIKey != "" {
		if cfg.OpenAIModel == "" {
			// Usar modelo por defecto si no se especifica
			cfg.OpenAIModel = "text-embedding-3-large"
		}
		return NewOpenAIEmbedder(cfg.OpenAIKey, cfg.OpenAIBaseURL, cfg.OpenAIModel)
	}

	return nil, fmt.Errorf("no valid embedder configuration found: either GGUF_MODEL_PATH, OLLAMA_URL, or OPENAI_API_KEY must be provided")
}

// NewEmbedderFromEnv crea una instancia de Embedder leyendo la configuración desde variables de entorno.
// Variables de entorno soportadas:
// - GGUF_MODEL_PATH: Ruta al archivo GGUF del modelo
// - GGUF_THREADS: Número de threads a usar (opcional)
// - GGUF_GPU_LAYERS: Número de capas GPU (opcional)
// - OLLAMA_URL: URL del servidor Ollama
// - OLLAMA_EMBEDDING_MODEL: Modelo de embedding de Ollama
// - OPENAI_API_KEY: Clave API de OpenAI
// - OPENAI_API_BASE: URL base para APIs compatibles con OpenAI
// - OPENAI_EMBEDDING_MODEL: Modelo de embedding de OpenAI
func NewEmbedderFromEnv() (Embedder, error) {
	cfg := &Config{
		GGUFModelPath: getEnv("GGUF_MODEL_PATH", ""),
		GGUFThreads:   getEnvInt("GGUF_THREADS", 0),
		GGUFGPULayers: getEnvInt("GGUF_GPU_LAYERS", 0),
		OllamaURL:     getEnv("OLLAMA_URL", ""),
		OllamaModel:   getEnv("OLLAMA_EMBEDDING_MODEL", ""),
		OpenAIKey:     getEnv("OPENAI_API_KEY", ""),
		OpenAIBaseURL: getEnv("OPENAI_API_BASE", ""),
		OpenAIModel:   getEnv("OPENAI_EMBEDDING_MODEL", ""),
	}

	return NewEmbedderFromConfig(cfg)
}

// ValidateConfig valida que la configuración del embedder sea válida.
func ValidateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	hasGGUF := cfg.GGUFModelPath != ""
	hasOllama := cfg.OllamaURL != ""
	hasOpenAI := cfg.OpenAIKey != ""

	if !hasGGUF && !hasOllama && !hasOpenAI {
		return fmt.Errorf("at least one embedder must be configured (GGUF, Ollama or OpenAI)")
	}

	// Validar configuración de GGUF
	if hasGGUF {
		if _, err := os.Stat(cfg.GGUFModelPath); err != nil {
			return fmt.Errorf("GGUF model file not found: %s", cfg.GGUFModelPath)
		}
	}

	// Validar configuración de Ollama
	if hasOllama {
		if cfg.OllamaModel == "" {
			return fmt.Errorf("ollama model is required when ollama URL is provided")
		}
		if !isValidURL(cfg.OllamaURL) {
			return fmt.Errorf("invalid ollama URL: %s", cfg.OllamaURL)
		}
	}

	// Validar configuración de OpenAI
	if hasOpenAI {
		if cfg.OpenAIKey == "" {
			return fmt.Errorf("openai API key cannot be empty")
		}
		if cfg.OpenAIBaseURL != "" && !isValidURL(cfg.OpenAIBaseURL) {
			return fmt.Errorf("invalid openai base URL: %s", cfg.OpenAIBaseURL)
		}
	}

	return nil
}

// GetEmbedderType devuelve el tipo de embedder que se usaría con la configuración dada.
func GetEmbedderType(cfg *Config) string {
	if cfg == nil {
		return "none"
	}

	if cfg.GGUFModelPath != "" {
		return "gguf"
	}

	if cfg.OllamaURL != "" {
		return "ollama"
	}

	if cfg.OpenAIKey != "" {
		return "openai"
	}

	return "none"
}

// getEnv lee una variable de entorno o devuelve un valor por defecto.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt lee una variable de entorno como entero o devuelve un valor por defecto.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// isValidURL realiza una validación básica de URL.
func isValidURL(url string) bool {
	if url == "" {
		return false
	}
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

// MainConfig representa la configuración principal de la aplicación.
// Esto es para integración con el sistema de configuración existente.
type MainConfig interface {
	GetGGUFModelPath() string
	GetGGUFThreads() int
	GetGGUFGPULayers() int
	GetOllamaURL() string
	GetOllamaModel() string
	GetOpenAIKey() string
	GetOpenAIURL() string
	GetOpenAIModel() string
}

// CodeMainConfig extends MainConfig with code-specific embedding model getters.
// This interface allows specialized code embedding models (e.g., CodeRankEmbed,
// Jina-code-embeddings) to be used for code indexing while using a different
// model for text/facts/vectors/events.
type CodeMainConfig interface {
	MainConfig
	GetCodeGGUFModelPath() string
	GetCodeOllamaModel() string
	GetCodeOpenAIModel() string
	HasCodeSpecificEmbedder() bool
}

// NewEmbedderFromMainConfig crea un embedder usando la configuración principal de la aplicación.
func NewEmbedderFromMainConfig(mainCfg MainConfig) (Embedder, error) {
	if mainCfg == nil {
		return nil, fmt.Errorf("main configuration is required")
	}

	cfg := &Config{
		GGUFModelPath: mainCfg.GetGGUFModelPath(),
		GGUFThreads:   mainCfg.GetGGUFThreads(),
		GGUFGPULayers: mainCfg.GetGGUFGPULayers(),
		OllamaURL:     mainCfg.GetOllamaURL(),
		OllamaModel:   mainCfg.GetOllamaModel(),
		OpenAIKey:     mainCfg.GetOpenAIKey(),
		OpenAIBaseURL: mainCfg.GetOpenAIURL(),
		OpenAIModel:   mainCfg.GetOpenAIModel(),
	}

	return NewEmbedderFromConfig(cfg)
}

// NewCodeEmbedderFromMainConfig creates an embedder specifically for code indexing.
// If a code-specific model is configured (code-gguf-model-path, code-ollama-model,
// or code-openai-model), it will use that model. Otherwise, it returns nil,
// indicating that the default embedder should be used for code as well.
//
// Priority: GGUF > Ollama > OpenAI (same as default embedder)
//
// This function returns (nil, nil) if no code-specific configuration is found,
// which signals to the caller that the default embedder should be reused.
func NewCodeEmbedderFromMainConfig(mainCfg CodeMainConfig) (Embedder, error) {
	if mainCfg == nil {
		return nil, fmt.Errorf("main configuration is required")
	}

	// If no code-specific embedder is configured, return nil (use default)
	if !mainCfg.HasCodeSpecificEmbedder() {
		return nil, nil
	}

	// Build config with code-specific model overrides
	// Use the same base config (threads, GPU layers, URLs, keys) as the default embedder
	cfg := &Config{
		GGUFModelPath: mainCfg.GetCodeGGUFModelPath(),
		GGUFThreads:   mainCfg.GetGGUFThreads(),
		GGUFGPULayers: mainCfg.GetGGUFGPULayers(),
		OllamaURL:     mainCfg.GetOllamaURL(),
		OllamaModel:   mainCfg.GetCodeOllamaModel(),
		OpenAIKey:     mainCfg.GetOpenAIKey(),
		OpenAIBaseURL: mainCfg.GetOpenAIURL(),
		OpenAIModel:   mainCfg.GetCodeOpenAIModel(),
	}

	return NewEmbedderFromConfig(cfg)
}
