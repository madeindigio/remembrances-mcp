// Package config holds the configuration structures for the Remembrances-MCP server.
package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/madeindigio/remembrances-mcp/pkg/version"
)

// Config holds the configuration for the Remembrances-MCP server.
type Config struct {
	// Deprecated: SSE transport is obsolete in MCP. Use MCPStreamableHTTP instead.
	SSE     bool   `mapstructure:"sse"`
	SSEAddr string `mapstructure:"sse-addr"`

	// MCPStreamableHTTP enables MCP over Streamable HTTP transport.
	// This is the recommended network transport for MCP (replaces SSE).
	MCPStreamableHTTP         bool   `mapstructure:"mcp-http"`
	MCPStreamableHTTPAddr     string `mapstructure:"mcp-http-addr"`
	MCPStreamableHTTPEndpoint string `mapstructure:"mcp-http-endpoint"`

	HTTP               bool   `mapstructure:"http"`
	HTTPAddr           string `mapstructure:"http-addr"`
	RestAPIServe       bool   `mapstructure:"rest-api-serve"`
	KnowledgeBase      string `mapstructure:"knowledge-base"`
	DbPath             string `mapstructure:"db-path"`
	SurrealDBURL       string `mapstructure:"surrealdb-url"`
	SurrealDBUser      string `mapstructure:"surrealdb-user"`
	SurrealDBPass      string `mapstructure:"surrealdb-pass"`
	SurrealDBNamespace string `mapstructure:"surrealdb-namespace"`
	SurrealDBDatabase  string `mapstructure:"surrealdb-database"`
	UseEmbeddedLibs    bool   `mapstructure:"use-embedded-libs"`
	EmbeddedLibsDir    string `mapstructure:"embedded-libs-dir"`
	// Command to start an external SurrealDB instance when connection cannot be
	// established. Can be set via CLI flag --surrealdb-start-cmd or
	// environment variable GOMEM_SURREALDB_START_CMD.
	SurrealDBStartCmd string `mapstructure:"surrealdb-start-cmd"`
	// GGUF local model configuration
	GGUFModelPath string `mapstructure:"gguf-model-path"`
	GGUFThreads   int    `mapstructure:"gguf-threads"`
	GGUFGPULayers int    `mapstructure:"gguf-gpu-layers"`
	// Ollama configuration
	OllamaURL   string `mapstructure:"ollama-url"`
	OllamaModel string `mapstructure:"ollama-model"`
	// OpenAI configuration
	OpenAIKey   string `mapstructure:"openai-key"`
	OpenAIURL   string `mapstructure:"openai-url"`
	OpenAIModel string `mapstructure:"openai-model"`
	// Code-specific embedding model configuration
	// These allow using specialized code embedding models (e.g., CodeRankEmbed, Jina-code-embeddings)
	// for code indexing while using a different model for text/facts/vectors/events
	CodeGGUFModelPath string `mapstructure:"code-gguf-model-path"`
	CodeOllamaModel   string `mapstructure:"code-ollama-model"`
	CodeOpenAIModel   string `mapstructure:"code-openai-model"`
	// Chunking configuration for embeddings
	ChunkSize    int    `mapstructure:"chunk-size"`
	ChunkOverlap int    `mapstructure:"chunk-overlap"`
	LogFile      string `mapstructure:"log"`
	// When true, disables all logging output to stdout/stderr.
	// Logs will only be written to the configured log file (if any).
	DisableOutputLog bool `mapstructure:"disable-output-log"`
	// Code monitoring configuration
	// When true, disables automatic code file watching for projects
	DisableCodeWatch bool `mapstructure:"disable-code-watch"`
	// Module configuration
	Modules        map[string]ModuleEntry `mapstructure:"modules"`
	DisableModules []string               `mapstructure:"disable"`
}

// ModuleEntry describes module configuration in config files.
type ModuleEntry struct {
	Enabled bool           `mapstructure:"enabled"`
	Config  map[string]any `mapstructure:"config"`
}

