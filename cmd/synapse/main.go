package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ediazs/synapse/internal/config"
	"github.com/ediazs/synapse/internal/embed"
	"github.com/ediazs/synapse/internal/engram"
	"github.com/ediazs/synapse/internal/mcp"
	"github.com/ediazs/synapse/internal/store"
)

func main() {
	// Load .env for local development. Missing file is not an error — in
	// production config comes from the real environment.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Printf("synapse: config: %v", err)
		os.Exit(1)
	}

	engramReader, err := engram.NewSQLiteEngramReader(cfg.EngramDBPath)
	if err != nil {
		log.Printf("synapse: warn: engram not reachable: %v", err)
		// Continue — health tool will report degraded
	}
	if engramReader != nil {
		defer engramReader.Close()
	}

	if err := os.MkdirAll(filepath.Dir(cfg.SynapseDBPath), 0o755); err != nil {
		log.Printf("synapse: warn: cannot create synapse dir: %v", err)
	}
	vecStore, err := store.NewSQLiteVectorStore(cfg.SynapseDBPath)
	if err != nil {
		log.Printf("synapse: vector store: %v", err)
		os.Exit(1)
	}
	defer vecStore.Close()

	embedder := embed.NewOpenAIEmbedder(cfg.OpenAIAPIKey, cfg.OpenAIEmbedModel)

	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "synapse",
		Version: "0.1.0",
	}, nil)

	deps := &mcp.Deps{
		EngramPath:  cfg.EngramDBPath,
		SynapsePath: cfg.SynapseDBPath,
		Reader:      engramReader,
		Store:       vecStore,
		Embedder:    embedder,
		Model:       cfg.OpenAIEmbedModel,
		APIKey:      cfg.OpenAIAPIKey,
	}
	mcp.RegisterTools(server, deps)

	if err := server.Run(context.Background(), &sdkmcp.StdioTransport{}); err != nil {
		log.Printf("synapse: server error: %v", err)
	}
}
