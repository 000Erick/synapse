package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ediazs/synapse/internal/domain"
)

// vec3072 builds a 3072-dim vector whose first element is set to v and the rest
// are zero. Distance between two such vectors is monotonic in |v1 - v2|, which
// makes nearest-neighbour ordering deterministic and easy to assert.
func vec3072(v float32) []float32 {
	out := make([]float32, Dims)
	out[0] = v
	return out
}

func newTestStore(t *testing.T) *SQLiteVectorStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "synapse.db")
	s, err := NewSQLiteVectorStore(path)
	if err != nil {
		t.Fatalf("NewSQLiteVectorStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestVectorStore_UpsertAndKNN_RoundTrip(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	rows := []domain.VecRow{
		{ObsID: 1, Embedding: vec3072(0.0), ContentHash: "h1", Model: "m"},
		{ObsID: 2, Embedding: vec3072(1.0), ContentHash: "h2", Model: "m"},
		{ObsID: 3, Embedding: vec3072(10.0), ContentHash: "h3", Model: "m"},
	}
	if err := s.Upsert(ctx, rows); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	// Query near obs 2 (value 1.0). Nearest should be id 2, then 1, then 3.
	res, err := s.KNN(ctx, vec3072(1.1), 3)
	if err != nil {
		t.Fatalf("KNN: %v", err)
	}
	if len(res) != 3 {
		t.Fatalf("expected 3 results, got %d", len(res))
	}
	if res[0].ID != 2 {
		t.Errorf("nearest = %d, want 2", res[0].ID)
	}
	if res[0].Source != domain.SourceVec {
		t.Errorf("source = %d, want SourceVec(%d)", res[0].Source, domain.SourceVec)
	}
	// Scores are negated distances → descending score = ascending distance.
	if !(res[0].Score >= res[1].Score && res[1].Score >= res[2].Score) {
		t.Errorf("scores not in descending order: %v", res)
	}
}

func TestVectorStore_KNN_HonorsK(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	for i := int64(1); i <= 5; i++ {
		if err := s.Upsert(ctx, []domain.VecRow{
			{ObsID: i, Embedding: vec3072(float32(i)), ContentHash: "h", Model: "m"},
		}); err != nil {
			t.Fatalf("Upsert %d: %v", i, err)
		}
	}
	res, err := s.KNN(ctx, vec3072(0), 2)
	if err != nil {
		t.Fatalf("KNN: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 results, got %d", len(res))
	}
}

func TestVectorStore_Upsert_IsIdempotent(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	row := domain.VecRow{ObsID: 7, Embedding: vec3072(2.0), ContentHash: "hA", Model: "m"}
	if err := s.Upsert(ctx, []domain.VecRow{row}); err != nil {
		t.Fatalf("Upsert 1: %v", err)
	}
	// Re-upsert same id with a new hash → no duplicate, hash updated.
	row.ContentHash = "hB"
	if err := s.Upsert(ctx, []domain.VecRow{row}); err != nil {
		t.Fatalf("Upsert 2: %v", err)
	}

	n, err := s.CountVectors(ctx)
	if err != nil {
		t.Fatalf("CountVectors: %v", err)
	}
	if n != 1 {
		t.Errorf("count = %d, want 1 (no duplicate)", n)
	}
	hashes, err := s.Hashes(ctx)
	if err != nil {
		t.Fatalf("Hashes: %v", err)
	}
	if hashes[7] != "hB" {
		t.Errorf("hash = %q, want hB (updated)", hashes[7])
	}
}

func TestVectorStore_DeleteByIDs(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	if err := s.Upsert(ctx, []domain.VecRow{
		{ObsID: 1, Embedding: vec3072(1), ContentHash: "h1", Model: "m"},
		{ObsID: 2, Embedding: vec3072(2), ContentHash: "h2", Model: "m"},
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := s.DeleteByIDs(ctx, []int64{1}); err != nil {
		t.Fatalf("DeleteByIDs: %v", err)
	}
	n, err := s.CountVectors(ctx)
	if err != nil {
		t.Fatalf("CountVectors: %v", err)
	}
	if n != 1 {
		t.Errorf("count = %d, want 1", n)
	}
	ids, err := s.AllObsIDs(ctx)
	if err != nil {
		t.Fatalf("AllObsIDs: %v", err)
	}
	if len(ids) != 1 || ids[0] != 2 {
		t.Errorf("remaining ids = %v, want [2]", ids)
	}
}

func TestVectorStore_CountOrphans(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	if err := s.Upsert(ctx, []domain.VecRow{
		{ObsID: 1, Embedding: vec3072(1), ContentHash: "h1", Model: "m"},
		{ObsID: 2, Embedding: vec3072(2), ContentHash: "h2", Model: "m"},
		{ObsID: 3, Embedding: vec3072(3), ContentHash: "h3", Model: "m"},
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	// Live IDs are 1 and 3 → obs 2 is an orphan.
	orphans, err := s.CountOrphans(ctx, []int64{1, 3})
	if err != nil {
		t.Fatalf("CountOrphans: %v", err)
	}
	if orphans != 1 {
		t.Errorf("orphans = %d, want 1", orphans)
	}
}
