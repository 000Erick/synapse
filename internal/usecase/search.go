package usecase

import (
	"context"
	"errors"

	"github.com/ediazs/synapse/internal/domain"
	"github.com/ediazs/synapse/internal/port"
)

// SearchUsecase performs hybrid FTS5+KNN search with RRF fusion.
type SearchUsecase struct {
	reader   port.EngramReader
	store    port.VectorStore
	embedder port.Embedder
	apiKey   string
}

// NewSearchUsecase creates a SearchUsecase.
func NewSearchUsecase(reader port.EngramReader, store port.VectorStore, embedder port.Embedder, apiKey string) *SearchUsecase {
	return &SearchUsecase{reader: reader, store: store, embedder: embedder, apiKey: apiKey}
}

// Run is a placeholder until slice 5 implements the real logic.
func (s *SearchUsecase) Run(ctx context.Context, query string, limit int) ([]domain.SearchResult, bool, error) {
	if query == "" {
		return nil, false, errors.New("query must not be empty")
	}
	return nil, false, nil
}
