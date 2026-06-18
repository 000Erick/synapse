package store

// Dims is the embedding dimensionality for text-embedding-3-large.
const Dims = 3072

// schemaVecObs creates the table holding the vectors as raw BLOB. Each row is
// keyed by the Engram observation id and stores the embedding as a
// little-endian float32 sequence. A normal table (not a virtual table) keeps
// the dependency on the pure-Go modernc.org/sqlite driver free of any C
// extension.
const schemaVecObs = `CREATE TABLE IF NOT EXISTS vec_obs (
    obs_id    INTEGER PRIMARY KEY,
    embedding BLOB NOT NULL,
    dims      INTEGER NOT NULL
)`

// schemaObsMeta is a companion table used for idempotency (content hash) and
// orphan detection. It lives in the same synapse.db, separate from vec_obs so
// the hot KNN path stays lean.
const schemaObsMeta = `CREATE TABLE IF NOT EXISTS obs_meta (
    obs_id       INTEGER PRIMARY KEY,
    content_hash TEXT NOT NULL,
    model        TEXT NOT NULL,
    created_at   TEXT NOT NULL DEFAULT (datetime('now'))
)`
