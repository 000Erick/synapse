package usecase

import (
	"context"
	"fmt"

	"github.com/ediazs/synapse/internal/port"
)

// StatsResult is the output of StatsUsecase.Run.
type StatsResult struct {
	LiveObservations int64
	Vectors          int64
	Orphans          int64
	Dims             int
	Model            string
}

// StatsUsecase collects statistics from both databases.
type StatsUsecase struct {
	store    port.VectorStore
	embedder port.Embedder
	model    string
}

// NewStatsUsecase creates a new StatsUsecase.
func NewStatsUsecase(store port.VectorStore, embedder port.Embedder, model string) *StatsUsecase {
	return &StatsUsecase{store: store, embedder: embedder, model: model}
}

// Run fetches statistics. liveObsCount is provided by the caller (from EngramReader).
func (s *StatsUsecase) Run(ctx context.Context, liveObsCount int64) (*StatsResult, error) {
	vectors, err := s.store.CountVectors(ctx)
	if err != nil {
		return nil, fmt.Errorf("stats: CountVectors: %w", err)
	}

	// For orphan count we need live IDs; we pass nil and the noop returns 0.
	orphans, err := s.store.CountOrphans(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("stats: CountOrphans: %w", err)
	}

	return &StatsResult{
		LiveObservations: liveObsCount,
		Vectors:          vectors,
		Orphans:          orphans,
		Dims:             s.embedder.Dims(),
		Model:            s.model,
	}, nil
}
