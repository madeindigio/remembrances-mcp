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

	"github.com/madeindigio/remembrances-mcp/v2/internal/config"
	"github.com/madeindigio/remembrances-mcp/v2/internal/indexer"
	"github.com/madeindigio/remembrances-mcp/v2/internal/kb"
	"github.com/madeindigio/remembrances-mcp/v2/internal/storage"
	"github.com/madeindigio/remembrances-mcp/v2/internal/transport"
	_ "github.com/madeindigio/remembrances-mcp/v2/modules/standard"
	"github.com/madeindigio/remembrances-mcp/v2/pkg/embedder"
	"github.com/madeindigio/remembrances-mcp/v2/pkg/mcp_tools"
	"github.com/madeindigio/remembrances-mcp/v2/pkg/modules"
	"github.com/madeindigio/remembrances-mcp/v2/pkg/principal"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	mcpserver "github.com/ThinkInAIXYZ/go-mcp/server"
	mcptransport "github.com/ThinkInAIXYZ/go-mcp/transport"
	"github.com/madeindigio/remembrances-mcp/v2/pkg/version"
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
	// Startup work with no request behind it: it lists the indexed projects to
	// put them in the server instructions, and with require_principal on that
	// read needs an identity like any other.
	ctx := storage.ServiceContext(context.Background())
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
	// Migration subcommands run before configuration is loaded: they take their
	// own flag sets and must be able to address a datastore the server itself
	// would refuse to open.
	if handled, code := runSubcommand(context.Background()); handled {
		os.Exit(code)
	}

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
	var mcpHTTPServer *transport.MCPHTTPServer

	// Setup MCP Streamable HTTP transport if enabled
	if cfg.MCPStreamableHTTP || cfg.SSE {
		// MCP Streamable HTTP transport.
		// Note: --sse is deprecated and treated as an alias to Streamable HTTP.
		addr := cfg.MCPStreamableHTTPAddr
		endpoint := cfg.MCPStreamableHTTPEndpoint
		if endpoint == "" {
			// The API document names the same constant, so the server and the
			// document cannot disagree about the endpoint (RM-5).
			endpoint = transport.DefaultMCPEndpoint
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
		// The endpoint is served by transport.MCPHTTPServer rather than by
		// go-mcp's own http.Server, so the authentication layer, the principal
		// layer and the request metadata run in front of the transport and
		// reach the tool handlers. Without that a tool call over this
		// transport has no identity at all and cannot work with
		// require_principal on.
		mcpHTTPServer, err = transport.NewMCPHTTPServer(addr, endpoint, mcptransport.Stateful, streamableHTTPLogger())
		if err != nil {
			slog.Error("failed to create MCP Streamable HTTP server", "error", err)
			os.Exit(1)
		}
		t = mcpHTTPServer.Transport()
	} else {
		slog.Info("Starting MCP over stdio (default)")
		t = mcptransport.NewStdioServerTransport()
	}

	// Tenancy (RM-2, RM-10). Two decisions are made here and nowhere else.
	//
	// require_principal closes the legacy no-principal path: with it on, a
	// storage call whose context carries no principal is an error rather than
	// a call running as the synthetic local identity.
	//
	// The tool principal resolver is what makes turning it on possible. In
	// stdio mode Remembrances is the single-user memory of one agent, there is
	// no token and no tenant, so every tool call runs as the local principal,
	// which is a real principal in the legacy org rather than an absence of
	// one. In a networked mode the identity has to come from the request, so
	// no default is installed and an auth-providing module (Alchemai Mem)
	// supplies one; without it those calls have no principal and, with
	// require_principal on, fail closed rather than borrowing an identity.
	storage.SetRequirePrincipal(cfg.RequirePrincipal)
	if !cfg.MCPStreamableHTTP && !cfg.SSE {
		mcp_tools.SetDefaultPrincipalResolver(func(context.Context, *protocol.CallToolRequest) (*principal.Principal, error) {
			return storage.LocalPrincipal(), nil
		})
	}
	slog.Info("Tenancy configured",
		"require_principal", storage.RequirePrincipal(),
		"tool_principal_resolver", mcp_tools.DefaultPrincipalResolver() != nil)
	// RM-10's notice, where the operator of a legacy deployment actually meets
	// it. Silent once the deployment is on the supported line.
	if notice := storage.LegacyLineNotice(); notice != "" {
		slog.Warn(notice, "end_of_support", storage.LegacyLineEndOfSupport)
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

	// Knowledge base path validation:
	// - if configured and missing, attempt to create it (mkdir -p)
	// - if creation fails (or path is not a directory), disable all KB features
	if cfg.KnowledgeBase != "" {
		if err := os.MkdirAll(cfg.KnowledgeBase, 0755); err != nil {
			slog.Warn("knowledge base disabled; failed to create directory", "path", cfg.KnowledgeBase, "error", err)
			cfg.KnowledgeBase = ""
		} else if info, err := os.Stat(cfg.KnowledgeBase); err != nil {
			slog.Warn("knowledge base disabled; failed to stat directory", "path", cfg.KnowledgeBase, "error", err)
			cfg.KnowledgeBase = ""
		} else if !info.IsDir() {
			slog.Warn("knowledge base disabled; path is not a directory", "path", cfg.KnowledgeBase)
			cfg.KnowledgeBase = ""
		}
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

	// Start the module lifecycles. RM-4 added Start and Stop to the module
	// contract; until they are called here a module's background work (Mem's
	// refinement worker, its JWKS refresher) never runs, so the lifecycle is
	// inert no matter what a module implements. A module that fails to start
	// stops the process rather than leaving a half configured server serving
	// requests it cannot honour.
	if err := modManager.StartModules(ctx); err != nil {
		slog.Error("failed to start modules", "error", err)
		os.Exit(1)
	}

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

	// The MCP Streamable HTTP endpoint authenticates with the same providers
	// the module HTTP routes use. They are only known now, once the modules
	// have loaded, which is why the server takes them here rather than at
	// construction. With no providers the layer is the identity and the
	// endpoint behaves exactly as before.
	if mcpHTTPServer != nil {
		mcpAuthProviders := modManager.GetHTTPAuthProviders()
		mcpHTTPServer.SetAuthProviders(mcpAuthProviders)
		slog.Info("MCP Streamable HTTP authentication configured",
			"auth_providers", len(mcpAuthProviders))
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
			// The auth providers must be passed in: RegisterModuleRoutes wraps
			// every module route in the authentication layer built from them,
			// and calling it without them registers the same routes with no
			// authentication at all.
			authProviders := modManager.GetHTTPAuthProviders()
			httpTransport.RegisterModuleRoutes(httpProviders, authProviders...)
			slog.Info("Registered HTTP endpoint providers",
				"count", len(httpProviders), "auth_providers", len(authProviders))
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

		// The MCP Streamable HTTP endpoint runs on a server this process owns,
		// so it is stopped here; srv.Shutdown below unwinds the transport.
		if mcpHTTPServer != nil {
			_ = mcpHTTPServer.Shutdown(shutdownCtx)
		}

		// Stop knowledge base watcher
		if kbWatcher != nil {
			kbWatcher.Stop()
		}

		// Stop the module lifecycles before releasing their resources: Stop
		// unwinds what StartModules started, Cleanup releases what Provision
		// allocated, and doing them in that order is what lets a module drain
		// in flight work instead of losing it.
		if err := modManager.StopModules(shutdownCtx); err != nil {
			slog.Warn("failed to stop one or more modules", "error", err)
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
	hasMCPHTTP := mcpHTTPServer != nil

	// The MCP Streamable HTTP endpoint is served by this process, not by
	// go-mcp, so its listener is started here; srv.Run() below still drives
	// the transport itself.
	if hasMCPHTTP {
		go func() {
			if err := mcpHTTPServer.Serve(); err != nil && err != http.ErrServerClosed {
				slog.Error("MCP Streamable HTTP listener error", "error", err)
				os.Exit(1)
			}
		}()
	}

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
		"tools.remember",
		"tools.events",
		"tools.knowledge_graph",
		"tools.code_indexing",
		"tools.code_search",
		"tools.code_manipulation",
	}
	if cfg.KnowledgeBase != "" {
		defaultModules = append(defaultModules, "tools.kb")
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
		if id == "tools.kb" && cfg.KnowledgeBase == "" {
			continue
		}
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
