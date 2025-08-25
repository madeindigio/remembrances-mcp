// Package main is the entry point for the Remembrances-MCP server.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
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
		mcpserver.WithInstructions("Remembrances-MCP server is ready."),
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

	// Connect to storage
	if err := storageInstance.Connect(ctx); err != nil {
		slog.Error("failed to connect to storage", "error", err)
		os.Exit(1)
	}
	defer storageInstance.Close()

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
	}()

	// Run the server (blocking)
	slog.Info("Starting Remembrances-MCP server")
	if err := srv.Run(); err != nil {
		slog.Error("server run error", "error", err)
		os.Exit(1)
	}
}
