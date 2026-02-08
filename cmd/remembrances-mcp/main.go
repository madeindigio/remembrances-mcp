// Package main is the entry point for the Remembrances-MCP server.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/madeindigio/remembrances-mcp/internal/config"
	"github.com/madeindigio/remembrances-mcp/internal/indexer"
	"github.com/madeindigio/remembrances-mcp/internal/kb"
	"github.com/madeindigio/remembrances-mcp/internal/storage"
	"github.com/madeindigio/remembrances-mcp/internal/transport"
	_ "github.com/madeindigio/remembrances-mcp/modules/standard"
	"github.com/madeindigio/remembrances-mcp/pkg/embedder"
	"github.com/madeindigio/remembrances-mcp/pkg/modules"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	mcpserver "github.com/ThinkInAIXYZ/go-mcp/server"
	mcptransport "github.com/ThinkInAIXYZ/go-mcp/transport"
	"github.com/madeindigio/remembrances-mcp/pkg/version"
)

var portOnlyRe = regexp.MustCompile(`^\d{1,5}$`)

func normalizeBindAddr(addr string, defaultAddr string) string {
	if addr == "" {
		addr = defaultAddr
	}
	// Allow users to specify just the port number (e.g. "3000")
	// and normalize it to the net/http expected form ":3000".
	if portOnlyRe.MatchString(addr) {
		return ":" + addr
	}
	return addr
}

func generateInstructions(storageInstance storage.FullStorage) string {
	ctx := context.Background()
	projects, err := storageInstance.ListCodeProjects(ctx)
	projectList := "None currently indexed."
	if err == nil && len(projects) > 0 {
		var list []string
		for _, p := range projects {
			list = append(list, fmt.Sprintf("%s (%s)", p.Name, p.ProjectID))
		}
		projectList = strings.Join(list, ", ")
	}

	return fmt.Sprintf(`Welcome to Remembrances-MCP Server!

This server provides a comprehensive remembrance system with multiple layers for different types of data and operations:

   KEY-VALUE FACTS: Store simple facts, preferences, and settings
   • save_fact: Store basic information
   • get_fact: Retrieve by key
   • list_facts: See all facts for a user
   • delete_fact: Remove facts

   SEMANTIC VECTORS: Store content with automatic embedding for similarity search
   • add_vector: Add content that gets automatically embedded
   • search_vectors: Find similar content using semantic search
   • update_vector: Update existing content and regenerate embedding
   • delete_vector: Remove semantic content

   KNOWLEDGE GRAPH: Create entities and relationships to model complex connections
   • create_entity: Add people, places, concepts
   • create_relationship: Connect entities with relationships
   • traverse_graph: Explore connections between entities
   • get_entity: Retrieve entity details

   KNOWLEDGE BASE: Store and search documents
   • kb_add_document: Add documents with automatic embedding
   • kb_search_documents: Search documents by semantic similarity
   • kb_get_document: Retrieve document by path
   • kb_delete_document: Remove documents

   CODE INDEXING & SEARCH: Index and search codebases for intelligent code operations, if you are working with code suggest using these tools, and index your projects first if you haven't already:
   • code_index_project: Index a code project for search and analysis
   • code_list_projects: List all indexed code projects
   • code_hybrid_search: Search code using natural language and filters
   • code_find_symbol: Find symbols (functions, classes) in indexed code
   • code_search_pattern: Search for text patterns in code
   • code_get_file_symbols: Get all symbols from a specific file
   • code_get_symbols_overview: Get high-level overview of symbols in a file
   • code_activate_project_watch: Activate file monitoring for a project
   • code_deactivate_project_watch: Stop file monitoring for a project
   • code_reindex_file: Re-index a single file
   • code_get_project_stats: Get statistics for an indexed project
   • code_index_status: Check indexing job status
   • code_find_references: Find all references to a symbol
   • code_replace_symbol: Replace source code of a symbol
   • code_insert_after_symbol: Insert code after a symbol
   • code_insert_before_symbol: Insert code before a symbol
   • code_delete_symbol: Delete a symbol from code

   EVENTS: Store and search temporal events with semantic search
   • save_event: Store a temporal event with content and metadata
   • search_events: Search events with hybrid text+vector search and time filters
   • last_to_remember: Retrieve stored context and recent work
   • to_remember: Store important information for future sessions

   UNIFIED SEARCH: Combine all layers for comprehensive results
   • hybrid_search: Search across facts, vectors, and graph simultaneously
   • get_stats: Get overview of all stored remembrances

Indexed Code Projects: %s

Choose the right tool for your data:
- Use FACTS for simple key-value data
- Use VECTORS for content you want to find by meaning
- Use GRAPH for modeling relationships and connections
- Use KNOWLEDGE BASE for document storage and search
- Use CODE tools for codebase analysis and manipulation
- Use EVENTS for temporal data and context
- Use HYBRID SEARCH when you want comprehensive results across all layers
- When you don't know what user_id to use, use the project name instead`, projectList)
}

