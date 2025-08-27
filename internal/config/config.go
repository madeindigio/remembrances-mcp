// Package config holds the configuration structures for the Remembrances-MCP server.
package config

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config holds the configuration for the Remembrances-MCP server.
type Config struct {
	SSE                bool   `mapstructure:"sse"`
	SSEAddr            string `mapstructure:"sse-addr"`
	RestAPIServe       bool   `mapstructure:"rest-api-serve"`
	KnowledgeBase      string `mapstructure:"knowledge-base"`
	DbPath             string `mapstructure:"db-path"`
	SurrealDBURL       string `mapstructure:"surrealdb-url"`
	SurrealDBUser      string `mapstructure:"surrealdb-user"`
	SurrealDBPass      string `mapstructure:"surrealdb-pass"`
	SurrealDBNamespace string `mapstructure:"surrealdb-namespace"`
	SurrealDBDatabase  string `mapstructure:"surrealdb-database"`
	// Command to start an external SurrealDB instance when connection cannot be
	// established. Can be set via CLI flag --surrealdb-start-cmd or
	// environment variable GOMEM_SURREALDB_START_CMD.
	SurrealDBStartCmd string `mapstructure:"surrealdb-start-cmd"`
	OllamaURL         string `mapstructure:"ollama-url"`
	OllamaModel       string `mapstructure:"ollama-model"`
	OpenAIKey         string `mapstructure:"openai-key"`
	OpenAIURL         string `mapstructure:"openai-url"`
	OpenAIModel       string `mapstructure:"openai-model"`
	LogFile           string `mapstructure:"log"`
}

// Load loads the configuration from CLI flags and environment variables.
func Load() (*Config, error) {
	// Define flags
	pflag.Bool("sse", false, "Enable SSE transport")
	pflag.String("sse-addr", ":3000", "Address to bind SSE transport (host:port), can also be set via GOMEM_SSE_ADDR")
	pflag.Bool("rest-api-serve", false, "Enable REST API server")
	pflag.String("knowledge-base", "", "Path to the knowledge base directory")
	pflag.String("db-path", "./remembrances.db", "Path to the embedded SurrealDB database")
	pflag.String("surrealdb-url", "", "URL for the remote SurrealDB instance")
	pflag.String("surrealdb-user", "root", "Username for SurrealDB")
	pflag.String("surrealdb-pass", "root", "Password for SurrealDB")
	pflag.String("surrealdb-namespace", "test", "Namespace for SurrealDB")
	pflag.String("surrealdb-database", "test", "Database for SurrealDB")
	pflag.String("surrealdb-start-cmd", "", "External command to start SurrealDB when connection fails")
	pflag.String("ollama-url", "http://localhost:11434", "URL for the Ollama server")
	pflag.String("ollama-model", "", "Ollama model to use for embeddings")
	pflag.String("openai-key", "", "OpenAI API key")
	pflag.String("openai-url", "https://api.openai.com/v1", "OpenAI base URL")
	pflag.String("openai-model", "text-embedding-3-large", "OpenAI model to use for embeddings")
	pflag.String("log", "", "Path to the log file (logs will be written to both stdout and file)")
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

// GetSurrealDBNamespace returns the SurrealDB namespace.
func (c *Config) GetSurrealDBNamespace() string {
	if c.SurrealDBNamespace == "" {
		return "test"
	}
	return c.SurrealDBNamespace
}

// GetSurrealDBDatabase returns the SurrealDB database.
func (c *Config) GetSurrealDBDatabase() string {
	if c.SurrealDBDatabase == "" {
		return "test"
	}
	return c.SurrealDBDatabase
}

// Getenv reads an environment variable or returns a default value.
func Getenv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// SetupLogging configures slog to write to both stdout and a log file if specified.
func (c *Config) SetupLogging() error {
	var writers []io.Writer

	// Always write to stdout
	writers = append(writers, os.Stdout)

	// If log file is specified, also write to file
	if c.LogFile != "" {
		logFile, err := os.OpenFile(c.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file %s: %w", c.LogFile, err)
		}
		writers = append(writers, logFile)
	}

	// Create a multi-writer that writes to all specified destinations
	multiWriter := io.MultiWriter(writers...)

	// Create a text handler with the multi-writer
	handler := slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false,
	})

	// Set the default logger
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return nil
}
