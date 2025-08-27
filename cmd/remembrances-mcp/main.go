// Package main is the entry point for the Remembrances-MCP server.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"remembrances-mcp/internal/config"
	"remembrances-mcp/internal/storage"
	"remembrances-mcp/pkg/embedder"
	"remembrances-mcp/pkg/mcp_tools"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	mcpserver "github.com/ThinkInAIXYZ/go-mcp/server"
	mcptransport "github.com/ThinkInAIXYZ/go-mcp/transport"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup logging
	if err := cfg.SetupLogging(); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting up logging: %v\n", err)
		os.Exit(1)
	}

	// Root context with graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Select transport: stdio (default) or SSE when --sse is passed or env set
	var t mcptransport.ServerTransport
	if cfg.SSE {
		addr := os.Getenv("GOMEM_SSE_ADDR")
		if addr == "" {
			addr = ":3000"
		}
		slog.Info("SSE transport enabled", "address", addr)
		t, err = mcptransport.NewSSEServerTransport(addr)
		if err != nil {
			slog.Error("failed to initialize SSE transport", "error", err)
			os.Exit(1)
		}
	} else {
		slog.Info("Starting MCP over stdio (default)")
		t = mcptransport.NewStdioServerTransport()
	}

	// Instantiate MCP server with basic metadata
	srv, err := mcpserver.NewServer(
		t,
		mcpserver.WithServerInfo(protocol.Implementation{
			Name:    "remembrances-mcp",
			Version: "0.1.0",
		}),
		mcpserver.WithInstructions(`Welcome to Remembrances-MCP Server!

This server provides a comprehensive remembrance system with three complementary layers:

🗂️ KEY-VALUE FACTS: Store simple facts, preferences, and settings that can be quickly retrieved by key
   • remembrance_save_fact: Store basic information
   • remembrance_get_fact: Retrieve by key
   • remembrance_list_facts: See all facts for a user
   • remembrance_delete_fact: Remove facts

🧠 SEMANTIC VECTORS: Store content with automatic embedding for similarity search
   • remembrance_add_vector: Add content that gets automatically embedded
   • remembrance_search_vectors: Find similar content using semantic search
   • remembrance_update_vector: Update existing content and regenerate embedding
   • remembrance_delete_vector: Remove semantic content

🕸️ KNOWLEDGE GRAPH: Create entities and relationships to model complex connections
   • remembrance_create_entity: Add people, places, concepts
   • remembrance_create_relationship: Connect entities with relationships
   • remembrance_traverse_graph: Explore connections between entities
   • remembrance_get_entity: Retrieve entity details

📚 KNOWLEDGE BASE: Store and search documents
   • kb_add_document: Add documents with automatic embedding
   • kb_search_documents: Search documents by semantic similarity
   • kb_get_document: Retrieve document by path
   • kb_delete_document: Remove documents

🔍 UNIFIED SEARCH: Combine all layers for comprehensive results
   • remembrance_hybrid_search: Search across facts, vectors, and graph simultaneously
   • remembrance_get_stats: Get overview of all stored remembrances

Choose the right tool for your data:
- Use FACTS for simple key-value data
- Use VECTORS for content you want to find by meaning
- Use GRAPH for modeling relationships and connections
- Use HYBRID SEARCH when you want comprehensive results across all layers`),
	)
	if err != nil {
		slog.Error("failed to create MCP server", "error", err)
		os.Exit(1)
	}

	// Initialize storage
	var storageInstance storage.StorageWithStats
	if cfg.SurrealDBURL != "" {
		// Use remote SurrealDB
		storageConfig := &storage.ConnectionConfig{
			URL:       cfg.SurrealDBURL,
			Username:  cfg.SurrealDBUser,
			Password:  cfg.SurrealDBPass,
			Namespace: cfg.GetSurrealDBNamespace(),
			Database:  cfg.GetSurrealDBDatabase(),
			Timeout:   30 * time.Second,
		}
		storageInstance = storage.NewSurrealDBStorage(storageConfig)
	} else {
		// Use embedded SurrealDB
		storageConfig := &storage.ConnectionConfig{
			DBPath:    cfg.DbPath,
			Namespace: cfg.GetSurrealDBNamespace(),
			Database:  cfg.GetSurrealDBDatabase(),
			Timeout:   30 * time.Second,
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

	// Initialize embedder using the main config interface
	embedderInstance, err := embedder.NewEmbedderFromMainConfig(cfg)
	if err != nil {
		slog.Error("failed to create embedder", "error", err)
		os.Exit(1)
	}

	// Register MCP tools
	toolManager := mcp_tools.NewToolManager(storageInstance, embedderInstance)
	if err := toolManager.RegisterTools(srv); err != nil {
		slog.Error("failed to register MCP tools", "error", err)
		os.Exit(1)
	}

	slog.Info("Remembrances-MCP server initialized successfully")

	// Graceful shutdown
	go func() {
		<-ctx.Done()
		slog.Info("Shutdown signal received, starting graceful shutdown")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)

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

	// Run the server (blocking)
	slog.Info("Starting Remembrances-MCP server")
	if err := srv.Run(); err != nil {
		slog.Error("server run error", "error", err)
		os.Exit(1)
	}
}
