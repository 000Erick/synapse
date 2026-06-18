package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sqlitevec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"

	"github.com/ediazs/synapse/internal/domain"
)

func init() {
	// Register the sqlite-vec extension for every SQLite connection opened in
	// this process. Safe to call once at package load.
	sqlitevec.Auto()
}

// SQLiteVectorStore is a sqlite-vec backed implementation of port.VectorStore.
// It owns synapse.db and never touches engram.db.
type SQLiteVectorStore struct {
	db *sql.DB
}

// NewSQLiteVectorStore opens (or creates) synapse.db at path and ensures the
// vec0 table and companion metadata table exist. WAL mode and a 5-second busy
// timeout are set so concurrent readers and the background backfill goroutine
// do not starve each other.
func NewSQLiteVectorStore(path string) (*SQLiteVectorStore, error) {
	dsn := path + "?_journal_mode=WAL&_busy_timeout=5000"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: open: %w", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: ping: %w", err)
	}
	if _, err := db.Exec(schemaVecObs); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: create vec_obs: %w", err)
	}
	if _, err := db.Exec(schemaObsMeta); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: create obs_meta: %w", err)
	}
	return &SQLiteVectorStore{db: db}, nil
}

// Close closes the underlying DB connection.
func (s *SQLiteVectorStore) Close() error {
	return s.db.Close()
}

// KNN returns up to k nearest neighbours to vec, ordered by ascending distance.
// The returned Ranked.Score is the negated distance so that higher = closer,
// consistent with the FTS path (which negates bm25).
func (s *SQLiteVectorStore) KNN(ctx context.Context, vec []float32, k int) ([]domain.Ranked, error) {
	blob, err := sqlitevec.SerializeFloat32(vec)
	if err != nil {
		return nil, fmt.Errorf("store: serialize query: %w", err)
	}
	const q = `SELECT obs_id, distance
               FROM vec_obs
               WHERE embedding MATCH ?
                 AND k = ?
               ORDER BY distance`
	rows, err := s.db.QueryContext(ctx, q, blob, k)
	if err != nil {
		return nil, fmt.Errorf("store: KNN: %w", err)
	}
	defer rows.Close()

	var out []domain.Ranked
	for rows.Next() {
		var (
			id   int64
			dist float64
		)
		if err := rows.Scan(&id, &dist); err != nil {
			return nil, fmt.Errorf("store: KNN scan: %w", err)
		}
		out = append(out, domain.Ranked{ID: id, Score: -dist, Source: domain.SourceVec})
	}
	return out, rows.Err()
}

// Upsert inserts or replaces vectors and their metadata in a single
// transaction. Re-upserting the same obs_id overwrites the prior vector/hash.
func (s *SQLiteVectorStore) Upsert(ctx context.Context, rows []domain.VecRow) error {
	if len(rows) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin: %w", err)
	}
	defer tx.Rollback()

	for _, r := range rows {
		blob, err := sqlitevec.SerializeFloat32(r.Embedding)
		if err != nil {
			return fmt.Errorf("store: serialize obs %d: %w", r.ObsID, err)
		}
		// vec0 virtual tables do not support UPSERT syntax; delete then insert.
		if _, err := tx.ExecContext(ctx, `DELETE FROM vec_obs WHERE obs_id = ?`, r.ObsID); err != nil {
			return fmt.Errorf("store: del vec %d: %w", r.ObsID, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO vec_obs(obs_id, embedding) VALUES (?, ?)`, r.ObsID, blob); err != nil {
			return fmt.Errorf("store: ins vec %d: %w", r.ObsID, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO obs_meta(obs_id, content_hash, model, created_at)
            VALUES (?, ?, ?, ?)
            ON CONFLICT(obs_id) DO UPDATE SET content_hash = excluded.content_hash, model = excluded.model`,
			r.ObsID, r.ContentHash, r.Model, time.Now().UTC().Format(time.RFC3339)); err != nil {
			return fmt.Errorf("store: ins meta %d: %w", r.ObsID, err)
		}
	}
	return tx.Commit()
}

// Hashes returns a map of obs_id -> content_hash for all stored vectors, used
// by backfill to skip unchanged observations.
func (s *SQLiteVectorStore) Hashes(ctx context.Context) (map[int64]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT obs_id, content_hash FROM obs_meta`)
	if err != nil {
		return nil, fmt.Errorf("store: Hashes: %w", err)
	}
	defer rows.Close()

	out := make(map[int64]string)
	for rows.Next() {
		var (
			id int64
			h  string
		)
		if err := rows.Scan(&id, &h); err != nil {
			return nil, fmt.Errorf("store: Hashes scan: %w", err)
		}
		out[id] = h
	}
	return out, rows.Err()
}

// DeleteByIDs removes vectors and metadata for the given obs_ids in one tx.
func (s *SQLiteVectorStore) DeleteByIDs(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin: %w", err)
	}
	defer tx.Rollback()

	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, `DELETE FROM vec_obs WHERE obs_id = ?`, id); err != nil {
			return fmt.Errorf("store: del vec %d: %w", id, err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM obs_meta WHERE obs_id = ?`, id); err != nil {
			return fmt.Errorf("store: del meta %d: %w", id, err)
		}
	}
	return tx.Commit()
}

// CountVectors returns the number of stored vectors.
func (s *SQLiteVectorStore) CountVectors(ctx context.Context) (int64, error) {
	var n int64
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM obs_meta`).Scan(&n); err != nil {
		return 0, fmt.Errorf("store: CountVectors: %w", err)
	}
	return n, nil
}

// CountOrphans returns how many stored vectors have an obs_id that is NOT in
// the given set of live IDs.
func (s *SQLiteVectorStore) CountOrphans(ctx context.Context, liveIDs []int64) (int64, error) {
	all, err := s.AllObsIDs(ctx)
	if err != nil {
		return 0, err
	}
	live := make(map[int64]struct{}, len(liveIDs))
	for _, id := range liveIDs {
		live[id] = struct{}{}
	}
	var orphans int64
	for _, id := range all {
		if _, ok := live[id]; !ok {
			orphans++
		}
	}
	return orphans, nil
}

// AllObsIDs returns every obs_id currently stored.
func (s *SQLiteVectorStore) AllObsIDs(ctx context.Context) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT obs_id FROM obs_meta ORDER BY obs_id`)
	if err != nil {
		return nil, fmt.Errorf("store: AllObsIDs: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("store: AllObsIDs scan: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