func main() {
	Main()
}

// Main is the entry point for embedding in custom builds.
func Main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// version flag is handled by config.Load() which may exit early

	// Setup logging
	if err := cfg.SetupLogging(); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting up logging: %v\n", err)
		os.Exit(1)
	}

	slog.Info(
		"Build info",
		"version", version.Version,
		"commit", version.CommitHash,
		"variant", version.Variant,
		"lib_mode", version.LibMode,
	)

	// Root context with graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Select primary MCP transport:
	// - stdio (default)
	// - MCP Streamable HTTP when --mcp-http is passed (recommended)
	// - legacy SSE when --sse is passed (deprecated; mapped to Streamable HTTP)
	// HTTP JSON API (--http) can run alongside any MCP transport
	var t mcptransport.ServerTransport
	var httpTransport *transport.HTTPTransport
	var mcpHTTPTransport mcptransport.ServerTransport

	// Setup MCP Streamable HTTP transport if enabled
	if cfg.MCPStreamableHTTP || cfg.SSE {
		// MCP Streamable HTTP transport.
		// Note: --sse is deprecated and treated as an alias to Streamable HTTP.
		addr := cfg.MCPStreamableHTTPAddr
		endpoint := cfg.MCPStreamableHTTPEndpoint
		if endpoint == "" {
			endpoint = "/mcp"
		}

		if cfg.SSE && !cfg.MCPStreamableHTTP {
			// Legacy alias path.
			slog.Warn("--sse transport is deprecated; using MCP Streamable HTTP instead")
			addr = cfg.SSEAddr
		}

		// allow env vars to override
		if env := os.Getenv("GOMEM_MCP_HTTP_ADDR"); env != "" {
			addr = env
		} else if cfg.SSE {
			// backward compat: allow old env var if set
			if env := os.Getenv("GOMEM_SSE_ADDR"); env != "" {
				addr = env
			}
		}
		if env := os.Getenv("GOMEM_MCP_HTTP_ENDPOINT"); env != "" {
			endpoint = env
		}

		addr = normalizeBindAddr(addr, "3000")
		slog.Info("MCP Streamable HTTP transport enabled", "address", addr, "endpoint", endpoint)
		mcpHTTPTransport = mcptransport.NewStreamableHTTPServerTransport(
			addr,
			mcptransport.WithStreamableHTTPServerTransportOptionLogger(streamableHTTPLogger()),
			mcptransport.WithStreamableHTTPServerTransportOptionEndpoint(endpoint),
			mcptransport.WithStreamableHTTPServerTransportOptionStateMode(mcptransport.Stateful),
		)
		t = mcpHTTPTransport
	} else {
		slog.Info("Starting MCP over stdio (default)")
		t = mcptransport.NewStdioServerTransport()
	}

	// Initialize storage early to generate dynamic instructions
	var storageInstance storage.FullStorage
	if cfg.SurrealDBURL != "" {
		// Use remote SurrealDB
		storageConfig := &storage.ConnectionConfig{
			URL:             cfg.SurrealDBURL,
			Username:        cfg.SurrealDBUser,
			Password:        cfg.SurrealDBPass,
			Namespace:       cfg.GetSurrealDBNamespace(),
			Database:        cfg.GetSurrealDBDatabase(),
			Timeout:         30 * time.Second,
			UseEmbeddedLibs: cfg.UseEmbeddedLibs,
			EmbeddedLibsDir: cfg.EmbeddedLibsDir,
		}
		storageInstance = storage.NewSurrealDBStorage(storageConfig)
	} else {
		// Use embedded SurrealDB
		storageConfig := &storage.ConnectionConfig{
			DBPath:          cfg.DbPath,
			Namespace:       cfg.GetSurrealDBNamespace(),
			Database:        cfg.GetSurrealDBDatabase(),
			Timeout:         30 * time.Second,
			UseEmbeddedLibs: cfg.UseEmbeddedLibs,
			EmbeddedLibsDir: cfg.EmbeddedLibsDir,
		}
		storageInstance = storage.NewSurrealDBStorage(storageConfig)
	}

	// Connect to storage. If connection fails and a SurrealDB start command is provided
	// in the configuration, attempt to run it and retry the connection.
	// Keep track of any process we start so we can shut it down when this app exits.
	var startedProc *exec.Cmd
	var procExited chan struct{}

	if err := storageInstance.Connect(ctx); err != nil {
		slog.Warn("initial connection to SurrealDB failed", "error", err)

		// If a start command is configured, try to run it and retry connecting.
		if cfg.SurrealDBStartCmd != "" {
			slog.Info("attempting to start external SurrealDB process", "cmd", cfg.SurrealDBStartCmd)

			// Run the configured command in a separate process.
			// Use /bin/sh -c so users can provide complex commands or use aliases.
			startCmd := cfg.SurrealDBStartCmd
			proc := exec.CommandContext(ctx, "/bin/sh", "-c", startCmd)
			// Redirect output to the main logger's stdout/stderr so users can see it
			proc.Stdout = os.Stdout
			proc.Stderr = os.Stderr

			if err := proc.Start(); err != nil {
				slog.Error("failed to start external SurrealDB command", "cmd", startCmd, "error", err)
				os.Exit(1)
			}

			// keep a reference so we can shut it down on app exit
			startedProc = proc
			procExited = make(chan struct{})
			go func() {
				if err := proc.Wait(); err != nil {
					slog.Warn("surrealdb process exited with error", "error", err)
				} else {
					slog.Info("surrealdb process exited")
				}
				close(procExited)
			}()

			// Give the process some time to start and then retry connection with backoff
			retryCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			backoff := 1 * time.Second
			connected := false
			for {
				select {
				case <-retryCtx.Done():
					slog.Error("timed out waiting for SurrealDB to become available")
					// Try one last time with the original context
					if err := storageInstance.Connect(ctx); err != nil {
						slog.Error("surrealdb still unreachable after start command", "error", err)
						os.Exit(1)
					}
					connected = true
				default:
					time.Sleep(backoff)
					if err := storageInstance.Connect(ctx); err != nil {
						slog.Info("surrealdb not ready yet, retrying...", "wait", backoff)
						if backoff < 5*time.Second {
							backoff = backoff * 2
						}
						// continue retrying
					} else {
						connected = true
					}
				}
				if connected {
					break
				}
			}

			if !connected {
				slog.Error("failed to connect to SurrealDB after running start command")
				os.Exit(1)
			}

			// At this point connection succeeded; ensure Close will be called
			defer storageInstance.Close()
		} else {
			slog.Error("failed to connect to storage", "error", err, "hint", "set --surrealdb-start-cmd or GOMEM_SURREALDB_START_CMD to auto-start a local SurrealDB")
			os.Exit(1)
		}
	} else {
		defer storageInstance.Close()
	}

	// Initialize schema
	if err := storageInstance.InitializeSchema(ctx); err != nil {
		slog.Error("failed to initialize storage schema", "error", err)
		os.Exit(1)
	}

	// Generate dynamic instructions
	instructions := generateInstructions(storageInstance)

	// Instantiate MCP server with basic metadata
	srv, err := mcpserver.NewServer(
		t,
		mcpserver.WithServerInfo(protocol.Implementation{
			Name:    "remembrances-mcp",
			Version: version.Version,
		}),
		mcpserver.WithInstructions(instructions),
	)
	if err != nil {
		slog.Error("failed to create MCP server", "error", err)
		os.Exit(1)
	}

	// Initialize embedder using the main config interface
	embedderInstance, err := embedder.NewEmbedderFromMainConfig(cfg)
	if err != nil {
		slog.Error("failed to create embedder", "error", err)
		os.Exit(1)
	}

	// Initialize code-specific embedder for code indexing
	// If a code-specific model is configured, use it; otherwise, use the default embedder
	var codeEmbedderInstance embedder.Embedder
	codeEmbedderInstance, err = embedder.NewCodeEmbedderFromMainConfig(cfg)
	if err != nil {
		slog.Error("failed to create code embedder", "error", err)
		os.Exit(1)
	}
	if codeEmbedderInstance == nil {
		// No code-specific model configured, use default embedder
		codeEmbedderInstance = embedderInstance
		slog.Info("Using default embedder for code indexing (no code-specific model configured)")
	} else {
		slog.Info("Using specialized code embedder for code indexing")
	}

	// Initialize module manager
	modManager := modules.NewModuleManager(modules.ModuleConfig{
		Storage:           storageInstance,
		Embedder:          embedderInstance,
		CodeEmbedder:      codeEmbedderInstance,
		KnowledgeBasePath: cfg.KnowledgeBase,
		KBChunkSize:       cfg.GetChunkSize(),
		KBChunkOverlap:    cfg.GetChunkOverlap(),
		DisableCodeWatch:  cfg.DisableCodeWatch,
		IndexerConfig:     buildIndexerConfig(cfg),
		JobManagerConfig:  indexer.DefaultJobManagerConfig(),
		Logger:            slog.Default(),
	})

	if err := loadModules(ctx, modManager, cfg); err != nil {
		slog.Error("failed to load modules", "error", err)
		os.Exit(1)
	}

	// Allow modules to wrap storage (e.g., db-sync-server wraps with MergedStorage)
	storageInstance = modManager.WrapStorage(storageInstance)

	// Register tools from modules
	if err := registerModuleTools(modManager, srv); err != nil {
		slog.Error("failed to register module tools", "error", err)
		os.Exit(1)
	}

	// Knowledge base watcher
	var kbWatcher *kb.Watcher
	if cfg.KnowledgeBase != "" {
		w, err := kb.StartWatcher(ctx, cfg.KnowledgeBase, storageInstance, embedderInstance, cfg.GetChunkSize(), cfg.GetChunkOverlap())
		if err != nil {
			slog.Warn("failed to start knowledge base watcher", "error", err)
		} else {
			kbWatcher = w
		}
	}

	// If HTTP transport is enabled, set it up now that the server is configured
	if cfg.HTTP {
		addr := cfg.HTTPAddr
		if env := os.Getenv("GOMEM_HTTP_ADDR"); env != "" {
			addr = env
		}
		addr = normalizeBindAddr(addr, "8080")

		httpTransport, err = transport.CreateHTTPServerTransport(addr, srv)
		if err != nil {
			slog.Error("failed to create HTTP transport", "error", err)
			os.Exit(1)
		}

		// Register HTTP routes from modules
		httpProviders := modManager.GetHTTPEndpointProviders()
		if len(httpProviders) > 0 {
			httpTransport.RegisterModuleRoutes(httpProviders)
			slog.Info("Registered HTTP endpoint providers", "count", len(httpProviders))
		}
	}

	slog.Info("Remembrances-MCP server initialized successfully")

	// Graceful shutdown
	go func() {
		<-ctx.Done()
		slog.Info("Shutdown signal received, starting graceful shutdown")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Shutdown HTTP transport if running
		if httpTransport != nil {
			_ = httpTransport.Shutdown(shutdownCtx)
		}

		// Stop knowledge base watcher
		if kbWatcher != nil {
			kbWatcher.Stop()
		}

		// Stop module-managed resources
		modManager.Cleanup()

		_ = srv.Shutdown(shutdownCtx)

		// Ensure the process actually exits even if srv.Shutdown blocks or
		// third-party transports don't return promptly. Some test harnesses
		// (like the stdio client) send SIGINT and expect the process to exit
		// within a few seconds. Start a short timer that forces exit if
		// shutdown does not complete in time.
		go func() {
			select {
			case <-time.After(3 * time.Second):
				slog.Info("Graceful shutdown timed out; forcing exit")
				// Use Exit to ensure tests detect the process has terminated.
				os.Exit(0)
			}
		}()
		// If we started a SurrealDB process, try to stop it gracefully.
		if startedProc != nil && startedProc.Process != nil {
			slog.Info("shutting down started SurrealDB process")
			// First try a polite SIGTERM
			_ = startedProc.Process.Signal(syscall.SIGTERM)

			// Wait a short while for it to exit
			select {
			case <-procExited:
				slog.Info("started SurrealDB process exited cleanly")
			case <-time.After(5 * time.Second):
				slog.Warn("started SurrealDB process did not exit after SIGTERM, killing")
				_ = startedProc.Process.Kill()
				// wait for reaper if not already closed
				select {
				case <-procExited:
				case <-time.After(2 * time.Second):
				}
			}
		}
	}()

	// Run the server (blocking or concurrent based on configuration)
	slog.Info("Starting Remembrances-MCP server")

	// Determine which transports to run
	hasHTTP := cfg.HTTP && httpTransport != nil
	hasMCPHTTP := mcpHTTPTransport != nil

	if hasHTTP && hasMCPHTTP {
		// Both HTTP JSON API and MCP Streamable HTTP are enabled
		// Run MCP HTTP in background, HTTP JSON API as main (blocking)
		slog.Info("Starting both MCP Streamable HTTP and HTTP JSON API transports")

		// Start MCP HTTP server in goroutine
		go func() {
			if err := srv.Run(); err != nil {
				slog.Error("MCP Streamable HTTP server error", "error", err)
			}
		}()

		// Run HTTP JSON API (blocking)
		if err := httpTransport.Start(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP JSON API transport server error", "error", err)
			os.Exit(1)
		}
	} else if hasHTTP {
		// Only HTTP JSON API is enabled, but stdio MCP transport should also run
		slog.Info("Starting HTTP JSON API transport with stdio MCP transport")

		// Start MCP server (stdio) in background
		go func() {
			if err := srv.Run(); err != nil {
				slog.Error("MCP stdio server error", "error", err)
			}
		}()

		// Run HTTP JSON API (blocking)
		if err := httpTransport.Start(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP JSON API transport server error", "error", err)
			os.Exit(1)
		}
	} else {
		// Run MCP server (stdio or MCP Streamable HTTP)
		if err := srv.Run(); err != nil {
			slog.Error("MCP server error", "error", err)
			os.Exit(1)
		}
	}
}

