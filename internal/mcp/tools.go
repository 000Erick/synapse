package mcp

import (
	"context"
	"encoding/json"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ediazs/synapse/internal/port"
	"github.com/ediazs/synapse/internal/usecase"
)

// Deps holds dependencies for MCP tool handlers.
type Deps struct {
	EngramPath  string
	SynapsePath string
	Reader      port.EngramReader
	Store       port.VectorStore
	Embedder    port.Embedder
	Model       string
	APIKey      string
}

// RegisterTools registers all five synapse MCP tools on the server.
func RegisterTools(server *sdkmcp.Server, d *Deps) {
	addHealthTool(server, d)
	addStatsTool(server, d)
	addBackfillTool(server, d)
	addSearchTool(server, d)
	addSyncTool(server, d)
}

// healthArgs has no required parameters.
type healthArgs struct{}

type healthOut struct {
	Status           string `json:"status"`
	EngramReachable  bool   `json:"engram_reachable"`
	SynapseReachable bool   `json:"synapse_reachable"`
	EngramPath       string `json:"engram_path"`
	SynapsePath      string `json:"synapse_path"`
	Version          string `json:"version"`
}

func addHealthTool(server *sdkmcp.Server, d *Deps) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "synapse_health",
		Description: "Liveness check and DB reachability for both engram.db and synapse.db.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, _ healthArgs) (*sdkmcp.CallToolResult, any, error) {
		uc := usecase.NewHealthUsecase(d.EngramPath, d.SynapsePath)
		r, err := uc.Run(ctx)
		if err != nil {
			return nil, nil, err
		}
		out := healthOut{
			Status:           r.Status,
			EngramReachable:  r.EngramReachable,
			SynapseReachable: r.SynapseReachable,
			EngramPath:       r.EngramPath,
			SynapsePath:      r.SynapsePath,
			Version:          r.Version,
		}
		b, _ := json.Marshal(out)
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: string(b)}},
		}, nil, nil
	})
}

type statsArgs struct{}

func addStatsTool(server *sdkmcp.Server, d *Deps) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "synapse_stats",
		Description: "Returns observation count, vector count, orphan count, dims, and model.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, _ statsArgs) (*sdkmcp.CallToolResult, any, error) {
		var liveCount int64
		if d.Reader != nil {
			obs, err := d.Reader.LiveObservations(ctx)
			if err == nil {
				liveCount = int64(len(obs))
			}
		}

		// Get live IDs for orphan calculation
		var liveIDs []int64
		if d.Reader != nil {
			liveIDs, _ = d.Reader.LiveIDs(ctx)
		}

		vectors, _ := d.Store.CountVectors(ctx)
		orphans, _ := d.Store.CountOrphans(ctx, liveIDs)

		out := map[string]any{
			"live_observations": liveCount,
			"vectors":           vectors,
			"orphans":           orphans,
			"dims":              d.Embedder.Dims(),
			"model":             d.Model,
		}
		b, _ := json.Marshal(out)
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: string(b)}},
		}, nil, nil
	})
}

type backfillArgs struct{}

func addBackfillTool(server *sdkmcp.Server, d *Deps) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "synapse_backfill",
		Description: "Idempotent batch-embed of all live Engram observations into synapse.db.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, _ backfillArgs) (*sdkmcp.CallToolResult, any, error) {
		if d.APIKey == "" {
			b, _ := json.Marshal(map[string]string{"error": "OPENAI_API_KEY is not set"})
			return &sdkmcp.CallToolResult{
				IsError: true,
				Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: string(b)}},
			}, nil, nil
		}
		uc := usecase.NewBackfillUsecase(d.Reader, d.Store, d.Embedder, d.APIKey, d.Model)
		result, err := uc.Run(ctx)
		if err != nil {
			return nil, nil, err
		}
		b, _ := json.Marshal(result)
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: string(b)}},
		}, nil, nil
	})
}

type searchArgs struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

func addSearchTool(server *sdkmcp.Server, d *Deps) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "synapse_search",
		Description: "Hybrid FTS5+KNN semantic search with RRF fusion. Returns ranked results.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, args searchArgs) (*sdkmcp.CallToolResult, any, error) {
		if d.APIKey == "" && d.Reader == nil {
			b, _ := json.Marshal(map[string]string{"error": "OPENAI_API_KEY is not set"})
			return &sdkmcp.CallToolResult{
				IsError: true,
				Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: string(b)}},
			}, nil, nil
		}
		limit := args.Limit
		if limit <= 0 {
			limit = 10
		}
		uc := usecase.NewSearchUsecase(d.Reader, d.Store, d.Embedder, d.APIKey)
		results, vectorUnavailable, err := uc.Run(ctx, args.Query, limit)
		if err != nil {
			return nil, nil, err
		}
		out := map[string]any{
			"results":            results,
			"vector_unavailable": vectorUnavailable,
		}
		b, _ := json.Marshal(out)
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: string(b)}},
		}, nil, nil
	})
}

type syncArgs struct{}

func addSyncTool(server *sdkmcp.Server, d *Deps) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "synapse_sync",
		Description: "Detects and removes orphan vectors whose observations were deleted from Engram.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, _ syncArgs) (*sdkmcp.CallToolResult, any, error) {
		uc := usecase.NewSyncUsecase(d.Reader, d.Store)
		result, err := uc.Run(ctx)
		if err != nil {
			return nil, nil, err
		}
		b, _ := json.Marshal(result)
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: string(b)}},
		}, nil, nil
	})
}
