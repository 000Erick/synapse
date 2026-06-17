# Synapse 🦤 (oxpecker)

> *A synapse is the connection that gives meaning to a memory.*

Standalone Go MCP server that adds **semantic search** on top of [Engram](https://github.com/Gentleman-Programming/engram) memory — without modifying Engram at all.

Engram remembers (lexical FTS5). Synapse is the **oxpecker**: a small companion that rides alongside, reads `engram.db` read-only, and gives it the semantic vision it lacks. A Spanish query like `"profesor"` can now surface an English observation about a `"teacher"` — something pure FTS5 can never do.

## How It Works

```
Agent (OpenCode / Claude Code / ...)
   │  MCP stdio
   ├──────────────► Engram (its own 20 tools, untouched)
   │                   └── engram.db (FTS5)  ◄── read-only
   └──────────────► Synapse (5 synapse_* tools)
                       └── synapse.db (sqlite-vec, 3072-dim vectors)
                            │  embeds via
                            └── OpenAI text-embedding-3-large
```

- **Sibling, not wrapper.** Synapse exposes NEW tools alongside Engram's; it never intercepts `mem_search`.
- **Two DBs, one owner each.** `engram.db` (read-only, never written) + `synapse.db` (vectors). This separation makes Synapse immune to Engram schema/version changes.
- **Hybrid search.** FTS5 (lexical) + vector KNN (semantic) fused with Reciprocal Rank Fusion (k=60).
- **Single static binary.** sqlite-vec is compiled in via CGO — no runtime `.dylib`, faithful to Engram's zero-dependency philosophy.

## Quick Start

```sh
cp .env.example .env
# Edit .env and add your OPENAI_API_KEY
make build
make test
```

## Configuration (`.env`)

| Variable | Default | Description |
|---|---|---|
| `OPENAI_API_KEY` | — | Required for backfill and semantic search |
| `OPENAI_EMBED_MODEL` | `text-embedding-3-large` | Embedding model |
| `SYNAPSE_DB_PATH` | `~/.synapse/synapse.db` | Path to Synapse vector DB |
| `ENGRAM_DB_PATH` | `~/.engram/engram.db` | Path to Engram DB (read-only) |

## MCP Tools

| Tool | Description |
|---|---|
| `synapse_health` | Liveness + DB reachability |
| `synapse_stats` | Counts: observations, vectors, orphans, dims |
| `synapse_backfill` | Idempotent batch-embed of existing observations |
| `synapse_search` | Hybrid lexical+vector search (RRF k=60) |
| `synapse_sync` | Detect and delete orphan vectors |

## Smoke Test (manual — requires real API key and engram.db)

```sh
cp .env.example .env && $EDITOR .env   # fill in OPENAI_API_KEY
make build
echo '{"method":"tools/call","params":{"name":"synapse_backfill"}}' | ./synapse
echo '{"method":"tools/call","params":{"name":"synapse_search","arguments":{"query":"profesor"}}}' | ./synapse
```

> **Note**: The smoke test above requires a real `OPENAI_API_KEY` and a populated `~/.engram/engram.db`. It is **not** part of the automated test suite.