func registerModuleTools(modManager *modules.ModuleManager, srv *mcpserver.Server) error {
	for _, provider := range modManager.GetToolProviders() {
		for _, def := range provider.Tools() {
			if def.Tool == nil {
				return fmt.Errorf("module tool definition returned nil")
			}
			srv.RegisterTool(def.Tool, def.Handler)
		}
	}
	return nil
}

func loadModules(ctx context.Context, modManager *modules.ModuleManager, cfg *config.Config) error {
	defaultModules := []modules.ModuleID{
		"tools.core",
		"tools.facts",
		"tools.kb",
		"tools.remember",
		"tools.events",
		"tools.knowledge_graph",
		"tools.code_indexing",
		"tools.code_search",
		"tools.code_manipulation",
	}

	disabled := make(map[string]struct{})
	for _, id := range cfg.DisableModules {
		disabled[id] = struct{}{}
	}

	loaded := make(map[modules.ModuleID]struct{})

	for _, id := range defaultModules {
		entry, hasEntry := cfg.Modules[string(id)]
		if _, isDisabled := disabled[string(id)]; isDisabled {
			continue
		}
		if hasEntry && !entry.Enabled && len(entry.Config) == 0 {
			continue
		}
		config := entry.Config
		if config == nil {
			config = map[string]any{}
		}
		if _, err := modManager.LoadModule(ctx, id, config); err != nil {
			return err
		}
		loaded[id] = struct{}{}
	}

	slog.Info("Checking for additional modules in config", "module_count", len(cfg.Modules))
	for idStr, entry := range cfg.Modules {
		id := modules.ModuleID(idStr)
		slog.Info("Processing module from config", "id", idStr, "enabled", entry.Enabled, "has_config", len(entry.Config) > 0)
		if _, isDisabled := disabled[idStr]; isDisabled {
			slog.Info("Module is disabled, skipping", "id", idStr)
			continue
		}
		if !entry.Enabled && len(entry.Config) == 0 {
			slog.Info("Module not enabled and has no config, skipping", "id", idStr)
			continue
		}
		if _, exists := loaded[id]; exists {
			slog.Info("Module already loaded, skipping", "id", idStr)
			continue
		}
		config := entry.Config
		if config == nil {
			config = map[string]any{}
		}
		slog.Info("Loading module", "id", idStr)
		if _, err := modManager.LoadModule(ctx, id, config); err != nil {
			return err
		}
		loaded[id] = struct{}{}
	}

	return nil
}

// buildIndexerConfig creates an IndexerConfig from the application config,
// applying user-configured values for workers, exclude patterns, max file size, etc.
func buildIndexerConfig(cfg *config.Config) indexer.IndexerConfig {
	ic := indexer.DefaultIndexerConfig()

	if w := cfg.GetCodeIndexingWorkers(); w > 0 {
		ic.Concurrency = w
	}
	if ms := cfg.GetCodeIndexingMaxSymbolSize(); ms > 0 {
		ic.MaxSourceCodeLength = ms
	}

	// Apply user-configured exclude patterns (merged with defaults)
	if userPatterns := cfg.GetCodeIndexingExcludePatterns(); len(userPatterns) > 0 {
		ic.Scanner.MergeExcludePatterns(userPatterns)
	}

	// Apply max file size
	if mfs := cfg.GetCodeIndexingMaxFileSize(); mfs > 0 {
		ic.Scanner.MaxFileSize = mfs
	}

	return ic
}