// Load loads the configuration from CLI flags and environment variables.
func Load() (*Config, error) {
	// Define flags
	// To add a new CLI flag:
	// 1) Register it here with pflag (or pflag.String/PBool/etc)
	// 2) Call pflag.Parse() (done below)
	// 3) Bind pflags to viper via v.BindPFlags(pflag.CommandLine)
	// 4) Read the value from the returned Config or via v.GetXXX
	// Note: flags that should cause the process to exit early (like --version)
	// can be handled immediately after parsing, before continuing with config
	// initialization.

	pflag.String("config", "", "Path to YAML configuration file")
	// Deprecated SSE flags: keep for backwards compatibility but migrate users to Streamable HTTP.
	pflag.Bool("sse", false, "[DEPRECATED] Enable SSE transport (obsolete). Use --mcp-http")
	pflag.String("sse-addr", ":3000", "[DEPRECATED] Address to bind SSE transport (obsolete). Use --mcp-http-addr")

	// MCP over Streamable HTTP (recommended)
	pflag.Bool("mcp-http", false, "Enable MCP Streamable HTTP transport")
	// Accept either plain port (e.g. "3000") or full address (e.g. "127.0.0.1:3000").
	pflag.String("mcp-http-addr", "3000", "Port or address to bind MCP Streamable HTTP transport (e.g. 3000 or 127.0.0.1:3000); can also be set via GOMEM_MCP_HTTP_ADDR")
	pflag.String("mcp-http-endpoint", "/mcp", "HTTP path for the MCP Streamable HTTP endpoint, can also be set via GOMEM_MCP_HTTP_ENDPOINT")

	pflag.Bool("http", false, "Enable HTTP JSON API transport")
	pflag.String("http-addr", ":8080", "Address to bind HTTP transport (host:port), can also be set via GOMEM_HTTP_ADDR")
	pflag.Bool("rest-api-serve", false, "Enable REST API server")
	pflag.String("knowledge-base", "", "Path to the knowledge base directory")
	pflag.String("db-path", "./remembrances.db", "Path to the embedded SurrealDB database")
	pflag.Bool("use-embedded-libs", true, "Extract and load embedded shared libraries (libsurrealdb, libllama, ggml)")
	pflag.String("embedded-libs-dir", "", "Destination directory for extracted embedded libraries (defaults to temporary dir)")
	pflag.String("surrealdb-url", "", "URL for the remote SurrealDB instance")
	pflag.String("surrealdb-user", "root", "Username for SurrealDB")
	pflag.String("surrealdb-pass", "root", "Password for SurrealDB")
	pflag.String("surrealdb-namespace", "test", "Namespace for SurrealDB")
	pflag.String("surrealdb-database", "test", "Database for SurrealDB")
	pflag.String("surrealdb-start-cmd", "", "External command to start SurrealDB when connection fails")
	pflag.String("gguf-model-path", "", "Path to GGUF model file for local embeddings")
	pflag.Int("gguf-threads", 0, "Number of threads for GGUF model (0 = auto-detect)")
	pflag.Int("gguf-gpu-layers", 0, "Number of GPU layers for GGUF model (0 = CPU only)")
	pflag.String("ollama-url", "http://localhost:11434", "URL for the Ollama server")
	pflag.String("ollama-model", "", "Ollama model to use for embeddings")
	pflag.String("openai-key", "", "OpenAI API key")
	pflag.String("openai-url", "https://api.openai.com/v1", "OpenAI base URL")
	pflag.String("openai-model", "text-embedding-3-large", "OpenAI model to use for embeddings")
	// Code-specific embedding model flags (for code indexing)
	pflag.String("code-gguf-model-path", "", "Path to GGUF model for code embeddings (e.g., CodeRankEmbed)")
	pflag.String("code-ollama-model", "", "Ollama model to use for code embeddings (e.g., jina/jina-embeddings-v2-base-code)")
	pflag.String("code-openai-model", "", "OpenAI model to use for code embeddings")
	pflag.Int("chunk-size", 800, "Maximum chunk size in characters for text splitting (default: 800)")
	pflag.Int("chunk-overlap", 100, "Overlap between chunks in characters (default: 100)")
	pflag.String("log", "", "Path to the log file (logs will be written to both stdout and file)")
	pflag.Bool("disable-output-log", false, "Disable logging to stdout/stderr; only write to log file if configured")
	pflag.Bool("disable-code-watch", false, "Disable automatic file watching for code projects")
	// Version flag is handled here so config package can manage early-exit flags
	// Also register a version flag with the standard library's flag set so
	// packages that use the stdlib flag package (or call flag.Parse)
	// won't error when users pass --version/-v to this binary.
	flag.Bool("version", false, "Print version and exit")

	// Make any flags registered with the stdlib visible to pflag so a single
	// unified parse will work for both kinds of flags.
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	// Do not re-register the "version" flag with pflag here — it is
	// registered via the standard library flag set above and copied into
	// pflag by AddGoFlagSet. Registering it twice causes a "flag redefined"
	// panic when parsing.
	pflag.Parse()

	// Handle early-exit flags (version) before binding to viper
	if ver := pflag.Lookup("version"); ver != nil && ver.Value.String() == "true" {
		fmt.Println(version.Describe())
		os.Exit(0)
	}

	// Initialize viper
	v := viper.New()

	// Read YAML config file if provided via --config flag
	configPath := pflag.Lookup("config").Value.String()
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		// No --config flag provided, try to find config.yaml in standard locations
		configFound := false

		if homeDir, err := os.UserHomeDir(); err == nil {
			var standardConfigPath string

			// Use OS-specific standard location
			if runtime.GOOS == "darwin" {
				// macOS: ~/Library/Application Support/remembrances/config.yaml
				standardConfigPath = filepath.Join(homeDir, "Library", "Application Support", "remembrances", "config.yaml")
			} else {
				// Linux/Unix: ~/.config/remembrances/config.yaml
				standardConfigPath = filepath.Join(homeDir, ".config", "remembrances", "config.yaml")
			}

			if _, err := os.Stat(standardConfigPath); err == nil {
				v.SetConfigFile(standardConfigPath)
				if err := v.ReadInConfig(); err == nil {
					configFound = true
					slog.Info("Using configuration file from standard location", "path", standardConfigPath)
				}
			}
		}

		// If no config file found in standard locations, continue without it
		// (environment variables and defaults will be used)
		if !configFound {
			slog.Info("No configuration file found, using environment variables and defaults")
		}
	}

	// Bind flags to viper
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
	if c.GGUFModelPath == "" && c.OllamaModel == "" && c.OpenAIKey == "" {
		return errors.New("at least one embedder (GGUF, Ollama or OpenAI) must be configured")
	}

	// Validate database configuration
	if c.DbPath == "" && c.SurrealDBURL == "" {
		return errors.New("either a database path or a SurrealDB URL must be provided")
	}

	return nil
}

