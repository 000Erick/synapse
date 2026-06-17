package usecase

import (
	"context"

	"github.com/ediazs/synapse/internal/port"
)

// SyncResult holds the counts from a sync run.
type SyncResult struct {
	Orphans       int64 `json:"removed"`
	LivePreserved int64 `json:"live_preserved"`
}

// SyncUsecase removes orphan vectors.
type SyncUsecase struct {
	reader port.EngramReader
	store  port.VectorStore
}

// NewSyncUsecase creates a SyncUsecase.
func NewSyncUsecase(reader port.EngramReader, store port.VectorStore) *SyncUsecase {
	return &SyncUsecase{reader: reader, store: store}
}

// Run is a placeholder until slice 6 implements the real logic.
func (s *SyncUsecase) Run(ctx context.Context) (*SyncResult, error) {
	return &SyncResult{}, nil
}
