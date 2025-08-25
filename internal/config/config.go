// Package config holds the configuration structures for the Remembrances-MCP server.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config holds the configuration for the Remembrances-MCP server.
type Config struct {
	SSE           bool   `mapstructure:"sse"`
	RestAPIServe  bool   `mapstructure:"rest-api-serve"`
	KnowledgeBase string `mapstructure:"knowledge-base"`
	DbPath        string `mapstructure:"db-path"`
	SurrealDBURL  string `mapstructure:"surrealdb-url"`
	SurrealDBUser string `mapstructure:"surrealdb-user"`
	SurrealDBPass string `mapstructure:"surrealdb-pass"`
	OllamaURL     string `mapstructure:"ollama-url"`
	OllamaModel   string `mapstructure:"ollama-model"`
	OpenAIKey     string `mapstructure:"openai-key"`
	OpenAIURL     string `mapstructure:"openai-url"`
	OpenAIModel   string `mapstructure:"openai-model"`
}

// Load loads the configuration from CLI flags and environment variables.
func Load() (*Config, error) {
	// Define flags
	pflag.Bool("sse", false, "Enable SSE transport")
	pflag.Bool("rest-api-serve", false, "Enable REST API server")
	pflag.String("knowledge-base", "", "Path to the knowledge base directory")
	pflag.String("db-path", "./remembrances.db", "Path to the embedded SurrealDB database")
	pflag.String("surrealdb-url", "", "URL for the remote SurrealDB instance")
	pflag.String("surrealdb-user", "root", "Username for SurrealDB")
	pflag.String("surrealdb-pass", "root", "Password for SurrealDB")
	pflag.String("ollama-url", "http://localhost:11434", "URL for the Ollama server")
	pflag.String("ollama-model", "", "Ollama model to use for embeddings")
	pflag.String("openai-key", "", "OpenAI API key")
	pflag.String("openai-url", "https://api.openai.com/v1", "OpenAI base URL")
	pflag.String("openai-model", "text-embedding-3-large", "OpenAI model to use for embeddings")
	pflag.Parse()

	// Bind flags to viper
	v := viper.New()
	if err := v.BindPFlags(pflag.CommandLine); err != nil {
		return nil, fmt.Errorf("failed to bind pflags: %w", err)
	}

	// Configure viper to read environment variables
	v.SetEnvPrefix("GOMEM")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	// Unmarshal the configuration
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	// Validate that at least one embedder is configured
	if c.OllamaModel == "" && c.OpenAIKey == "" {
		return errors.New("at least one embedder (Ollama or OpenAI) must be configured")
	}

	// Validate database configuration
	if c.DbPath == "" && c.SurrealDBURL == "" {
		return errors.New("either a database path or a SurrealDB URL must be provided")
	}

	return nil
}

// GetOllamaURL returns the Ollama server URL.
func (c *Config) GetOllamaURL() string {
	return c.OllamaURL
}

// GetOllamaModel returns the Ollama model name.
func (c *Config) GetOllamaModel() string {
	return c.OllamaModel
}

// GetOpenAIKey returns the OpenAI API key.
func (c *Config) GetOpenAIKey() string {
	return c.OpenAIKey
}

// GetOpenAIURL returns the OpenAI base URL.
func (c *Config) GetOpenAIURL() string {
	return c.OpenAIURL
}

// GetOpenAIModel returns the OpenAI model name.
func (c *Config) GetOpenAIModel() string {
	return c.OpenAIModel
}

// Getenv reads an environment variable or returns a default value.
func Getenv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
