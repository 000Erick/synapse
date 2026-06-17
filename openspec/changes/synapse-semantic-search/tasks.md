# Tasks: Synapse Semantic Search over Engram

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1 200–1 600 lines (new repo from zero) |
| 400-line budget risk | High |
| Chained PRs recommended | No (user chose single-PR) |
| Suggested split | Single PR, 7 sequential commits (one per slice) |
| Delivery strategy | single-pr |
| Chain strategy | size:exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

> **size:exception already accepted** — single PR, 7 work-unit commits, one per slice.
> Each commit must leave `go test ./...` GREEN before the next begins.

### Suggested Work Units (commits, not PRs)

| Commit | Theme | Gate |
|--------|-------|------|
| 0 | setup + config.yaml reconcile | `go build ./...` |
| 1 | scaffold-health | `go test ./...` GREEN (no OpenAI/sqlite-vec) |
| 2 | vector-store | KNN round-trip temp-DB test |
| 3 | embeddings | httptest mocked tests GREEN |
| 4 | backfill | re-run = 0 duplicates test |
| 5 | hybrid-search | RRF unit + cross-language integration test |
| 6 | sync | orphan-pruned test |

---

## Slice 0 — Setup (commit 0: `chore(setup): init module, config, Makefile, README`)

- [ ] **0.0** DECISION RECORD: tool names are `synapse_*` (proposal is source of truth; config.yaml listed `semantic_*`). Update `openspec/config.yaml` description line to use `synapse_health`, `synapse_stats`, `synapse_backfill`, `synapse_search`, `synapse_sync`.
- [ ] **0.1** Create `go.mod` — module `github.com/ediazs/synapse`, go 1.22+ (check local `go version`); add deps: `modelcontextprotocol/go-sdk`, `mattn/go-sqlite3`, `nlpodyssey/sqlite-vec-go-bindings/cgo`.
- [ ] **0.2** Create `go.sum` — run `go mod tidy` with `CGO_ENABLED=1`.
- [ ] **0.3** Create `Makefile` — targets: `build` (`CGO_ENABLED=1 go build -o synapse ./cmd/synapse`), `test` (`CGO_ENABLED=1 go test ./...`), `run` (`CGO_ENABLED=1 go run ./cmd/synapse`).
- [ ] **0.4** Create `.env.example` listing `OPENAI_API_KEY=`, `OPENAI_EMBED_MODEL=`, `SYNAPSE_DB_PATH=`, `ENGRAM_DB_PATH=`.
- [ ] **0.5** Create `README.md` skeleton — project name "Synapse (oxpecker)", one-line description, `make build` / `make test` quickstart, .env setup note.
- [ ] **0.6** Verify `.gitignore` already exists; add `.env` and `synapse` binary entries if missing.
- **GATE**: `CGO_ENABLED=1 go build ./...` exits 0 (even if only go.mod exists).

---

## Slice 1 — Scaffold + Health (commit 1: `feat(scaffold): domain, ports, noop adapters, health+stats tools`)

