//go:build tools

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/madeindigio/remembrances-mcp/internal/config"
	"github.com/madeindigio/remembrances-mcp/internal/kb"
	"github.com/madeindigio/remembrances-mcp/internal/storage"
	"github.com/madeindigio/remembrances-mcp/pkg/embedder"
)

func main() {
	// Load config
	cfg, err := config.LoadConfig("/tmp/test-config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create storage
	st, err := storage.NewSurrealDBStorage(cfg)
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}

	ctx := context.Background()
	if err := st.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer st.Close()

	if err := st.InitializeSchema(ctx); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	// Create embedder
	emb, err := embedder.NewEmbedder(cfg)
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}

	// Start KB watcher
	watcher, err := kb.StartWatcher(ctx, cfg.GetKBPath(), st, emb, cfg.GetChunkSize(), cfg.GetChunkOverlap())
	if err != nil {
		log.Fatalf("Failed to start watcher: %v", err)
	}
	defer watcher.Stop()

	// Wait for processing
	fmt.Println("Waiting for document processing (15 seconds)...")
	time.Sleep(15 * time.Second)

	// Query to check stored chunks
	fmt.Println("\n=== Verification Complete ===")
	fmt.Println("Check the logs above for 'kb document synced' messages")
	fmt.Println("Expected: chunks=4 or chunks=5 for a 2164 byte document with chunk-size=500")
}
