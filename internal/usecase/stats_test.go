package usecase_test

import (
	"context"
	"testing"

	"github.com/ediazs/synapse/internal/embed"
	"github.com/ediazs/synapse/internal/store"
	"github.com/ediazs/synapse/internal/usecase"
)

func TestStatsUsecase_ReturnsAllFields(t *testing.T) {
	noopStore := &store.NoopVectorStore{}
	noopEmbed := &embed.NoopEmbedder{}

	uc := usecase.NewStatsUsecase(noopStore, noopEmbed, "text-embedding-3-large")
	result, err := uc.Run(context.Background(), 5)
	if err != nil {
		t.Fatalf("stats.Run: %v", err)
	}
	if result.LiveObservations != 5 {
		t.Errorf("expected live_observations=5, got %d", result.LiveObservations)
	}
	if result.Vectors != 0 {
		t.Errorf("expected vectors=0, got %d", result.Vectors)
	}
	if result.Dims != 3072 {
		t.Errorf("expected dims=3072, got %d", result.Dims)
	}
	if result.Model != "text-embedding-3-large" {
		t.Errorf("expected model=text-embedding-3-large, got %q", result.Model)
	}
}

func TestStatsUsecase_ZeroVectors(t *testing.T) {
	noopStore := &store.NoopVectorStore{}
	noopEmbed := &embed.NoopEmbedder{}

	uc := usecase.NewStatsUsecase(noopStore, noopEmbed, "text-embedding-3-large")
	result, err := uc.Run(context.Background(), 0)
	if err != nil {
		t.Fatalf("stats.Run: %v", err)
	}
	if result.LiveObservations != 0 {
		t.Errorf("expected live_observations=0, got %d", result.LiveObservations)
	}
	if result.Orphans != 0 {
		t.Errorf("expected orphans=0, got %d", result.Orphans)
	}
}