- [ ] **1.1** Create `internal/domain/observation.go` — structs `Observation{ID, Title, Content, CreatedAt, DeletedAt}` and `Ranked{ID, Score, Source}`, `Source` bitmask constants (`SourceFTS=1`, `SourceVec=2`), `SearchResult{Ranked, Observation, Snippet}`.
- [ ] **1.2** Create `internal/domain/rrf.go` — pure `RRF(lists [][]Ranked, k int) []Ranked`; map accumulate, stable sort by score desc.
- [ ] **1.3** Create `internal/domain/rrf_test.go` — table-driven: two lists → correct fused order per spec scenarios; pure, no I/O.
- [ ] **1.4** Create `internal/port/ports.go` — interfaces `EngramReader`, `Embedder`, `VectorStore` exactly as designed (with `CountVectors`, `CountOrphans`, `AllObsIDs`).
- [ ] **1.5** Create `internal/engram/reader.go` — `SQLiteEngramReader` using `mattn/go-sqlite3` (no sqlite-vec); opens `file:<path>?mode=ro&_journal=WAL`; explicit `SELECT id, title, content, created_at, deleted_at FROM observations WHERE deleted_at IS NULL`; implements `LiveObservations`, `LiveIDs`, `FTS`.
- [ ] **1.6** Create `internal/engram/reader_test.go` — seed a `t.TempDir()` SQLite with Engram schema (`observations` + `observations_fts` FTS5 shadow); assert `LiveObservations` returns only non-deleted rows; assert `FTS` returns token matches; never touches `~/.engram/engram.db`.
- [ ] **1.7** Create `internal/embed/noop.go` — `NoopEmbedder` implementing `Embedder`; `Embed` returns zero-vectors of len 3072; `Dims` returns 3072.
- [ ] **1.8** Create `internal/store/noop.go` — `NoopVectorStore` implementing `VectorStore`; all methods return zero values / nil error (stub).
- [ ] **1.9** Create `internal/usecase/health.go` — `HealthUsecase.Run(ctx)`: pings both DBs (simple `SELECT 1`); returns `{engram: ok/err, synapse: ok/err}`.
- [ ] **1.10** Create `internal/usecase/stats.go` — `StatsUsecase.Run(ctx)`: calls `EngramReader.LiveObservations` (count), `VectorStore.CountVectors`, `VectorStore.CountOrphans`, `Embedder.Dims`; returns flat struct.
- [ ] **1.11** Create `internal/config/config.go` — reads env: `OPENAI_API_KEY`, `OPENAI_EMBED_MODEL` (default `text-embedding-3-large`), `SYNAPSE_DB_PATH` (default `~/.synapse/synapse.db`), `ENGRAM_DB_PATH` (default `~/.engram/engram.db`); expands `~` with `os.UserHomeDir`.
- [ ] **1.12** Create `internal/mcp/tools.go` — registers all five tools (`synapse_health`, `synapse_stats`, `synapse_backfill`, `synapse_search`, `synapse_sync`); backfill/search/sync handlers return `"OPENAI_API_KEY is not set"` stub for now.
- [ ] **1.13** Create `cmd/synapse/main.go` — composition root: load config → open DBs → wire adapters (use Noop store+embedder) → register MCP tools → `server.ServeStdio()`.
- [ ] **1.14** Create `internal/usecase/health_test.go` and `stats_test.go` — use temp DBs + NoopEmbedder + NoopStore; assert return shapes; no network.
- **GATE**: `CGO_ENABLED=1 go test ./...` GREEN. No OpenAI or sqlite-vec dependency yet.

---

## Slice 2 — Vector Store (commit 2: `feat(store): sqlite-vec vec0 schema, VectorStore impl, KNN round-trip`)

- [ ] **2.1** Create `internal/store/schema.go` — `CREATE VIRTUAL TABLE IF NOT EXISTS vec_obs USING vec0(obs_id INTEGER PRIMARY KEY, embedding FLOAT[3072])` + `CREATE TABLE IF NOT EXISTS obs_meta (obs_id INTEGER PRIMARY KEY, content_hash TEXT NOT NULL, model TEXT NOT NULL, created_at TEXT NOT NULL)`; apply via `sqlite_vec.Auto()`.
- [ ] **2.2** Create `internal/store/vec.go` — `SQLiteVectorStore` implementing `VectorStore`: `Upsert` (tx per batch, INSERT OR REPLACE into both tables), `KNN` (`SELECT obs_id, distance FROM vec_obs WHERE embedding MATCH ? ORDER BY distance LIMIT ?`), `Hashes` (SELECT all from obs_meta), `DeleteByIDs` (DELETE from both in tx), `CountVectors`, `CountOrphans` (LEFT JOIN with `EngramReader.LiveIDs`), `AllObsIDs`.
- [ ] **2.3** Update `cmd/synapse/main.go` — swap `NoopVectorStore` for `SQLiteVectorStore`; run `sqlite_vec.Auto()` before opening synapse.db.
- [ ] **2.4** Create `internal/store/vec_test.go` — `t.TempDir()` DB; `Upsert` one vec → `KNN` → assert obs_id returned; re-Upsert same → `Hashes` shows same hash (no dupe); `DeleteByIDs` → count drops; all under `CGO_ENABLED=1`.
- **GATE**: `CGO_ENABLED=1 go test ./...` GREEN; KNN round-trip passes in temp DB.

