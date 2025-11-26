package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/madeindigio/remembrances-mcp/internal/storage"
	embedderpkg "github.com/madeindigio/remembrances-mcp/pkg/embedder"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: reprocess-knowledge-base <kb-path> <gguf-model-path>")
		fmt.Println("Example: ./reprocess-knowledge-base .serena/memories /www/Remembrances/nomic-embed-text-v1.5.Q4_K_M.gguf")
		os.Exit(1)
	}

	kbPath := os.Args[1]
	modelPath := os.Args[2]

	ctx := context.Background()

	// Create storage
	cfg := &storage.ConnectionConfig{
		DBPath:    "surrealkv://~/www/MCP/remembrances-mcp/remembrances.db",
		Namespace: "test",
		Database:  "test",
	}

	store := storage.NewSurrealDBStorage(cfg)
	if err := store.Connect(ctx); err != nil {
		log.Fatal("Failed to connect to storage:", err)
	}
	defer store.Close()

	// Create embedder
	emb, err := embedderpkg.NewGGUFEmbedder(modelPath, 0, 32) // 0 threads (auto), 32 GPU layers
	if err != nil {
		log.Fatal("Failed to create embedder:", err)
	}
	defer emb.Close()

	fmt.Printf("Scanning knowledge base: %s\n", kbPath)

	// Find all .md files
	var files []string
	err = filepath.Walk(kbPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		log.Fatal("Failed to scan kb directory:", err)
	}

	fmt.Printf("Found %d markdown files\n\n", len(files))

	// Process each file
	for i, fullPath := range files {
		rel, _ := filepath.Rel(kbPath, fullPath)
		fmt.Printf("[%d/%d] Processing: %s\n", i+1, len(files), rel)

		// Read file
		content, err := os.ReadFile(fullPath)
		if err != nil {
			log.Printf("  ✗ Failed to read: %v\n", err)
			continue
		}

		fileInfo, _ := os.Stat(fullPath)
		contentStr := string(content)

		// Generate chunks and embeddings
		chunks, embeddings, err := embedderpkg.EmbedTextChunksWithOverlap(ctx, emb, contentStr, 1500, 200)
		if err != nil {
			log.Printf("  ✗ Failed to generate embeddings: %v\n", err)
			continue
		}

		// Save with proper metadata
		metadata := map[string]interface{}{
			"source":        "reprocess-script",
			"total_size":    len(content),
			"last_modified": fileInfo.ModTime().Format("2006-01-02T15:04:05Z07:00"),
		}

		if err := store.SaveDocumentChunks(ctx, rel, chunks, embeddings, metadata); err != nil {
			log.Printf("  ✗ Failed to save: %v\n", err)
			continue
		}

		fmt.Printf("  ✓ Saved %d chunks\n", len(chunks))
	}

	fmt.Println("\n✅ Reprocessing complete!")
}
