// Package main is the entry point for the Remembrances-MCP server.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"remembrances-mcp/internal/config"

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
		log.Printf("SSE transport enabled, listening on %s", addr)
		t, err = mcptransport.NewSSEServerTransport(addr)
		if err != nil {
			log.Fatalf("failed to initialize SSE transport: %v", err)
		}
	} else {
		log.Println("Starting MCP over stdio (default)")
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
		log.Fatalf("failed to create MCP server: %v", err)
	}

	// TODO: Register tools (Mem0-like ops) and resources here

	// Graceful shutdown
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	// Run the server (blocking)
	if err := srv.Run(); err != nil {
		log.Fatalf("server run error: %v", err)
	}
}