---

## Slice 3 — Embeddings (commit 3: `feat(embed): OpenAI HTTP embedder, batch, retry, mock`)

- [ ] **3.1** Create `internal/embed/openai.go` — `OpenAIEmbedder`: `POST https://api.openai.com/v1/embeddings` with `input[]`, `model`; parse response `data[].embedding`; exponential backoff on 429 (respect `Retry-After`, max 5 tries, 1s/2s/4s/8s/16s); fail-fast on 4xx other than 429; return `ErrNoAPIKey` sentinel when key empty.
- [ ] **3.2** Create `internal/embed/mock.go` — `MockEmbedder`: configurable via func field `EmbedFn func([]string)([][]float32, error)`; default returns deterministic zero-vectors; records calls for assertions.
- [ ] **3.3** Create `internal/embed/openai_test.go` — `httptest.NewServer` stubs: (a) success returns 3072-dim vectors; (b) 429 with `Retry-After:1` → retried → succeeds; (c) 4xx non-429 → fails fast; (d) batch of 250 split into `ceil(250/100)` calls; table-driven with `testing.Short()` skip guard for retry tests.
- [ ] **3.4** Update `cmd/synapse/main.go` — swap `NoopEmbedder` for `OpenAIEmbedder` (returns `ErrNoAPIKey` gracefully at call time; not a startup failure).
- **GATE**: `CGO_ENABLED=1 go test ./...` GREEN; all mocked embedder tests pass; no real API calls.

---

## Slice 4 — Backfill (commit 4: `feat(backfill): idempotent batch-embed usecase + synapse_backfill tool`)

- [ ] **4.1** Create `internal/usecase/backfill.go` — `BackfillUsecase.Run(ctx)`: check API key (return `ErrNoAPIKey` early); `engram.LiveObservations()` → `store.Hashes()` → filter todo (hash mismatch or missing); `chunk(todo, 100)` → loop: `embedder.Embed(texts)` → `store.Upsert(batch)` per tx; accumulate embedded/skipped/failed; return `BackfillResult{Embedded, Skipped, Failed}`.
- [ ] **4.2** Wire `synapse_backfill` tool handler in `internal/mcp/tools.go` to call `BackfillUsecase.Run`; return JSON result.
- [ ] **4.3** Create `internal/usecase/backfill_test.go` — seed temp engram DB (10 obs); use `MockEmbedder` + `SQLiteVectorStore` (temp dir): (a) first run → 10 embedded, 0 skipped; (b) re-run → 0 embedded, 10 skipped; (c) partial (7 already in store) → 3 embedded, 7 skipped; (d) second-batch error → 20 embedded + 10 failed, no panic; (e) missing API key → immediate error, 0 reads.
- [ ] **4.4** Content hash helper: `sha256(title + "\x00" + content)` — inline in backfill or `internal/domain/hash.go`; unit test for determinism.
- **GATE**: `CGO_ENABLED=1 go test ./...` GREEN; re-run = 0 duplicates test passes.

---

## Slice 5 — Hybrid Search (commit 5: `feat(search): hybrid FTS5+KNN, RRF k=60, synapse_search tool`)

