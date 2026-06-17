# Synapse (oxpecker)

Standalone MCP server that adds **semantic search** on top of [Engram](https://github.com/ediazs/engram) memory. Reads `engram.db` read-only and stores embedding vectors in its own `synapse.db`.

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