// GetGGUFModelPath returns the GGUF model file path.
func (c *Config) GetGGUFModelPath() string {
	return c.GGUFModelPath
}

// GetGGUFThreads returns the number of threads for GGUF model.
func (c *Config) GetGGUFThreads() int {
	return c.GGUFThreads
}

// GetGGUFGPULayers returns the number of GPU layers for GGUF model.
func (c *Config) GetGGUFGPULayers() int {
	return c.GGUFGPULayers
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

// GetCodeGGUFModelPath returns the GGUF model path for code embeddings.
// If not set, returns the default GGUF model path.
func (c *Config) GetCodeGGUFModelPath() string {
	if c.CodeGGUFModelPath != "" {
		return c.CodeGGUFModelPath
	}
	return c.GGUFModelPath
}

// GetCodeOllamaModel returns the Ollama model for code embeddings.
// If not set, returns the default Ollama model.
func (c *Config) GetCodeOllamaModel() string {
	if c.CodeOllamaModel != "" {
		return c.CodeOllamaModel
	}
	return c.OllamaModel
}

// GetCodeOpenAIModel returns the OpenAI model for code embeddings.
// If not set, returns the default OpenAI model.
func (c *Config) GetCodeOpenAIModel() string {
	if c.CodeOpenAIModel != "" {
		return c.CodeOpenAIModel
	}
	return c.OpenAIModel
}

// HasCodeSpecificEmbedder returns true if a code-specific embedding model is configured.
func (c *Config) HasCodeSpecificEmbedder() bool {
	return c.CodeGGUFModelPath != "" || c.CodeOllamaModel != "" || c.CodeOpenAIModel != ""
}

// GetChunkSize returns the chunk size for text splitting.
func (c *Config) GetChunkSize() int {
	if c.ChunkSize <= 0 {
		return 800 // Default chunk size - CRITICAL: must stay under 512 token limit
	}
	return c.ChunkSize
}

// GetChunkOverlap returns the overlap between chunks.
func (c *Config) GetChunkOverlap() int {
	if c.ChunkOverlap < 0 {
		return 200 // Default overlap
	}
	return c.ChunkOverlap
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

// SetupLogging configures slog output.
//
// Important: when running MCP over stdio, stdout must be reserved for protocol
// messages. Therefore, console logs default to stderr in stdio mode.
func (c *Config) SetupLogging() error {
	var writers []io.Writer

	// Console logging (stdout/stderr)
	if !c.DisableOutputLog {
		// If we're running in stdio mode (default: no http/sse/rest), avoid stdout.
		// This prevents logs from corrupting MCP protocol messages.
		stdioMode := !c.SSE && !c.MCPStreamableHTTP && !c.HTTP && !c.RestAPIServe
		if stdioMode {
			writers = append(writers, os.Stderr)
		} else {
			writers = append(writers, os.Stdout)
		}
	}

	// If log file is specified, also write to file
	if c.LogFile != "" {
		logFile, err := os.OpenFile(c.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file %s: %w", c.LogFile, err)
		}
		writers = append(writers, logFile)
	}

	// If nothing is configured (disable-output-log=true and no file), discard logs.
	if len(writers) == 0 {
		writers = append(writers, io.Discard)
	}

	// Create a multi-writer that writes to all specified destinations
	multiWriter := io.MultiWriter(writers...)

	// Create a text handler with the multi-writer
	handler := slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		Level:     slog.LevelInfo, // Change this to desired log level
		AddSource: false,
	})

	// Set the default logger
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return nil
}