- [ ] **5.1** Create `internal/usecase/search.go` — `SearchUsecase.Run(ctx, query, limit)`: validate non-empty query; `errgroup` fanout: (a) `engram.FTS(ctx, query, limit*2)`, (b) `embedder.Embed([query])` → `store.KNN(vec, limit*2)` — if embed fails set `vectorUnavailable=true` and use empty KNN list; `domain.RRF([fts, knn], 60)` → top-N; hydrate each `obs_id` from a map of live obs; return `[]SearchResult` + `VectorUnavailable bool`.
- [ ] **5.2** Wire `synapse_search` tool handler in `internal/mcp/tools.go`: parse `query` + `limit` (default 10, max 50); call `SearchUsecase.Run`; serialize results including `source` field.
- [ ] **5.3** Create `internal/usecase/search_test.go`:
  - (a) Empty query → error `"query must not be empty"`.
  - (b) RRF fusion: mock FTS=[A,B,C], mock KNN=[B,A,D] → verify order and scores per spec.
  - (c) Embed failure → FTS-only results + `vector_unavailable:true`.
  - (d) Cross-language: seed temp engram with obs title="teacher"; `MockEmbedder` returns vec[0.9,0,…] for "profesor" and identical vec for "teacher" embed; KNN returns "teacher" obs → appears in top 5; source = "vec" or "both".
  - (e) Default limit=10, custom limit=5 honored.
  - All tests: `MockEmbedder` + temp DBs; no real API key.
- **GATE**: `CGO_ENABLED=1 go test ./...` GREEN; cross-language scenario (d) passes.

---

## Slice 6 — Sync (commit 6: `feat(sync): orphan-vector cleanup usecase + synapse_sync tool`)

- [ ] **6.1** Create `internal/usecase/sync.go` — `SyncUsecase.Run(ctx)`: `engram.LiveIDs()` → `store.AllObsIDs()` → set-difference orphans; if any: `store.DeleteByIDs(orphans)` in tx; return `SyncResult{Orphans, Kept}`.
- [ ] **6.2** Wire `synapse_sync` tool handler in `internal/mcp/tools.go`.
- [ ] **6.3** Create `internal/usecase/sync_test.go` — temp engram (3 live obs ids 1,2,3) + temp synapse (vectors for ids 1,2,3,4,5); `SyncUsecase.Run` → orphans=[4,5] deleted; live [1,2,3] kept; second run → 0 orphans.
- **GATE**: `CGO_ENABLED=1 go test ./...` GREEN; orphan-pruned test + live-kept assertion pass.

---

## Slice 7 — Wire .env + Final Polish (commit 7: `chore(config): godotenv local run, smoke test instructions`)

- [ ] **7.1** Add `github.com/joho/godotenv` to `go.mod`; in `cmd/synapse/main.go` call `godotenv.Load()` (ignore `os.IsNotExist` error — optional file).
- [ ] **7.2** Confirm `go test ./...` still GREEN (godotenv must not affect test isolation — tests never read `.env`).
- [ ] **7.3** Update `README.md` with smoke test instructions (manual/optional — not part of `go test`):
  ```
  # Smoke test (requires real OPENAI_API_KEY in .env and real ~/.engram/engram.db)
  cp .env.example .env && $EDITOR .env   # fill in OPENAI_API_KEY
  make build
  echo '{"method":"tools/call","params":{"name":"synapse_backfill"}}' | ./synapse
  echo '{"method":"tools/call","params":{"name":"synapse_search","arguments":{"query":"profesor"}}}' | ./synapse
  ```
  Mark clearly as **manual / optional** — not gated in CI.
- **GATE**: `CGO_ENABLED=1 go test ./...` GREEN (no change in test behavior).

---

## Dependency Order Summary

```
0 (setup) → 1 (scaffold+health) → 2 (vector-store) → 3 (embeddings)
                                                              ↓
                                                    4 (backfill) → 5 (hybrid-search)
                                                                            ↓
                                                                    6 (sync) → 7 (polish)
```

Slices 4, 5, 6 can be re-ordered independently once slices 1–3 are green, but sequential order maximizes incremental testability.
