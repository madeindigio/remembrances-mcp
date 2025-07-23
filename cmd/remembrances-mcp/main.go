// Package main is the entry point for the Remembrances-MCP server.
package main

import (
	"fmt"
	"os"

	"remembrances-mcp/internal/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Use the configuration
	if cfg.SSE {
		fmt.Println("SSE transport enabled")
	}

	if cfg.RestAPIServe {
		fmt.Printf("Serving REST API\n")
	}

	if cfg.KnowledgeBase != "" {
		fmt.Printf("Using knowledge base at %s\n", cfg.KnowledgeBase)
	}

	if cfg.DbPath != "" {
		fmt.Printf("Using database at %s\n", cfg.DbPath)
	} else if cfg.SurrealDBURL != "" {
		fmt.Printf("Connecting to SurrealDB at %s\n", cfg.SurrealDBURL)
	}

	if cfg.OllamaModel != "" {
		fmt.Printf("Using Ollama with model %s\n", cfg.OllamaModel)
	} else if cfg.OpenAIKey != "" {
		fmt.Printf("Using OpenAI with model %s\n", cfg.OpenAIModel)
	}
}
