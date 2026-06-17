# Design: Synapse Semantic Search over Engram

## Technical Approach

Standalone Go MCP server (stdio) that reads `engram.db` read-only and owns
`synapse.db` (sqlite-vec). Clean-architecture rings: **domain** (entities + RRF,
pure) → **ports** (`EngramReader`, `Embedder`, `VectorStore`) → **adapters**
(`engram`, `embed`, `store`, `mcp`) → **main** (`cmd/synapse`, composition root).
The Dependency Rule points inward: tools depend on ports, never on `database/sql`,
sqlite-vec, or `net/http`. Maps the proposal's seven capabilities; each spec
domain is one adapter or use case.

## Module Layout & Dependency Rule

```
cmd/synapse                 main: load config, open DBs, wire adapters, serve stdio
  └─ internal/config        env parsing (OPENAI_API_KEY, *_DB_PATH, model)
  └─ internal/domain        Observation, SearchResult, Ranked, RRF()   [no imports out]
  └─ internal/port          EngramReader, Embedder, VectorStore        [interfaces]
  └─ internal/usecase       Search, Backfill, Sync, Stats, Health
  └─ internal/engram        adapter: read-only sqlite reader + FTS5  -> EngramReader
  └─ internal/embed         adapter: OpenAI HTTP client + mock        -> Embedder
  └─ internal/store         adapter: sqlite-vec vec0 KNN              -> VectorStore
  └─ internal/mcp           adapter: tool registration + handlers

Import arrows ───────────────► always point inward:
mcp ─► usecase ─► port ◄─ engram | embed | store ;  all ─► domain ; domain ─► (nothing)
```

Ports (segregated, ISP):

```go
type EngramReader interface {                  // internal/engram fulfills
    LiveObservations(ctx) ([]domain.Observation, error)   // SELECT explicit cols
    LiveIDs(ctx) ([]int64, error)                          // for sync LEFT JOIN
    FTS(ctx, query string, k int) ([]domain.Ranked, error)
}
type Embedder interface {                       // internal/embed fulfills
    Embed(ctx, inputs []string) ([][]float32, error)       // batched
    Dims() int                                             // 3072
}
type VectorStore interface {                    // internal/store fulfills
    KNN(ctx, vec []float32, k int) ([]domain.Ranked, error)
    Upsert(ctx, []domain.VecRow) error                     // tx per batch
    Hashes(ctx) (map[int64]string, error)                  // idempotency
    DeleteByIDs(ctx, ids []int64) error                    // sync
    CountVectors/CountOrphans(...)
}
```

## Data Design (synapse.db)

Two tables, joined on `obs_id`. vec0 holds only vectors; companion holds metadata
for idempotency and orphan detection (information hiding: hash logic lives in one place).

```sql
CREATE VIRTUAL TABLE vec_obs USING vec0(
    obs_id INTEGER PRIMARY KEY,        -- = engram observations.id
    embedding FLOAT[3072]              -- text-embedding-3-large
);
CREATE TABLE obs_meta (
    obs_id       INTEGER PRIMARY KEY,
    content_hash TEXT NOT NULL,        -- sha256(title + "\x00" + content)
    model        TEXT NOT NULL,
    created_at   TEXT NOT NULL
);
```

**KNN + join**: `vec_obs` returns `(obs_id, distance)` ranked; usecase joins each
`obs_id` back to the live Engram observation (already fetched, mapped by id) to
hydrate title/content. `obs_meta` is never on the KNN hot path — only backfill/sync.

## synapse_search Flow (hybrid + RRF)

```
                    ┌─► EngramReader.FTS(q, K)  ──► []Ranked(source=fts)
query string ──┐    │
               ├────┤   (run in parallel: errgroup)
               │    └─► Embedder.Embed([q]) ─► vec ─► VectorStore.KNN(vec,K) ─► []Ranked(source=vec)
               │
               └─► RRF(ftsList, vecList, k=60) ─► dedupe by obs_id ─► hydrate ─► top-N
```

Query is embedded ONCE (single-element batch), then KNN. FTS and KNN run
concurrently via `errgroup`; if the embed call fails (e.g. no API key), search
degrades to FTS-only with a warning rather than erroring out.

RRF (pure `domain` function, fully testable):

```
func RRF(lists [][]Ranked, k int) []Fused:
    score := map[obsID]float64{}
    seen  := map[obsID]Source{}
    for _, list := range lists:
        for rank, item := range list:      # rank is 0-based
            score[item.ID] += 1.0 / float64(k + rank + 1)
            seen[item.ID] |= item.Source   # fts|vec -> both if in both
    sort by score desc, stable; emit Fused{ID, Score, Source}
```

`Source` is a bitmask: `fts=1`, `vec=2`, `both=3`. Dedupe is implicit in the map.

## Idempotent Backfill

```
hashes := store.Hashes()                       # obs_id -> content_hash
todo := []obs where hash(title+content) != hashes[id]   # new OR changed
for batch in chunk(todo, 256):                 # ~256 <= 2048 OpenAI limit
    vecs := embedder.Embed(batch.texts)        # 1 HTTP call
    BEGIN; store.Upsert(rows{obs_id,vec,hash,model,now}); COMMIT   # tx per batch
```

