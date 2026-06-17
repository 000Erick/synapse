# Proposal: Synapse Semantic Search over Engram

## Intent

Engram search is FTS5 **lexical-only**: it matches tokens, not meaning. A query
for `"profesor"` never finds an observation that says `"teacher"`, and
`"circuit breaker"` misses notes about `"resilience pattern"`. As the memory
store grows (~1040 observations), this cross-language and synonym gap makes
recall unreliable.

Synapse (codename *oxpecker*) adds **semantic search** as a standalone sibling
MCP server that reads Engram read-only and stores its own vectors. Engram is
never modified, so it stays immune to Synapse's schema and version drift.

## Scope

### In Scope
- Standalone Go MCP server (stdio) with clean-architecture layout.
- Read-only Engram reader (explicit column SELECT, live = `deleted_at IS NULL`).
- Vector store on `sqlite-vec` (vec0, 3072 dims) in its own DB.
- OpenAI `text-embedding-3-large` embedder (HTTP, batched, mockable).
- Tools: `synapse_health`, `synapse_stats`, `synapse_backfill`, `synapse_search`, `synapse_sync`.
- Hybrid retrieval: Engram FTS5 + vector KNN fused via RRF (k=60).
- Idempotent backfill and orphan-vector cleanup.

### Out of Scope
- **Modifying Engram** — read-only contract, no writes to `engram.db`.
- **Replacing `mem_search`** — Synapse augments, it does not replace Engram tools.
- **Cloud service / API** — local single binary over stdio only.
- Re-ranking models, multi-provider embeddings, UI, auth.

## Capabilities

### New Capabilities
- `mcp-server`: stdio MCP server lifecycle, tool registration, config from env.
- `engram-reader`: read-only access to Engram observations + FTS5 search.
- `vector-store`: sqlite-vec persistence and KNN over 3072-dim embeddings.
- `embeddings`: OpenAI embedder interface with batching and a mock.
- `backfill`: one-time idempotent embedding of existing observations.
- `hybrid-search`: RRF fusion of lexical + vector results.
- `vector-sync`: orphan-vector detection and removal.

### Modified Capabilities
- None (first change; `openspec/specs/` is empty).

## Approach

Sibling-MCP architecture. Two SQLite databases, one owner each:

| DB | Path | Access | Owner |
|----|------|--------|-------|
| `engram.db` | `~/.engram/engram.db` | `file:...?mode=ro` (WAL, safe concurrent reads) | Engram |
| `synapse.db` | `~/.synapse/synapse.db` | read-write (`vec0`, ~12MB/1040 vecs) | Synapse |

Clean-architecture rings: **domain** (Observation, SearchResult, RRF) → **ports**
(`EngramReader`, `VectorStore`, `Embedder`) → **adapters** (sqlite, OpenAI HTTP,
MCP) → **main** (composition root). The Dependency Rule points inward; tools
depend on ports, never on concrete drivers.

Driver: `sqlite-vec-go-bindings/cgo` + `mattn/go-sqlite3`, static link,
`CGO_ENABLED=1`, no runtime dylib. MCP via official
`modelcontextprotocol/go-sdk`. Module path: `github.com/ediazs/synapse`.

### Slice / Commit Plan (one PR, sequential commits)

| Slice | Commit theme | Deliverable | Gate |
|-------|--------------|-------------|------|
| 1 | scaffold-health | go.mod, clean-arch layout, stdio server, `synapse_health`+`synapse_stats`, read-only `EngramReader`, Noop stores | `go test ./...` GREEN; no OpenAI/sqlite-vec |
| 2 | vector-store | sqlite-vec (CGO+mattn), `VectorStore` impl, `synapse.db` vec0 schema (3072 dims) | KNN round-trips in-memory test |
| 3 | embeddings | OpenAI `text-embedding-3-large` HTTP client, `Embedder` port, mock, batching | Mocked unit tests GREEN |
| 4 | backfill | `synapse_backfill`: read ~1040 obs read-only, embed in ~11 batches, store; idempotent | Re-run inserts 0 duplicates |
| 5 | hybrid-search | `synapse_search`: FTS5 + vector KNN fused via RRF k=60; cross-language | `"profesor"` surfaces `"teacher"` |
| 6 | sync | `synapse_sync`: remove orphan vectors (obs missing or `deleted_at`) | Orphans pruned, live kept |

### MCP Tools

| Tool | Purpose |
|------|---------|
| `synapse_health` | Liveness + DB reachability (both DBs) |
| `synapse_stats` | Counts: observations, vectors, orphans, dims |
| `synapse_backfill` | Idempotent batch-embed of existing observations |
| `synapse_search` | Hybrid lexical+vector search, RRF-fused results |
| `synapse_sync` | Detect and delete orphan vectors |

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/synapse/` | New | Composition root, stdio entrypoint |
| `internal/domain/` | New | Entities, RRF, search results |
| `internal/ports/` | New | `EngramReader`, `VectorStore`, `Embedder` |
| `internal/adapter/engram/` | New | Read-only sqlite reader + FTS5 |
| `internal/adapter/vector/` | New | sqlite-vec store |
| `internal/adapter/openai/` | New | Embedder HTTP client |
| `internal/adapter/mcp/` | New | Tool registration + handlers |
| `~/.engram/engram.db` | Read-only | Never written |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| CGO static link fails on darwin/arm64 | Med | Pin `sqlite-vec-go-bindings/cgo`+`mattn`; CI gate on `go test ./...` with CGO_ENABLED=1 |
| Engram schema drift breaks reader | Med | Explicit column SELECT (never `SELECT *`); isolated vector DB |
| OpenAI cost/latency on backfill | Low | Batched (~11 calls, ~$0.08); `Embedder` mocked in tests |
| Concurrent write to engram.db | Low | Open `mode=ro`; WAL makes reads safe |
| Orphan accumulation after deletes | Med | `synapse_sync` mandatory; `synapse_stats` exposes orphan count |
| 3072-dim mismatch vs model | Low | Schema dims pinned to model; validate on insert |

## Rollback Plan

Synapse is additive and isolated. To roll back: stop the server and delete
`~/.synapse/synapse.db`. Engram is untouched (read-only), so nothing in the
memory store needs reverting. Per-slice rollback = revert that commit; earlier
slices stay GREEN because each ends at a passing `go test ./...`.

## Dependencies

- OpenAI API key (`OPENAI_API_KEY`).
- Existing `~/.engram/engram.db` with `observations` + `observations_fts`.
- `CGO_ENABLED=1` toolchain (darwin/arm64).

## Success Criteria

- [ ] `go test ./...` GREEN at the end of every slice (strict TDD).
- [ ] Engram DB opened read-only; zero writes to `engram.db`.
- [ ] `synapse_backfill` embeds ~1040 obs idempotently (re-run = 0 dupes).
- [ ] `synapse_search "profesor"` surfaces English `"teacher"` observations.
- [ ] RRF (k=60) fuses FTS5 + KNN into one ranked list.
- [ ] `synapse_sync` removes orphan vectors; live vectors preserved.
- [ ] All five MCP tools registered and reachable over stdio.
