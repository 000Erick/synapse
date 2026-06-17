package usecase

import (
	"context"
	"errors"

	"github.com/ediazs/synapse/internal/port"
)

// ErrNoAPIKey is returned when OPENAI_API_KEY is not set.
var ErrNoAPIKey = errors.New("OPENAI_API_KEY is not set")

// BackfillResult holds the counts from a backfill run.
type BackfillResult struct {
	Embedded int64 `json:"embedded"`
	Skipped  int64 `json:"skipped"`
	Failed   int64 `json:"failed"`
}

// BackfillUsecase embeds live observations idempotently.
type BackfillUsecase struct {
	reader   port.EngramReader
	store    port.VectorStore
	embedder port.Embedder
	apiKey   string
}

// NewBackfillUsecase creates a BackfillUsecase.
func NewBackfillUsecase(reader port.EngramReader, store port.VectorStore, embedder port.Embedder, apiKey string) *BackfillUsecase {
	return &BackfillUsecase{reader: reader, store: store, embedder: embedder, apiKey: apiKey}
}

// Run is a placeholder until slice 4 implements the real logic.
func (b *BackfillUsecase) Run(ctx context.Context) (*BackfillResult, error) {
	if b.apiKey == "" {
		return nil, ErrNoAPIKey
	}
	return &BackfillResult{}, nil
}
