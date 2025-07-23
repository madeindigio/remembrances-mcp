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
}

// NewEmbedderFromConfig crea una instancia de Embedder basada en la configuración disponible.
// Prioridad: si OLLAMA_URL está configurado, usa Ollama; si OPENAI_API_KEY está configurado, usa OpenAI.
// Retorna error si no se encuentra ninguna configuración válida.
func NewEmbedderFromConfig(cfg *Config) (Embedder, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	// Prioridad 1: Ollama (si URL está disponible)
	if cfg.OllamaURL != "" {
		if cfg.OllamaModel == "" {
			return nil, fmt.Errorf("ollama URL provided but model is missing")
		}
		return NewOllamaEmbedder(cfg.OllamaURL, cfg.OllamaModel)
	}

	// Prioridad 2: OpenAI (si API key está disponible)
	if cfg.OpenAIKey != "" {
		if cfg.OpenAIModel == "" {
			// Usar modelo por defecto si no se especifica
			cfg.OpenAIModel = "text-embedding-3-large"
		}
		return NewOpenAIEmbedder(cfg.OpenAIKey, cfg.OpenAIBaseURL, cfg.OpenAIModel)
	}

	return nil, fmt.Errorf("no valid embedder configuration found: either OLLAMA_URL or OPENAI_API_KEY must be provided")
}

// NewEmbedderFromEnv crea una instancia de Embedder leyendo la configuración desde variables de entorno.
// Variables de entorno soportadas:
// - OLLAMA_URL: URL del servidor Ollama
// - OLLAMA_EMBEDDING_MODEL: Modelo de embedding de Ollama
// - OPENAI_API_KEY: Clave API de OpenAI
// - OPENAI_API_BASE: URL base para APIs compatibles con OpenAI
// - OPENAI_EMBEDDING_MODEL: Modelo de embedding de OpenAI
func NewEmbedderFromEnv() (Embedder, error) {
	cfg := &Config{
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

	hasOllama := cfg.OllamaURL != ""
	hasOpenAI := cfg.OpenAIKey != ""

	if !hasOllama && !hasOpenAI {
		return fmt.Errorf("at least one embedder must be configured (Ollama or OpenAI)")
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
	GetOllamaURL() string
	GetOllamaModel() string
	GetOpenAIKey() string
	GetOpenAIURL() string
	GetOpenAIModel() string
}

// NewEmbedderFromMainConfig crea un embedder usando la configuración principal de la aplicación.
func NewEmbedderFromMainConfig(mainCfg MainConfig) (Embedder, error) {
	if mainCfg == nil {
		return nil, fmt.Errorf("main configuration is required")
	}

	cfg := &Config{
		OllamaURL:     mainCfg.GetOllamaURL(),
		OllamaModel:   mainCfg.GetOllamaModel(),
		OpenAIKey:     mainCfg.GetOpenAIKey(),
		OpenAIBaseURL: mainCfg.GetOpenAIURL(),
		OpenAIModel:   mainCfg.GetOpenAIModel(),
	}

	return NewEmbedderFromConfig(cfg)
}
