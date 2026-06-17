package store

// Dims is the embedding dimensionality for text-embedding-3-large.
const Dims = 3072

// schemaVecObs creates the sqlite-vec virtual table holding the vectors.
// vec0 stores a fixed-dimension float vector keyed by the Engram observation id.
const schemaVecObs = `CREATE VIRTUAL TABLE IF NOT EXISTS vec_obs USING vec0(
    obs_id INTEGER PRIMARY KEY,
    embedding FLOAT[3072]
)`

// schemaObsMeta is a companion table used for idempotency (content hash) and
// orphan detection. It lives in the same synapse.db, separate from vec0 so the
// hot KNN path stays lean.
const schemaObsMeta = `CREATE TABLE IF NOT EXISTS obs_meta (
    obs_id       INTEGER PRIMARY KEY,
    content_hash TEXT NOT NULL,
    model        TEXT NOT NULL,
    created_at   TEXT NOT NULL DEFAULT (datetime('now'))
)`