Resumable: each committed batch persists hash, so a crash mid-run resumes from
the next unhashed/changed obs. Re-run with no changes = 0 work, 0 dupes
(PRIMARY KEY + hash compare). Edited observation → hash differs → re-embedded.

## Orphan Sync

```
liveIDs := engram.LiveIDs()                    # deleted_at IS NULL
vecIDs  := store.AllObsIDs()
orphans := vecIDs \ liveIDs                     # set difference (LEFT JOIN equiv)
BEGIN; store.DeleteByIDs(orphans) from vec_obs AND obs_meta; COMMIT
```

Covers both hard-deleted rows and soft-deletes (`deleted_at` set). Live vectors
never touched. `synapse_stats` exposes `orphans` so drift is observable before sync.

## Error Handling & Config

| Condition | Behavior |
|-----------|----------|
| Missing `OPENAI_API_KEY` | `health` = OK (DBs reachable); `backfill` + embed-path of `search` fail with explicit "OPENAI_API_KEY not set"; search falls back to FTS-only |
| OpenAI 429 / 5xx | Exponential backoff (e.g. 1s,2s,4s, max 5 tries), honor `Retry-After`; surface final error with status code |
| OpenAI 4xx (bad request) | Fail fast, no retry; clear message |
| `engram.db` locked | Shouldn't occur: opened `file:<path>?mode=ro` + WAL = safe concurrent reads. If it does, return wrapped error, no retry |
| Dim mismatch (vec ≠ 3072) | Reject at Upsert; validate `Embedder.Dims()` at startup |

Config from env only (`.env` git-ignored): `OPENAI_API_KEY`,
`OPENAI_EMBED_MODEL` (default `text-embedding-3-large`), `SYNAPSE_DB_PATH`,
`ENGRAM_DB_PATH`.

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `go.mod` | Create | module `github.com/ediazs/synapse`; deps: go-sdk, sqlite-vec-go-bindings/cgo, mattn/go-sqlite3 |
| `cmd/synapse/main.go` | Create | composition root, stdio serve |
| `internal/config/config.go` | Create | env parsing |
| `internal/domain/*.go` | Create | Observation, SearchResult, Ranked, RRF |
| `internal/port/ports.go` | Create | three port interfaces |
| `internal/usecase/*.go` | Create | Search, Backfill, Sync, Stats, Health |
| `internal/engram/reader.go` | Create | RO sqlite + FTS5, explicit SELECT |
| `internal/embed/openai.go`, `mock.go` | Create | HTTP client + test mock |
| `internal/store/vec.go` | Create | sqlite-vec vec0 + obs_meta |
| `internal/mcp/tools.go` | Create | register 5 tools |

## Testing Strategy (strict TDD — RED→GREEN→REFACTOR)

| Layer | What | Approach |
|-------|------|----------|
| Unit | `RRF` fusion, dedupe, source bitmask, hash | Pure table-driven; no I/O |
| Unit | Backfill/Sync use cases | Mock `Embedder` + temp-DB `VectorStore`; assert 0-dupe re-run, orphan delete |
| Integration | `VectorStore` KNN round-trip | `sqlite_vec.Auto()` + temp `t.TempDir()` DB; insert→KNN→assert order |
| Integration | OpenAI client | `httptest.Server`: success, 429+Retry-After, 4xx, batching |
| Integration | `EngramReader` | seed a throwaway temp sqlite with the Engram schema — **NEVER touch real engram.db** |

`Embedder` always mocked in use-case tests (no network, no cost). Both DBs use
`t.TempDir()`. `go test ./...` GREEN ends every slice.

## Build / Run

```
PATH includes /opt/homebrew/bin
CGO_ENABLED=1 go build -o synapse ./cmd/synapse   # static link, no runtime dylib
env via .env (git-ignored); sqlite_vec.Auto() registers the extension
```

## Slice → Files Map (one PR, 6 sequential commits)

| Slice | Creates | Gate |
|-------|---------|------|
| 1 scaffold-health | go.mod, cmd/synapse, config, domain, port, usecase(Health/Stats), engram reader, Noop store/embedder | `go test ./...` GREEN, no OpenAI/vec |
| 2 vector-store | internal/store (vec0+obs_meta), VectorStore impl | KNN round-trip in-memory |
| 3 embeddings | internal/embed (openai.go + mock.go), Embedder port | httptest mocked tests GREEN |
| 4 backfill | usecase/backfill, synapse_backfill tool | re-run = 0 dupes |
| 5 hybrid-search | usecase/search, RRF wiring, synapse_search tool | "profesor" surfaces "teacher" |
| 6 sync | usecase/sync, synapse_sync tool | orphans pruned, live kept |

## Open Questions

- [ ] OpenAI batch size: brief says up to ~2048; design picks 256 for tx granularity/retry safety — confirm acceptable.
- [ ] `internal/usecase` vs folding tool logic into `internal/mcp`: design keeps use cases separate (testable without MCP); confirm this extra ring is wanted.
