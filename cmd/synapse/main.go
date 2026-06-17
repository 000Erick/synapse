package main

import (
	"context"
	"log"
	"os"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ediazs/synapse/internal/config"
	"github.com/ediazs/synapse/internal/embed"
	"github.com/ediazs/synapse/internal/engram"
	"github.com/ediazs/synapse/internal/mcp"
	"github.com/ediazs/synapse/internal/store"
)

func main() {
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

	noopStore := &store.NoopVectorStore{}
	noopEmbed := &embed.NoopEmbedder{}

	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "synapse",
		Version: "0.1.0",
	}, nil)

	deps := &mcp.Deps{
		EngramPath:  cfg.EngramDBPath,
		SynapsePath: cfg.SynapseDBPath,
		Reader:      engramReader,
		Store:       noopStore,
		Embedder:    noopEmbed,
		Model:       cfg.OpenAIEmbedModel,
		APIKey:      cfg.OpenAIAPIKey,
	}
	mcp.RegisterTools(server, deps)

	if err := server.Run(context.Background(), &sdkmcp.StdioTransport{}); err != nil {
		log.Printf("synapse: server error: %v", err)
	}
}
