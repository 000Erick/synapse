package usecase

import (
	"context"
	"errors"
	"sync"

	"github.com/ediazs/synapse/internal/domain"
	"github.com/ediazs/synapse/internal/port"
)

const (
	rrfK            = 60
	defaultLimit    = 10
	candidateFactor = 5 // pull this many * limit candidates from each source
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

// Run executes a hybrid search. It runs FTS5 (lexical) and vector KNN (semantic)
// concurrently, fuses them with RRF (k=60), and hydrates the top results from
// Engram. The bool return reports whether the vector path was used (false when
// it was unavailable and the result is FTS-only).
func (s *SearchUsecase) Run(ctx context.Context, query string, limit int) ([]domain.SearchResult, bool, error) {
	if query == "" {
		return nil, false, errors.New("query must not be empty")
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	candidates := limit * candidateFactor

	var (
		wg         sync.WaitGroup
		ftsHits    []domain.Ranked
		ftsErr     error
		vecHits    []domain.Ranked
		vectorUsed bool
	)

	// FTS path (always attempted).
	wg.Add(1)
	go func() {
		defer wg.Done()
		ftsHits, ftsErr = s.reader.FTS(ctx, query, candidates)
	}()

	// Vector path (only when an API key is configured; otherwise FTS-only).
	if s.apiKey != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			vecs, err := s.embedder.Embed(ctx, []string{query})
			if err != nil || len(vecs) == 0 {
				return // non-fatal: degrade to FTS-only
			}
			hits, err := s.store.KNN(ctx, vecs[0], candidates)
			if err != nil {
				return // non-fatal: degrade to FTS-only
			}
			vecHits = hits
			vectorUsed = true
		}()
	}

	wg.Wait()

	if ftsErr != nil {
		return nil, false, ftsErr
	}
	// Vector failure is non-fatal: degrade to FTS-only rather than erroring.

	lists := [][]domain.Ranked{ftsHits}
	if vectorUsed {
		lists = append(lists, vecHits)
	}
	fused := domain.RRF(lists, rrfK)
	if len(fused) > limit {
		fused = fused[:limit]
	}

	results, err := s.hydrate(ctx, fused)
	if err != nil {
		return nil, vectorUsed, err
	}
	return results, vectorUsed, nil
}

// hydrate turns ranked IDs into SearchResults by looking up titles/content from
// the live Engram observations.
func (s *SearchUsecase) hydrate(ctx context.Context, ranked []domain.Ranked) ([]domain.SearchResult, error) {
	if len(ranked) == 0 {
		return []domain.SearchResult{}, nil
	}
	obs, err := s.reader.LiveObservations(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]domain.Observation, len(obs))
	for _, o := range obs {
		byID[o.ID] = o
	}

	out := make([]domain.SearchResult, 0, len(ranked))
	for _, r := range ranked {
		o, ok := byID[r.ID]
		if !ok {
			continue // ranked id no longer live; skip
		}
		out = append(out, domain.SearchResult{
			ID:      o.ID,
			Title:   o.Title,
			Snippet: snippet(o.Content, 200),
			Score:   r.Score,
			Source:  r.Source,
		})
	}
	return out, nil
}

func snippet(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
