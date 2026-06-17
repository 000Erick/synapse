// Command smoke runs a real backfill + search against the live Engram DB.
// It is a manual tool, not part of the test suite. Requires OPENAI_API_KEY.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"

	"github.com/ediazs/synapse/internal/config"
	"github.com/ediazs/synapse/internal/embed"
	"github.com/ediazs/synapse/internal/engram"
	"github.com/ediazs/synapse/internal/store"
	"github.com/ediazs/synapse/internal/usecase"
)

func main() {
	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.OpenAIAPIKey == "" {
		log.Fatal("OPENAI_API_KEY not set")
	}

	ctx := context.Background()

	reader, err := engram.NewSQLiteEngramReader(cfg.EngramDBPath)
	if err != nil {
		log.Fatalf("engram: %v", err)
	}
	defer reader.Close()

	if err := os.MkdirAll(filepath.Dir(cfg.SynapseDBPath), 0o755); err != nil {
		log.Fatalf("mkdir: %v", err)
	}
	vstore, err := store.NewSQLiteVectorStore(cfg.SynapseDBPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer vstore.Close()

	embedder := embed.NewOpenAIEmbedder(cfg.OpenAIAPIKey, cfg.OpenAIEmbedModel)

	// --- BACKFILL ---
	fmt.Println("== backfill starting ==")
	t0 := time.Now()
	bf := usecase.NewBackfillUsecase(reader, vstore, embedder, cfg.OpenAIAPIKey, cfg.OpenAIEmbedModel)
	res, err := bf.Run(ctx)
	if err != nil {
		log.Fatalf("backfill: %v", err)
	}
	fmt.Printf("backfill done in %s: embedded=%d skipped=%d failed=%d\n",
		time.Since(t0).Round(time.Millisecond), res.Embedded, res.Skipped, res.Failed)

	n, _ := vstore.CountVectors(ctx)
	fmt.Printf("vectors stored: %d\n", n)

	// --- SEARCH ---
	queries := os.Args[1:]
	if len(queries) == 0 {
		queries = []string{"profesor", "zombie", "arquitectura limpia"}
	}
	search := usecase.NewSearchUsecase(reader, vstore, embedder, cfg.OpenAIAPIKey)
	for _, q := range queries {
		fmt.Printf("\n== search %q ==\n", q)
		hits, vectorUsed, err := search.Run(ctx, q, 5)
		if err != nil {
			fmt.Printf("  error: %v\n", err)
			continue
		}
		fmt.Printf("  vectorUsed=%v, %d hits:\n", vectorUsed, len(hits))
		for i, h := range hits {
			src := map[int]string{1: "fts", 2: "vec", 3: "both"}[int(h.Source)]
			fmt.Printf("  %d. [%s score=%.4f] #%d %s\n", i+1, src, h.Score, h.ID, h.Title)
		}
	}
}
