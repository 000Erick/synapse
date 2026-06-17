package engram

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"

	"github.com/ediazs/synapse/internal/domain"
)

// SQLiteEngramReader is a read-only adapter over engram.db.
type SQLiteEngramReader struct {
	db *sql.DB
}

// NewSQLiteEngramReader opens engram.db at path in read-only WAL mode.
func NewSQLiteEngramReader(path string) (*SQLiteEngramReader, error) {
	dsn := fmt.Sprintf("file:%s?mode=ro", path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("engram: open: %w", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("engram: ping: %w", err)
	}
	return &SQLiteEngramReader{db: db}, nil
}

// Close closes the underlying DB connection.
func (r *SQLiteEngramReader) Close() error {
	return r.db.Close()
}

// LiveObservations returns all non-deleted observations.
func (r *SQLiteEngramReader) LiveObservations(ctx context.Context) ([]domain.Observation, error) {
	const q = `SELECT id, title, content, project, scope, type, topic_key, updated_at
               FROM observations
               WHERE deleted_at IS NULL`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("engram: LiveObservations: %w", err)
	}
	defer rows.Close()

	var obs []domain.Observation
	for rows.Next() {
		var o domain.Observation
		if err := rows.Scan(&o.ID, &o.Title, &o.Content, &o.Project, &o.Scope, &o.Type, &o.TopicKey, &o.UpdatedAt); err != nil {
			return nil, fmt.Errorf("engram: LiveObservations scan: %w", err)
		}
		obs = append(obs, o)
	}
	return obs, rows.Err()
}

// LiveIDs returns the IDs of all non-deleted observations.
func (r *SQLiteEngramReader) LiveIDs(ctx context.Context) ([]int64, error) {
	const q = `SELECT id FROM observations WHERE deleted_at IS NULL`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("engram: LiveIDs: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("engram: LiveIDs scan: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// FTS performs an FTS5 MATCH search and returns up to k results in BM25 order.
func (r *SQLiteEngramReader) FTS(ctx context.Context, query string, k int) ([]domain.Ranked, error) {
	const q = `SELECT o.id, -bm25(observations_fts) AS score
               FROM observations_fts
               JOIN observations o ON o.id = observations_fts.rowid
               WHERE observations_fts MATCH ?
                 AND o.deleted_at IS NULL
               ORDER BY bm25(observations_fts)
               LIMIT ?`
	rows, err := r.db.QueryContext(ctx, q, query, k)
	if err != nil {
		return nil, fmt.Errorf("engram: FTS: %w", err)
	}
	defer rows.Close()

	var results []domain.Ranked
	for rows.Next() {
		var r domain.Ranked
		r.Source = domain.SourceFTS
		if err := rows.Scan(&r.ID, &r.Score); err != nil {
			return nil, fmt.Errorf("engram: FTS scan: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// ExecForTest executes an arbitrary SQL statement — only used in tests to
// verify that the read-only connection rejects writes.
func (r *SQLiteEngramReader) ExecForTest(sql string) error {
	_, err := r.db.Exec(sql)
	return err
}
